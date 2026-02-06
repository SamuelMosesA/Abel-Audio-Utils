package main

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/portaudio"
	"behringerRecorder/lib/types"
	"behringerRecorder/lib/web"
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
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	if abs, err := filepath.Abs(cfg.StorageLocation); err == nil {
		cfg.StorageLocation = abs
	}

	pa.Initialize()
	defer pa.Terminate()

	state := &types.AppState{
		Clients:      make(map[*websocket.Conn]bool),
		ChLeft:       0,
		ChRight:      1,
		Boost:        1.0,
		RecordChan:   make(chan []float32, 100),
		PlaybackChan: make(chan []float32, 100),
	}

	state.Devices, _ = pa.Devices()

	// Start workers
	web.StartBroadcaster(state, state.PlaybackChan)
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

	http.HandleFunc("/api/devices", web.NewDevicesHandler(state))
	http.HandleFunc("/api/status", web.NewStatusHandler(state))
	http.HandleFunc("/api/control", web.NewControlHandler(state, cfg))
	http.HandleFunc("/ws", web.NewWSHandler(state))

	PrintGreen(fmt.Sprintf("UI: http://%s:%s", web.GetLocalIP(), cfg.Port))
	log.Fatal(http.ListenAndServe("0.0.0.0:"+cfg.Port, nil))
}
