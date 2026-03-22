package main

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/gemini"
	"behringerRecorder/lib/portaudio"
	"behringerRecorder/lib/types"
	"behringerRecorder/lib/web"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	pa "github.com/gordonklaus/portaudio"

	_ "behringerRecorder/docs" // Ignore if swagger hasn't generated yet
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Behringer Audio Recorder API
// @version 1.0
// @description REST API for controlling audio interfaces, recording, and Gemini models.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.basic BasicAuth
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name behringer_session
func main() {
	fmt.Println("Starting Behringer Audio Recorder...")
	cfgPath := flag.String("config", "config.yaml", "path to config YAML file")
	flag.Parse()

	cfg, err := config.LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	if abs, err := filepath.Abs(cfg.StorageLocation); err == nil {
		cfg.StorageLocation = abs
	}
	if abs, err := filepath.Abs(cfg.CloudDriveLocation); err == nil {
		cfg.CloudDriveLocation = abs
	}

	fmt.Printf("[CONFIG] Loaded: L:%d, R:%d, Boost:%.1f, Storage:%s\n",
		cfg.DefaultChL, cfg.DefaultChR, cfg.DefaultBoost, cfg.StorageLocation)

	pa.Initialize()
	defer pa.Terminate()

	state := &types.AppState{
		StorageLocation:    cfg.StorageLocation,
		CloudDriveLocation: cfg.CloudDriveLocation,
		RecordChan:         make(chan []float32, 100),
		PlaybackChan:       make(chan []float32, 100),
	}
	state.ChLeft = int32(cfg.DefaultChL)
	state.ChRight = int32(cfg.DefaultChR)
	state.SetBoost(cfg.DefaultBoost)
	state.DeviceID = -1 // No device selected initially

	// Initialize Translation Manager if API key provided
	if cfg.GeminiAPIKey != "" {
		tm, err := gemini.NewTranslationManager(cfg.GeminiAPIKey, cfg.GeminiModel)
		if err != nil {
			fmt.Printf("[GEMINI] Warning: Failed to init Translation Manager: %v\n", err)
		} else {
			state.Translator = tm
			state.Translator.SetEnabled(true)
			state.GeminiEnabled = true
			fmt.Printf("[GEMINI] Translation enabled using model: %s\n", cfg.GeminiModel)
		}
	}

	allDevices, _ := pa.Devices()
	for _, d := range allDevices {
		if d.MaxInputChannels > 0 {
			state.Devices = append(state.Devices, d)
		}
	}

	// Start workers
	web.StartAudioBroadcaster(state, cfg, state.PlaybackChan)
	portaudio.StartStorageWorker(state, state.RecordChan)

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
	r.Use(sessions.Sessions("behringer_session", store))

	// Static & HTML routes
	r.Static("/_app", "./static/_app")

	htmlRoutes := map[string]string{
		"/":       "static/index.html",
		"/admin":  "static/admin.html",
		"/login":  "static/login.html",
		"/stream": "static/stream.html",
	}

	for path, file := range htmlRoutes {
		r.StaticFile(path, file)
	}

	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// RESTful API Handlers
	api := r.Group("/api")
	{
		// Auth
		api.POST("/auth/session", web.LoginHandler(cfg, state))

		// Audio
		audio := api.Group("/audio")
		{
			audio.GET("/devices", web.DevicesHandler(state))
			audio.GET("/config", web.GetAudioConfig(state))
			audio.GET("/stream", web.StreamHandler(state, cfg))
			audio.GET("/stream/*lang", web.StreamHandler(state, cfg))
		}

		// Recordings
		recordings := api.Group("/recordings")
		{
			// Public status
			recordings.GET("", web.GetRecordingStatus(state))
		}

		// AI
		ai := api.Group("/ai")
		{
			ai.GET("/subtitles", web.SubtitlesHandler(state))
			ai.GET("/subtitles/*lang", web.SubtitlesHandler(state))
			ai.GET("/streams", web.GetAIStreamsStatus(state))
		}

		// System
		system := api.Group("/system")
		{
			system.GET("/connection", web.GetSystemConnection(cfg))
		}

		// Admin/Protected routes (session authenticated)
		web.RegisterAdminRoutes(api, state, cfg)
	}

	r.GET("/ws", web.NewWSHandler(state, cfg))

	fmt.Printf("\033[32mUI: http://%s:%s\033[0m\n", web.GetLocalIP(), cfg.Port)
	log.Fatal(r.Run("0.0.0.0:" + cfg.Port))
}
