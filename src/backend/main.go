package main

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/openai"
	"abel/src/backend/lib/audioengine"
	"abel/src/backend/lib/state"
	"abel/src/backend/lib/web"
	"abel/src/backend/lib/telemetry"
	"context"
	"embed"
	"log/slog"
	"os"
	"path/filepath"

	pa "github.com/gordonklaus/portaudio"
)

//go:embed all:static
var staticFiles embed.FS

// @title Abel API
// @version 1.0
// @description REST API for controlling audio interfaces, recording, and OpenAI Realtime models.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.basic BasicAuth
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name abel_session
func main() {
	logger := slog.With("component", "main")

	logger.Info("Starting Abel...")

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Could not determine user home directory", slog.Any("error", err))
		os.Exit(1)
	}

	resolvedPath := filepath.Join(home, ".config", "abel", "config.yaml")
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		logger.Error("Configuration file not found. Only configuration from ~/.config/abel/config.yaml is allowed.", slog.String("expected_path", resolvedPath))
		os.Exit(1)
	}

	logger.Info("Loading configuration", slog.String("path", resolvedPath))
	cfg, err := config.LoadConfig(resolvedPath)
	if err != nil {
		logger.Error("Error loading config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx := context.Background()
	tel, err := telemetry.InitTelemetry(ctx, cfg.OTLPEndpoint)
	if err != nil {
		logger.Error("Failed to initialize OpenTelemetry", slog.Any("error", err))
	} else {
		defer tel.Shutdown(ctx)
	}
	if abs, err := filepath.Abs(cfg.StorageLocation); err == nil {
		cfg.StorageLocation = abs
	}
	if abs, err := filepath.Abs(cfg.CloudDriveLocation); err == nil {
		cfg.CloudDriveLocation = abs
	}

	logger.Info("Config loaded",
		slog.Int("default_ch_l", cfg.DefaultChL),
		slog.Int("default_ch_r", cfg.DefaultChR),
		slog.Float64("default_boost", cfg.DefaultBoost),
		slog.String("storage_location", cfg.StorageLocation),
	)

	pa.Initialize()
	defer pa.Terminate()

	appState := state.NewAppState(cfg.StorageLocation, cfg.CloudDriveLocation)
	
	state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
		s.SetChL(int32(cfg.DefaultChL))
		s.SetChR(int32(cfg.DefaultChR))
		s.SetBoost(cfg.DefaultBoost)
		s.SetDeviceID(-1)
		s.SetSampleRate(int32(cfg.SampleRate))
	})

	// Initialize AI Translation/Transcription Manager
	if cfg.OpenAIAPIKey == "" {
		logger.Warn("OpenAI API Key not set. Real-time translation/transcription will be unavailable.")
	}

	tm, initErr := openai.NewOpenAIManager(cfg, appState, cfg.OpenAIAPIKey, cfg.OpenAITranslateModel, cfg.OpenAITranscribeModel, cfg.OpenAIVoice, cfg.AIOriginalLanguage)
	if initErr != nil {
		logger.Warn("Failed to initialize Translation Manager", slog.Any("error", initErr))
	} else if tm != nil {
		appState.Translator = tm
		tm.SetOnStateChange(func() {
			appState.Broadcast(state.SectionAI)
		})
		state.Update[state.AIConfig](appState, state.SectionAI, func(s *state.AIConfig) {
			s.SetEnabled(false)
		})
		logger.Info("Translation manager ready", slog.String("provider", "openai"))
	}

	allDevices, _ := pa.Devices()
	for _, d := range allDevices {
		if d.MaxInputChannels > 0 {
			appState.Devices = append(appState.Devices, d)
		}
	}

	// Start workers
	audioengine.StartAudioBroadcaster(appState, cfg, appState.PlaybackChan)
	audioengine.StartStorageWorker(appState, appState.RecordChan)

	r := web.NewRouter(appState, cfg, staticFiles)

	logger.Info("Web UI active", slog.String("url", "http://"+web.GetLocalIP()+":"+cfg.Port))
	if err := r.Run("0.0.0.0:" + cfg.Port); err != nil {
		logger.Error("Web server terminated", slog.Any("error", err))
		os.Exit(1)
	}
}
