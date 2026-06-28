package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(appState *state.AppState, cfg *config.Config, staticFiles embed.FS) *gin.Engine {
	// Switch from default to release mode by default, standard logger in gin is noisy
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Custom compact logger format
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[API] %v | %3d | %13v | %15s | %-7s %#v %s\n",
			param.TimeStamp.Format(time.RFC1123),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	}))

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true if using HTTPS
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("abel_session", store))

	subFS, _ := fs.Sub(staticFiles, "static")
	content := http.FS(subFS)

	// Custom subFS for _app to ensure paths match (e.g. /_app/immutable should look for immutable in the sub-FS)
	appFS, _ := fs.Sub(staticFiles, "static/_app")
	appContent := http.FS(appFS)

	// Static assets using standard Gin helpers where possible
	r.StaticFileFS("/favicon.png", "favicon.png", content)
	r.StaticFS("/_app", appContent)

	// Helper for serving HTML files without redirects
	serveHTML := func(filename string) gin.HandlerFunc {
		return func(c *gin.Context) {
			data, err := fs.ReadFile(subFS, filename)
			if err != nil {
				c.String(http.StatusNotFound, "File not found")
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		}
	}

	// Multi-page entry points (Prerendered)
	r.GET("/", serveHTML("index.html"))
	r.GET("/admin", serveHTML("admin.html"))
	r.GET("/login", serveHTML("login.html"))
	r.GET("/stream", serveHTML("stream.html"))

	// Consolidated AI Live Audio route - serves index.html for any subpath
	r.GET("/ai_live_audio/*any", serveHTML("index.html"))

	// SPA fallback for dynamic routes
	r.NoRoute(serveHTML("index.html"))

	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// RESTful API Handlers
	api := r.Group("/api")
	{
		// Auth
		api.POST("/auth/session", LoginHandler(cfg, appState))

		// Audio
		audio := api.Group("/audio")
		{
			audio.GET("/devices", DevicesHandler(appState))
			audio.GET("/config", GetAudioConfig(appState))
			audio.GET("/stream", StreamHandler(appState, cfg))
			audio.GET("/stream/*lang", StreamHandler(appState, cfg))
		}

		// Recordings
		recordings := api.Group("/recordings")
		{
			// Public status
			recordings.GET("", GetRecordingStatus(appState))
		}

		// AI
		ai := api.Group("/ai")
		{
			ai.GET("/subtitles", SubtitlesHandler(appState, cfg))
			ai.GET("/subtitles/*lang", SubtitlesHandler(appState, cfg))
			ai.GET("/streams", GetAIStreamsStatus(appState))
			ai.GET("/config", GetAIConfig(cfg))
		}

		// System
		system := api.Group("/system")
		{
			system.GET("/connection", GetSystemConnection(cfg))
		}

		// Telemetry
		api.POST("/telemetry/errors", ErrorLogHandler())

		// Admin/Protected routes (session authenticated)
		RegisterAdminRoutes(api, appState, cfg)
	}

	r.GET("/ws", NewWSHandler(appState, cfg))

	return r
}

func RegisterAdminRoutes(r *gin.RouterGroup, appState *state.AppState, cfg *config.Config) {
	r.Use(SessionAuthMiddleware())
	{
		r.PATCH("/audio/config", UpdateAudioConfig(appState, cfg))
		r.POST("/recordings", CreateRecording(appState, cfg))
		r.POST("/ai/streams", UpdateAIStreams(appState, cfg))
		r.GET("/system/changelog", ChangeLogHandler(appState))
		r.GET("/recordings/files", ListRecordingFiles(cfg))
		r.POST("/recordings/push", PushRecordingToCloud(cfg))
		r.StaticFS("/recordings/raw", http.Dir(cfg.StorageLocation))
	}
}
