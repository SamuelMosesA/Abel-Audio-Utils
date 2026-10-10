package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAudioConfig(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{}
	router := setupTestRouter(appState, cfg)

	body := map[string]interface{}{"chL": 1, "chR": 2, "boost": 2.5}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("PATCH", "/api/audio/config", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int32(1), appState.Config().ChL())
	assert.Equal(t, 2.5, appState.Config().Boost())
}

type stubAudioBroadcaster struct {
	subscribedLang string
	chunk          []byte
}

func (s *stubAudioBroadcaster) Subscribe(language string, sampleRate int, source <-chan []float32) (<-chan []byte, func(), error) {
	s.subscribedLang = language
	ch := make(chan []byte, 2)
	ch <- s.chunk
	return ch, func() { close(ch) }, nil
}

func TestStreamHandler_ProgressiveMP3Contract(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{SampleRate: 48000}
	mp3Payload := []byte{0xFF, 0xFB, 0x90, 0x64, 0x00, 0x01, 0x02}
	broadcaster := &stubAudioBroadcaster{chunk: mp3Payload}

	router := gin.New()
	router.GET("/api/audio/stream", StreamHandler(appState, cfg, broadcaster))
	router.GET("/api/audio/stream/*lang", StreamHandler(appState, cfg, broadcaster))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/api/audio/stream", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "audio/mpeg", w.Header().Get("Content-Type"))
	assert.Equal(t, "chunked", w.Header().Get("Transfer-Encoding"))
	assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	assert.Contains(t, w.Body.Bytes(), mp3Payload[0])
	assert.Equal(t, "default", broadcaster.subscribedLang)
}

func TestStreamHandler_RejectsBlockedLanguage(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{
		SampleRate:         48000,
		AIOriginalLanguage: "en",
		AILanguages: []config.AILanguage{
			{Code: "es", Name: "Spanish"},
		},
	}
	state.Update[state.AIConfig](appState, state.SectionAI, func(s *state.AIConfig) {
		s.SetBlocked("es", true)
	})
	broadcaster := &stubAudioBroadcaster{chunk: []byte{0xFF}}

	router := gin.New()
	router.GET("/api/audio/stream/*lang", StreamHandler(appState, cfg, broadcaster))

	req := httptest.NewRequest(http.MethodGet, "/api/audio/stream/es", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "blocked")
}

func TestRestartAudioEngineRequiresLogin(t *testing.T) {
	router := setupTestRouter(state.NewAppState("", ""), &config.Config{})

	req, _ := http.NewRequest(http.MethodPost, "/api/audio/restart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRestartAudioEngineRefusedWhileRecording(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) { s.SetRecording(true) })
	router := setupTestRouter(appState, &config.Config{})

	req, _ := http.NewRequest(http.MethodPost, "/api/audio/restart", nil)
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "while recording")
	assert.True(t, appState.IsRecording(), "recording must be untouched")
}

func TestRestartAudioEngineRejectsConcurrentInFlightRequests(t *testing.T) {
	appState := state.NewAppState("", "")
	router := setupTestRouter(appState, &config.Config{})

	restartingAudio.Store(true)
	defer restartingAudio.Store(false)

	req, _ := http.NewRequest(http.MethodPost, "/api/audio/restart", nil)
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already in progress")
}

func TestUpdateAudioConfigInvalidDeviceDoesNotPanic(t *testing.T) {
	appState := state.NewAppState("", "")
	router := setupTestRouter(appState, &config.Config{})

	jsonBody, _ := json.Marshal(map[string]interface{}{"deviceID": -1})
	req, _ := http.NewRequest("PATCH", "/api/audio/config", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, appState.Config().IsRunning())
	assert.False(t, appState.Engine().IsRunning())
}
