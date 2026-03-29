package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/state"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestStreamHandler(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{SampleRate: 48000}
	router := setupTestRouter(appState, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "/stream", nil)
	w := httptest.NewRecorder()
	go router.ServeHTTP(w, req)

	time.Sleep(100 * time.Millisecond)
	found := false
	appState.StreamChannels.Range(func(key, value interface{}) bool {
		found = true
		ch := key.(chan []float32)
		ch <- []float32{0.1, 0.1}
		return false
	})
	assert.True(t, found)
	time.Sleep(100 * time.Millisecond)
	cancel()
}
