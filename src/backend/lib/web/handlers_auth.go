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
	"github.com/gorilla/securecookie"
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
		sessionID := fmt.Sprintf("%x", b)

		session := sessions.Default(c)
		session.Set("authenticated", true)
		session.Set("session_id", sessionID)
		session.Set("username", req.Username)
		session.Save()

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"session": sessionID,
		})
	}
}

// @Summary End auth session
// @Description Logs out, revokes active server-side session, and clears auth cookies
// @Tags Auth
// @Produce json
// @Success 200 {object} object "Logout Success"
// @Router /api/auth/session [delete]
func LogoutHandler(appState *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		if sessionID, ok := session.Get("session_id").(string); ok && sessionID != "" {
			if appState != nil {
				appState.RevokeSession(sessionID)
			}
		}

		session.Clear()
		session.Options(sessions.Options{
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		session.Save()

		// Also explicitly instruct browser to clear cookie via standard response header
		c.SetCookie("abel_session", "", -1, "/", "", false, true)

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	}
}

// @Summary Get current session info
// @Description Returns the authenticated user session details
// @Tags Auth
// @Produce json
// @Success 200 {object} object "Session Info"
// @Router /api/auth/session [get]
func GetSessionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		username := session.Get("username")
		sessionID := session.Get("session_id")

		c.JSON(http.StatusOK, gin.H{
			"status":     "authenticated",
			"username":   username,
			"session":    sessionID,
			"session_id": sessionID,
		})
	}
}

// SessionAuthMiddleware protects routes using Gin sessions and verifies session revocation
func SessionAuthMiddleware(appState *state.AppState) gin.HandlerFunc {
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

// CookieSanitizerMiddleware pre-validates signed session cookies before third-party session middleware.
// If an incoming cookie was signed by an old key or has an invalid signature, this middleware strips
// it from the request header (preventing third-party library ERROR logs) and sets a deletion header
// so the client immediately discards the expired cookie.
func CookieSanitizerMiddleware(cookieName string, sessionSecret []byte) gin.HandlerFunc {
	codec := securecookie.New(sessionSecret, nil)
	return func(c *gin.Context) {
		cookie, err := c.Request.Cookie(cookieName)
		if err == nil && cookie.Value != "" {
			var dst map[interface{}]interface{}
			if err := codec.Decode(cookieName, cookie.Value, &dst); err != nil {
				// Cookie signature or decoding failed (e.g. key rotated or expired).
				// 1. Strip the invalid cookie from the incoming request headers so gin-contrib/sessions sees no cookie.
				cookies := c.Request.Cookies()
				c.Request.Header.Del("Cookie")
				for _, ck := range cookies {
					if ck.Name != cookieName {
						c.Request.AddCookie(ck)
					}
				}

				// 2. Instruct the browser to immediately clear the dead cookie.
				c.SetCookie(cookieName, "", -1, "/", "", false, true)
			}
		}
		c.Next()
	}
}
