package gemini

import (
	"behringerRecorder/lib/types"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/genai"
)

type TranslationSession struct {
	Language string
	AudioIn  chan []byte // PCM 16kHz Mono bytes
	AudioOut chan []float32
	ctx      context.Context
	cancel   context.CancelFunc
}

type TranslationManager struct {
	client *genai.Client
	model  string
	
	sessions sync.Map // map[string]*TranslationSession
	mu       sync.Mutex
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
		client: client,
		model:  model,
	}, nil
}

func (m *TranslationManager) GetChannel(language string) chan []float32 {
	log.Printf("[GEMINI] Getting channel for language: %s", language)
	if language == "" || language == "default" {
		return nil
	}

	if val, ok := m.sessions.Load(language); ok {
		return val.(*TranslationSession).AudioOut
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
		Language: language,
		AudioIn:  audioIn,
		AudioOut: audioOut,
		ctx:      ctx,
		cancel:   cancel,
	}
	m.sessions.Store(language, session)

	go m.runSession(session)

	return audioOut
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
			Listeners: 0, // Simplified: not tracking individual stream listeners anymore
		})
		return true
	})
	return list
}

func (m *TranslationManager) StopSession(language string) {
	log.Printf("[GEMINI] Force stopping session for %s", language)
	if val, ok := m.sessions.Load(language); ok {
		val.(*TranslationSession).cancel()
	}
}

func (m *TranslationManager) runSession(s *TranslationSession) {
	defer m.sessions.Delete(s.Language)
	defer close(s.AudioOut)

	log.Printf("[GEMINI] Starting translation session for %s", s.Language)

	config := &genai.LiveConnectConfig{
		ResponseModalities: []genai.Modality{genai.ModalityAudio},
		RealtimeInputConfig: &genai.RealtimeInputConfig{
			ActivityHandling: genai.ActivityHandlingNoInterruption,
			TurnCoverage:     genai.TurnCoverageTurnIncludesAllInput,
		},
		SpeechConfig: &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
					VoiceName: "Puck",
				},
			},
		},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText(fmt.Sprintf(`I will send a live audio stream of a sermon in English. The church is located in Amsterdam. It is a reformed baptist tradition. So try to understand the theological concepts and find the apt way to explain in the target language.
You are to translate to the language %[1]s.
Only output the translated audio. NEVER output text. Do not add any extra comments or explanations. 
Do not stop your output in the translated language if the person starts talking again. You must output side by side with the English audio.
Try to understand the idioms and phrases and cultural contexts of British and American english speaker and translate the meaning as an explanation in language %[1]s. For example
"out of shape" is translated to something like unfit etc`, s.Language)),
			},
			Role: "system",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	liveSession, err := m.client.Live.Connect(ctx, m.model, config)
	if err != nil {
		log.Printf("[GEMINI] Failed to connect live session for %s: %v", s.Language, err)
		return
	}

	// Receive loop
	go func() {
		for {
			resp, err := liveSession.Receive()
			if err != nil {
				log.Printf("[GEMINI] Session %s receive error: %v", s.Language, err)
				s.cancel()
				return
			}
			if resp.ServerContent != nil {
				if resp.ServerContent.ModelTurn != nil {
					log.Printf("[GEMINI] Session %s received model turn with %d parts", s.Language, len(resp.ServerContent.ModelTurn.Parts))
					for _, part := range resp.ServerContent.ModelTurn.Parts {
						if part.InlineData != nil {
							pcmData := part.InlineData.Data
							log.Printf("[GEMINI] Session %s received %d bytes of audio data", s.Language, len(pcmData))
							floatData := convertInt16ToFloat32(pcmData)
							s.AudioOut <- floatData
						}
						if part.Text != "" {
							log.Printf("[GEMINI] Session %s received text: %s", s.Language, part.Text)
						}
					}
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
				s.cancel()
				return
			}
			sendCount++
			if time.Since(lastSend) > 5*time.Second {
				// Calculate peak for this chunk to see if it's silence
				var maxVal int16
				for i := 0; i < len(data); i += 2 {
					val := int16(data[i]) | int16(data[i+1])<<8
					if val < 0 { val = -val }
					if val > maxVal { maxVal = val }
				}
				log.Printf("[GEMINI] Session %s sent %d chunks in last 5s (last chunk peak: %d)", s.Language, sendCount, maxVal)
				sendCount = 0
				lastSend = time.Now()
			}
		case <-s.ctx.Done():
			liveSession.Close()
			return
		}
	}
}

func (m *TranslationManager) PushAudio(chunk []float32) {
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
		floats[base] = f     // Sample A - Left
		floats[base+1] = f   // Sample A - Right
		floats[base+2] = f   // Sample B - Left
		floats[base+3] = f   // Sample B - Right
	}
	return floats
}
