package web

import (
	"behringerRecorder/lib/audioengine"
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/state"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
func CreateRecording(appState *state.AppState, cfg *config.Config) gin.HandlerFunc {
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
				if errHeader := audioengine.WritePlaceholderHeader(file, 2, cfg.SampleRate); errHeader != nil {
					file.Close()
					err = errHeader
					return
				}

				appState.Engine().SetFile(file)
				appState.Engine().ResetSamples()
				s.SetRecording(true)
				if req.Boost != nil {
					state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(si *state.InterfaceConfig) {
						si.SetBoost(*req.Boost)
					})
				}
				fmt.Printf("[RECORDING] START - File: %s\n", filename)
				respStatus = "Recording started"
				respFile = filename

			} else if req.Action == "stop" {
				if !isRecording {
					err = fmt.Errorf("not currently recording")
					return
				}

				file := appState.Engine().File()
				appState.Engine().SetFile(nil)
				samplesWrote := appState.Engine().SamplesWrote()

				if file == nil {
					err = fmt.Errorf("no file to finalize")
					return
				}

				filename := filepath.Base(file.Name())
				if errFinalize := audioengine.FinalizeWavHeader(file, 2, samplesWrote, cfg.SampleRate); errFinalize != nil {
					file.Close()
					err = errFinalize
					return
				}
				if errClose := file.Close(); errClose != nil {
					err = errClose
					return
				}

				s.SetRecording(false)
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
func GetRecordingStatus(appState *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{
			"isRecording": appState.IsRecording(),
			"samples":     appState.Engine().SamplesWrote(),
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
