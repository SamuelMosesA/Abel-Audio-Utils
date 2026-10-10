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

type LanguageStatus struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Blocked   bool   `json:"blocked"`
	Active    bool   `json:"active"`
	Listeners int    `json:"listeners"`
}

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
			Blocked   *bool  `json:"blocked"`
			Subtitles bool   `json:"subtitles"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		var shouldCloseAll bool
		var shouldStopLanguage bool
		var resolvedStopLanguage string

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

			if req.Action == "toggle_language" && req.Language != "" && req.Blocked != nil {
				code := cfg.ResolveLanguageCode(req.Language)
				s.SetBlocked(code, *req.Blocked)
				s.SetBlocked(req.Language, *req.Blocked)
				if *req.Blocked {
					shouldStopLanguage = true
					resolvedStopLanguage = cfg.ResolveLanguageName(req.Language)
				}
				logger.Info("Language killswitch toggled",
					slog.String("ai.language", req.Language),
					slog.Bool("ai.blocked", *req.Blocked),
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

		if shouldStopLanguage && appState.Translator != nil {
			appState.Translator.StopSession(resolvedStopLanguage, true)
			appState.Translator.StopSession(req.Language, true)
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
// @Description Returns active sessions, configured languages, and master state
// @Tags AI
// @Produce json
// @Success 200 {object} object "AI Streams Status"
// @Failure 401 {object} string "Unauthorized"
// @Router /api/ai/streams [get]
func GetAIStreamsStatus(appState *state.AppState, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{
			"masterEnabled": appState.AI().IsEnabled(),
		}
		var sessions []state.SessionInfo
		if appState.Translator != nil {
			sessions = appState.Translator.ListSessions()
		} else {
			sessions = []state.SessionInfo{}
		}
		status["sessions"] = sessions

		languages := make([]LanguageStatus, 0)
		if cfg != nil {
			activeLanguages := make(map[string]bool)
			for _, s := range sessions {
				activeLanguages[strings.ToLower(s.Language)] = true
				if code := cfg.ResolveLanguageCode(s.Language); code != "" {
					activeLanguages[strings.ToLower(code)] = true
				}
			}

			aiState := appState.AI()
			for _, lang := range cfg.AILanguages {
				codeLower := strings.ToLower(lang.Code)
				nameLower := strings.ToLower(lang.Name)
				blocked := aiState.IsBlocked(lang.Code) || aiState.IsBlocked(lang.Name)
				active := activeLanguages[codeLower] || activeLanguages[nameLower]

				listeners := 0
				if appState.Translator != nil {
					listeners = appState.Translator.GetListenerCount(lang.Code)
					if listeners == 0 && lang.Code != lang.Name {
						listeners = appState.Translator.GetListenerCount(lang.Name)
					}
				}

				languages = append(languages, LanguageStatus{
					Code:      lang.Code,
					Name:      lang.Name,
					Blocked:   blocked,
					Active:    active,
					Listeners: listeners,
				})
			}
		}
		status["languages"] = languages

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

		if appState.IsLanguageBlocked(lang) || appState.IsLanguageBlocked(resolvedLang) {
			connLogger.Warn("Subtitles connection rejected: language is blocked via killswitch")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Language translation is blocked"})
			return
		}

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
