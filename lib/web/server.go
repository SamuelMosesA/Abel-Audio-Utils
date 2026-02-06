package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/portaudio"
	"behringerRecorder/lib/types"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func NewDevicesHandler(state *types.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state.Mu.RLock()
		devices := state.Devices
		state.Mu.RUnlock()
		list := portaudio.GetDevices(devices)
		json.NewEncoder(w).Encode(list)
	}
}

func NewControlHandler(state *types.AppState, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type Req struct {
			Action   string
			DeviceID int
			ChL      int
			ChR      int
			Folder   string
			Boost    float64
		}
		var req Req
		json.NewDecoder(r.Body).Decode(&req)

		state.Mu.Lock()
		defer state.Mu.Unlock()

		if req.Action == "connect" {
			state.Mu.Unlock()
			err := portaudio.StartAudioEngine(state, cfg, req.DeviceID, state.RecordChan, state.PlaybackChan)
			state.Mu.Lock()
			if err != nil {
				http.Error(w, err.Error(), 500)
			}
		} else if req.Action == "start" && !state.IsRecording {
			folder := req.Folder
			if folder == "" {
				folder = cfg.StorageLocation
			}
			os.MkdirAll(folder, 0755)
			base := filepath.Join(folder, fmt.Sprintf("rec_%d", time.Now().Unix()))
			state.File, _ = os.Create(base + ".wav")
			portaudio.WritePlaceholderHeader(state.File)

			state.SamplesWrote = 0
			state.IsRecording = true
			if req.Boost > 0 {
				state.Boost = req.Boost
			}
		} else if req.Action == "stop" && state.IsRecording {
			portaudio.FinalizeWavHeader(state.File, 2, state.SamplesWrote, cfg.SampleRate)
			state.File = nil
			state.IsRecording = false
		} else if req.Action == "update" {
			state.ChLeft = req.ChL
			state.ChRight = req.ChR
			state.Boost = req.Boost
		}
	}
}

func NewWSHandler(state *types.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		state.Mu.Lock()
		state.Clients[conn] = true
		state.Mu.Unlock()
	}
}
