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

	"github.com/stretchr/testify/assert"
)

func TestGetAIStreamsStatus(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.AIConfig](appState, state.SectionAI, func(s *state.AIConfig) {
		s.SetEnabled(true)
	})
	cfg := &config.Config{
		AILanguages: []config.AILanguage{
			{Code: "es", Name: "Spanish"},
			{Code: "fr", Name: "French"},
		},
	}
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

	languages, ok := resp["languages"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, languages, 2)
}

func TestToggleLanguageKillswitch(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{
		AILanguages: []config.AILanguage{
			{Code: "es", Name: "Spanish"},
		},
	}
	mockTranslator := &MockTranslator{}
	appState.Translator = mockTranslator
	router := setupTestRouter(appState, cfg)

	// Block Spanish
	blockReqBody := map[string]interface{}{
		"action":   "toggle_language",
		"language": "es",
		"blocked":  true,
	}
	jsonBody, _ := json.Marshal(blockReqBody)
	req, _ := http.NewRequest("POST", "/api/ai/streams", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, appState.IsLanguageBlocked("es"))

	// Status reflects blocked
	statusReq, _ := http.NewRequest("GET", "/api/ai/streams", nil)
	statusReq.Header.Set("X-Test-Auth", "true")
	statusW := httptest.NewRecorder()
	router.ServeHTTP(statusW, statusReq)

	var resp map[string]interface{}
	json.Unmarshal(statusW.Body.Bytes(), &resp)
	languages := resp["languages"].([]interface{})
	lang0 := languages[0].(map[string]interface{})
	assert.Equal(t, "es", lang0["code"])
	assert.Equal(t, true, lang0["blocked"])

	// Unblock Spanish
	unblockReqBody := map[string]interface{}{
		"action":   "toggle_language",
		"language": "es",
		"blocked":  false,
	}
	jsonBody, _ = json.Marshal(unblockReqBody)
	req, _ = http.NewRequest("POST", "/api/ai/streams", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-Test-Auth", "true")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, appState.IsLanguageBlocked("es"))
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
	mockTranslator.SendSubtitle("Hello")
	time.Sleep(100 * time.Millisecond)
	cancel()
}

func TestSubtitlesHandlerRejectsBlockedLanguage(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.AIConfig](appState, state.SectionAI, func(s *state.AIConfig) {
		s.SetEnabled(true)
		s.SetBlocked("es", true)
	})
	mockTranslator := &MockTranslator{}
	appState.Translator = mockTranslator
	router := setupTestRouter(appState, testCfg)

	req, _ := http.NewRequest("GET", "/subtitles/es", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "blocked")
}
