package main

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/portaudio"
	"behringerRecorder/lib/types"
	"behringerRecorder/lib/web"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	pa "github.com/gordonklaus/portaudio"
)

func PrintGreen(msg string) {
	fmt.Printf("\033[32m%s\033[0m\n", msg)
}

func main() {
	// ... (rest of main initialization)
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
	state.ChLeft.Store(int32(cfg.DefaultChL))
	state.ChRight.Store(int32(cfg.DefaultChR))
	state.SetBoost(cfg.DefaultBoost)
	state.DeviceID.Store(-1) // No device selected initially

	state.Devices, _ = pa.Devices()

	// Start workers
	web.StartAudioBroadcaster(state, state.PlaybackChan)
	portaudio.StartStorageWorker(state, state.RecordChan)

	// 1. Static Assets (JS, CSS, etc.)
	http.Handle("/static/", http.FileServer(http.Dir("static")))

	// 2. Main HTML Routes
	htmlRoutes := map[string]string{
		"/":       "static/index.html",
		"/admin":  "static/admin.html",
		"/login":  "static/login.html",
		"/stream": "static/stream.html",
	}

	for path, file := range htmlRoutes {
		f := file // closure capture
		http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, f)
		})
	}

	http.HandleFunc("/api/devices", web.DevicesHandler(state))
	http.Handle("/api/recordings/", http.StripPrefix("/api/recordings/", http.FileServer(http.Dir(cfg.StorageLocation))))
	http.HandleFunc("/api/files", web.FilesHandler(cfg))
	http.HandleFunc("/api/status", web.NewStatusHandler(state, cfg))
	http.HandleFunc("/api/control", web.NewControlHandler(state, cfg))
	http.HandleFunc("/api/push", web.PushHandler(cfg))
	http.HandleFunc("/api/login", web.LoginHandler(cfg))
	http.HandleFunc("/api/stream", web.StreamHandler(state, cfg))
	http.HandleFunc("/api/stream/", web.StreamHandler(state, cfg))
	http.HandleFunc("/ws", web.NewWSHandler(state, cfg))

	PrintGreen(fmt.Sprintf("UI: http://%s:%s", web.GetLocalIP(), cfg.Port))
	log.Fatal(http.ListenAndServe("0.0.0.0:"+cfg.Port, nil))
}
