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

func TestGetAIStreamsStatus(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.AIConfig](appState, state.SectionAI, func(s *state.AIConfig) {
		s.SetEnabled(true)
	})
	cfg := &config.Config{}
	mockTranslator := &MockTranslator{}
	appState.Translator = mockTranslator
	router := setupTestRouter(appState, cfg)

	req, _ := http.NewRequest("GET", "/api/ai/streams", nil)
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, true, resp["masterEnabled"])
	assert.NotEmpty(t, resp["sessions"])
}

func TestSubtitlesHandler(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.AIConfig](appState, state.SectionAI, func(s *state.AIConfig) {
		s.SetEnabled(true)
	})
	mockTranslator := &MockTranslator{}
	appState.Translator = mockTranslator
	router := setupTestRouter(appState, testCfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "/subtitles/en", nil)
	w := httptest.NewRecorder()
	go router.ServeHTTP(w, req)

	time.Sleep(100 * time.Millisecond)
	if mockTranslator.subtitleChan != nil {
		mockTranslator.subtitleChan <- "Hello"
	}
	time.Sleep(100 * time.Millisecond)
	cancel()
}
