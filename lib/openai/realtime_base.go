package openai

import (
	"context"
	"encoding/base64"
	"sync/atomic"

	"behringerRecorder/lib/config"
	"behringerRecorder/lib/state"
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
	Type         string       `json:"type,omitempty"`
	Modalities   []string     `json:"modalities,omitempty"`
	Instructions string       `json:"instructions,omitempty"`
	Audio        *AudioConfig `json:"audio,omitempty"`
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

func NewOpenAIManager(cfg *config.Config, apiKey, translateModel, transcribeModel, voice, originalLang string, audioInSize, audioOutSize, subtitleSize int) (*OpenAIManager, error) {
	langCode := cfg.ResolveLanguageCode(originalLang)
	transcriber, _ := NewTranscriptionManager(cfg, apiKey, transcribeModel, langCode, audioInSize, audioOutSize, subtitleSize)
	translator, _ := NewTranslationManager(cfg, apiKey, translateModel, voice, langCode, audioInSize, audioOutSize, subtitleSize)
	
	return &OpenAIManager{
		Config:           cfg,
		OriginalLanguage: langCode,
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

func DecodeAudioDelta(delta64 string) ([]float32, error) {
	data, err := base64.StdEncoding.DecodeString(delta64)
	if err != nil {
		return nil, err
	}
	count := len(data) / 2
	// 24kHz Mono -> 48kHz Stereo
	// We need 2x for rate and 2x for channels = 4x
	floats := make([]float32, count*4)
	for i := 0; i < count; i++ {
		v := int16(data[i*2]) | int16(data[i*2+1])<<8
		f := float32(v) / 32767.0
		base := i * 4
		floats[base] = f     // L
		floats[base+1] = f   // R
		floats[base+2] = f   // L (next 48k sample)
		floats[base+3] = f   // R (next 48k sample)
	}
	return floats, nil
}
