package web

import (
	"abel/src/backend/lib/audioengine"
	"abel/src/backend/lib/audioengine/audio_processing"
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

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
	logger := slog.With("component", "engine")
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

type HLSProvider interface {
	EnsureStream(language string, sampleRate int, source <-chan []float32) error
	WaitForPlaylist(ctx context.Context, language string, timeout time.Duration) ([]byte, error)
	ReadSegment(language, name string) ([]byte, error)
}

func streamLanguage(path string) string {
	language := filepath.Base(path)
	if language == "stream" || language == "." || language == "/" || language == "" {
		return "default"
	}
	return language
}

func StreamHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		language := streamLanguage(c.Param("lang"))
		target := "/api/audio/hls/" + url.PathEscape(language) + "/index.m3u8"
		c.Redirect(http.StatusTemporaryRedirect, target)
	}
}

func HLSPlaylistHandler(appState *state.AppState, cfg *config.Config, publisher HLSProvider) gin.HandlerFunc {
	logger := slog.With("component", "hls")
	return func(c *gin.Context) {
		language := streamLanguage(c.Param("lang"))
		sampleRate := int(appState.Config().SampleRate())
		if sampleRate <= 0 {
			sampleRate = cfg.SampleRate
		}

		if language != "default" {
			resolved := cfg.ResolveLanguageName(language)
			if appState.IsLanguageBlocked(language) || appState.IsLanguageBlocked(resolved) {
				logger.Warn("HLS stream requested for blocked language", slog.String("stream.language", language))
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "translation audio is blocked"})
				return
			}
		}

		if language != "default" && appState.Translator == nil {
			c.Redirect(http.StatusTemporaryRedirect, "/api/audio/hls/default/index.m3u8")
			return
		}

		var source <-chan []float32
		if language == "default" {
			source = appState.PlaybackChan
		} else {
			source = appState.Translator.GetChannel(language)
			if source == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "translation audio is unavailable"})
				return
			}
		}

		if err := publisher.EnsureStream(language, sampleRate, source); err != nil {
			logger.Error("Failed to start HLS stream", slog.String("stream.language", language), slog.Any("error", err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "audio stream is unavailable"})
			return
		}

		playlist, err := publisher.WaitForPlaylist(c.Request.Context(), language, 8*time.Second)
		if err != nil {
			logger.Warn("HLS playlist not ready", slog.String("stream.language", language), slog.Any("error", err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "audio stream is not ready"})
			return
		}

		c.Header("Cache-Control", "no-store")
		c.Header("X-Accel-Buffering", "no")
		c.Data(http.StatusOK, "application/vnd.apple.mpegurl", playlist)
	}
}

// HLSSegmentHandler serves live .ts audio segments generated by FFmpeg.
func HLSSegmentHandler(publisher HLSProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := publisher.ReadSegment(c.Param("lang"), c.Param("segment"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) || errors.Is(err, audio_processing.ErrHLSNotReady) {
				c.Status(http.StatusNotFound)
				return
			}
			c.Status(http.StatusBadRequest)
			return
		}
		c.Header("Cache-Control", "public, max-age=30, immutable")
		c.Data(http.StatusOK, "video/mp2t", data)
	}
}
