package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"context"
	"encoding/json"
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

func TestGetSystemConnection(t *testing.T) {
	cfg := &config.Config{Port: "8080"}
	router := setupTestRouter(state.NewAppState("", ""), cfg)

	req, _ := http.NewRequest("GET", "/api/system/connection", nil)
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp SystemConnectionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Contains(t, resp.ServerURL, "8080")
	assert.Equal(t, "8080", resp.Port)
	assert.NotEmpty(t, resp.Host)
	assert.Equal(t, resp.Host+":8080", resp.DisplayEndpoint)
}
