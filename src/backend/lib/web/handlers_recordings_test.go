package web

import (
	"abel/src/backend/lib/audioengine"
	"abel/src/backend/lib/audioengine/audio_processing"
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/recording"
	"abel/src/backend/lib/state"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRecordingHelper(t *testing.T, dir string) string {
	t.Helper()
	name := "rec_test.wav"
	f, err := os.Create(filepath.Join(dir, name))
	require.NoError(t, err)
	defer f.Close()
	const rate = 44100
	const frames = rate * 6
	require.NoError(t, audio_processing.WritePlaceholderWavHeader(f, 2, rate))
	data := make([]byte, frames*4)
	for i := 0; i < frames; i++ {
		seconds := float64(i) / rate
		sample := int16(0)
		if seconds >= 1.2 && seconds < 3.2 {
			sample = int16(math.Sin(seconds*2*math.Pi*440) * 12000)
		}
		binary.LittleEndian.PutUint16(data[i*4:], uint16(sample))
		binary.LittleEndian.PutUint16(data[i*4+2:], uint16(sample))
	}
	_, err = f.Write(data)
	require.NoError(t, err)
	dataBytes := int64(frames * 2 * audio_processing.WavBytesPerSample)
	require.NoError(t, audio_processing.FinalizeWavHeader(f, 2, dataBytes, rate))
	return name
}

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
	processor := recording.NewRecordingProcessor(cfg)
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
	finalStage := ""
	for time.Now().Before(deadline) {
		entries, _ := processor.Library()
		for _, e := range entries {
			for _, j := range e.Jobs {
				if j.ID == jobID {
					finalStage = j.Stage
				}
			}
		}
		if finalStage == "failed" || finalStage == "completed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
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
	data, err := os.ReadFile(filepath.Join(cfg.CloudDriveLocation, recording.CloudFilename("rec.wav", "rec-processed.mp3", info.ModTime())))
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

func TestRecordingLibraryAndProcessingRequireAuthentication(t *testing.T) {
	dir := t.TempDir()
	name := testRecordingHelper(t, dir)
	cfg := &config.Config{StorageLocation: dir, CloudDriveLocation: filepath.Join(dir, "cloud")}
	router := setupTestRouter(state.NewAppState(dir, cfg.CloudDriveLocation), cfg)

	for _, path := range []string{"/api/recordings/library", "/api/recordings/process", "/api/recordings/process/cancel"} {
		method := http.MethodGet
		if path != "/api/recordings/library" {
			method = http.MethodPost
		}
		req := httptest.NewRequest(method, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s without session returned %d", path, response.Code)
		}
	}

	body, _ := json.Marshal(map[string]interface{}{"source": name, "startSeconds": 4, "endSeconds": 2})
	req := httptest.NewRequest(http.MethodPost, "/api/recordings/process", bytes.NewReader(body))
	req.Header.Set("X-Test-Auth", "true")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid trim returned %d: %s", response.Code, response.Body.String())
	}
}

func TestUploadAudioStartsAutomaticProcessing(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	inputDir := t.TempDir()
	wav := filepath.Join(inputDir, testRecordingHelper(t, inputDir))
	mp3 := filepath.Join(inputDir, "guest-sermon.mp3")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-i", wav, "-codec:a", "libmp3lame", mp3).CombinedOutput(); err != nil {
		t.Fatalf("prepare MP3: %v: %s", err, out)
	}
	storage := t.TempDir()
	cfg := &config.Config{StorageLocation: storage, CloudDriveLocation: filepath.Join(storage, "cloud")}
	processor := recording.NewRecordingProcessor(cfg)
	router := setupTestRouterWithProcessor(state.NewAppState(storage, cfg.CloudDriveLocation), cfg, processor)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "guest-sermon.mp3")
	require.NoError(t, err)
	f, err := os.Open(mp3)
	require.NoError(t, err)
	_, err = io.Copy(part, f)
	require.NoError(t, err)
	f.Close()
	writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/recordings/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-Auth", "true")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload returned %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		File  string `json:"file"`
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || payload.JobID == "" {
		t.Fatalf("upload response: %s: %v", response.Body.String(), err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		entries, err := processor.Library()
		if err == nil {
			for _, e := range entries {
				for _, j := range e.Jobs {
					if j.ID == payload.JobID {
						if j.Stage == "completed" {
							sourceInfo, err := os.Stat(filepath.Join(storage, payload.File))
							require.NoError(t, err)
							_, err = os.Stat(filepath.Join(cfg.CloudDriveLocation, recording.CloudFilename(payload.File, j.Output, sourceInfo.ModTime())))
							require.NoError(t, err)
							goto done
						}
						if j.Stage == "failed" {
							t.Fatalf("import job ended in %s: %s", j.Stage, j.Error)
						}
					}
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for import job completion")
		}
		time.Sleep(20 * time.Millisecond)
	}
done:
	library, err := processor.Library()
	if err != nil || len(library) != 1 || library[0].Display != "guest-sermon.mp3" {
		t.Fatalf("import library: %+v, %v", library, err)
	}
}

func TestUploadRejectsUnreadableAudioAndRequiresSession(t *testing.T) {
	storage := t.TempDir()
	cfg := &config.Config{StorageLocation: storage, CloudDriveLocation: filepath.Join(storage, "cloud")}
	router := setupTestRouter(state.NewAppState(storage, cfg.CloudDriveLocation), cfg)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "broken.wav")
	require.NoError(t, err)
	part.Write([]byte("not audio"))
	writer.Close()
	request := func(auth bool) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/api/recordings/upload", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		if auth {
			req.Header.Set("X-Test-Auth", "true")
		}
		return req
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request(false))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated upload returned %d", response.Code)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request(true))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unreadable audio returned %d: %s", response.Code, response.Body.String())
	}
	library, err := os.ReadDir(storage)
	if err != nil || len(library) != 0 {
		t.Fatalf("invalid upload left files: %+v, %v", library, err)
	}
}
