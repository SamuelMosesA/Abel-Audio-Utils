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

func TestStreamHandlerRedirectsToHLS(t *testing.T) {
	router := gin.New()
	router.GET("/api/audio/stream/*lang", StreamHandler())

	req, _ := http.NewRequest(http.MethodGet, "/api/audio/stream/fr", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/api/audio/hls/fr/index.m3u8", w.Header().Get("Location"))
}

type stubHLSProvider struct {
	ensuredLanguage string
	playlist        []byte
	segment         []byte
}

func (s *stubHLSProvider) EnsureStream(language string, sampleRate int, source <-chan []float32) error {
	s.ensuredLanguage = language
	return nil
}

func (s *stubHLSProvider) WaitForPlaylist(ctx context.Context, language string, timeout time.Duration) ([]byte, error) {
	return s.playlist, nil
}

func (s *stubHLSProvider) ReadSegment(language, name string) ([]byte, error) {
	return s.segment, nil
}

func TestHLSHandlersServeSafariCompatibleContract(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{SampleRate: 48000, AIOriginalLanguage: "en"}
	provider := &stubHLSProvider{
		playlist: []byte("#EXTM3U\n#EXT-X-VERSION:3\n#EXTINF:1.0,\nsegment-000000001.ts\n"),
		segment:  []byte{0x47, 0x40, 0x00},
	}
	router := gin.New()
	router.GET("/api/audio/hls/:lang/index.m3u8", HLSPlaylistHandler(appState, cfg, provider))
	router.GET("/api/audio/hls/:lang/:segment", HLSSegmentHandler(provider))

	playlistRequest := httptest.NewRequest(http.MethodGet, "/api/audio/hls/default/index.m3u8", nil)
	playlistResponse := httptest.NewRecorder()
	router.ServeHTTP(playlistResponse, playlistRequest)

	require.Equal(t, http.StatusOK, playlistResponse.Code)
	assert.Equal(t, "application/vnd.apple.mpegurl", playlistResponse.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", playlistResponse.Header().Get("Cache-Control"))
	assert.Contains(t, playlistResponse.Body.String(), "#EXTM3U")
	assert.Equal(t, "default", provider.ensuredLanguage)

	segmentRequest := httptest.NewRequest(http.MethodGet, "/api/audio/hls/default/segment-000000001.ts", nil)
	segmentResponse := httptest.NewRecorder()
	router.ServeHTTP(segmentResponse, segmentRequest)

	require.Equal(t, http.StatusOK, segmentResponse.Code)
	assert.Equal(t, "video/mp2t", segmentResponse.Header().Get("Content-Type"))
	assert.Equal(t, provider.segment, segmentResponse.Body.Bytes())
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

func TestHLSPlaylistHandlerRejectsBlockedLanguage(t *testing.T) {
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
	provider := &stubHLSProvider{}
	router := gin.New()
	router.GET("/api/audio/hls/:lang/index.m3u8", HLSPlaylistHandler(appState, cfg, provider))

	req := httptest.NewRequest(http.MethodGet, "/api/audio/hls/es/index.m3u8", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "blocked")
}
