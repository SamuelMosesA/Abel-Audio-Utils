package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/audioengine"
	"behringerRecorder/lib/state"
	"encoding/binary"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func DevicesHandler(state *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("API: Device list requested")
		devices := state.Devices
		list := audioengine.GetDevices(devices)
		c.JSON(http.StatusOK, list)
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
func UpdateAudioConfig(appState *state.AppState, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if appState.IsRecording() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot change configuration while recording"})
			return
		}

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

		state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
			if req.DeviceID != nil {
				err := audioengine.StartAudioEngine(nil, appState, cfg, *req.DeviceID, appState.RecordChan, appState.PlaybackChan)
				if err != nil {
					fmt.Printf("[ENGINE] Error starting: %v\n", err)
				} else {
					s.SetIsRunning(true)
					s.SetDeviceID(int32(*req.DeviceID))
					fmt.Printf("[ENGINE] Started with Device ID: %d\n", *req.DeviceID)
				}
			}

			if req.ChL != nil {
				s.SetChL(int32(*req.ChL))
			}
			if req.ChR != nil {
				s.SetChR(int32(*req.ChR))
			}
			if req.Boost != nil {
				s.SetBoost(*req.Boost)
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
func GetAudioConfig(appState *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		conf := appState.Config()
		loc := appState.Locations()
		c.JSON(http.StatusOK, gin.H{
			"deviceID":           conf.DeviceID(),
			"isRunning":          conf.IsRunning(),
			"isRecording":        appState.IsRecording(),
			"chL":                conf.ChL(),
			"chR":                conf.ChR(),
			"boost":              conf.Boost(),
			"storageLocation":    loc.Storage(),
			"cloudDriveLocation": loc.CloudDrive(),
		})
	}
}

func StreamHandler(appState *state.AppState, cfg *config.Config) gin.HandlerFunc {
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
		
		// Use ISO codes for AI stream lookup
		if lang != "default" && lang != cfg.AIOriginalLanguage && appState.Translator != nil {
			ch = appState.Translator.GetChannel(lang)
			if ch != nil {
				isTranslated = true
			}
		}

		if ch == nil {
			// fallback to standard stream
			ch = make(chan []float32, 100)
			appState.StreamChannels.Store(ch, true)
			defer appState.StreamChannels.Delete(ch)
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
