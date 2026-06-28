package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"abel/src/backend/lib/telemetry"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary Control AI streams
// @Description Toggles master AI switch or stops a specific language translation
// @Tags AI
// @Accept json
// @Produce json
// @Param request body object true "AI Action"
// @Success 200 {object} string "Success"
// @Failure 400 {object} string "Invalid Action"
// @Failure 401 {object} string "Unauthorized"
// @Security CookieAuth
// @Security BasicAuth
// @Router /api/ai/streams [post]
func UpdateAIStreams(appState *state.AppState, cfg *config.Config) gin.HandlerFunc {
	logger := slog.With("component", "ai")
	return func(c *gin.Context) {
		var req struct {
			Action    string `json:"action"`
			Enabled   *bool  `json:"enabled"`
			Language  string `json:"language"`
			Subtitles bool   `json:"subtitles"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		var shouldCloseAll bool
		state.Update[state.AIConfig](appState, state.SectionAI, func(s *state.AIConfig) {
			if req.Action == "toggle_master" && req.Enabled != nil {
				logger.Info("Toggling master AI state",
					slog.Bool("ai.enabled", *req.Enabled),
					slog.Bool("ai.current_state", s.IsEnabled()),
				)
				s.SetEnabled(*req.Enabled)
				if !*req.Enabled {
					shouldCloseAll = true
				}
				logger.Info("Master AI state updated",
					slog.Bool("ai.enabled", s.IsEnabled()),
				)
			}
		})

		// Perform side-effects OUTSIDE of the lock
		if req.Action == "toggle_master" && req.Enabled != nil && appState.Translator != nil {
			appState.Translator.SetEnabled(*req.Enabled)
			if shouldCloseAll {
				appState.Translator.CloseAll()
			}
		}

		if req.Action == "stop_translation" && req.Language != "" {
			if appState.Translator != nil {
				resolved := cfg.ResolveLanguageName(req.Language)
				logger.Info("Stopping translation",
					slog.String("ai.language", req.Language),
					slog.String("ai.resolved", resolved),
				)
				appState.Translator.StopSession(resolved, true)
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "AI action completed"})
	}
}

// @Summary Get AI streams status
// @Description Returns active sessions and master state
// @Tags AI
// @Produce json
// @Success 200 {object} object "AI Streams Status"
// @Failure 401 {object} string "Unauthorized"
// @Router /api/ai/streams [get]
func GetAIStreamsStatus(appState *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{
			"masterEnabled": appState.AI().IsEnabled(),
		}
		if appState.Translator != nil {
			status["sessions"] = appState.Translator.ListSessions()
		} else {
			status["sessions"] = []state.SessionInfo{}
		}
		c.JSON(http.StatusOK, status)
	}
}

// @Summary Get AI configuration
// @Description Returns the list of configured languages and the original language
// @Tags AI
// @Produce json
// @Success 200 {object} object "AI Configuration"
// @Failure 401 {object} string "Unauthorized"
// @Router /api/ai/config [get]
func GetAIConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"languages":        cfg.AILanguages,
			"originalLanguage": cfg.AIOriginalLanguage,
		})
	}
}

// @Summary Get subtitles stream
// @Description Real-time SSE stream of subtitles for a specific language
// @Tags AI
// @Produce text/event-stream
// @Param lang query string false "Language"
// @Success 200 {object} string "SSE Stream"
// @Router /api/ai/subtitles [get]
func SubtitlesHandler(appState *state.AppState, cfg *config.Config) gin.HandlerFunc {
	logger := slog.With("component", "server")
	return func(c *gin.Context) {
		lang := c.Query("lang")
		if lang == "" {
			lang = c.Param("lang")
			lang = strings.TrimPrefix(lang, "/")
		}

		// Use the language code directly
		if lang == "" || lang == "default" || lang == "subtitles" {
			lang = cfg.AIOriginalLanguage
		}
		resolvedLang := cfg.ResolveLanguageName(lang)

		connLogger := logger.With(
			slog.String("client.language", lang),
			slog.String("client.resolved_language", resolvedLang),
		)

		connLogger.Info("Subtitles connection requested",
			slog.String("client.ip", c.Request.RemoteAddr),
		)

		if appState.Translator == nil {
			connLogger.Error("Subtitles handler aborted: Translator is nil")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Translation not available"})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		ch, cleanup := appState.Translator.GetSubtitles(lang)
		if ch == nil {
			connLogger.Error("Failed to get subtitle channel")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subtitle channel"})
			return
		}
		defer func() {
			connLogger.Info("Subtitles connection closed")
			cleanup()
		}()

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
			return
		}

		// Initial keep-alive or state check
		if !appState.AI().IsEnabled() {
			fmt.Fprintf(c.Writer, "data: %s\n\n", `{"error": "AI Master Switch is OFF"}`)
			flusher.Flush()
		}

		for {
			select {
			case text, ok := <-ch:
				if !ok {
					connLogger.Info("Subtitles channel closed")
					return
				}
				if telemetry.SubtitlesSent != nil {
					telemetry.SubtitlesSent.Add(c.Request.Context(), 1)
				}
				fmt.Fprintf(c.Writer, "data: %s\n\n", text)
				flusher.Flush()
			case <-c.Request.Context().Done():
				return
			case <-time.After(30 * time.Second):
				// Keep-alive
				fmt.Fprintf(c.Writer, ": keep-alive\n\n")
				flusher.Flush()
			}
		}
	}
}
