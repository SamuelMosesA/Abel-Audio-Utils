package main

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/gemini"
	"behringerRecorder/lib/audioengine"
	"behringerRecorder/lib/state"
	"behringerRecorder/lib/web"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	pa "github.com/gordonklaus/portaudio"
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

	appState := state.NewAppState(cfg.StorageLocation, cfg.CloudDriveLocation)
	
	state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
		s.SetChL(int32(cfg.DefaultChL))
		s.SetChR(int32(cfg.DefaultChR))
		s.SetBoost(cfg.DefaultBoost)
		s.SetDeviceID(-1)
	})

	// Initialize Translation Manager if API key provided
	if cfg.GeminiAPIKey != "" {
		tm, err := gemini.NewTranslationManager(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiVoice)
		if err != nil {
			fmt.Printf("[GEMINI] Warning: Failed to init Translation Manager: %v\n", err)
		} else {
			appState.Translator = tm
			state.Update[state.GeminiConfig](appState, state.SectionGemini, func(s *state.GeminiConfig) {
				s.SetEnabled(false)
			})
			fmt.Printf("[GEMINI] Translation manager ready (disabled by default) using model: %s\n", cfg.GeminiModel)
		}
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

	r := web.NewRouter(appState, cfg)

	fmt.Printf("\033[32mUI: http://%s:%s\033[0m\n", web.GetLocalIP(), cfg.Port)
	log.Fatal(r.Run("0.0.0.0:" + cfg.Port))
}
