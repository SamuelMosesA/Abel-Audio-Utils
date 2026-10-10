package web

import (
	"abel/src/backend/lib/audioengine"
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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
func CreateRecording(appState *state.AppState, cfg *config.Config, processor *RecordingProcessor) gin.HandlerFunc {
	logger := slog.With("component", "recording")
	return func(c *gin.Context) {
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

		sampleRate := int(appState.Config().SampleRate())
		if sampleRate <= 0 {
			sampleRate = cfg.SampleRate
		}

		var shouldUpdateBoost bool
		state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) {
			isRecording := s.IsRecording()

			if req.Action == "start" {
				if isRecording {
					err = fmt.Errorf("already recording")
					return
				}
				folder := req.Folder
				if folder == "" {
					folder = appState.Locations().Storage()
				}
				os.MkdirAll(folder, 0755)
				filename := fmt.Sprintf("rec_%d.wav", time.Now().Unix())
				base := filepath.Join(folder, filename)
				file, errCreate := os.Create(base)
				if errCreate != nil {
					err = errCreate
					return
				}
				if errHeader := audioengine.WritePlaceholderHeader(file, 2, sampleRate); errHeader != nil {
					file.Close()
					err = errHeader
					return
				}

				appState.Engine().SetFile(file)
				appState.Engine().ResetSamples()
				s.SetRecording(true)
				if req.Boost != nil {
					shouldUpdateBoost = true
				}
				logger.Info("Recording started",
					slog.String("recording.file", filename),
				)
				respStatus = "Recording started"
				respFile = filename

			} else if req.Action == "stop" {
				if !isRecording {
					err = fmt.Errorf("not currently recording")
					return
				}

				file, samplesWrote := appState.Engine().TakeFile()
				s.SetRecording(false)

				if file == nil {
					err = fmt.Errorf("no file to finalize")
					return
				}

				filename := filepath.Base(file.Name())
				if errFinalize := audioengine.FinalizeWavHeader(file, 2, samplesWrote, sampleRate); errFinalize != nil {
					file.Close()
					err = errFinalize
					return
				}
				if errClose := file.Close(); errClose != nil {
					err = errClose
					return
				}

				logger.Info("Recording stopped",
					slog.String("recording.file", filename),
					slog.Int("recording.samples", int(samplesWrote)),
				)
				respStatus = "Recording stopped"
				respFile = filename
			} else {
				err = fmt.Errorf("invalid action")
			}
		})

		if err == nil && shouldUpdateBoost && req.Boost != nil {
			state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(si *state.InterfaceConfig) {
				si.SetBoost(*req.Boost)
			})
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		response := gin.H{"status": respStatus, "file": respFile}
		if req.Action == "stop" && processor != nil {
			job, processErr := processor.enqueue(respFile, 0, 0, true)
			if processErr != nil {
				logger.Error("Could not queue recording processing", "file", respFile, "error", processErr)
				response["processingError"] = processErr.Error()
			} else {
				response["jobId"] = job.ID
			}
		}
		c.JSON(http.StatusOK, response)
	}
}

// @Summary Get recording status
// @Description Returns whether the system is recording
// @Tags Recordings
// @Produce json
// @Success 200 {object} object "Recording Status"
// @Failure 401 {object} string "Unauthorized"
// @Router /api/recordings [get]
func GetRecordingStatus(appState *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{
			"isRecording": appState.IsRecording(),
			"samples":     appState.Engine().SamplesWrote(),
		}
		c.JSON(http.StatusOK, status)
	}
}

// ListRecordingFiles enumerates saved audio recordings available for download.
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

func PushRecordingToCloud(processor *RecordingProcessor) gin.HandlerFunc {
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
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if err := processor.Push(req.Source); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, errInvalidExport) {
				status = http.StatusBadRequest
			} else if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
			}
			if status == http.StatusInternalServerError {
				slog.Error("audio push failed", "file", req.Source, "error", err)
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}

func ListRecordingLibrary(processor *RecordingProcessor) gin.HandlerFunc {
	return func(c *gin.Context) {
		entries, err := processor.Library()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list recordings"})
			return
		}
		c.JSON(http.StatusOK, entries)
	}
}

func ProcessRecording(processor *RecordingProcessor) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Source       string  `json:"source"`
			StartSeconds float64 `json:"startSeconds"`
			EndSeconds   float64 `json:"endSeconds"`
			Automatic    bool    `json:"automatic"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		if !req.Automatic && req.StartSeconds == 0 && req.EndSeconds == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Manual trim needs a start or end time"})
			return
		}
		job, err := processor.enqueue(req.Source, req.StartSeconds, req.EndSeconds, req.Automatic)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, errQueueFull) {
				status = http.StatusServiceUnavailable
			} else if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, job)
	}
}

// CancelRecordingProcessing stops a queued or running FFmpeg job.
// @Summary Stop recording processing
// @Tags Recordings
// @Accept json
// @Produce json
// @Success 200 {object} string "Cancelled"
// @Security CookieAuth
// @Security BasicAuth
// @Router /api/recordings/process/cancel [post]
func CancelRecordingProcessing(processor *RecordingProcessor) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ID string `json:"id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Processing job ID is required"})
			return
		}
		if err := processor.Cancel(req.ID); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
	}
}

const maxRecordingUpload = int64(4 << 30)

func UploadRecording(processor *RecordingProcessor) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRecordingUpload+(1<<20))
		reader, err := c.Request.MultipartReader()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Expected an audio file upload"})
			return
		}
		part, err := reader.NextPart()
		if err != nil || part.FormName() != "file" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing audio file"})
			return
		}
		defer part.Close()
		base := filepath.Base(strings.ReplaceAll(part.FileName(), "\\", "/"))
		if len(base) > 180 || !validAudioName(base) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Supported files: WAV, MP3, M4A, FLAC, AAC, OGG (name up to 180 bytes)"})
			return
		}
		if err := os.MkdirAll(processor.cfg.StorageLocation, 0755); err != nil {
			slog.Error("Could not create recordings directory", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store audio file"})
			return
		}
		tmp, err := os.CreateTemp(processor.cfg.StorageLocation, ".abel-upload-*")
		if err != nil {
			slog.Error("Could not create upload file", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store audio file"})
			return
		}
		defer os.Remove(tmp.Name())
		copied, copyErr := io.Copy(tmp, part)
		if closeErr := tmp.Close(); copyErr == nil {
			copyErr = closeErr
		}
		if copyErr != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(copyErr, &tooLarge) || copied > maxRecordingUpload {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Audio file exceeds 4 GB"})
			} else {
				slog.Error("Audio upload failed", "name", base, "error", copyErr)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store audio file"})
			}
			return
		}
		if copied > maxRecordingUpload {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Audio file exceeds 4 GB"})
			return
		}
		if copied == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Audio file is empty"})
			return
		}
		if err := validateAudioFile(tmp.Name()); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File does not contain readable audio"})
			return
		}
		name := fmt.Sprintf("import-%d-%s", time.Now().UnixNano(), base)
		if err := os.Chmod(tmp.Name(), 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store audio file"})
			return
		}
		if err := os.Rename(tmp.Name(), filepath.Join(processor.cfg.StorageLocation, name)); err != nil {
			slog.Error("Could not publish uploaded audio", "name", base, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store audio file"})
			return
		}
		slog.Info("Audio file imported", "file", name, "bytes", copied)
		response := gin.H{"file": name}
		job, err := processor.enqueue(name, 0, 0, true)
		if err != nil {
			slog.Error("Could not queue imported audio", "file", name, "error", err)
			response["processingError"] = err.Error()
		} else {
			response["jobId"] = job.ID
		}
		c.JSON(http.StatusCreated, response)
	}
}
