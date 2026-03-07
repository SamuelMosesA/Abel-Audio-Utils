package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/portaudio"
	"behringerRecorder/lib/types"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

func GetWiFiSSID() string {
	if runtime.GOOS == "darwin" {
		// macOS
		cmd := exec.Command("/System/Library/PrivateFrameworks/Apple80211.framework/Resources/airport", "-I")
		out, err := cmd.Output()
		if err != nil {
			return "N/A"
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "SSID: ") {
				return strings.TrimPrefix(line, "SSID: ")
			}
		}
	} else if runtime.GOOS == "linux" {
		// Linux
		cmd := exec.Command("nmcli", "-t", "-f", "active,ssid", "dev", "wifi")
		out, err := cmd.Output()
		if err != nil {
			return "N/A"
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "yes:") {
				return strings.TrimPrefix(line, "yes:")
			}
		}
	}
	return "N/A"
}

func DevicesHandler(state *types.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("API: Device list requested")
		state.Mu.RLock()
		devices := state.Devices
		state.Mu.RUnlock()
		list := portaudio.GetDevices(devices)
		json.NewEncoder(w).Encode(list)
	}
}

// Update all handlers to check for authentication or remove state broadcasts
func NewControlHandler(state *types.AppState, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Basic Auth check for all control actions
		pass := r.Header.Get("X-Admin-Password")
		if pass == "" {
			// Also check body for password if header is missing (convenience for some clients)
			// But header is better for utility fetcher
		}
		if pass != cfg.AdminPassword {
			http.Error(w, "Unauthorized", 401)
			return
		}

		type Req struct {
			Action   string
			DeviceID int
			ChL      *int
			ChR      *int
			Folder   string
			Boost    *float64
			Language string
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", 400)
			return
		}

		// Lock for atomic read of recording state
		isRecording := state.IsRecording.Load()

		if req.Action == "connect" {
			err := portaudio.StartAudioEngine(state, cfg, req.DeviceID, state.RecordChan, state.PlaybackChan)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			state.IsRunning.Store(true)
			state.DeviceID.Store(int32(req.DeviceID))
			if req.ChL != nil {
				state.ChLeft.Store(int32(*req.ChL))
			}
			if req.ChR != nil {
				state.ChRight.Store(int32(*req.ChR))
			}
			if req.Boost != nil {
				state.SetBoost(*req.Boost)
			}
			fmt.Printf("[ENGINE] Started with Device ID: %d\n", req.DeviceID)

		} else if req.Action == "start" {
			if isRecording {
				http.Error(w, "Already recording", 400)
				return
			}
			folder := req.Folder
			if folder == "" {
				folder = cfg.StorageLocation
			}
			os.MkdirAll(folder, 0755)
			filename := fmt.Sprintf("rec_%d.wav", time.Now().Unix())
			base := filepath.Join(folder, filename)
			file, err := os.Create(base)
			if err != nil {
				http.Error(w, "Failed to create file", 500)
				return
			}
			portaudio.WritePlaceholderHeader(file)

			state.Mu.Lock()
			state.File = file
			state.Mu.Unlock()
			state.SamplesWrote.Store(0)
			state.IsRecording.Store(true)
			if req.Boost != nil {
				state.SetBoost(*req.Boost)
			}
			fmt.Printf("[RECORDING] START - File: %s\n", filename)

		} else if req.Action == "stop" {
			if !isRecording {
				http.Error(w, "Not currently recording", 400)
				return
			}
			state.Mu.Lock()
			file := state.File
			state.File = nil
			state.Mu.Unlock()

			samplesWrote := state.SamplesWrote.Load()

			if file == nil {
				http.Error(w, "No file to finalize", 500)
				return
			}

			filename := filepath.Base(file.Name())
			portaudio.FinalizeWavHeader(file, 2, samplesWrote, cfg.SampleRate)
			file.Close()

			state.IsRecording.Store(false)
			fmt.Printf("[RECORDING] STOP - File: %s, Samples: %d\n", filename, samplesWrote)

		} else if req.Action == "update" && !isRecording {
			if req.ChL != nil {
				state.ChLeft.Store(int32(*req.ChL))
			}
			if req.ChR != nil {
				state.ChRight.Store(int32(*req.ChR))
			}
			if req.Boost != nil {
				state.SetBoost(*req.Boost)
			}
		} else if req.Action == "stop_translation" {
			if state.Translator != nil && req.Language != "" {
				state.Translator.StopSession(req.Language)
			}
		}
	}
}

func NewStatusHandler(state *types.AppState, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := struct {
			IsRunning          bool                `json:"isRunning"`
			IsRecording        bool                `json:"isRecording"`
			ChL                int                 `json:"chL"`
			ChR                int                 `json:"chR"`
			Boost              float64             `json:"boost"`
			DeviceId           int                 `json:"deviceId"`
			StorageLocation    string              `json:"storageLocation"`
			CloudDriveLocation string              `json:"cloudDriveLocation"`
			Translations       []types.SessionInfo `json:"translations"`
			ServerURL          string              `json:"serverUrl"`
			SSID               string              `json:"ssid"`
		}{
			IsRunning:          state.IsRunning.Load(),
			IsRecording:        state.IsRecording.Load(),
			ChL:                int(state.ChLeft.Load()),
			ChR:                int(state.ChRight.Load()),
			Boost:              state.GetBoost(),
			DeviceId:           int(state.DeviceID.Load()),
			StorageLocation:    state.StorageLocation,
			CloudDriveLocation: state.CloudDriveLocation,
			ServerURL:          fmt.Sprintf("http://%s:%s", GetLocalIP(), cfg.Port),
			SSID:               GetWiFiSSID(),
		}
		if state.Translator != nil {
			status.Translations = state.Translator.ListSessions()
		}
		json.NewEncoder(w).Encode(status)
	}
}

func FilesHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files, err := os.ReadDir(cfg.StorageLocation)
		if err != nil {
			http.Error(w, "Failed to read recordings directory", 500)
			return
		}

		type FileInfo struct {
			Name    string    `json:"name"`
			Size    int64     `json:"size"`
			ModTime time.Time `json:"modTime"`
		}

		var list []FileInfo
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".wav" {
				info, err := f.Info()
				if err == nil {
					list = append(list, FileInfo{
						Name:    f.Name(),
						Size:    info.Size(),
						ModTime: info.ModTime(),
					})
				}
			}
		}
		json.NewEncoder(w).Encode(list)
	}
}

func PushHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Source string `json:"source"`
			Target string `json:"target"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", 400)
			return
		}

		sourcePath := filepath.Join(cfg.StorageLocation, req.Source)
		targetPath := filepath.Join(cfg.CloudDriveLocation, req.Target)

		// Ensure target directory exists
		if err := os.MkdirAll(cfg.CloudDriveLocation, 0755); err != nil {
			http.Error(w, "Failed to create target directory", 500)
			return
		}

		// Copy file
		src, err := os.Open(sourcePath)
		if err != nil {
			http.Error(w, "Source file not found", 404)
			return
		}
		defer src.Close()

		dst, err := os.Create(targetPath)
		if err != nil {
			http.Error(w, "Failed to create destination file", 500)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			http.Error(w, "Failed to copy file", 500)
			return
		}

		fmt.Printf("[CLOUD] Pushed %s -> %s\n", req.Source, req.Target)
		w.WriteHeader(http.StatusOK)
	}
}

func NewWSHandler(state *types.AppState, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only admins allowed to connect via WebSocket
		pass := r.URL.Query().Get("pass")
		if pass != cfg.AdminPassword {
			fmt.Printf("[WS] Denied connection attempt from %s (invalid or missing password)\n", r.RemoteAddr)
			http.Error(w, "Unauthorized", 401)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		wsClient := &types.WSClient{Conn: conn, Type: "admin"}
		state.Clients.Store(wsClient, true)

		// Disconnect old admin if exists (exclusive access)
		if oldAdmin := state.AdminClient.Swap(wsClient); oldAdmin != nil {
			fmt.Printf("[ADMIN] New admin connecting, kicking out old admin %p\n", oldAdmin)
			// Explicitly notify the client they are being kicked
			oldAdmin.WriteJSON(map[string]string{"type": "kickout"})
			oldAdmin.Close()
			state.Clients.Delete(oldAdmin)
		}
		fmt.Printf("[CLIENT] New admin connected (ID: %p).\n", wsClient)

		// Listen for client messages (mostly for disconnect)
		go func() {
			for {
				_, _, err := wsClient.Conn.ReadMessage()
				if err != nil {
					// Client disconnected
					state.Clients.Delete(wsClient)
					if state.AdminClient.Load() == wsClient {
						state.AdminClient.CompareAndSwap(wsClient, nil)
					}
					fmt.Printf("[CLIENT] Admin disconnected (ID: %p)\n", wsClient)
					wsClient.Close()
					return
				}
			}
		}()
	}
}

func StreamHandler(state *types.AppState, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := filepath.Base(r.URL.Path)
		if lang == "stream" || lang == "/" {
			lang = "default"
		}

		fmt.Printf("[STREAM] New listener connected: %s (lang: %s)\n", r.RemoteAddr, lang)

		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")

		// Write a dummy WAV header for a "forever" stream
		// 44 bytes header
		header := make([]byte, 44)
		copy(header[0:4], "RIFF")
		binary.LittleEndian.PutUint32(header[4:8], 0xFFFFFFFF) // File size
		copy(header[8:12], "WAVE")
		copy(header[12:16], "fmt ")
		binary.LittleEndian.PutUint32(header[16:20], 16) // fmt chunk size
		binary.LittleEndian.PutUint16(header[20:22], 1)  // PCM
		binary.LittleEndian.PutUint16(header[22:24], 2)  // Channels
		binary.LittleEndian.PutUint32(header[24:28], uint32(cfg.SampleRate))
		binary.LittleEndian.PutUint32(header[28:32], uint32(cfg.SampleRate*2*2)) // Byte rate
		binary.LittleEndian.PutUint16(header[32:34], 4)                          // Block align
		binary.LittleEndian.PutUint16(header[34:36], 16)                         // Bits per sample
		copy(header[36:40], "data")
		binary.LittleEndian.PutUint32(header[40:44], 0xFFFFFFFF) // Data size

		w.Write(header)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// Create or get the audio channel
		var ch chan []float32
		isTranslated := false
		if lang != "default" && state.Translator != nil {
			ch = state.Translator.GetChannel(lang)
			if ch != nil {
				isTranslated = true
			}
		}

		if ch == nil {
			// fallback to standard stream
			ch = make(chan []float32, 100)
			state.StreamChannels.Store(ch, true)
			defer state.StreamChannels.Delete(ch)
		}

		fmt.Printf("[STREAM] New listener connected: %s (lang: %s, translated: %v)\n", r.RemoteAddr, lang, isTranslated)
		defer fmt.Printf("[STREAM] Listener disconnected: %s (lang: %s)\n", r.RemoteAddr, lang)

		// Write PCM data
		for {
			select {
			case chunk, ok := <-ch:
				if !ok {
					return
				}
				// Convert float32 [-1, 1] to i16 for classic WAV
				pcmBuf := make([]byte, len(chunk)*2)
				for i, v := range chunk {
					s := int16(v * 32767)
					binary.LittleEndian.PutUint16(pcmBuf[i*2:], uint16(s))
				}
				_, err := w.Write(pcmBuf)
				if err != nil {
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			case <-r.Context().Done():
				return
			}
		}
	}
}

func LoginHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		if req.Password == cfg.AdminPassword {
			w.WriteHeader(http.StatusOK)
		} else {
			http.Error(w, "Invalid password", 401)
		}
	}
}

// broadcastStateUpdate and sendConfigStateUpdate are removed as they are no longer needed
// JSON state is now fetched via /api/status
