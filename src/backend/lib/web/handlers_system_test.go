package web

import (
	"abel/src/backend/lib/audioengine"
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeLogHandler(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{}
	router := setupTestRouter(appState, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "/api/system/changelog", nil)
	req.Header.Set("X-Test-Auth", "true")

	w := httptest.NewRecorder()
	stateObj := appState
	go router.ServeHTTP(w, req)

	time.Sleep(100 * time.Millisecond)
	state.Update[state.RecordIntent](stateObj, state.SectionRecording, func(s *state.RecordIntent) {})
	time.Sleep(100 * time.Millisecond)
	cancel()
}

type stubRestarter struct {
	calls  int
	result audioengine.RestartResult
	err    error
}

func (s *stubRestarter) Restart() (audioengine.RestartResult, error) {
	s.calls++
	return s.result, s.err
}

func postRestart(router http.Handler, authenticated bool) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, "/api/system/restart", nil)
	if authenticated {
		req.Header.Set("X-Test-Auth", "true")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestRestartEngineRequiresLogin(t *testing.T) {
	restarter := &stubRestarter{}
	router := setupTestRouterWithRestarter(state.NewAppState("", ""), &config.Config{}, restarter)

	w := postRestart(router, false)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Zero(t, restarter.calls)
}

func TestRestartEngineReturnsResult(t *testing.T) {
	restarter := &stubRestarter{result: audioengine.RestartResult{
		Devices:          []audioengine.AudioDevice{{ID: 0, Name: "Behringer UMC404HD", In: 4}},
		DeviceID:         0,
		RestoredDevice:   "Behringer UMC404HD",
		ConfigReloaded:   true,
		NeedsFullRestart: []string{"port"},
	}}
	router := setupTestRouterWithRestarter(state.NewAppState("", ""), &config.Config{}, restarter)

	w := postRestart(router, true)
	require.Equal(t, http.StatusOK, w.Code)

	var resp audioengine.RestartResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, restarter.result, resp)
}

func TestRestartEngineConflicts(t *testing.T) {
	for _, err := range []error{audioengine.ErrRecordingInProgress, audioengine.ErrRestartInProgress} {
		router := setupTestRouterWithRestarter(state.NewAppState("", ""), &config.Config{}, &stubRestarter{err: err})

		w := postRestart(router, true)
		assert.Equal(t, http.StatusConflict, w.Code, err.Error())
		assert.Contains(t, w.Body.String(), err.Error())
	}
}

func TestRestartEngineFailure(t *testing.T) {
	router := setupTestRouterWithRestarter(state.NewAppState("", ""), &config.Config{}, &stubRestarter{err: errors.New("reinitialize audio: host error")})

	w := postRestart(router, true)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "host error")
}

func TestRestartEngineUnavailable(t *testing.T) {
	router := setupTestRouterWithRestarter(state.NewAppState("", ""), &config.Config{}, nil)

	w := postRestart(router, true)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// Real Restarter end to end through HTTP: refused while recording.
func TestRestartEngineRefusedWhileRecording(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) { s.SetRecording(true) })
	restarter := audioengine.NewRestarter(appState, &config.Config{}, "")
	restarter.ReinitAudio = func() error { t.Fatal("audio must not be re-initialized while recording"); return nil }
	router := setupTestRouterWithRestarter(appState, &config.Config{}, restarter)

	w := postRestart(router, true)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.True(t, appState.IsRecording(), "recording must be untouched")
}

func TestGetSystemConnection(t *testing.T) {
	cfg := &config.Config{Port: "8080"}
	router := setupTestRouter(state.NewAppState("", ""), cfg)

	req, _ := http.NewRequest("GET", "/api/system/connection", nil)
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["serverUrl"], "8080")
}
