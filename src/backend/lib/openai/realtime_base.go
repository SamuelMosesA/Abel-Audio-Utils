package openai

import (
	"context"
	"encoding/base64"
	"sync/atomic"

	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
)

// Common structures for OpenAI Realtime WebSocket API
type Event struct {
	Type    string `json:"type"`
	EventID string `json:"event_id,omitempty"`
}

type SessionUpdateEvent struct {
	Type    string        `json:"type"`
	Session SessionConfig `json:"session"`
}

type SessionConfig struct {
	Type              string       `json:"type,omitempty"`
	Modalities        []string     `json:"modalities,omitempty"`
	Instructions      string       `json:"instructions,omitempty"`
	Voice             string       `json:"voice,omitempty"`
	InputAudioFormat  string       `json:"input_audio_format,omitempty"`
	OutputAudioFormat string       `json:"output_audio_format,omitempty"`
	Audio             *AudioConfig `json:"audio,omitempty"`
}

type AudioConfig struct {
	Input  *InputConfig  `json:"input,omitempty"`
	Output *OutputConfig `json:"output,omitempty"`
}

type InputConfig struct {
	Format        *InputFormat         `json:"format,omitempty"`
	Transcription *TranscriptionConfig `json:"transcription,omitempty"`
	TurnDetection *TurnDetectionConfig `json:"turn_detection,omitempty"`
}

type TurnDetectionConfig struct {
	Type              string  `json:"type,omitempty"`
	Threshold         float64 `json:"threshold,omitempty"`
	PrefixPaddingMs   int     `json:"prefix_padding_ms,omitempty"`
	SilenceDurationMs int     `json:"silence_duration_ms,omitempty"`
}

type InputFormat struct {
	Type string `json:"type,omitempty"`
	Rate int    `json:"rate,omitempty"`
}

type TranscriptionConfig struct {
	Model    string `json:"model,omitempty"`
	Language string `json:"language,omitempty"`
}

type OutputConfig struct {
	Format   *OutputFormat `json:"format,omitempty"`
	Language string        `json:"language,omitempty"`
	Voice    string        `json:"voice,omitempty"`
}

type OutputFormat struct {
	Type string `json:"type,omitempty"`
	Rate int    `json:"rate,omitempty"`
}

type InputAudioAppendEvent struct {
	Type  string `json:"type"`
	Audio string `json:"audio"` // base64
}

type RealtimeSession struct {
	Language     string
	AudioIn      chan []byte 
	AudioOut     chan []float32
	ctx          context.Context
	cancel       context.CancelFunc
	lastTokens   int64
}

// OpenAIManager is a wrapper that delegates to separate Transcription and Translation managers
type OpenAIManager struct {
	Config           *config.Config
	OriginalLanguage   string // Normalized code (e.g., "en")
	Transcriber      *TranscriptionManager
	Translator       *TranslationManager
	Enabled          atomic.Bool
	OnStateChange    func()
}

func (m *OpenAIManager) isOriginalLanguage(lang string) bool {
	if lang == "" || lang == "default" {
		return true
	}
	code := m.Config.ResolveLanguageCode(lang)
	return code == m.OriginalLanguage
}

func NewOpenAIManager(cfg *config.Config, appState *state.AppState, apiKey, translateModel, transcribeModel, voice, originalLang string) (*OpenAIManager, error) {
	transcriber, _ := NewTranscriptionManager(cfg, appState, apiKey, transcribeModel, originalLang, 100, 1000, 100)
	translator, _ := NewTranslationManager(cfg, appState, apiKey, translateModel, voice, originalLang, 100, 1000, 100)
	
	return &OpenAIManager{
		Config:           cfg,
		OriginalLanguage: originalLang,
		Transcriber:      transcriber,
		Translator:       translator,
	}, nil
}

func (m *OpenAIManager) SetEnabled(enabled bool) {
	m.Enabled.Store(enabled)
	m.Transcriber.SetEnabled(enabled)
	m.Translator.SetEnabled(enabled)
}

func (m *OpenAIManager) SetOnStateChange(fn func()) {
	m.OnStateChange = fn
	m.Transcriber.SetOnStateChange(fn)
	m.Translator.SetOnStateChange(fn)
}

func (m *OpenAIManager) CloseAll() {
	m.Transcriber.CloseAll()
	m.Translator.CloseAll()
}

func (m *OpenAIManager) ListSessions() []state.SessionInfo {
	list := m.Transcriber.ListSessions()
	list = append(list, m.Translator.ListSessions()...)
	return list
}

func (m *OpenAIManager) StopSession(language string, subtitles bool) {
	if m.isOriginalLanguage(language) {
		m.Transcriber.StopSession(language, subtitles)
	} else {
		m.Translator.StopSession(language, subtitles)
	}
}

func (m *OpenAIManager) GetSubtitles(language string) (chan string, func()) {
	if m.isOriginalLanguage(language) {
		return m.Transcriber.GetSubtitles(language)
	}
	return m.Translator.GetSubtitles(language)
}

func (m *OpenAIManager) GetChannel(language string) chan []float32 {
	if m.isOriginalLanguage(language) {
		return m.Transcriber.GetChannel(language)
	}
	return m.Translator.GetChannel(language)
}

func (m *OpenAIManager) PushAudio(chunk []float32) {
	if !m.Enabled.Load() {
		return
	}
	// Push to both - they handle their own language filtering
	m.Transcriber.PushAudio(chunk)
	m.Translator.PushAudio(chunk)
}

func DecodeAudioDelta(delta64 string, targetRate int) ([]float32, error) {
	data, err := base64.StdEncoding.DecodeString(delta64)
	if err != nil {
		return nil, err
	}
	srcLen := len(data) / 2
	if srcLen == 0 {
		return []float32{}, nil
	}

	src := make([]float32, srcLen)
	for i := 0; i < srcLen; i++ {
		v := int16(data[i*2]) | int16(data[i*2+1])<<8
		src[i] = float32(v) / 32767.0
	}

	if targetRate <= 0 {
		targetRate = 48000
	}

	ratio := 24000.0 / float64(targetRate)
	dstLen := int(float64(srcLen) / ratio)
	floats := make([]float32, dstLen*2)

	for i := 0; i < dstLen; i++ {
		srcIdx := float64(i) * ratio
		idx0 := int(srcIdx)
		idx1 := idx0 + 1
		if idx1 >= srcLen {
			idx1 = srcLen - 1
		}
		t := srcIdx - float64(idx0)

		var val float32
		if idx0 < srcLen {
			val = src[idx0]*(1.0-float32(t)) + src[idx1]*float32(t)
		}

		floats[i*2] = val     // Left channel
		floats[i*2+1] = val   // Right channel
	}

	return floats, nil
}
