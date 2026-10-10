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

func TestGetSessionHandler(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{
		Credentials: map[string]string{"admin": "password"},
	}
	router := setupTestRouter(appState, cfg)

	t.Run("Unauthenticated request returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/auth/session", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Unauthorized session")
	})

	t.Run("Authenticated session returns 200 with session info", func(t *testing.T) {
		// Log in first
		body := map[string]string{"username": "admin", "password": "password"}
		jsonBody, _ := json.Marshal(body)
		loginReq, _ := http.NewRequest("POST", "/api/auth/session", bytes.NewBuffer(jsonBody))
		loginW := httptest.NewRecorder()
		router.ServeHTTP(loginW, loginReq)

		assert.Equal(t, http.StatusOK, loginW.Code)
		var loginResp map[string]string
		json.Unmarshal(loginW.Body.Bytes(), &loginResp)
		expectedSessID := loginResp["session"]
		cookie := loginW.Header().Get("Set-Cookie")
		assert.NotEmpty(t, cookie)

		// Check session
		req, _ := http.NewRequest("GET", "/api/auth/session", nil)
		req.Header.Set("Cookie", cookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "authenticated", resp["status"])
		assert.Equal(t, "admin", resp["username"])
		assert.Equal(t, expectedSessID, resp["session"])
	})

	t.Run("Revoked session returns 401", func(t *testing.T) {
		// Log in
		body := map[string]string{"username": "admin", "password": "password"}
		jsonBody, _ := json.Marshal(body)
		loginReq, _ := http.NewRequest("POST", "/api/auth/session", bytes.NewBuffer(jsonBody))
		loginW := httptest.NewRecorder()
		router.ServeHTTP(loginW, loginReq)

		var loginResp map[string]string
		json.Unmarshal(loginW.Body.Bytes(), &loginResp)
		sessID := loginResp["session"]
		cookie := loginW.Header().Get("Set-Cookie")

		// Revoke session server-side
		appState.RevokeSession(sessID)

		// Check session
		req, _ := http.NewRequest("GET", "/api/auth/session", nil)
		req.Header.Set("Cookie", cookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Session revoked")
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

func TestLogoutHandlerAndRevocation(t *testing.T) {
	appState := state.NewAppState("", "")
	storageDir := t.TempDir()
	cfg := &config.Config{
		Credentials:     map[string]string{"admin": "password"},
		StorageLocation: storageDir,
	}
	router := setupTestRouter(appState, cfg)

	t.Run("Logout with DELETE /api/auth/session revokes token and clears cookie", func(t *testing.T) {
		// 1. Log in
		loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password"})
		loginReq, _ := http.NewRequest("POST", "/api/auth/session", bytes.NewBuffer(loginBody))
		loginRec := httptest.NewRecorder()
		router.ServeHTTP(loginRec, loginReq)
		assert.Equal(t, http.StatusOK, loginRec.Code)
		var loginResp map[string]string
		json.Unmarshal(loginRec.Body.Bytes(), &loginResp)
		sessID := loginResp["session"]
		assert.NotEmpty(t, sessID)
		cookie := loginRec.Header().Get("Set-Cookie")

		// 2. Access protected route -> 200 OK
		reqProtected, _ := http.NewRequest("GET", "/api/recordings/files", nil)
		reqProtected.Header.Set("Cookie", cookie)
		recProtected := httptest.NewRecorder()
		router.ServeHTTP(recProtected, reqProtected)
		assert.Equal(t, http.StatusOK, recProtected.Code)

		// 3. Logout via DELETE /api/auth/session
		logoutReq, _ := http.NewRequest("DELETE", "/api/auth/session", nil)
		logoutReq.Header.Set("Cookie", cookie)
		logoutRec := httptest.NewRecorder()
		router.ServeHTTP(logoutRec, logoutReq)
		assert.Equal(t, http.StatusOK, logoutRec.Code)
		var logoutResp map[string]string
		json.Unmarshal(logoutRec.Body.Bytes(), &logoutResp)
		assert.Equal(t, "logged_out", logoutResp["status"])

		// Verify cookie was cleared
		clearedCookie := logoutRec.Header().Get("Set-Cookie")
		assert.NotEmpty(t, clearedCookie)

		// 4. Verify server-side revocation on AppState
		assert.True(t, appState.IsSessionRevoked(sessID))

		// 5. Replaying original session cookie must now be rejected with 401 Unauthorized
		replayReq, _ := http.NewRequest("GET", "/api/recordings/files", nil)
		replayReq.Header.Set("Cookie", cookie)
		replayRec := httptest.NewRecorder()
		router.ServeHTTP(replayRec, replayReq)
		assert.Equal(t, http.StatusUnauthorized, replayRec.Code)
		assert.Contains(t, replayRec.Body.String(), "Session revoked")
	})

	t.Run("Logout with no active session succeeds safely", func(t *testing.T) {
		logoutReq, _ := http.NewRequest("DELETE", "/api/auth/session", nil)
		logoutRec := httptest.NewRecorder()
		router.ServeHTTP(logoutRec, logoutReq)
		assert.Equal(t, http.StatusOK, logoutRec.Code)
	})
}
