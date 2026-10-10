package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"abel/src/backend/lib/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/gorilla/websocket"
	sync_map "github.com/zolstein/sync-map"
)

type TranscriptionManager struct {
	Config             *config.Config
	AppState           *state.AppState
	APIKey             string
	Model              string
	OriginalLanguage   string
	AudioInBufferSize  int
	AudioOutBufferSize int
	SubtitleBufferSize int
	Enabled            atomic.Bool

	Sessions      sync_map.Map[string, *RealtimeSession]
	Subscribers   sync_map.Map[string, []chan string]
	LastRestart   sync_map.Map[string, time.Time]
	Mu            sync.Mutex
	OnStateChange func()
}

// NewTranscriptionManager creates a live speech-to-text session manager.
func NewTranscriptionManager(cfg *config.Config, appState *state.AppState, apiKey, model, originalLang string, audioInSize, audioOutSize, subtitleSize int) (*TranscriptionManager, error) {
	return &TranscriptionManager{
		Config:             cfg,
		AppState:           appState,
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
	m.Sessions.Range(func(_ string, session *RealtimeSession) bool {
		session.cancel()
		return true
	})
}

func (m *TranscriptionManager) ListSessions() []state.SessionInfo {
	var list []state.SessionInfo
	m.Sessions.Range(func(lang string, _ *RealtimeSession) bool {
		list = append(list, state.SessionInfo{
			Language:  lang,
			Listeners: m.GetListenerCount(lang),
			Subtitles: true,
		})
		return true
	})
	return list
}

func (m *TranscriptionManager) GetListenerCount(language string) int {
	if language == "" {
		language = m.OriginalLanguage
	}
	code := m.Config.ResolveLanguageCode(language)
	name := m.Config.ResolveLanguageName(language)

	m.Mu.Lock()
	defer m.Mu.Unlock()

	count := 0
	if subs, ok := m.Subscribers.Load(code); ok {
		count += len(subs)
	}
	if strings.ToLower(code) != strings.ToLower(name) {
		if subs, ok := m.Subscribers.Load(name); ok {
			count += len(subs)
		}
	}
	return count
}

func (m *TranscriptionManager) StopSession(language string, subtitles bool) {
	if session, ok := m.Sessions.Load(language); ok {
		session.cancel()
	}
}

func (m *TranscriptionManager) GetSubtitles(language string) (chan string, func()) {
	if language == "" {
		language = m.OriginalLanguage
	}
	subCh := make(chan string, m.SubtitleBufferSize)
	m.Mu.Lock()
	subs, _ := m.Subscribers.Load(language)
	subs = append(subs, subCh)
	m.Subscribers.Store(language, subs)
	m.Mu.Unlock()

	cleanup := func() {
		m.Mu.Lock()
		defer m.Mu.Unlock()
		if oldSubs, ok := m.Subscribers.Load(language); ok {
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
	if !m.Enabled.Load() {
		return nil
	}
	if language == "" {
		language = m.OriginalLanguage
	}
	if session, ok := m.Sessions.Load(language); ok {
		return session.AudioOut
	}

	m.Mu.Lock()
	defer m.Mu.Unlock()
	if session, ok := m.Sessions.Load(language); ok {
		return session.AudioOut
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
	pcm := DownsampleChunkForAI(chunk, m.AppState, m.Config)
	if len(pcm) == 0 {
		return
	}

	logger := slog.With("component", "openai")

	// Auto-start if subscribers exist
	m.Subscribers.Range(func(lang string, _ []chan string) bool {
		if _, ok := m.Sessions.Load(lang); !ok {
			now := time.Now()
			if last, ok := m.LastRestart.Load(lang); ok && now.Sub(last) < 5*time.Second {
				return true
			}
			m.LastRestart.Store(lang, now)
			logger.Info("Auto-starting transcription", slog.String("ai.language", lang))
			m.GetChannel(lang)
		}
		return true
	})

	m.Sessions.Range(func(_ string, s *RealtimeSession) bool {
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

func (m *TranscriptionManager) runSession(s *RealtimeSession) {
	logger := slog.With("component", "openai")
	logger.Info("Starting transcription session", slog.String("ai.language", s.Language))
	defer m.Sessions.Delete(s.Language)
	defer close(s.AudioOut)

	url := fmt.Sprintf("wss://api.openai.com/v1/realtime?model=%s", m.Model)
	header := http.Header{}
	header.Add("Authorization", "Bearer "+m.APIKey)
	header.Add("OpenAI-Safety-Identifier", "abel-recorder-v1")

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
		logger.Error("Transcription dial error",
			slog.Int("openai.status_code", status),
			slog.Any("openai.error", err),
			slog.String("openai.response_body", body),
		)
		return
	}
	defer conn.Close()

	config := SessionUpdateEvent{
		Type: "session.update",
		Session: SessionConfig{
			Type: "transcription",
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
					delta, _ = raw["transcript"].(string)
				}
				if delta != "" {
					BroadcastSubtitle(&m.Subscribers, s.Language, delta)
				}
			} else if raw["type"] == "response.done" {
				if respObj, ok := raw["response"].(map[string]interface{}); ok {
					if usage, ok := respObj["usage"].(map[string]interface{}); ok {
						if totalTokensVal, ok := usage["total_tokens"].(float64); ok {
							totalTokens := int64(totalTokensVal)
							delta := totalTokens - s.lastTokens
							if delta > 0 {
								s.lastTokens = totalTokens
								if telemetry.AITokensConsumed != nil {
									telemetry.AITokensConsumed.Add(context.Background(), delta,
										metric.WithAttributes(
											attribute.String("provider", "openai"),
											attribute.String("language", s.Language),
										),
									)
								}
							}
						}
					}
				}
			} else if raw["type"] == "error" {
				logger.Error("Transcription error", slog.Any("openai.error", raw["error"]))
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
