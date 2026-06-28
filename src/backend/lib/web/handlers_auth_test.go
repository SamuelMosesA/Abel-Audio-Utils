package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginHandler(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{
		Credentials: map[string]string{"admin": "password"},
	}
	router := setupTestRouter(appState, cfg)

	t.Run("Successful Login", func(t *testing.T) {
		body := map[string]string{"username": "admin", "password": "password"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/auth/session", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "success", resp["status"])
		assert.NotEmpty(t, resp["session"])
	})

	t.Run("Failed Login", func(t *testing.T) {
		body := map[string]string{"username": "admin", "password": "wrong"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/auth/session", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAdminRoutesProtection(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{}
	router := setupTestRouter(appState, cfg)

	t.Run("Unauthorized Access", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/recordings/files", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
