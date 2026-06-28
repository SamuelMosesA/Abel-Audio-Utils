package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"abel/src/backend/lib/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/gorilla/websocket"
)

type TranslationManager struct {
	Config             *config.Config
	AppState           *state.AppState
	APIKey             string
	Model              string
	Voice              string
	AudioInBufferSize  int
	AudioOutBufferSize int
	SubtitleBufferSize int
	OriginalLanguage   string
	Enabled            atomic.Bool

	Sessions      sync.Map // map[string]*RealtimeSession
	Subscribers   sync.Map // map[string][]chan string
	LastRestart   sync.Map // map[string]time.Time
	Mu            sync.Mutex
	OnStateChange func()
}

func NewTranslationManager(cfg *config.Config, appState *state.AppState, apiKey, model, voice, originalLang string, audioInSize, audioOutSize, subtitleSize int) (*TranslationManager, error) {
	return &TranslationManager{
		Config:             cfg,
		AppState:           appState,
		APIKey:             apiKey,
		Model:              model,
		Voice:              voice,
		OriginalLanguage:   originalLang,
		AudioInBufferSize:  audioInSize,
		AudioOutBufferSize: audioOutSize,
		SubtitleBufferSize: subtitleSize,
	}, nil
}

func (m *TranslationManager) SetEnabled(enabled bool) {
	m.Enabled.Store(enabled)
}

func (m *TranslationManager) SetOnStateChange(fn func()) {
	m.OnStateChange = fn
}

func (m *TranslationManager) CloseAll() {
	m.Sessions.Range(func(key, value interface{}) bool {
		s := value.(*RealtimeSession)
		s.cancel()
		return true
	})
}

func (m *TranslationManager) ListSessions() []state.SessionInfo {
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

func (m *TranslationManager) StopSession(language string, subtitles bool) {
	if val, ok := m.Sessions.Load(language); ok {
		val.(*RealtimeSession).cancel()
	}
}

func (m *TranslationManager) isOriginalLanguage(lang string) bool {
	if lang == "default" {
		return true
	}
	code := m.Config.ResolveLanguageCode(lang)
	return code == m.OriginalLanguage
}

func (m *TranslationManager) GetSubtitles(language string) (chan string, func()) {
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

func (m *TranslationManager) GetChannel(language string) chan []float32 {
	if !m.Enabled.Load() {
		return nil
	}
	if m.isOriginalLanguage(language) || language == "" {
		return nil
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

func (m *TranslationManager) PushAudio(chunk []float32) {
	if !m.Enabled.Load() {
		return
	}
	pcm := m.downsample(chunk)
	if len(pcm) == 0 {
		return
	}

	m.Sessions.Range(func(key, value interface{}) bool {
		lang := key.(string)
		if m.isOriginalLanguage(lang) {
			return true
		}
		s := value.(*RealtimeSession)
		select {
		case s.AudioIn <- pcm:
		default:
		}
		return true
	})
}

func (m *TranslationManager) downsample(chunk []float32) []byte {
	srcRate := int(m.AppState.Config().SampleRate())
	if srcRate <= 0 {
		srcRate = m.Config.SampleRate
	}
	dstRate := 24000 // OpenAI Realtime expects 24kHz

	// If sample rates are already equal to 24000 (mono conversion only)
	if srcRate == dstRate {
		downsampled := make([]int16, len(chunk)/2)
		for i := 0; i < len(chunk)/2; i++ {
			avg := (chunk[i*2] + chunk[i*2+1]) / 2.0
			if avg > 1.0 { avg = 1.0 } else if avg < -1.0 { avg = -1.0 }
			downsampled[i] = int16(avg * 32767)
		}
		bytes := make([]byte, len(downsampled)*2)
		for i, v := range downsampled {
			bytes[i*2] = byte(v & 0xff)
			bytes[i*2+1] = byte(v >> 8)
		}
		return bytes
	}

	// General downsampling using accumulators/ratios
	ratio := float64(srcRate) / float64(dstRate)
	srcFrames := len(chunk) / 2
	dstFrames := int(float64(srcFrames) / ratio)
	if dstFrames <= 0 {
		return nil
	}

	downsampled := make([]int16, dstFrames)
	for i := 0; i < dstFrames; i++ {
		startFrame := int(float64(i) * ratio)
		endFrame := int(float64(i+1) * ratio)
		if endFrame > srcFrames {
			endFrame = srcFrames
		}
		if endFrame <= startFrame {
			endFrame = startFrame + 1
		}

		var sum float32
		count := 0
		for f := startFrame; f < endFrame; f++ {
			sum += chunk[f*2] + chunk[f*2+1]
			count += 2
		}
		avg := sum / float32(count)
		if avg > 1.0 { avg = 1.0 } else if avg < -1.0 { avg = -1.0 }
		downsampled[i] = int16(avg * 32767)
	}

	bytes := make([]byte, len(downsampled)*2)
	for i, v := range downsampled {
		bytes[i*2] = byte(v & 0xff)
		bytes[i*2+1] = byte(v >> 8)
	}
	return bytes
}

func (m *TranslationManager) runSession(s *RealtimeSession) {
	logger := slog.With("component", "openai")
	logger.Info("Starting translation session", slog.String("ai.language", s.Language))
	defer m.Sessions.Delete(s.Language)
	defer close(s.AudioOut)

	url := fmt.Sprintf("wss://api.openai.com/v1/realtime/translations?model=%s", m.Model)
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
		logger.Error("Translation dial error",
			slog.String("ai.language", s.Language),
			slog.String("ai.resolved_name", m.Config.ResolveLanguageName(s.Language)),
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
			Audio: &AudioConfig{
				Output: &OutputConfig{
					Language: s.Language,
				},
			},
		},
	}
	if err := conn.WriteJSON(config); err != nil { return }

	// Receive
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil { return }
			var raw map[string]interface{}
			if err := json.Unmarshal(message, &raw); err != nil { continue }
			
			if telemetry.AIEventsReceived != nil {
				eventType := "unknown"
				if t, ok := raw["type"].(string); ok {
					eventType = t
				}
				telemetry.AIEventsReceived.Add(context.Background(), 1,
					metric.WithAttributes(
						attribute.String("language", s.Language),
						attribute.String("event_type", eventType),
					),
				)
			}

			// Revert to original event names for translation session
			if raw["type"] == "session.output_audio.delta" {
				if telemetry.AIAudioDeltasReceived != nil {
					telemetry.AIAudioDeltasReceived.Add(context.Background(), 1,
						metric.WithAttributes(
							attribute.String("language", s.Language),
						),
					)
				}
				delta, _ := raw["delta"].(string)
				targetRate := int(m.AppState.Config().SampleRate())
				if targetRate <= 0 {
					targetRate = m.Config.SampleRate
				}
				floats, err := DecodeAudioDelta(delta, targetRate)
				if err == nil {
					select {
					case s.AudioOut <- floats:
					default:
					}
				}
			}

			if raw["type"] == "session.output_transcript.delta" {
				delta, _ := raw["delta"].(string)
				m.broadcastSubtitle(s.Language, delta)
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
				logger.Error("Translation error",
					slog.String("ai.language", s.Language),
					slog.Any("openai.error", raw["error"]),
				)
				return
			}
		}
	}()

	// Send
	for {
		select {
		case pcm := <-s.AudioIn:
			event := map[string]interface{}{
				"type": "session.input_audio_buffer.append",
				"audio": base64.StdEncoding.EncodeToString(pcm),
			}
			if err := conn.WriteJSON(event); err != nil { return }
		case <-s.ctx.Done():
			return
		}
	}
}

func (m *TranslationManager) broadcastSubtitle(language, text string) {
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
