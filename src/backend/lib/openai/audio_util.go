package openai

import (
	"abel/src/backend/lib/audioengine/audio_processing"
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
)

// ResolveSampleRate determines the current active hardware sample rate,
// falling back to configured sample rate or standard default if unavailable.
func ResolveSampleRate(appState *state.AppState, cfg *config.Config) int {
	if appState != nil {
		if rate := int(appState.Config().SampleRate()); rate > 0 {
			return rate
		}
	}
	if cfg != nil && cfg.SampleRate > 0 {
		return cfg.SampleRate
	}
	return audio_processing.DefaultWavSampleRate
}

// DownsampleChunkForAI converts an incoming stereo float32 audio chunk to 24 kHz mono 16-bit PCM bytes
// suitable for OpenAI Realtime speech recognition and translation.
func DownsampleChunkForAI(chunk []float32, appState *state.AppState, cfg *config.Config) []byte {
	srcRate := ResolveSampleRate(appState, cfg)
	return audio_processing.DownsampleStereoToMonoPCM24k(chunk, srcRate)
}

// DecodeAIDelta decodes a base64-encoded OpenAI audio delta into stereo float32 samples at the active engine sample rate.
func DecodeAIDelta(delta64 string, appState *state.AppState, cfg *config.Config) ([]float32, error) {
	targetRate := ResolveSampleRate(appState, cfg)
	return audio_processing.DecodeAudioDelta(delta64, targetRate)
}
