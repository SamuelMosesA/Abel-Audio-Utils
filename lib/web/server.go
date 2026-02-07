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
		fmt.Println("API: Device list requested")
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

		// Lock for atomic read of recording state
		state.Mu.RLock()
		isRecording := state.IsRecording
		state.Mu.RUnlock()

		if req.Action == "connect" {
			// Start engine without holding lock (long operation)
			err := portaudio.StartAudioEngine(state, cfg, req.DeviceID, state.RecordChan, state.PlaybackChan)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			// Update state
			state.Mu.Lock()
			state.IsRunning = true
			state.DeviceID = req.DeviceID
			state.ChLeft = req.ChL
			state.ChRight = req.ChR
			state.Mu.Unlock()
			fmt.Printf("[ENGINE] Started with Device ID: %d\n", req.DeviceID)
			// Notify all clients
			broadcastStateUpdate(state)

		} else if req.Action == "start" {
			if isRecording {
				fmt.Printf("[RECORDING] START request rejected - already recording\n")
				http.Error(w, "Already recording", 400)
				return
			}
			// Create recording file
			folder := req.Folder
			if folder == "" {
				folder = cfg.StorageLocation
			}
			os.MkdirAll(folder, 0755)
			filename := fmt.Sprintf("rec_%d.wav", time.Now().Unix())
			base := filepath.Join(folder, filename)
			file, err := os.Create(base)
			if err != nil {
				fmt.Printf("[RECORDING] START failed - could not create file: %v\n", err)
				http.Error(w, "Failed to create file", 500)
				return
			}
			portaudio.WritePlaceholderHeader(file)

			// Update state atomically
			state.Mu.Lock()
			state.File = file
			state.SamplesWrote = 0
			state.IsRecording = true
			if req.Boost > 0 {
				state.Boost = req.Boost
			}
			state.Mu.Unlock()
			fmt.Printf("[RECORDING] START - File: %s\n", filename)
			// Notify all clients
			broadcastStateUpdate(state)

		} else if req.Action == "stop" {
			if !isRecording {
				fmt.Printf("[RECORDING] STOP request rejected - not recording\n")
				http.Error(w, "Not currently recording", 400)
				return
			}
			// Read file and sample count
			state.Mu.RLock()
			file := state.File
			samplesWrote := state.SamplesWrote
			state.Mu.RUnlock()

			if file == nil {
				fmt.Printf("[RECORDING] STOP failed - no file handle\n")
				http.Error(w, "No file to finalize", 500)
				return
			}

			// Get filename for logging
			filename := filepath.Base(file.Name())

			// Finalize file (without lock)
			portaudio.FinalizeWavHeader(file, 2, samplesWrote, cfg.SampleRate)
			file.Close()

			// Update state
			state.Mu.Lock()
			state.File = nil
			state.IsRecording = false
			state.Mu.Unlock()
			fmt.Printf("[RECORDING] STOP - File: %s, Samples: %d\n", filename, samplesWrote)
			// Notify all clients
			broadcastStateUpdate(state)

		} else if req.Action == "update" && !isRecording {
			// Only allow config updates when not recording
			state.Mu.Lock()
			state.ChLeft = req.ChL
			state.ChRight = req.ChR
			state.Boost = req.Boost
			state.Mu.Unlock()
			// Notify all clients
			broadcastStateUpdate(state)
		}
	}
}

func NewStatusHandler(state *types.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()
		status := struct {
			IsRunning   bool    `json:"isRunning"`
			IsRecording bool    `json:"isRecording"`
			ChL         int     `json:"chL"`
			ChR         int     `json:"chR"`
			Boost       float64 `json:"boost"`
		}{
			IsRunning:   state.IsRunning,
			IsRecording: state.IsRecording,
			ChL:         state.ChLeft,
			ChR:         state.ChRight,
			Boost:       state.Boost,
		}
		json.NewEncoder(w).Encode(status)
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
		clientCount := len(state.Clients)
		// First client becomes primary, or if only 1 client ensure it's primary
		if state.PrimaryClient == nil || clientCount == 1 {
			state.PrimaryClient = conn
			state.Mu.Unlock()
			fmt.Printf("[CLIENT] New client connected (ID: %p) - Assigned as PRIMARY. Total clients: %d\n", conn, clientCount)
		} else {
			state.Mu.Unlock()
			fmt.Printf("[CLIENT] New client connected (ID: %p) - Secondary. Total clients: %d\n", conn, clientCount)
		}

		// Send initial state to newly connected client
		sendStateUpdate(conn, state)
		broadcastStateUpdate(state)

		// Listen for client messages
		go func() {
			for {
				_, data, err := conn.ReadMessage()
				if err != nil {
					// Client disconnected
					state.Mu.Lock()
					delete(state.Clients, conn)
					clientCount := len(state.Clients)
					// If primary client disconnected, assign new primary
					if state.PrimaryClient == conn {
						state.PrimaryClient = nil
						// Assign first available client as primary
						for c := range state.Clients {
							state.PrimaryClient = c
							break
						}
						if state.PrimaryClient != nil {
							state.Mu.Unlock()
							fmt.Printf("[CLIENT] Primary client disconnected (ID: %p). Assigned new PRIMARY (ID: %p). Remaining clients: %d\n", conn, state.PrimaryClient, clientCount)
						} else {
							state.Mu.Unlock()
							fmt.Printf("[CLIENT] Primary client disconnected (ID: %p). No clients remaining\n", conn)
						}
					} else {
						state.Mu.Unlock()
						fmt.Printf("[CLIENT] Secondary client disconnected (ID: %p). Remaining clients: %d\n", conn, clientCount)
					}
					// If only 1 client remains, ensure it's primary
					if clientCount == 1 && state.PrimaryClient == nil {
						state.Mu.Lock()
						for c := range state.Clients {
							state.PrimaryClient = c
							break
						}
						state.Mu.Unlock()
					}
					conn.Close()
					// Notify remaining clients of primary change
					broadcastStateUpdate(state)
					return
				}

				// Handle incoming messages from clients
				var msg struct {
					Type string `json:"type"`
				}
				if err := json.Unmarshal(data, &msg); err != nil {
					continue
				}

				if msg.Type == "requestPrimary" {
					// Secondary client requesting primary - disconnect old primary and promote requester
					state.Mu.Lock()
					oldPrimary := state.PrimaryClient
					state.PrimaryClient = conn
					state.Mu.Unlock()

					// Disconnect the old primary client
					if oldPrimary != nil && oldPrimary != conn {
						fmt.Printf("[PRIMARY] Client %p requested primary control. Disconnecting old PRIMARY %p\n", conn, oldPrimary)
						oldPrimary.Close()
						state.Mu.Lock()
						delete(state.Clients, oldPrimary)
						state.Mu.Unlock()
					}

					// Notify all clients of the change
					fmt.Printf("[PRIMARY] New PRIMARY assigned: %p\n", conn)
					broadcastStateUpdate(state)
				}
			}
		}()
	}
}

// sendStateUpdate sends the current engine state to a single client.
// Must be called while holding the state mutex to prevent concurrent writes.
func sendStateUpdate(conn *websocket.Conn, state *types.AppState) {
	isPrimary := state.PrimaryClient == conn

	update := struct {
		Type        string  `json:"type"`
		IsRunning   bool    `json:"isRunning"`
		IsRecording bool    `json:"isRecording"`
		IsPrimary   bool    `json:"isPrimary"`
		DeviceID    int     `json:"deviceId"`
		ChL         int     `json:"chL"`
		ChR         int     `json:"chR"`
		Boost       float64 `json:"boost"`
	}{
		Type:        "state",
		IsRunning:   state.IsRunning,
		IsRecording: state.IsRecording,
		IsPrimary:   isPrimary,
		DeviceID:    state.DeviceID,
		ChL:         state.ChLeft,
		ChR:         state.ChRight,
		Boost:       state.Boost,
	}

	conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
	conn.WriteJSON(update)
}

// broadcastStateUpdate sends the current engine state to all connected clients.
// This ensures all clients stay in sync when any client performs an action.
func broadcastStateUpdate(state *types.AppState) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	for c := range state.Clients {
		sendStateUpdate(c, state)
	}
}
