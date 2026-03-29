package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/state"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRecordingStatus(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) {
		s.SetRecording(true)
	})
	appState.Engine().AddSamples(1234)

	cfg := &config.Config{}
	router := setupTestRouter(appState, cfg)

	req, _ := http.NewRequest("GET", "/api/recordings", nil)
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, true, resp["isRecording"])
	assert.Equal(t, float64(1234), resp["samples"])
}

func TestListRecordingFiles(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "test_recordings_*")
	defer os.RemoveAll(tmpDir)
	os.WriteFile(tmpDir+"/rec1.wav", []byte("data"), 0644)

	appState := state.NewAppState("", "")
	cfg := &config.Config{StorageLocation: tmpDir}
	router := setupTestRouter(appState, cfg)

	req, _ := http.NewRequest("GET", "/api/recordings/files", nil)
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	t.Run("Empty Directory", func(t *testing.T) {
		emptyDir, _ := os.MkdirTemp("", "empty_*")
		defer os.RemoveAll(emptyDir)
		cfg.StorageLocation = emptyDir
		req, _ := http.NewRequest("GET", "/api/recordings/files", nil)
		req.Header.Set("X-Test-Auth", "true")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestPushRecordingToCloud(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "test_push_*")
	defer os.RemoveAll(tmpDir)
	os.WriteFile(tmpDir+"/rec.wav", []byte("data"), 0644)

	appState := state.NewAppState("", "")
	cfg := &config.Config{StorageLocation: tmpDir, CloudDriveLocation: tmpDir + "/cloud"}
	os.Mkdir(cfg.CloudDriveLocation, 0755)

	router := setupTestRouter(appState, cfg)
	body := map[string]interface{}{"source": "rec.wav", "target": "pushed.wav"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/recordings/push", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
