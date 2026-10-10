package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"admin"`
}

// @Summary Create auth session
// @Description Logs in and establishes a session
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login Credentials"
// @Success 200 {object} object "Login Success"
// @Router /api/auth/session [post]
func LoginHandler(cfg *config.Config, appState *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if req.Username == "" {
			req.Username = "admin" // Default if empty
		}

		authorized := false
		if p, ok := cfg.Credentials[req.Username]; ok && p == req.Password {
			authorized = true
		}

		if !authorized {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		// Generate new session ID
		b := make([]byte, 16)
		rand.Read(b)
		newSessionID := fmt.Sprintf("%x", b)

		// Set Gin Session
		session := sessions.Default(c)
		session.Set("authenticated", true)
		session.Set("username", req.Username)
		session.Set("session_id", newSessionID)
		session.Save()

		c.JSON(http.StatusOK, gin.H{"status": "success", "session": newSessionID})
	}
}

// @Summary Terminate auth session
// @Description Logs out, invalidates cookie, and revokes session ID server-side
// @Tags Auth
// @Produce json
// @Success 200 {object} object "Logout Success"
// @Router /api/auth/session [delete]
// @Router /api/auth/logout [post]
func LogoutHandler(appState *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		if sessID, ok := session.Get("session_id").(string); ok && sessID != "" {
			if appState != nil {
				appState.RevokeSession(sessID)
			}
		}

		session.Options(sessions.Options{
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		session.Clear()
		session.Save()

		c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
	}
}

// SessionAuthMiddleware protects routes using Gin sessions and verifies session revocation
func SessionAuthMiddleware(stateObj ...*state.AppState) gin.HandlerFunc {
	var appState *state.AppState
	if len(stateObj) > 0 {
		appState = stateObj[0]
	}

	logger := slog.With("component", "auth")
	return func(c *gin.Context) {
		session := sessions.Default(c)
		auth := session.Get("authenticated")
		reqLogger := logger.With(
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)
		if auth != true {
			reqLogger.Warn("Auth request denied",
				slog.String("reason", "no authenticated session"),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized session"})
			c.Abort()
			return
		}

		sessionID, _ := session.Get("session_id").(string)
		if appState != nil && sessionID != "" && appState.IsSessionRevoked(sessionID) {
			reqLogger.Warn("Auth request denied",
				slog.String("reason", "session has been revoked"),
				slog.String("session_id", sessionID),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session revoked"})
			c.Abort()
			return
		}

		user := session.Get("username")
		reqLogger.Info("Auth request granted",
			slog.Any("user", user),
		)
		c.Next()
	}
}
