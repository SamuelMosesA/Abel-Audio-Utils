package main

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/portaudio"
	"behringerRecorder/lib/types"
	"behringerRecorder/lib/web"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	pa "github.com/gordonklaus/portaudio"
	"github.com/gorilla/websocket"
)

func PrintGreen(msg string) {
	fmt.Printf("\033[32m%s\033[0m\n", msg)
}

func main() {
	// Allow providing a config file path via CLI: `-config /path/to/config.yaml`.
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
		Clients:      make(map[*websocket.Conn]bool),
		ChLeft:       cfg.DefaultChL,
		ChRight:      cfg.DefaultChR,
		Boost:        cfg.DefaultBoost,
		RecordChan:   make(chan []float32, 100),
		PlaybackChan: make(chan []float32, 100),
	}

	state.Devices, _ = pa.Devices()

	// Start workers
	web.StartAudioBroadcaster(state, state.PlaybackChan)
	portaudio.StartStorageWorker(state, state.RecordChan)

	tmpl := template.Must(template.ParseFiles("static/index.html"))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/assets/", http.FileServer(http.Dir("static"))) // Vite assets are in static/assets
	http.HandleFunc("/vite.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/vite.svg")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.FileServer(http.Dir("static")).ServeHTTP(w, r)
			return
		}
		tmpl.Execute(w, cfg)
	})

	http.HandleFunc("/api/devices", web.DevicesHandler(state))
	http.Handle("/api/recordings/", http.StripPrefix("/api/recordings/", http.FileServer(http.Dir(cfg.StorageLocation))))
	http.HandleFunc("/api/files", web.FilesHandler(cfg))
	http.HandleFunc("/api/status", web.NewStatusHandler(state, cfg))
	http.HandleFunc("/api/control", web.NewControlHandler(state, cfg))
	http.HandleFunc("/api/push", web.PushHandler(cfg))
	http.HandleFunc("/ws", web.NewWSHandler(state))

	PrintGreen(fmt.Sprintf("UI: http://%s:%s", web.GetLocalIP(), cfg.Port))
	log.Fatal(http.ListenAndServe("0.0.0.0:"+cfg.Port, nil))
}
