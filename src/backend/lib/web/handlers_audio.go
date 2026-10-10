package web

import (
	"abel/src/backend/lib/audioengine"
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

var restartingAudio atomic.Bool

func DevicesHandler(state *state.AppState) gin.HandlerFunc {
	logger := slog.With("component", "api")
	return func(c *gin.Context) {
		logger.Info("Device list requested")
		devices := state.Devices
		list := audioengine.GetDevices(devices)
		c.JSON(http.StatusOK, list)
	}
}

type UpdateAudioConfigRequest struct {
	DeviceID *int     `json:"deviceID"`
	ChL      *int     `json:"chL"`
	ChR      *int     `json:"chR"`
	Boost    *float64 `json:"boost"`
}

// @Summary Update audio configuration
// @Description Updates active audio input device, channel mapping, or boost
// @Tags Audio
// @Accept json
// @Produce json
// @Param body body UpdateAudioConfigRequest true "Audio configuration updates"
// @Success 200 {object} object "Interface updated"
// @Failure 400 {object} string "Invalid request body"
// @Failure 401 {object} string "Unauthorized"
// @Failure 500 {object} string "Internal Error"
// @Security CookieAuth
// @Security BasicAuth
// @Router /api/audio/config [patch]
func UpdateAudioConfig(appState *state.AppState, cfg *config.Config) gin.HandlerFunc {
	logger := slog.With("component", "api")
	return func(c *gin.Context) {
		var req UpdateAudioConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		state.Update[state.AudioEngineUIConfig](appState, state.SectionInterface, func(s *state.AudioEngineUIConfig) {
			if req.DeviceID != nil {
				err := audioengine.StartAudioEngine(nil, appState, cfg, *req.DeviceID, appState.RecordChan, appState.PlaybackChan)
				if err != nil {
					logger.Error("Error starting audio engine",
						slog.Any("audio.error", err),
					)
				} else {
					s.SetIsRunning(true)
					s.SetDeviceID(int32(*req.DeviceID))
					logger.Info("Audio engine started",
						slog.Int("audio.device_id", *req.DeviceID),
					)
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

// @Summary Restart audio engine
// @Description Re-scans audio devices so hardware connected after startup appears, reloads config.yaml and the credentials file, and reconnects to the previous device by name. Refused while recording or while a restart is in progress.
// @Tags Audio
// @Produce json
// @Success 200 {object} audioengine.RestartResult
// @Failure 401 {object} string "Unauthorized"
// @Failure 409 {object} string "Recording or restart in progress"
// @Failure 500 {object} string "Internal Error"
// @Security CookieAuth
// @Security BasicAuth
// @Router /api/audio/restart [post]
func RestartAudioEngine(appState *state.AppState, cfg *config.Config) gin.HandlerFunc {
	logger := slog.With("component", "engine")
	return func(c *gin.Context) {
		if appState.IsRecording() {
			c.JSON(http.StatusConflict, gin.H{"error": "Cannot restart the engine while recording"})
			return
		}

		if !restartingAudio.CompareAndSwap(false, true) {
			c.JSON(http.StatusConflict, gin.H{"error": "Audio engine restart already in progress"})
			return
		}
		defer restartingAudio.Store(false)

		result, err := audioengine.RestartEngine(nil, appState, cfg)
		if err != nil {
			logger.Error("Engine restart failed", slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		logger.Info("Engine restarted",
			slog.Int("audio.device_count", len(result.Devices)),
			slog.String("audio.reconnected", result.Reconnected),
			slog.String("config.error", result.ConfigError),
		)
		c.JSON(http.StatusOK, result)
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
			"sampleRate":         conf.SampleRate(),
			"storageLocation":    loc.Storage(),
			"cloudDriveLocation": loc.CloudDrive(),
		})
	}
}

// AudioStreamBroadcaster defines the interface for subscribing to real-time progressive audio streams.
type AudioStreamBroadcaster interface {
	Subscribe(language string, sampleRate int, source <-chan []float32) (<-chan []byte, func(), error)
}

func streamLanguage(path string) string {
	clean := strings.TrimPrefix(path, "/")
	clean = strings.TrimSpace(clean)
	if clean == "" || clean == "." || clean == "stream" {
		return "default"
	}
	return filepath.Base(clean)
}

// StreamHandler streams live captured or translated audio as progressive chunked MP3 frames.
func StreamHandler(appState *state.AppState, cfg *config.Config, broadcaster AudioStreamBroadcaster) gin.HandlerFunc {
	logger := slog.With("component", "audio_stream")
	return func(c *gin.Context) {
		language := streamLanguage(c.Param("lang"))
		sampleRate := int(appState.Config().SampleRate())
		if sampleRate <= 0 {
			sampleRate = cfg.SampleRate
		}
		if sampleRate <= 0 {
			sampleRate = 48000
		}

		if language != "default" {
			resolved := cfg.ResolveLanguageName(language)
			if appState.IsLanguageBlocked(language) || appState.IsLanguageBlocked(resolved) {
				logger.Warn("Audio stream requested for blocked language", slog.String("stream.language", language))
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "translation audio is blocked"})
				return
			}
		}

		var source <-chan []float32
		if language == "default" {
			source = appState.PlaybackChan
		} else {
			if appState.Translator == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "translation audio is unavailable"})
				return
			}
			source = appState.Translator.GetChannel(language)
			if source == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "translation audio is unavailable"})
				return
			}
		}

		if broadcaster == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "audio streaming is disabled"})
			return
		}

		ch, unsubscribe, err := broadcaster.Subscribe(language, sampleRate, source)
		if err != nil {
			logger.Error("Failed to subscribe to audio stream", slog.String("stream.language", language), slog.Any("error", err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "audio stream is unavailable"})
			return
		}
		defer unsubscribe()

		c.Header("Content-Type", "audio/mpeg")
		c.Header("Transfer-Encoding", "chunked")
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)

		for {
			select {
			case <-c.Request.Context().Done():
				return
			case chunk, ok := <-ch:
				if !ok {
					return
				}
				if _, err := c.Writer.Write(chunk); err != nil {
					return
				}
				c.Writer.Flush()
			}
		}
	}
}
