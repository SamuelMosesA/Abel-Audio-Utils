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
	cfg := &config.Config{
		StorageLocation: t.TempDir(),
	}
	router := setupTestRouter(appState, cfg)

	t.Run("Unauthorized Access", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/recordings/files", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestSessionSecretValidation(t *testing.T) {
	appState := state.NewAppState("", "")
	storageDir := t.TempDir()
	cfgA := &config.Config{
		Credentials:     map[string]string{"admin": "password"},
		SessionSecret:   "secret-key-instance-a-0123456789",
		StorageLocation: storageDir,
	}
	routerA := setupTestRouter(appState, cfgA)

	// 1. Log in on Router A to acquire valid session cookie
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password"})
	loginReq, _ := http.NewRequest("POST", "/api/auth/session", bytes.NewBuffer(loginBody))
	loginRec := httptest.NewRecorder()
	routerA.ServeHTTP(loginRec, loginReq)
	assert.Equal(t, http.StatusOK, loginRec.Code)

	cookie := loginRec.Header().Get("Set-Cookie")
	assert.NotEmpty(t, cookie)

	// 2. Request protected route on Router A using this cookie -> 200 OK
	reqA, _ := http.NewRequest("GET", "/api/recordings/files", nil)
	reqA.Header.Set("Cookie", cookie)
	recA := httptest.NewRecorder()
	routerA.ServeHTTP(recA, reqA)
	assert.Equal(t, http.StatusOK, recA.Code)

	// 3. Request protected route on Router B with DIFFERENT secret using cookie from Router A -> 401 Unauthorized
	cfgB := &config.Config{
		Credentials:     map[string]string{"admin": "password"},
		SessionSecret:   "secret-key-instance-b-9876543210",
		StorageLocation: storageDir,
	}
	routerB := setupTestRouter(appState, cfgB)

	reqB, _ := http.NewRequest("GET", "/api/recordings/files", nil)
	reqB.Header.Set("Cookie", cookie)
	recB := httptest.NewRecorder()
	routerB.ServeHTTP(recB, reqB)
	assert.Equal(t, http.StatusUnauthorized, recB.Code)

	// 4. Request protected route on a router initialized with legacy "secret" -> 401 Unauthorized
	cfgLegacy := &config.Config{
		SessionSecret:   "secret",
		StorageLocation: storageDir,
	}
	routerLegacy := setupTestRouter(appState, cfgLegacy)
	reqLegacy, _ := http.NewRequest("GET", "/api/recordings/files", nil)
	reqLegacy.Header.Set("Cookie", cookie)
	recLegacy := httptest.NewRecorder()
	routerLegacy.ServeHTTP(recLegacy, reqLegacy)
	assert.Equal(t, http.StatusUnauthorized, recLegacy.Code)
}
