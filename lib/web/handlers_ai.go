package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/state"
	"fmt"
	"log"
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
				fmt.Printf("[AI] Toggling master to: %v (Current state: %v)\n", *req.Enabled, s.IsEnabled())
				s.SetEnabled(*req.Enabled)
				if !*req.Enabled {
					shouldCloseAll = true
				}
				fmt.Printf("[AI] Master state now: %v\n", s.IsEnabled())
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
				fmt.Printf("[AI] Stopping translation for: %s (resolved: %s)\n", req.Language, resolved)
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

		log.Printf("[SERVER] Subtitles connection requested for lang: %s (name: %s) (IP: %s)", lang, resolvedLang, c.Request.RemoteAddr)

		if appState.Translator == nil {
			log.Printf("[SERVER] Subtitles handler aborted: Translator is nil")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Translation not available"})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		ch, cleanup := appState.Translator.GetSubtitles(lang)
		if ch == nil {
			log.Printf("[SERVER] Failed to get subtitle channel for lang: %s (resolved: %s)", lang, resolvedLang)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subtitle channel"})
			return
		}
		defer func() {
			log.Printf("[SERVER] Subtitles connection closed for lang: %s (resolved: %s)", lang, resolvedLang)
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
					log.Printf("[SERVER] Subtitles channel closed for lang: %s (resolved: %s)", lang, resolvedLang)
					return
				}
				log.Printf("[SERVER] Sending subtitle to %s (%s): %s", lang, resolvedLang, text)
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
