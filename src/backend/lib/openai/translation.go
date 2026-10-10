package openai

import (
	"context"
	"encoding/base64"
	"abel/src/backend/lib/audioengine/conversion"
	"encoding/json"
	"errors"
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

	endpoint string // overrides the OpenAI realtime URL (tests)
}

// NewTranslationManager creates a live multilingual speech translation session manager.
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
			Language:  lang,
			Listeners: m.GetListenerCount(lang),
			Subtitles: true,
		})
		return true
	})
	return list
}

func (m *TranslationManager) GetListenerCount(language string) int {
	code := m.Config.ResolveLanguageCode(language)
	name := m.Config.ResolveLanguageName(language)

	m.Mu.Lock()
	defer m.Mu.Unlock()

	count := 0
	if val, ok := m.Subscribers.Load(code); ok {
		if subs, ok := val.([]chan string); ok {
			count += len(subs)
		}
	}
	if strings.ToLower(code) != strings.ToLower(name) {
		if val, ok := m.Subscribers.Load(name); ok {
			if subs, ok := val.([]chan string); ok {
				count += len(subs)
			}
		}
	}
	return count
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

	// Auto-start a session for any language that has subtitle subscribers but
	// no live session (subtitle-only viewers never call GetChannel). Debounced
	// so a failing dial cannot spin. Mirrors TranscriptionManager.PushAudio.
	m.Subscribers.Range(func(key, value interface{}) bool {
		lang := key.(string)
		if m.isOriginalLanguage(lang) {
			return true
		}
		if _, ok := m.Sessions.Load(lang); ok {
			return true
		}
		now := time.Now()
		if last, ok := m.LastRestart.Load(lang); ok && now.Sub(last.(time.Time)) < 5*time.Second {
			return true
		}
		m.LastRestart.Store(lang, now)
		slog.With("component", "openai").Info("Auto-starting translation for subtitle subscribers",
			slog.String("ai.language", lang))
		m.GetChannel(lang)
		return true
	})

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
	return conversion.DownsampleStereoToMonoPCM24k(chunk, srcRate)
}

// Reconnect policy for a translation session. A session is supervised for
// its whole lifetime: when OpenAI closes the socket (the 60-minute cap, a
// network drop) or reports session_expired, the session object and its
// channels stay alive, audio arriving meanwhile is buffered, and the socket
// is re-dialled. Only ctx cancellation (StopSession / CloseAll) ends it.
const (
	defaultTranslationEndpoint = "wss://api.openai.com/v1/realtime/translations"
	reconnectMinBackoff        = 1 * time.Second
	reconnectMaxBackoff        = 30 * time.Second
	// Audio held while disconnected: ~15 s of 24 kHz PCM16 mono. Oldest dropped first.
	pendingAudioMaxBytes = 24000 * 2 * 15
)

var errSessionExpired = errors.New("openai session expired")

// pendingAudio is a bounded FIFO of downsampled audio kept while no socket is open.
type pendingAudio struct {
	chunks [][]byte
	bytes  int
}

func (p *pendingAudio) push(chunk []byte) {
	p.chunks = append(p.chunks, chunk)
	p.bytes += len(chunk)
	for p.bytes > pendingAudioMaxBytes && len(p.chunks) > 0 {
		p.bytes -= len(p.chunks[0])
		p.chunks = p.chunks[1:]
	}
}

func (p *pendingAudio) pop() {
	p.bytes -= len(p.chunks[0])
	p.chunks = p.chunks[1:]
}

func (m *TranslationManager) endpointURL() string {
	base := m.endpoint
	if base == "" {
		base = defaultTranslationEndpoint
	}
	return fmt.Sprintf("%s?model=%s", base, m.Model)
}

func (m *TranslationManager) runSession(s *RealtimeSession) {
	logger := slog.With("component", "openai", slog.String("ai.language", s.Language))
	logger.Info("Starting translation session")
	defer m.Sessions.Delete(s.Language)
	defer close(s.AudioOut)

	var pending pendingAudio
	backoff := reconnectMinBackoff
	attempt := 0

	for {
		if s.ctx.Err() != nil {
			return
		}
		conn, err := m.dialBuffering(s, &pending)
		if err != nil {
			if s.ctx.Err() != nil {
				return
			}
			attempt++
			logger.Error("Translation dial error",
				slog.String("ai.resolved_name", m.Config.ResolveLanguageName(s.Language)),
				slog.Any("openai.error", err),
				slog.Int("attempt", attempt),
				slog.Duration("retry_in", backoff),
			)
			if !m.waitBuffering(s, backoff, &pending) {
				return
			}
			backoff *= 2
			if backoff > reconnectMaxBackoff {
				backoff = reconnectMaxBackoff
			}
			continue
		}
		if attempt > 0 {
			logger.Info("Translation session reconnected", slog.Int("attempts", attempt))
		}
		attempt = 0
		backoff = reconnectMinBackoff
		s.lastTokens = 0 // usage counters restart with a new OpenAI session

		reason := m.serve(s, conn, &pending)
		conn.Close()
		if s.ctx.Err() != nil {
			return
		}
		logger.Warn("Translation session dropped, reconnecting", slog.Any("reason", reason))
	}
}

type dialResult struct {
	conn *websocket.Conn
	err  error
}

// dialBuffering opens the socket while continuing to drain AudioIn into pending,
// so audio produced during the handshake is replayed rather than dropped.
func (m *TranslationManager) dialBuffering(s *RealtimeSession, pending *pendingAudio) (*websocket.Conn, error) {
	header := http.Header{}
	header.Add("Authorization", "Bearer "+m.APIKey)
	header.Add("OpenAI-Safety-Identifier", "abel-recorder-v1")

	resultCh := make(chan dialResult, 1)
	go func() {
		conn, resp, err := websocket.DefaultDialer.DialContext(s.ctx, m.endpointURL(), header)
		if err != nil && resp != nil {
			defer resp.Body.Close()
			buf := make([]byte, 1024)
			n, _ := resp.Body.Read(buf)
			err = fmt.Errorf("%w (status %d: %s)", err, resp.StatusCode, string(buf[:n]))
		}
		resultCh <- dialResult{conn: conn, err: err}
	}()

	for {
		select {
		case r := <-resultCh:
			return r.conn, r.err
		case pcm := <-s.AudioIn:
			pending.push(pcm)
		case <-s.ctx.Done():
			go func() {
				if r := <-resultCh; r.conn != nil {
					r.conn.Close()
				}
			}()
			return nil, s.ctx.Err()
		}
	}
}

// waitBuffering sleeps for the backoff while draining AudioIn into pending.
// Returns false if the session was cancelled.
func (m *TranslationManager) waitBuffering(s *RealtimeSession, d time.Duration, pending *pendingAudio) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return true
		case pcm := <-s.AudioIn:
			pending.push(pcm)
		case <-s.ctx.Done():
			return false
		}
	}
}

// serve runs one socket's lifetime: configure, replay pending audio, then pump
// AudioIn until the socket fails, OpenAI expires the session, or ctx ends.
func (m *TranslationManager) serve(s *RealtimeSession, conn *websocket.Conn, pending *pendingAudio) error {
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
	if err := conn.WriteJSON(config); err != nil {
		return err
	}

	readErr := make(chan error, 1)
	go m.readLoop(s, conn, readErr)

	for len(pending.chunks) > 0 {
		if err := writeAudio(conn, pending.chunks[0]); err != nil {
			return err // unsent chunks stay buffered for the next socket
		}
		pending.pop()
	}

	for {
		select {
		case pcm := <-s.AudioIn:
			if err := writeAudio(conn, pcm); err != nil {
				pending.push(pcm) // do not lose the chunk that hit the dead socket
				return err
			}
		case err := <-readErr:
			return err
		case <-s.ctx.Done():
			return s.ctx.Err()
		}
	}
}

func writeAudio(conn *websocket.Conn, pcm []byte) error {
	return conn.WriteJSON(map[string]interface{}{
		"type":  "session.input_audio_buffer.append",
		"audio": base64.StdEncoding.EncodeToString(pcm),
	})
}

// readLoop consumes server events until the socket errors or OpenAI expires
// the session. Non-fatal error events are logged and the session continues;
// a fatal one is followed by the server closing the socket, which surfaces
// here as a read error and triggers a reconnect.
func (m *TranslationManager) readLoop(s *RealtimeSession, conn *websocket.Conn, done chan<- error) {
	logger := slog.With("component", "openai", slog.String("ai.language", s.Language))
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			done <- err
			return
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(message, &raw); err != nil {
			continue
		}
		eventType, _ := raw["type"].(string)

		if telemetry.AIEventsReceived != nil {
			if eventType == "" {
				eventType = "unknown"
			}
			telemetry.AIEventsReceived.Add(context.Background(), 1,
				metric.WithAttributes(
					attribute.String("language", s.Language),
					attribute.String("event_type", eventType),
				),
			)
		}

		switch eventType {
		case "session.output_audio.delta":
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
		case "session.output_transcript.delta":
			delta, _ := raw["delta"].(string)
			m.broadcastSubtitle(s.Language, delta)
		case "response.done":
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
		case "error":
			logger.Error("Translation error", slog.Any("openai.error", raw["error"]))
			if errObj, ok := raw["error"].(map[string]interface{}); ok {
				if code, _ := errObj["code"].(string); code == "session_expired" {
					done <- errSessionExpired
					return
				}
			}
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
