package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/portaudio"
	"behringerRecorder/lib/types"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
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

func DevicesHandler(state *types.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("API: Device list requested")
		devices := state.Devices
		list := portaudio.GetDevices(devices)
		c.JSON(http.StatusOK, list)
	}
}

// SessionAuthMiddleware protects routes using Gin sessions
func SessionAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		auth := session.Get("authenticated")
		if auth != true {
			fmt.Printf("[AUTH] Denied request: %s %s (No authenticated session)\n", c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized session"})
			c.Abort()
			return
		}
		user := session.Get("username")
		fmt.Printf("[AUTH] Granted: %s %s (User: %v)\n", c.Request.Method, c.Request.URL.Path, user)
		c.Next()
	}
}

// @Summary Get system change log
// @Description Real-time SSE stream of state changes across the console
// @Tags System
// @Produce text/event-stream
// @Success 200 {object} string "SSE Stream"
// @Security CookieAuth
// @Security BasicAuth
// @Router /api/system/changelog [get]
func ChangeLogHandler(state *types.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		ch := make(chan types.StateChange, 10)
		state.BroadcastHub.Store(ch, true)
		defer state.BroadcastHub.Delete(ch)

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
			return
		}

		for {
			select {
			case change, ok := <-ch:
				if !ok {
					return
				}
				payload, _ := json.Marshal(change)
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(payload))
				flusher.Flush()
			case <-c.Request.Context().Done():
				return
			case <-time.After(30 * time.Second):
				fmt.Fprintf(c.Writer, ": keep-alive\n\n")
				flusher.Flush()
			}
		}
	}
}

// @Summary Update audio config
// @Description Boots up the device engine or updates running config
// @Tags Audio
// @Accept json
// @Produce json
// @Param request body object true "Interface Config"
// @Success 200 {object} string "Success"
// @Failure 400 {object} string "Invalid Request"
// @Failure 401 {object} string "Unauthorized"
// @Failure 500 {object} string "Internal Error"
// @Security CookieAuth
// @Security BasicAuth
// @Router /api/audio/config [patch]
func UpdateAudioConfig(state *types.AppState, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		state.Mu.Lock()
		if state.IsRecording {
			state.Mu.Unlock()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot change configuration while recording"})
			return
		}
		state.Mu.Unlock()

		session := sessions.Default(c)
		sessionID, _ := session.Get("session_id").(string)
		var req struct {
			DeviceID *int     `json:"deviceID"`
			ChL      *int     `json:"chL"`
			ChR      *int     `json:"chR"`
			Boost    *float64 `json:"boost"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		state.UpdateState(sessionID, "interface", func() {
			if req.DeviceID != nil {
				err := portaudio.StartAudioEngine(nil, state, cfg, *req.DeviceID, state.RecordChan, state.PlaybackChan)
				if err != nil {
					// Note: we can't easily return early from the closure with an error to Gin
					// but we can log it. For now, we'll keep it simple.
					fmt.Printf("[ENGINE] Error starting: %v\n", err)
				} else {
					state.IsRunning = true
					state.DeviceID = int32(*req.DeviceID)
					fmt.Printf("[ENGINE] Started with Device ID: %d\n", *req.DeviceID)
				}
			}

			if req.ChL != nil {
				state.ChLeft = int32(*req.ChL)
			}
			if req.ChR != nil {
				state.ChRight = int32(*req.ChR)
			}
			if req.Boost != nil {
				state.SetBoost(*req.Boost)
			}
		})

		c.JSON(http.StatusOK, gin.H{"status": "Interface updated"})
	}
}

// @Summary Get audio config
// @Description Returns current channels, boost, device info, and recording status
// @Tags Audio
// @Produce json
// @Success 200 {object} object "Audio Configuration"
// @Failure 401 {object} string "Unauthorized"
// @Router /api/audio/config [get]
func GetAudioConfig(state *types.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()
		c.JSON(http.StatusOK, gin.H{
			"deviceID":           state.DeviceID,
			"isRunning":          state.IsRunning,
			"isRecording":        state.IsRecording,
			"chL":                state.ChLeft,
			"chR":                state.ChRight,
			"boost":              state.GetBoost(),
			"storageLocation":    state.StorageLocation,
			"cloudDriveLocation": state.CloudDriveLocation,
		})
	}
}

// @Summary Control recording
// @Description Starts or stops a recording
// @Tags Recordings
// @Accept json
// @Produce json
// @Param request body object true "Recording Action"
// @Success 200 {object} string "Success"
// @Failure 400 {object} string "Invalid Action"
// @Failure 401 {object} string "Unauthorized"
// @Failure 500 {object} string "Internal Error"
// @Security CookieAuth
// @Security BasicAuth
// @Router /api/recordings [post]
func CreateRecording(state *types.AppState, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionID, _ := session.Get("session_id").(string)

		var req struct {
			Action string   `json:"action"`
			Folder string   `json:"folder"`
			Boost  *float64 `json:"boost"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		var respStatus string
		var respFile string
		var err error

		state.UpdateState(sessionID, "recording", func() {
			isRecording := state.IsRecording

			if req.Action == "start" {
				if isRecording {
					err = fmt.Errorf("already recording")
					return
				}
				folder := req.Folder
				if folder == "" {
					folder = cfg.StorageLocation
				}
				os.MkdirAll(folder, 0755)
				filename := fmt.Sprintf("rec_%d.wav", time.Now().Unix())
				base := filepath.Join(folder, filename)
				file, errCreate := os.Create(base)
				if errCreate != nil {
					err = errCreate
					return
				}
				portaudio.WritePlaceholderHeader(file)

				state.File = file
				state.SamplesWrote = 0
				state.IsRecording = true
				if req.Boost != nil {
					state.SetBoost(*req.Boost)
				}
				fmt.Printf("[RECORDING] START - File: %s\n", filename)
				respStatus = "Recording started"
				respFile = filename

			} else if req.Action == "stop" {
				if !isRecording {
					err = fmt.Errorf("not currently recording")
					return
				}
				
				file := state.File
				state.File = nil
				samplesWrote := state.SamplesWrote

				if file == nil {
					err = fmt.Errorf("no file to finalize")
					return
				}

				filename := filepath.Base(file.Name())
				portaudio.FinalizeWavHeader(file, 2, samplesWrote, cfg.SampleRate)
				file.Close()

				state.IsRecording = false
				fmt.Printf("[RECORDING] STOP - File: %s, Samples: %d\n", filename, samplesWrote)
				respStatus = "Recording stopped"
				respFile = filename
			} else {
				err = fmt.Errorf("invalid action")
			}
		})

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": respStatus, "file": respFile})
	}
}

// @Summary Get recording status
// @Description Returns whether the system is recording
// @Tags Recordings
// @Produce json
// @Success 200 {object} object "Recording Status"
// @Failure 401 {object} string "Unauthorized"
// @Router /api/recordings [get]
func GetRecordingStatus(state *types.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()
		status := gin.H{
			"isRecording": state.IsRecording,
			"samples":     state.SamplesWrote,
		}
		c.JSON(http.StatusOK, status)
	}
}

// @Summary Control AI streams
// @Description Toggles master gemini switch or stops a specific language translation
// @Tags AI
// @Accept json
// @Produce json
// @Param request body object true "AI Action"
// @Success 200 {object} string "Success"
// @Failure 400 {object} string "Invalid Action"
// @Failure 401 {object} string "Unauthorized"
// @Security CookieAuth
// @Security BasicAuth
// @Router /api/ai/streams [post]
func UpdateAIStreams(state *types.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionID, _ := session.Get("session_id").(string)

		var req struct {
			Action    string `json:"action"`
			Enabled   *bool  `json:"enabled"`
			Language  string `json:"language"`
			Subtitles bool   `json:"subtitles"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		state.UpdateState(sessionID, "gemini", func() {
			if req.Action == "toggle_master" {
				if req.Enabled != nil {
					state.GeminiEnabled = *req.Enabled
					if state.Translator != nil {
						state.Translator.SetEnabled(*req.Enabled)
						if !*req.Enabled {
							state.Translator.CloseAll()
						}
					}
				}
			} else if req.Action == "stop_translation" {
				if state.Translator != nil && req.Language != "" {
					state.Translator.StopSession(req.Language, req.Subtitles)
				}
			}
		})

		c.JSON(http.StatusOK, gin.H{"status": "Gemini action completed"})
	}
}

// @Summary Get AI streams status
// @Description Returns active sessions and master state
// @Tags AI
// @Produce json
// @Success 200 {object} object "AI Streams Status"
// @Failure 401 {object} string "Unauthorized"
// @Router /api/ai/streams [get]
func GetAIStreamsStatus(state *types.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{
			"masterEnabled": state.GeminiEnabled,
		}
		if state.Translator != nil {
			status["sessions"] = state.Translator.ListSessions()
		} else {
			status["sessions"] = []types.SessionInfo{}
		}
		c.JSON(http.StatusOK, status)
	}
}

func RegisterAdminRoutes(r *gin.RouterGroup, state *types.AppState, cfg *config.Config) {
	r.Use(SessionAuthMiddleware())
	{
		r.PATCH("/audio/config", UpdateAudioConfig(state, cfg))
		r.POST("/recordings", CreateRecording(state, cfg))
		r.POST("/ai/streams", UpdateAIStreams(state))
		r.GET("/system/changelog", ChangeLogHandler(state))
		r.GET("/recordings/files", ListRecordingFiles(cfg))
		r.POST("/recordings/push", PushRecordingToCloud(cfg))
		r.StaticFS("/recordings/raw", http.Dir(cfg.StorageLocation))
	}
}

func GetSystemConnection(cfg *config.Config) gin.HandlerFunc {
	// @Summary Get system connection info
	// @Description Returns local IP and WiFi SSID
	// @Tags System
	// @Produce json
	// @Success 200 {object} object "Connection Status"
	// @Security CookieAuth
	// @Security BasicAuth
	// @Router /api/system/connection [get]
	return func(c *gin.Context) {
		status := struct {
			ServerURL string `json:"serverUrl"`
			SSID      string `json:"ssid"`
		}{
			ServerURL: fmt.Sprintf("http://%s:%s", GetLocalIP(), cfg.Port),
			SSID:      GetWiFiSSID(),
		}
		c.JSON(http.StatusOK, status)
	}
}


func ListRecordingFiles(cfg *config.Config) gin.HandlerFunc {
	// @Summary List recording files
	// @Description Returns a list of WAV files in storage
	// @Tags Recordings
	// @Produce json
	// @Success 200 {array} object "File List"
	// @Security CookieAuth
	// @Security BasicAuth
	// @Router /api/recordings/files [get]
	return func(c *gin.Context) {
		files, err := os.ReadDir(cfg.StorageLocation)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read recordings directory"})
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
		c.JSON(http.StatusOK, list)
	}
}

func PushRecordingToCloud(cfg *config.Config) gin.HandlerFunc {
	// @Summary Push recording to cloud
	// @Description Copies a local file to the cloud drive location
	// @Tags Recordings
	// @Accept json
	// @Produce json
	// @Param request body object true "Push Request"
	// @Success 200 {object} string "Success"
	// @Security CookieAuth
	// @Security BasicAuth
	// @Router /api/recordings/push [post]
	return func(c *gin.Context) {
		var req struct {
			Source string `json:"source"`
			Target string `json:"target"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		sourcePath := filepath.Join(cfg.StorageLocation, req.Source)
		targetPath := filepath.Join(cfg.CloudDriveLocation, req.Target)

		// Ensure target directory exists
		if err := os.MkdirAll(cfg.CloudDriveLocation, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create target directory"})
			return
		}

		// Copy file
		src, err := os.Open(sourcePath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Source file not found"})
			return
		}
		defer src.Close()

		dst, err := os.Create(targetPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create destination file"})
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to copy file"})
			return
		}

		fmt.Printf("[CLOUD] Pushed %s -> %s\n", req.Source, req.Target)
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}


func NewWSHandler(state *types.AppState, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionID, _ := session.Get("session_id").(string)
		auth := session.Get("authenticated")

		if auth != true {
			// Fallback for non-session clients (e.g. CLI tools if any)
			pass := c.Query("pass")
			if pass == cfg.AdminPassword && pass != "" {
				// OK
			} else {
				user, password, hasAuth := c.Request.BasicAuth()
				if hasAuth {
					if p, ok := cfg.Credentials[user]; ok && p == password {
						// OK
					} else {
						c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
						return
					}
				} else {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized session"})
					return
				}
			}
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		wsClient := &types.WSClient{Conn: conn, Type: "admin"}
		state.Clients.Store(wsClient, true)

		// Set the active admin client (multiple can now connect, but let's keep track of 'the' admin if needed)
		state.AdminClient = wsClient
		fmt.Printf("[CLIENT] New admin connected (ID: %p, Session: %s).\n", wsClient, sessionID)

		// Listen for client messages (mostly for disconnect)
		go func() {
			for {
				_, _, err := wsClient.Conn.ReadMessage()
				if err != nil {
					// Client disconnected
					state.Clients.Delete(wsClient)
					if state.AdminClient == wsClient {
						state.AdminClient = nil
						// state.MasterSessionID = "" // Removed
					}
					fmt.Printf("[CLIENT] Admin disconnected (ID: %p). Session lock released.\n", wsClient)
					wsClient.Close()
					return
				}
			}
		}()
	}
}


func StreamHandler(state *types.AppState, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := filepath.Base(c.Request.URL.Path)
		if lang == "stream" || lang == "/" {
			lang = "default"
		}

		fmt.Printf("[STREAM] New listener connected: %s (lang: %s)\n", c.Request.RemoteAddr, lang)

		c.Header("Content-Type", "audio/wav")
		c.Header("Connection", "keep-alive")
		c.Header("Cache-Control", "no-cache")
		c.Header("X-Accel-Buffering", "no")

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

		c.Writer.Write(header)
		if f, ok := c.Writer.(http.Flusher); ok {
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

		fmt.Printf("[STREAM] New listener connected: %s (lang: %s, translated: %v)\n", c.Request.RemoteAddr, lang, isTranslated)
		defer fmt.Printf("[STREAM] Listener disconnected: %s (lang: %s)\n", c.Request.RemoteAddr, lang)

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
				_, err := c.Writer.Write(pcmBuf)
				if err != nil {
					return
				}
				if f, ok := c.Writer.(http.Flusher); ok {
					f.Flush()
				}
			case <-c.Request.Context().Done():
				return
			}
		}
	}
}


func SubtitlesHandler(state *types.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.Query("lang")
		if lang == "" {
			lang = c.Param("lang")
			lang = strings.TrimPrefix(lang, "/")
		}
		if lang == "" || lang == "default" || lang == "subtitles" {
			lang = "English"
		}

		log.Printf("[SERVER] Subtitles connection requested for lang: %s (IP: %s)", lang, c.Request.RemoteAddr)

		if state.Translator == nil {
			log.Printf("[SERVER] Subtitles handler aborted: Translator is nil")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Translation not available"})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		ch, cleanup := state.Translator.GetSubtitles(lang)
		if ch == nil {
			log.Printf("[SERVER] Failed to get subtitle channel for lang: %s", lang)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subtitle channel"})
			return
		}
		defer func() {
			log.Printf("[SERVER] Subtitles connection closed for lang: %s", lang)
			cleanup()
		}()

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
			return
		}

		// Initial keep-alive or state check
		if !state.GeminiEnabled {
			fmt.Fprintf(c.Writer, "data: %s\n\n", `{"error": "Gemini Master Switch is OFF"}`)
			flusher.Flush()
		}

		for {
			select {
			case text, ok := <-ch:
				if !ok {
					log.Printf("[SERVER] Subtitles channel closed for lang: %s", lang)
					return
				}
				log.Printf("[SERVER] Sending subtitle to %s: %s", lang, text)
				payload, _ := json.Marshal(map[string]string{"text": text})
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(payload))
				flusher.Flush()
			case <-c.Request.Context().Done():
				return
			case <-time.After(30 * time.Second):
				// Keep-alive
				fmt.Fprintf(c.Writer, ": keep-alive\n\n")
				flusher.Flush()
			}
		}
	}
}
type LoginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"admin"`
}

// @Summary Create auth session
// @Description Logs in and establishes a session
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login Credentials"
// @Success 200 {object} object "Login Success"
// @Router /api/auth/session [post]
func LoginHandler(cfg *config.Config, state *types.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if req.Username == "" {
			req.Username = "admin" // Default if empty
		}

		authorized := false
		if p, ok := cfg.Credentials[req.Username]; ok && p == req.Password {
			authorized = true
		} else if req.Username == "admin" && req.Password == cfg.AdminPassword {
			authorized = true
		}

		if !authorized {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}
		
		// Generate new session ID
		b := make([]byte, 16)
		rand.Read(b)
		newSessionID := fmt.Sprintf("%x", b)

		// Set Gin Session
		session := sessions.Default(c)
		session.Set("authenticated", true)
		session.Set("username", req.Username)
		session.Set("session_id", newSessionID)
		session.Save()
		
		c.JSON(http.StatusOK, gin.H{
			"status":   "success",
			"session":  newSessionID,
			"username": req.Username,
		})
	}
}

// broadcastStateUpdate and sendConfigStateUpdate are removed as they are no longer needed
// JSON state is now fetched via /api/status
