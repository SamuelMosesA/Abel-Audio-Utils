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

func TestAdminPageProtection(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{
		Credentials: map[string]string{"admin": "password"},
	}
	router := setupTestRouter(appState, cfg)

	t.Run("Unauthenticated GET /admin redirects to /login with no-store cache headers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/login", w.Header().Get("Location"))
		assert.Equal(t, "no-store, private", w.Header().Get("Cache-Control"))
		assert.Equal(t, "Cookie", w.Header().Get("Vary"))
		assert.NotContains(t, w.Body.String(), "Admin Dashboard")
	})

	t.Run("Authenticated GET /admin returns 200 with no-store cache headers and admin HTML", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin", nil)
		req.Header.Set("X-Test-Auth", "true")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "no-store, private", w.Header().Get("Cache-Control"))
		assert.Equal(t, "Cookie", w.Header().Get("Vary"))
		assert.Contains(t, w.Body.String(), "Admin Dashboard")
	})
}

func TestSessionEndpoints(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{
		Credentials: map[string]string{"admin": "password"},
	}
	router := setupTestRouter(appState, cfg)

	t.Run("Unauthenticated GET /api/auth/session returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/auth/session", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "no-store, private", w.Header().Get("Cache-Control"))
		assert.Equal(t, "Cookie", w.Header().Get("Vary"))
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, false, resp["authenticated"])
	})

	t.Run("Authenticated GET /api/auth/session returns 200 with session details", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/auth/session", nil)
		req.Header.Set("X-Test-Auth", "true")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "no-store, private", w.Header().Get("Cache-Control"))
		assert.Equal(t, "Cookie", w.Header().Get("Vary"))
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, true, resp["authenticated"])
		assert.Equal(t, "admin", resp["username"])
		assert.Equal(t, "test-session", resp["session_id"])
	})
}

func TestLogoutAndSubsequentAdminAccess(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{
		Credentials: map[string]string{"admin": "password"},
	}
	router := setupTestRouter(appState, cfg)

	// 1. Log in to obtain the session cookie
	body := map[string]string{"username": "admin", "password": "password"}
	jsonBody, _ := json.Marshal(body)
	loginReq, _ := http.NewRequest("POST", "/api/auth/session", bytes.NewBuffer(jsonBody))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	assert.Equal(t, http.StatusOK, loginRec.Code)
	cookies := loginRec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "abel_session" {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie, "Session cookie abel_session must be set after login")

	// 2. Access /admin with cookie -> should succeed
	adminReq, _ := http.NewRequest("GET", "/admin", nil)
	adminReq.AddCookie(sessionCookie)
	adminRec := httptest.NewRecorder()
	router.ServeHTTP(adminRec, adminReq)

	assert.Equal(t, http.StatusOK, adminRec.Code)
	assert.Contains(t, adminRec.Body.String(), "Admin Dashboard")

	// 3. Check /api/auth/session with cookie -> should be authenticated
	statusReq, _ := http.NewRequest("GET", "/api/auth/session", nil)
	statusReq.AddCookie(sessionCookie)
	statusRec := httptest.NewRecorder()
	router.ServeHTTP(statusRec, statusReq)

	assert.Equal(t, http.StatusOK, statusRec.Code)
	var statusResp map[string]any
	json.Unmarshal(statusRec.Body.Bytes(), &statusResp)
	assert.Equal(t, true, statusResp["authenticated"])

	// 4. Logout via DELETE /api/auth/session
	logoutReq, _ := http.NewRequest("DELETE", "/api/auth/session", nil)
	logoutReq.AddCookie(sessionCookie)
	logoutRec := httptest.NewRecorder()
	router.ServeHTTP(logoutRec, logoutReq)

	assert.Equal(t, http.StatusOK, logoutRec.Code)
	assert.Equal(t, "no-store, private", logoutRec.Header().Get("Cache-Control"))
	assert.Equal(t, "Cookie", logoutRec.Header().Get("Vary"))

	// Extract the cookie set after logout (should clear session)
	logoutCookies := logoutRec.Result().Cookies()
	var postLogoutCookie *http.Cookie
	for _, c := range logoutCookies {
		if c.Name == "abel_session" {
			postLogoutCookie = c
			break
		}
	}

	// 5. Subsequent request to /admin with post-logout cookie must redirect to /login
	adminAfterLogoutReq, _ := http.NewRequest("GET", "/admin", nil)
	if postLogoutCookie != nil {
		adminAfterLogoutReq.AddCookie(postLogoutCookie)
	}
	adminAfterLogoutRec := httptest.NewRecorder()
	router.ServeHTTP(adminAfterLogoutRec, adminAfterLogoutReq)

	assert.Equal(t, http.StatusFound, adminAfterLogoutRec.Code)
	assert.Equal(t, "/login", adminAfterLogoutRec.Header().Get("Location"))
	assert.NotContains(t, adminAfterLogoutRec.Body.String(), "Admin Dashboard")

	// 6. Subsequent request to /api/auth/session with post-logout cookie must return 401
	statusAfterLogoutReq, _ := http.NewRequest("GET", "/api/auth/session", nil)
	if postLogoutCookie != nil {
		statusAfterLogoutReq.AddCookie(postLogoutCookie)
	}
	statusAfterLogoutRec := httptest.NewRecorder()
	router.ServeHTTP(statusAfterLogoutRec, statusAfterLogoutReq)

	assert.Equal(t, http.StatusUnauthorized, statusAfterLogoutRec.Code)
}

