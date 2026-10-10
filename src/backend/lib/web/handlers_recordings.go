package web

import (
	"abel/src/backend/lib/audioengine/audio_processing"
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/recording"
	"abel/src/backend/lib/state"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
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
func createRecordingWavFile(folder string, sampleRate int) (*os.File, string, error) {
	if err := os.MkdirAll(folder, 0755); err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("rec_%d.wav", time.Now().Unix())
	base := filepath.Join(folder, filename)
	file, err := os.Create(base)
	if err != nil {
		return nil, "", err
	}
	if err := audio_processing.WritePlaceholderWavHeader(file, 2, sampleRate); err != nil {
		file.Close()
		return nil, "", err
	}
	return file, filename, nil
}

func startRecordingSession(appState *state.AppState, sampleRate int, folder string, boost *float64) (string, error) {
	if folder == "" {
		folder = appState.Locations().Storage()
	}
	file, filename, err := createRecordingWavFile(folder, sampleRate)
	if err != nil {
		return "", err
	}
	var startErr error
	state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) {
		if s.IsRecording() {
			startErr = fmt.Errorf("already recording")
			return
		}
		appState.Engine().SetFile(file)
		appState.Engine().ResetSamples()
		s.SetRecording(true)
	})
	if startErr != nil {
		file.Close()
		os.Remove(filepath.Join(folder, filename))
		return "", startErr
	}
	if boost != nil {
		state.Update[state.AudioEngineUIConfig](appState, state.SectionInterface, func(si *state.AudioEngineUIConfig) {
			si.SetBoost(*boost)
		})
	}
	slog.With("component", "recording").Info("Recording started", slog.String("recording.file", filename))
	return filename, nil
}

func stopRecordingSession(appState *state.AppState, sampleRate int) (string, error) {
	var file *os.File
	var samplesWrote int64
	var stopErr error

	state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) {
		if !s.IsRecording() {
			stopErr = fmt.Errorf("not currently recording")
			return
		}
		file, samplesWrote = appState.Engine().TakeFile()
		s.SetRecording(false)
	})

	if stopErr != nil {
		return "", stopErr
	}
	if file == nil {
		return "", fmt.Errorf("no file to finalize")
	}

	filename := filepath.Base(file.Name())
	if errFinalize := audio_processing.FinalizeWavHeaderWithSamples(file, 2, samplesWrote, sampleRate); errFinalize != nil {
		file.Close()
		return "", errFinalize
	}
	if errClose := file.Close(); errClose != nil {
		return "", errClose
	}

	slog.With("component", "recording").Info("Recording stopped",
		slog.String("recording.file", filename),
		slog.Int("recording.samples", int(samplesWrote)),
	)
	return filename, nil
}

func handleStartRecording(c *gin.Context, appState *state.AppState, sampleRate int, folder string, boost *float64) {
	filename, err := startRecordingSession(appState, sampleRate, folder, boost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "Recording started", "file": filename})
}

func handleStopRecording(c *gin.Context, appState *state.AppState, sampleRate int, processor *recording.RecordingProcessor) {
	filename, err := stopRecordingSession(appState, sampleRate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response := gin.H{"status": "Recording stopped", "file": filename}
	if processor != nil {
		if job, processErr := processor.Enqueue(filename, 0, 0, true); processErr != nil {
			slog.With("component", "recording").Error("Could not queue recording processing", "file", filename, "error", processErr)
			response["processingError"] = processErr.Error()
		} else {
			response["jobId"] = job.ID
		}
	}
	c.JSON(http.StatusOK, response)
}

func CreateRecording(appState *state.AppState, cfg *config.Config, processor *recording.RecordingProcessor) gin.HandlerFunc {
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

		sampleRate := int(appState.Config().SampleRate())
		if sampleRate <= 0 {
			sampleRate = cfg.SampleRate
		}

		switch req.Action {
		case "start":
			handleStartRecording(c, appState, sampleRate, req.Folder, req.Boost)
		case "stop":
			handleStopRecording(c, appState, sampleRate, processor)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
		}
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

func PushRecordingToCloud(processor *recording.RecordingProcessor) gin.HandlerFunc {
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
			if errors.Is(err, recording.ErrInvalidExport) {
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

func ListRecordingLibrary(processor *recording.RecordingProcessor) gin.HandlerFunc {
	return func(c *gin.Context) {
		entries, err := processor.Library()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list recordings"})
			return
		}
		c.JSON(http.StatusOK, entries)
	}
}

func ProcessRecording(processor *recording.RecordingProcessor) gin.HandlerFunc {
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
		job, err := processor.Enqueue(req.Source, req.StartSeconds, req.EndSeconds, req.Automatic)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, recording.ErrQueueFull) {
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
func CancelRecordingProcessing(processor *recording.RecordingProcessor) gin.HandlerFunc {
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

func storeUploadedAudio(part *multipart.Part, storageDir string) (string, int64, error) {
	base := filepath.Base(strings.ReplaceAll(part.FileName(), "\\", "/"))
	if len(base) > 180 || !recording.ValidAudioName(base) {
		return "", 0, errors.New("Supported files: WAV, MP3, M4A, FLAC, AAC, OGG (name up to 180 bytes)")
	}
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return "", 0, fmt.Errorf("create directory: %w", err)
	}
	tmp, err := os.CreateTemp(storageDir, ".abel-upload-*")
	if err != nil {
		return "", 0, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	copied, copyErr := io.Copy(tmp, part)
	if closeErr := tmp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return "", copied, copyErr
	}
	if copied == 0 {
		return "", 0, errors.New("Audio file is empty")
	}
	if err := recording.ValidateAudioFile(tmp.Name()); err != nil {
		return "", copied, errors.New("File does not contain readable audio")
	}
	name := fmt.Sprintf("import-%d-%s", time.Now().UnixNano(), base)
	if err := os.Chmod(tmp.Name(), 0644); err != nil {
		return "", copied, err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(storageDir, name)); err != nil {
		return "", copied, fmt.Errorf("publish audio: %w", err)
	}
	return name, copied, nil
}

func respondUploadError(c *gin.Context, err error, bytesWritten int64) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) || bytesWritten > maxRecordingUpload {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Audio file exceeds 4 GB"})
	} else if strings.Contains(err.Error(), "Supported files") || strings.Contains(err.Error(), "empty") || strings.Contains(err.Error(), "readable") {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	} else {
		slog.Error("Audio upload failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store audio file"})
	}
}

func UploadRecording(processor *recording.RecordingProcessor) gin.HandlerFunc {
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

		name, bytesWritten, err := storeUploadedAudio(part, processor.StorageLocation())
		if err != nil {
			respondUploadError(c, err, bytesWritten)
			return
		}

		slog.Info("Audio file imported", "file", name, "bytes", bytesWritten)
		response := gin.H{"file": name}
		if job, err := processor.Enqueue(name, 0, 0, true); err != nil {
			slog.Error("Could not queue imported audio", "file", name, "error", err)
			response["processingError"] = err.Error()
		} else {
			response["jobId"] = job.ID
		}
		c.JSON(http.StatusCreated, response)
	}
}
