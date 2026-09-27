package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"crypto/rand"
	"fmt"
	"io/fs"
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

// SessionAuthMiddleware protects routes using Gin sessions
func SessionAuthMiddleware() gin.HandlerFunc {
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
		user := session.Get("username")
		reqLogger.Info("Auth request granted",
			slog.Any("user", user),
		)
		c.Next()
	}
}

// @Summary Get auth session status
// @Description Checks if the current session is authenticated
// @Tags Auth
// @Produce json
// @Success 200 {object} object "Session valid"
// @Failure 401 {object} object "Unauthorized"
// @Router /api/auth/session [get]
func GetSessionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, private")
		c.Header("Vary", "Cookie")

		session := sessions.Default(c)
		auth := session.Get("authenticated")
		if auth != true {
			c.JSON(http.StatusUnauthorized, gin.H{"authenticated": false, "error": "Unauthorized session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"authenticated": true,
			"username":      session.Get("username"),
			"session_id":    session.Get("session_id"),
		})
	}
}

// @Summary Terminate auth session
// @Description Logs out and invalidates the session
// @Tags Auth
// @Produce json
// @Success 200 {object} object "Logout Success"
// @Router /api/auth/session [delete]
func LogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, private")
		c.Header("Vary", "Cookie")

		session := sessions.Default(c)
		session.Clear()
		session.Options(sessions.Options{
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})
		if err := session.Save(); err != nil {
			slog.Error("Failed to clear session during logout", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
	}
}

// AdminPageHandler serves admin.html if the session is authenticated,
// otherwise redirecting to /login with cache prevention headers.
func AdminPageHandler(fsys fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, private")
		c.Header("Vary", "Cookie")

		session := sessions.Default(c)
		auth := session.Get("authenticated")
		if auth != true {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		data, err := fs.ReadFile(fsys, "admin.html")
		if err != nil {
			c.String(http.StatusNotFound, "File not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	}
}

