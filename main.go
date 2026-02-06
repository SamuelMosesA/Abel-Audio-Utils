package main

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/portaudio"
	"behringerRecorder/lib/types"
	"behringerRecorder/lib/web"
	"fmt"
	"log"
	"net/http"

	pa "github.com/gordonklaus/portaudio"
	"github.com/gorilla/websocket"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
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

	// Start workers
	web.StartBroadcaster(state, state.PlaybackChan)
	portaudio.StartStorageWorker(state, state.RecordChan)

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/api/devices", web.HandleGetDevices)
	http.HandleFunc("/api", web.NewControlHandler(state, cfg))
	http.HandleFunc("/ws", web.NewWSHandler(state))

	fmt.Printf("UI: http://%s:%s\n", web.GetLocalIP(), cfg.Port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+cfg.Port, nil))
}
