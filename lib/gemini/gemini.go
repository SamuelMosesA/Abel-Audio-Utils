package gemini

import (
	"behringerRecorder/lib/types"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/genai"
)

type TranslationSession struct {
	Language  string
	AudioIn   chan []byte // PCM 16kHz Mono bytes
	AudioOut  chan []float32
	Subtitles bool
	ctx       context.Context
	cancel    context.CancelFunc
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
	client  GeminiClient
	model   string
	Enabled atomic.Bool

	sessions    sync.Map // map[string]*TranslationSession
	subscribers sync.Map // map[string][]chan string
	mu          sync.Mutex
}


func NewTranslationManager(apiKey, model string) (*TranslationManager, error) {
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
		client: &RealGeminiClient{client: client},
		model:  model,
	}, nil
}



func (m *TranslationManager) GetChannel(language string) chan []float32 {
	return m.GetChannels(language, false)
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

	audioOut := make(chan []float32, 100)
	audioIn := make(chan []byte, 100)
	ctx, cancel := context.WithCancel(context.Background())
	session := &TranslationSession{
		Language:  language,
		AudioIn:   audioIn,
		AudioOut:  audioOut,
		Subtitles: subtitles,
		ctx:       ctx,
		cancel:    cancel,
	}
	m.sessions.Store(language, session)

	go m.runSession(session)

	return audioOut
}

func (m *TranslationManager) GetSubtitles(language string) (chan string, func()) {
	if language == "" {
		language = "default"
	}

	// Ensure session exists
	m.GetChannels(language, true)

	subCh := make(chan string, 10)
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
}

func (m *TranslationManager) ListSessions() []types.SessionInfo {
	var list []types.SessionInfo
	m.sessions.Range(func(key, value interface{}) bool {
		lang := key.(string)
		list = append(list, types.SessionInfo{
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
	}
}

func (m *TranslationManager) runSession(s *TranslationSession) {
	log.Printf("[GEMINI] runSession started for language: %s, model: %s", s.Language, m.model)
	defer log.Printf("[GEMINI] runSession exited for language: %s", s.Language)
	defer m.sessions.Delete(s.Language)

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
		InputAudioTranscription:  &genai.AudioTranscriptionConfig{},
		OutputAudioTranscription: &genai.AudioTranscriptionConfig{},
		RealtimeInputConfig: &genai.RealtimeInputConfig{
			ActivityHandling: genai.ActivityHandlingNoInterruption,
			TurnCoverage:     genai.TurnCoverageTurnIncludesAllInput,
		},
		SpeechConfig: &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
					VoiceName: "Kore",
				},
			},
		},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				func() *genai.Part {
					if displayLang == "English" {
						return genai.NewPartFromText(`You are a professional real-time transcriber. 
Your task: Transcribe the incoming English audio stream into English text.

Rules:
1. Output ONLY the English transcription.
2. The sermon is from a reformed baptist tradition in Amsterdam; use appropriate theological terminology.
3. If there is silence or no speech, remain silent.`)
					}
					return genai.NewPartFromText(fmt.Sprintf(`You are a professional real-time translator. 
Your task: Translate the incoming English audio stream into %s.

Rules:
1. Output ONLY the %s translation audio/text.
2. NEVER output English or extra comments.
3. The sermon is from a reformed baptist tradition in Amsterdam; use appropriate theological terminology in %s.
4. If there is silence or no speech, remain silent.
5. Pay attention to the pauses and try not to rush the pauses.
6. Avoid complex and less used words in the language.
7. For languages which take more time to convey the same meaning, speak faster
8. There might be multiple people speaking. Do not wait for them to finish before audio output. Try to get different voices for each speaker
9. There will be continuous conversation. DO NOT STOP OUTPUT OF TRANSLATED AUDIO & SUBTITLES WITHOUT WAITING FOR PAUSE`, displayLang, displayLang, displayLang))
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
				if ot := resp.ServerContent.OutputTranscription; ot != nil && ot.Text != "" {
					log.Printf("[GEMINI] Session %s -> OutputTranscription: %s", s.Language, ot.Text)
					m.broadcastSubtitle(s.Language, ot.Text)
				}
				if resp.ServerContent.TurnComplete {
					log.Printf("[GEMINI] Session %s turn complete", s.Language)
				}
				if resp.ServerContent.Interrupted {
					log.Printf("[GEMINI] Session %s interrupted", s.Language)
				}
			}
			if resp.ToolCall != nil {
				log.Printf("[GEMINI] Session %s received tool call", s.Language)
			}
			if resp.GoAway != nil {
				log.Printf("[GEMINI] Session %s received GoAway", s.Language)
			}
		}
	}()

	// Send loop
	lastSend := time.Now()
	sendCount := 0
Loop:
	for {
		select {
		case data := <-s.AudioIn:
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
			sendCount++
			if time.Since(lastSend) > 5*time.Second {
				// Calculate peak for this chunk to see if it's silence
				var maxVal int16
				for i := 0; i < len(data); i += 2 {
					val := int16(data[i]) | int16(data[i+1])<<8
					if val < 0 {
						val = -val
					}
					if val > maxVal {
						maxVal = val
					}
				}
				log.Printf("[GEMINI] Session %s sent %d chunks in last 5s (last chunk peak: %d)", s.Language, sendCount, maxVal)
				sendCount = 0
				lastSend = time.Now()
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

func (m *TranslationManager) broadcastSubtitle(language string, text string) {
	if val, ok := m.subscribers.Load(language); ok {
		subs := val.([]chan string)
		for _, ch := range subs {
			select {
			case ch <- text:
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

	// Downsample 48kHz to 16kHz (take 1 of every 3 pairs)
	downsampled := make([]int16, 0, len(chunk)/6)
	for i := 0; i < len(chunk)-5; i += 6 {
		// Average L and R for one sample
		avg := (chunk[i] + chunk[i+1]) / 2.0
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

	m.sessions.Range(func(key, value interface{}) bool {
		s := value.(*TranslationSession)
		select {
		case s.AudioIn <- bytes:
			// OK
		default:
			log.Printf("[GEMINI] Session %s buffer full, dropping chunk", s.Language)
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
