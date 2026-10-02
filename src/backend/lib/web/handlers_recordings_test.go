package web

import (
	"abel/src/backend/lib/audioengine"
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		emptyRouter := setupTestRouter(appState, &config.Config{StorageLocation: emptyDir})
		req, _ := http.NewRequest("GET", "/api/recordings/files", nil)
		req.Header.Set("X-Test-Auth", "true")
		w := httptest.NewRecorder()
		emptyRouter.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestCreateRecordingWritesValidWavHeader(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_recording_wav_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	appState := state.NewAppState(tmpDir, "")
	cfg := &config.Config{SampleRate: 44100, StorageLocation: tmpDir}
	processor := NewRecordingProcessor(cfg)
	router := setupTestRouterWithProcessor(appState, cfg, processor)

	startBody, _ := json.Marshal(map[string]string{"action": "start"})
	startReq, _ := http.NewRequest("POST", "/api/recordings", bytes.NewBuffer(startBody))
	startReq.Header.Set("X-Test-Auth", "true")
	startResp := httptest.NewRecorder()
	router.ServeHTTP(startResp, startReq)

	assert.Equal(t, http.StatusOK, startResp.Code)
	var startPayload map[string]string
	assert.NoError(t, json.Unmarshal(startResp.Body.Bytes(), &startPayload))

	file := appState.Engine().File()
	assert.NotNil(t, file)
	n, err := audioengine.WriteAudio(file, []float32{1.0, -1.0, 0.5, -0.5})
	assert.NoError(t, err)
	appState.Engine().AddSamples(int64(n))

	stopBody, _ := json.Marshal(map[string]string{"action": "stop"})
	stopReq, _ := http.NewRequest("POST", "/api/recordings", bytes.NewBuffer(stopBody))
	stopReq.Header.Set("X-Test-Auth", "true")
	stopResp := httptest.NewRecorder()
	router.ServeHTTP(stopResp, stopReq)

	assert.Equal(t, http.StatusOK, stopResp.Code)
	var stopPayload map[string]interface{}
	assert.NoError(t, json.Unmarshal(stopResp.Body.Bytes(), &stopPayload))
	assert.NotEmpty(t, stopPayload["jobId"])
	jobID, ok := stopPayload["jobId"].(string)
	assert.True(t, ok)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		processor.mu.RLock()
		job := processor.jobs[jobID]
		processor.mu.RUnlock()
		if job.Stage == "failed" || job.Stage == "completed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	processor.mu.RLock()
	finalStage := processor.jobs[jobID].Stage
	processor.mu.RUnlock()
	assert.Contains(t, []string{"failed", "completed"}, finalStage)

	recordingPath := filepath.Join(tmpDir, startPayload["file"])
	data, err := os.ReadFile(recordingPath)
	assert.NoError(t, err)
	assert.Len(t, data, 52)
	assert.Equal(t, "RIFF", string(data[0:4]))
	assert.Equal(t, uint32(44), binary.LittleEndian.Uint32(data[4:8]))
	assert.Equal(t, "WAVE", string(data[8:12]))
	assert.Equal(t, "fmt ", string(data[12:16]))
	assert.Equal(t, uint16(1), binary.LittleEndian.Uint16(data[20:22]))
	assert.Equal(t, uint16(2), binary.LittleEndian.Uint16(data[22:24]))
	assert.Equal(t, uint32(44100), binary.LittleEndian.Uint32(data[24:28]))
	assert.Equal(t, uint32(176400), binary.LittleEndian.Uint32(data[28:32]))
	assert.Equal(t, uint16(4), binary.LittleEndian.Uint16(data[32:34]))
	assert.Equal(t, uint16(16), binary.LittleEndian.Uint16(data[34:36]))
	assert.Equal(t, "data", string(data[36:40]))
	assert.Equal(t, uint32(8), binary.LittleEndian.Uint32(data[40:44]))
}

func TestPushRecordingToCloud(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "test_push_*")
	defer os.RemoveAll(tmpDir)
	os.WriteFile(tmpDir+"/rec.wav", []byte("data"), 0644)
	os.WriteFile(tmpDir+"/rec-processed.mp3", []byte("mp3 data"), 0644)

	appState := state.NewAppState("", "")
	cfg := &config.Config{StorageLocation: tmpDir, CloudDriveLocation: tmpDir + "/cloud"}
	os.Mkdir(cfg.CloudDriveLocation, 0755)

	router := setupTestRouter(appState, cfg)
	body := map[string]interface{}{"source": "rec-processed.mp3"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/recordings/push", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	info, err := os.Stat(filepath.Join(tmpDir, "rec.wav"))
	assert.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(cfg.CloudDriveLocation, cloudFilename("rec.wav", "rec-processed.mp3", info.ModTime())))
	assert.NoError(t, err)
	assert.Equal(t, "mp3 data", string(data))

	body["source"] = "../rec.wav"
	jsonBody, _ = json.Marshal(body)
	req, _ = http.NewRequest("POST", "/api/recordings/push", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-Test-Auth", "true")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateRecordingWithActiveStorageWorker(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_worker_recording_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	appState := state.NewAppState(tmpDir, "")
	cfg := &config.Config{SampleRate: 48000, StorageLocation: tmpDir}
	router := setupTestRouter(appState, cfg)

	audioengine.StartStorageWorker(appState, appState.RecordChan)

	startBody, _ := json.Marshal(map[string]string{"action": "start"})
	startReq, _ := http.NewRequest("POST", "/api/recordings", bytes.NewBuffer(startBody))
	startReq.Header.Set("X-Test-Auth", "true")
	startResp := httptest.NewRecorder()
	router.ServeHTTP(startResp, startReq)
	assert.Equal(t, http.StatusOK, startResp.Code)

	var startPayload map[string]string
	assert.NoError(t, json.Unmarshal(startResp.Body.Bytes(), &startPayload))

	// Send audio chunks to RecordChan
	for i := 0; i < 10; i++ {
		appState.RecordChan <- []float32{0.5, -0.5, 0.25, -0.25} // 2 stereo frames (4 samples)
	}

	time.Sleep(50 * time.Millisecond)

	stopBody, _ := json.Marshal(map[string]string{"action": "stop"})
	stopReq, _ := http.NewRequest("POST", "/api/recordings", bytes.NewBuffer(stopBody))
	stopReq.Header.Set("X-Test-Auth", "true")
	stopResp := httptest.NewRecorder()
	router.ServeHTTP(stopResp, stopReq)
	assert.Equal(t, http.StatusOK, stopResp.Code)

	recordingPath := filepath.Join(tmpDir, startPayload["file"])
	data, err := os.ReadFile(recordingPath)
	assert.NoError(t, err)

	assert.Equal(t, "RIFF", string(data[0:4]))
	assert.Equal(t, "WAVE", string(data[8:12]))
	assert.Equal(t, "fmt ", string(data[12:16]))
	assert.Equal(t, uint16(1), binary.LittleEndian.Uint16(data[20:22]))
	assert.Equal(t, uint16(2), binary.LittleEndian.Uint16(data[22:24]))
	assert.Equal(t, uint32(48000), binary.LittleEndian.Uint32(data[24:28]))
	assert.Equal(t, "data", string(data[36:40]))

	dataSize := binary.LittleEndian.Uint32(data[40:44])
	assert.Equal(t, uint32(len(data)-44), dataSize)
	assert.Equal(t, uint32(len(data)-8), binary.LittleEndian.Uint32(data[4:8]))
}
