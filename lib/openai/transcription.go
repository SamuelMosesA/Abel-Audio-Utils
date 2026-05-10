package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"behringerRecorder/lib/config"
	"behringerRecorder/lib/state"

	"github.com/gorilla/websocket"
)

type TranscriptionManager struct {
	Config             *config.Config
	APIKey             string
	Model              string
	OriginalLanguage   string
	AudioInBufferSize  int
	AudioOutBufferSize int
	SubtitleBufferSize int
	Enabled            atomic.Bool

	Sessions      sync.Map // map[string]*RealtimeSession
	Subscribers   sync.Map // map[string][]chan string
	LastRestart   sync.Map // map[string]time.Time
	Mu            sync.Mutex
	OnStateChange func()
}

func NewTranscriptionManager(cfg *config.Config, apiKey, model, originalLang string, audioInSize, audioOutSize, subtitleSize int) (*TranscriptionManager, error) {
	return &TranscriptionManager{
		Config:             cfg,
		APIKey:             apiKey,
		Model:              model,
		OriginalLanguage:   originalLang,
		AudioInBufferSize:  audioInSize,
		AudioOutBufferSize: audioOutSize,
		SubtitleBufferSize: subtitleSize,
	}, nil
}

func (m *TranscriptionManager) SetEnabled(enabled bool) {
	m.Enabled.Store(enabled)
}

func (m *TranscriptionManager) SetOnStateChange(fn func()) {
	m.OnStateChange = fn
}

func (m *TranscriptionManager) CloseAll() {
	m.Sessions.Range(func(key, value interface{}) bool {
		s := value.(*RealtimeSession)
		s.cancel()
		return true
	})
}

func (m *TranscriptionManager) ListSessions() []state.SessionInfo {
	var list []state.SessionInfo
	m.Sessions.Range(func(key, value interface{}) bool {
		lang := key.(string)
		list = append(list, state.SessionInfo{
			Language: lang,
		})
		return true
	})
	return list
}

func (m *TranscriptionManager) StopSession(language string, subtitles bool) {
	if val, ok := m.Sessions.Load(language); ok {
		val.(*RealtimeSession).cancel()
	}
}

func (m *TranscriptionManager) GetSubtitles(language string) (chan string, func()) {
	if language == "" {
		language = m.OriginalLanguage
	}
	subCh := make(chan string, m.SubtitleBufferSize)
	m.Mu.Lock()
	var subs []chan string
	if val, ok := m.Subscribers.Load(language); ok {
		subs = val.([]chan string)
	}
	subs = append(subs, subCh)
	m.Subscribers.Store(language, subs)
	m.Mu.Unlock()

	cleanup := func() {
		m.Mu.Lock()
		defer m.Mu.Unlock()
		if val, ok := m.Subscribers.Load(language); ok {
			oldSubs := val.([]chan string)
			newSubs := make([]chan string, 0, len(oldSubs))
			for _, ch := range oldSubs {
				if ch != subCh {
					newSubs = append(newSubs, ch)
				}
			}
			if len(newSubs) == 0 {
				m.Subscribers.Delete(language)
			} else {
				m.Subscribers.Store(language, newSubs)
			}
		}
		close(subCh)
	}
	return subCh, cleanup
}

func (m *TranscriptionManager) GetChannel(language string) chan []float32 {
	if language == "" {
		language = m.OriginalLanguage
	}
	if val, ok := m.Sessions.Load(language); ok {
		return val.(*RealtimeSession).AudioOut
	}

	m.Mu.Lock()
	defer m.Mu.Unlock()
	if val, ok := m.Sessions.Load(language); ok {
		return val.(*RealtimeSession).AudioOut
	}

	audioOut := make(chan []float32, m.AudioOutBufferSize)
	audioIn := make(chan []byte, m.AudioInBufferSize)
	ctx, cancel := context.WithCancel(context.Background())
	session := &RealtimeSession{
		Language: language,
		AudioIn:  audioIn,
		AudioOut: audioOut,
		ctx:      ctx,
		cancel:   cancel,
	}
	m.Sessions.Store(language, session)
	go m.runSession(session)
	return audioOut
}

func (m *TranscriptionManager) PushAudio(chunk []float32) {
	if !m.Enabled.Load() {
		return
	}
	// Downsample 48kHz to 24kHz
	pcm := m.downsample(chunk)
	if len(pcm) == 0 {
		return
	}

	// Auto-start if subscribers exist
	m.Subscribers.Range(func(key, value interface{}) bool {
		lang := key.(string)
		if _, ok := m.Sessions.Load(lang); !ok {
			now := time.Now()
			if last, ok := m.LastRestart.Load(lang); ok {
				if now.Sub(last.(time.Time)) < 5*time.Second {
					return true
				}
			}
			m.LastRestart.Store(lang, now)
			log.Printf("[OPENAI] Auto-starting transcription for %s", lang)
			m.GetChannel(lang)
		}
		return true
	})

	m.Sessions.Range(func(key, value interface{}) bool {
		s := value.(*RealtimeSession)
		
		// Push raw audio directly to output for low-latency bypass
		select {
		case s.AudioOut <- chunk:
		default:
		}

		// Push downsampled audio to AI for transcription
		select {
		case s.AudioIn <- pcm:
		default:
		}
		return true
	})
}

func (m *TranscriptionManager) downsample(chunk []float32) []byte {
	downsampled := make([]int16, 0, len(chunk)/4)
	for i := 0; i < len(chunk)-3; i += 4 {
		avg := (chunk[i] + chunk[i+1] + chunk[i+2] + chunk[i+3]) / 4.0
		if avg > 1.0 {
			avg = 1.0
		} else if avg < -1.0 {
			avg = -1.0
		}
		downsampled = append(downsampled, int16(avg*32767))
	}
	bytes := make([]byte, len(downsampled)*2)
	for i, v := range downsampled {
		bytes[i*2] = byte(v & 0xff)
		bytes[i*2+1] = byte(v >> 8)
	}
	return bytes
}

func (m *TranscriptionManager) runSession(s *RealtimeSession) {
	log.Printf("[OPENAI] Starting transcription session for %s", s.Language)
	defer m.Sessions.Delete(s.Language)
	defer close(s.AudioOut)

	url := "wss://api.openai.com/v1/realtime?intent=transcription"
	header := http.Header{}
	header.Add("Authorization", "Bearer "+m.APIKey)
	header.Add("OpenAI-Safety-Identifier", "behringer-recorder-v1")

	conn, resp, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		status := 0
		body := ""
		if resp != nil {
			status = resp.StatusCode
			defer resp.Body.Close()
			buf := make([]byte, 1024)
			n, _ := resp.Body.Read(buf)
			body = string(buf[:n])
		}
		log.Printf("[OPENAI] Transcription dial error (Status %d): %v (Body: %s)", status, err, body)
		return
	}
	defer conn.Close()

	config := SessionUpdateEvent{
		Type: "session.update",
		Session: SessionConfig{
			Type:       "transcription",
			Audio: &AudioConfig{
				Input: &InputConfig{
					Format: &InputFormat{Type: "audio/pcm", Rate: 24000},
					Transcription: &TranscriptionConfig{
						Model:    m.Model,
						Language: m.OriginalLanguage,
					},
				},
			},
		},
	}
	if err := conn.WriteJSON(config); err != nil {
		return
	}

	// Receive
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var raw map[string]interface{}
			if err := json.Unmarshal(message, &raw); err != nil {
				continue
			}

			if raw["type"] == "conversation.item.input_audio_transcription.delta" ||
				raw["type"] == "conversation.item.input_audio_transcription.completed" {
				delta, _ := raw["delta"].(string)
				if delta == "" {
					// For 'completed', the text is in 'transcript' field
					delta, _ = raw["transcript"].(string)
				}
				if delta != "" {
					m.broadcastSubtitle(s.Language, delta)
				}
			} else if raw["type"] == "error" {
				log.Printf("[OPENAI] Transcription error: %v", raw["error"])
				return
			}
		}
	}()

	// Send
	for {
		select {
		case pcm := <-s.AudioIn:
			event := map[string]interface{}{
				"type":  "input_audio_buffer.append",
				"audio": base64.StdEncoding.EncodeToString(pcm),
			}
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (m *TranscriptionManager) broadcastSubtitle(language, text string) {
	if val, ok := m.Subscribers.Load(language); ok {
		subs := val.([]chan string)
		payload, _ := json.Marshal(map[string]interface{}{"text": text, "tokens": 0})
		for _, ch := range subs {
			select {
			case ch <- string(payload):
			default:
			}
		}
	}
}
