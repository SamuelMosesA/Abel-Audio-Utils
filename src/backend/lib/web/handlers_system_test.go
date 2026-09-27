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
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["serverUrl"], "8080")
}
