package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"behringerRecorder/lib/state"

	"google.golang.org/genai"
)

func ptr[T any](v T) *T {
	return &v
}

type TranslationSession struct {
	Language     string
	AudioIn      chan []byte // PCM 16kHz Mono bytes
	AudioOut     chan []float32
	Subtitles    bool
	ctx          context.Context
	cancel       context.CancelFunc
	lastDropLog  time.Time
	lastResponse time.Time
}

type GeminiClient interface {
	Connect(ctx context.Context, model string, config *genai.LiveConnectConfig) (GeminiSession, error)
}

type GeminiSession interface {
	SendRealtimeInput(input genai.LiveRealtimeInput) error
	Receive() (*genai.LiveServerMessage, error)
	Close() error
}

type RealGeminiClient struct {
	client *genai.Client
}

func (c *RealGeminiClient) Connect(ctx context.Context, model string, config *genai.LiveConnectConfig) (GeminiSession, error) {
	session, err := c.client.Live.Connect(ctx, model, config)
	if err != nil {
		return nil, err
	}
	return session, nil
}

type TranslationManager struct {
	client             GeminiClient
	model              string
	voice              string
	audioInBufferSize  int
	audioOutBufferSize int
	subtitleBufferSize int
	Enabled            atomic.Bool

	lastRestart   sync.Map // map[string]time.Time
	sessions      sync.Map // map[string]*TranslationSession
	subscribers   sync.Map // map[string][]chan string
	mu            sync.Mutex
	OnStateChange func()
}

func NewTranslationManager(apiKey, model, voice string, audioInSize, audioOutSize, subtitleSize int) (*TranslationManager, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{
			APIVersion: "v1beta",
		},
	})
	if err != nil {
		return nil, err
	}

	return &TranslationManager{
		client:             &RealGeminiClient{client: client},
		model:              model,
		voice:              voice,
		audioInBufferSize:  audioInSize,
		audioOutBufferSize: audioOutSize,
		subtitleBufferSize: subtitleSize,
	}, nil
}

func (m *TranslationManager) GetChannel(language string) chan []float32 {
	return m.GetChannels(language, true)
}

func (m *TranslationManager) GetChannels(language string, subtitles bool) chan []float32 {
	if language == "" {
		language = "default"
	}
	log.Printf("[GEMINI] Getting channel for language index: %s (subtitlesRequested: %v)", language, subtitles)

	if val, ok := m.sessions.Load(language); ok {
		s := val.(*TranslationSession)
		// If we already have a session but with different subtitle setting, we might need to restart it
		// or just accept it as is. For now, if it exists, return it.
		return s.AudioOut
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check
	if val, ok := m.sessions.Load(language); ok {
		return val.(*TranslationSession).AudioOut
	}

	audioOut := make(chan []float32, m.audioOutBufferSize)
	audioIn := make(chan []byte, m.audioInBufferSize)
	ctx, cancel := context.WithCancel(context.Background())
	session := &TranslationSession{
		Language:     language,
		AudioIn:      audioIn,
		AudioOut:     audioOut,
		Subtitles:    true,
		ctx:          ctx,
		cancel:       cancel,
		lastResponse: time.Now(),
	}
	m.sessions.Store(language, session)
	if m.OnStateChange != nil {
		m.OnStateChange()
	}
	go m.runSession(session)

	return audioOut
}

func (m *TranslationManager) GetSubtitles(language string) (chan string, func()) {
	if language == "" {
		language = "default"
	}

	// Ensure session exists
	m.GetChannels(language, true)

	subCh := make(chan string, m.subtitleBufferSize)
	m.mu.Lock()
	var subs []chan string
	if val, ok := m.subscribers.Load(language); ok {
		subs = val.([]chan string)
	}
	subs = append(subs, subCh)
	m.subscribers.Store(language, subs)
	m.mu.Unlock()

	cleanup := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if val, ok := m.subscribers.Load(language); ok {
			oldSubs := val.([]chan string)
			newSubs := make([]chan string, 0, len(oldSubs))
			for _, ch := range oldSubs {
				if ch != subCh {
					newSubs = append(newSubs, ch)
				}
			}
			if len(newSubs) == 0 {
				m.subscribers.Delete(language)
			} else {
				m.subscribers.Store(language, newSubs)
			}
		}
		close(subCh)
	}

	return subCh, cleanup
}

func (m *TranslationManager) CloseAll() {
	log.Printf("[GEMINI] Closing all translation sessions")
	m.sessions.Range(func(key, value interface{}) bool {
		s := value.(*TranslationSession)
		s.cancel()
		return true
	})
	if m.OnStateChange != nil {
		m.OnStateChange()
	}
}

func (m *TranslationManager) ListSessions() []state.SessionInfo {
	var list []state.SessionInfo
	m.sessions.Range(func(key, value interface{}) bool {
		lang := key.(string)
		list = append(list, state.SessionInfo{
			Language:  lang,
			Listeners: 0, // Simplified
			Subtitles: value.(*TranslationSession).Subtitles,
		})
		return true
	})
	return list
}

func (m *TranslationManager) StopSession(language string, subtitles bool) {
	log.Printf("[GEMINI] Force stopping session for %s", language)
	if val, ok := m.sessions.Load(language); ok {
		val.(*TranslationSession).cancel()
		if m.OnStateChange != nil {
			m.OnStateChange()
		}
	}
}

func (m *TranslationManager) runSession(s *TranslationSession) {
	log.Printf("[GEMINI] runSession started for language: %s, model: %s", s.Language, m.model)
	defer log.Printf("[GEMINI] runSession exited for language: %s", s.Language)
	defer func() {
		m.sessions.Delete(s.Language)
		if m.OnStateChange != nil {
			m.OnStateChange()
		}
	}()

	log.Printf("[GEMINI] Starting translation session for %s", s.Language)

	modalities := []genai.Modality{genai.ModalityAudio}

	targetLang := s.Language
	if targetLang == "default" || targetLang == "" {
		targetLang = "English"
	}

	// Simple capitalization
	displayLang := targetLang
	if len(displayLang) > 0 {
		displayLang = strings.ToUpper(displayLang[:1]) + displayLang[1:]
	}

	config := &genai.LiveConnectConfig{
		ResponseModalities:       modalities,
		MediaResolution:          genai.MediaResolutionMedium,
		InputAudioTranscription:  &genai.AudioTranscriptionConfig{},
		OutputAudioTranscription: &genai.AudioTranscriptionConfig{},
		ContextWindowCompression: &genai.ContextWindowCompressionConfig{
			TriggerTokens: ptr(int64(104857)),
			SlidingWindow: &genai.SlidingWindow{
				TargetTokens: ptr(int64(52428)),
			},
		},
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: genai.ThinkingLevelMedium,
		},
		RealtimeInputConfig: &genai.RealtimeInputConfig{
			ActivityHandling: genai.ActivityHandlingNoInterruption,
			TurnCoverage:     genai.TurnCoverageTurnIncludesAllInput,
		},
		SpeechConfig: &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
					VoiceName: m.voice,
				},
			},
		},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				func() *genai.Part {
					if displayLang == "English" {
						return genai.NewPartFromText(`You are a professional real-time transcriber. 
Your task: Transcribe the incoming English audio stream and repeat it.

1. Transcribe EVERY single word you hear. 
2. Do NOT skip any sentences, even if they seem repetitive or out of context.
3. Maintain 100% fidelity. Include filler words, stutters, and all spoken content.
4. If you are unsure, transcribe your best guess. NEVER stay silent while audio is playing.
5. DO NOT WAIT FOR PAUSES.
6. Continuously output without waiting for a pause. I want lower latency between input and output.
7. Stream word by word, token by token, as you hear it.`)
					}
					return genai.NewPartFromText(fmt.Sprintf(`You are a professional real-time translator. 
Your task: Translate the incoming English audio stream into %s.

Rules:
1. Translate EVERY single sentence you hear. 
2. Do NOT skip any content, even if it seems repetitive.
3. The sermon is from a reformed baptist tradition in Amsterdam; use appropriate theological terminology in %s.
4. DO NOT WAIT FOR PAUSES.
5. Avoid complex and less used words in the language.
6. For languages which take more time to convey the same meaning, speak faster.
7. Continuously output without waiting for a pause. I want lower latency between input and output.
8. Stream word by word, token by token, as you hear it.
`, displayLang, displayLang))
				}(),
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	liveSession, err := m.client.Connect(ctx, m.model, config)
	if err != nil {
		log.Printf("[GEMINI] Failed to connect live session for %s: %v", s.Language, err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	receiveLoopCtx, receiveLoopCancel := context.WithCancel(ctx)

	// Receive loop
	go func() {
		defer wg.Done()
		defer receiveLoopCancel()
		for {
			resp, err := liveSession.Receive()
			if err != nil {
				log.Printf("[GEMINI] Session %s receive error: %v", s.Language, err)
				return
			}
			// log.Printf("[GEMINI] Session %s received message", s.Language) // Too noisy
			if resp.ServerContent != nil {
				if resp.ServerContent.ModelTurn != nil {
					log.Printf("[GEMINI] Session %s received model turn with %d parts", s.Language, len(resp.ServerContent.ModelTurn.Parts))
					for _, part := range resp.ServerContent.ModelTurn.Parts {
						if part.InlineData != nil {
							pcmData := part.InlineData.Data
							floatData := convertInt16ToFloat32(pcmData)
							select {
							case s.AudioOut <- floatData:
							default:
								// Drop if nobody is listening
							}
						}
					}
				}
				if it := resp.ServerContent.InputTranscription; it != nil && it.Text != "" {
					log.Printf("[GEMINI] Session %s <- InputTranscription: %s", s.Language, it.Text)
					s.lastResponse = time.Now()
				}
				if ot := resp.ServerContent.OutputTranscription; ot != nil && ot.Text != "" {
					log.Printf("[GEMINI] Session %s -> OutputTranscription: %s", s.Language, ot.Text)
					m.broadcastSubtitle(s.Language, ot.Text, 0)
					s.lastResponse = time.Now()
				}
				if resp.ServerContent.TurnComplete {
					log.Printf("[GEMINI] Session %s turn complete", s.Language)
				}
				if resp.ServerContent.Interrupted {
					log.Printf("[GEMINI] Session %s interrupted", s.Language)
				}
			}
			if resp.UsageMetadata != nil {
				// Broadcast usage metadata as a "meta" subtitle chunk
				m.broadcastSubtitle(s.Language, "", int(resp.UsageMetadata.TotalTokenCount))
			}
			if resp.ToolCall != nil {
				log.Printf("[GEMINI] Session %s received tool call", s.Language)
			}
			if resp.GoAway != nil {
				log.Printf("[GEMINI] Session %s received GoAway: %v", s.Language, resp.GoAway)
			}
			if resp.SetupComplete != nil {
				log.Printf("[GEMINI] Session %s setup complete", s.Language)
			}
		}
	}()

	// Send loop
	// Send initial activity start since AAD is disabled
	err = liveSession.SendRealtimeInput(genai.LiveRealtimeInput{
		ActivityStart: &genai.ActivityStart{},
	})
	if err != nil {
		log.Printf("[GEMINI] Failed to send activity start for %s: %v", s.Language, err)
	}

Loop:
	for {
		select {
		case data := <-s.AudioIn:
			// Send to Gemini
			err := liveSession.SendRealtimeInput(genai.LiveRealtimeInput{
				Audio: &genai.Blob{
					MIMEType: "audio/pcm;rate=16000",
					Data:     data,
				},
			})
			if err != nil {
				log.Printf("[GEMINI] Session %s send error: %v", s.Language, err)
				break Loop
			}
		case <-s.ctx.Done():
			break Loop
		case <-receiveLoopCtx.Done():
			break Loop
		}
	}

	liveSession.Close()
	wg.Wait()
	close(s.AudioOut)
}

func (m *TranslationManager) SetEnabled(enabled bool) {
	m.Enabled.Store(enabled)
}

func (m *TranslationManager) SetOnStateChange(fn func()) {
	m.OnStateChange = fn
}

func (m *TranslationManager) broadcastSubtitle(language string, text string, tokens int) {
	if val, ok := m.subscribers.Load(language); ok {
		subs := val.([]chan string)
		payload, _ := json.Marshal(map[string]interface{}{
			"text":   text,
			"tokens": tokens,
		})
		payloadStr := string(payload)
		for _, ch := range subs {
			select {
			case ch <- payloadStr:
			default:
				// Buffer full
			}
		}
	}
}

func (m *TranslationManager) PushAudio(chunk []float32) {
	if !m.Enabled.Load() {
		return
	}
	if len(chunk) == 0 {
		return
	}

	// Downsample 48kHz to 16kHz (average 3 frames of stereo)
	downsampled := make([]int16, 0, len(chunk)/6)
	for i := 0; i < len(chunk)-5; i += 6 {
		// Average all 6 samples in the 3-frame window (L+R for 3 frames)
		avg := (chunk[i] + chunk[i+1] + chunk[i+2] + chunk[i+3] + chunk[i+4] + chunk[i+5]) / 6.0

		// Clamp to prevent overflow
		if avg > 1.0 {
			avg = 1.0
		} else if avg < -1.0 {
			avg = -1.0
		}

		// Convert to int16
		s := int16(avg * 32767)
		downsampled = append(downsampled, s)
	}

	if len(downsampled) == 0 {
		return
	}

	// Convert to bytes for send
	bytes := make([]byte, len(downsampled)*2)
	for i, v := range downsampled {
		bytes[i*2] = byte(v & 0xff)
		bytes[i*2+1] = byte(v >> 8)
	}

	m.subscribers.Range(func(key, value interface{}) bool {
		lang := key.(string)
		if _, ok := m.sessions.Load(lang); !ok {
			// Session is dead but has subscribers - restart it with a cooldown
			now := time.Now()
			if last, ok := m.lastRestart.Load(lang); ok {
				if now.Sub(last.(time.Time)) < 5*time.Second {
					return true // Skip restart, too soon
				}
			}
			log.Printf("[GEMINI] Watchdog: Restarting dead session for %s", lang)
			m.lastRestart.Store(lang, now)
			m.GetChannels(lang, true)
		}
		return true
	})

	m.sessions.Range(func(key, value interface{}) bool {
		s := value.(*TranslationSession)
		select {
		case s.AudioIn <- bytes:
		default:
			if time.Since(s.lastDropLog) > 5*time.Second {
				log.Printf("[GEMINI] Session %s audio buffer full, dropping chunk", s.Language)
				s.lastDropLog = time.Now()
			}
		}
		return true
	})
}

func convertInt16ToFloat32(data []byte) []float32 {
	count := len(data) / 2
	// Gemini outputs 24kHz Mono. System expects 48kHz Stereo.
	// Factor of 2 for upsampling (24->48) and factor of 2 for stereo (mono->L+R).
	floats := make([]float32, count*4)
	for i := 0; i < count; i++ {
		v := int16(data[i*2]) | int16(data[i*2+1])<<8
		f := float32(v) / 32767.0

		// Upsample 1:2 by repeating samples for 48kHz, and mono->stereo
		base := i * 4
		floats[base] = f   // Sample A - Left
		floats[base+1] = f // Sample A - Right
		floats[base+2] = f // Sample B - Left
		floats[base+3] = f // Sample B - Right
	}
	return floats
}
