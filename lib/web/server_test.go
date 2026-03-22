package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/types"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter(state *types.AppState, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("behringer_session", store))

	// Mock auth session if header is present
	r.Use(func(c *gin.Context) {
		if c.GetHeader("X-Test-Auth") == "true" {
			session := sessions.Default(c)
			session.Set("authenticated", true)
			session.Set("session_id", "test-session")
			session.Set("username", "admin")
			session.Save()
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.POST("/auth/session", LoginHandler(cfg, state))
		RegisterAdminRoutes(api, state, cfg)
		api.GET("/recordings", GetRecordingStatus(state))
		api.GET("/ai/streams", GetAIStreamsStatus(state))
		api.GET("/system/connection", GetSystemConnection(cfg))
	}
	r.GET("/stream", StreamHandler(state, cfg))
	r.GET("/subtitles/:lang", SubtitlesHandler(state))
	r.GET("/ws", NewWSHandler(state, cfg))
	
	return r
}

func TestLoginHandler(t *testing.T) {
	state := &types.AppState{}
	cfg := &config.Config{
		Credentials: map[string]string{"admin": "password"},
	}
	router := setupTestRouter(state, cfg)

	t.Run("Successful Login", func(t *testing.T) {
		body := map[string]string{"username": "admin", "password": "password"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/auth/session", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "success", resp["status"])
		assert.NotEmpty(t, resp["session"])
	})

	t.Run("Failed Login", func(t *testing.T) {
		body := map[string]string{"username": "admin", "password": "wrong"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/auth/session", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestGetRecordingStatus(t *testing.T) {
	state := &types.AppState{
		IsRecording:  true,
		SamplesWrote: 1234,
	}
	cfg := &config.Config{}
	router := setupTestRouter(state, cfg)

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

func TestAdminRoutesProtection(t *testing.T) {
	state := &types.AppState{}
	cfg := &config.Config{}
	router := setupTestRouter(state, cfg)

	t.Run("Unauthorized Access", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/recordings/files", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestChangeLogHandler(t *testing.T) {
	state := &types.AppState{}
	cfg := &config.Config{}
	router := setupTestRouter(state, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "/api/system/changelog", nil)
	req.Header.Set("X-Test-Auth", "true")

	w := httptest.NewRecorder()
	go router.ServeHTTP(w, req)

	time.Sleep(100 * time.Millisecond)
	state.UpdateState("test-session", "test-section", func() {})
	time.Sleep(100 * time.Millisecond)
	cancel()
}

func TestGetAIStreamsStatus(t *testing.T) {
	state := &types.AppState{GeminiEnabled: true}
	cfg := &config.Config{}
	mockTranslator := &MockTranslator{}
	state.Translator = mockTranslator
	router := setupTestRouter(state, cfg)

	req, _ := http.NewRequest("GET", "/api/ai/streams", nil)
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, true, resp["masterEnabled"])
	assert.NotEmpty(t, resp["sessions"])
}

func TestGetSystemConnection(t *testing.T) {
	cfg := &config.Config{Port: "8080"}
	router := setupTestRouter(&types.AppState{}, cfg)

	req, _ := http.NewRequest("GET", "/api/system/connection", nil)
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["serverUrl"], "8080")
}

func TestStreamHandler(t *testing.T) {
	state := &types.AppState{}
	cfg := &config.Config{SampleRate: 48000}
	router := setupTestRouter(state, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "/stream", nil)
	w := httptest.NewRecorder()
	go router.ServeHTTP(w, req)

	time.Sleep(100 * time.Millisecond)
	found := false
	state.StreamChannels.Range(func(key, value interface{}) bool {
		found = true
		ch := key.(chan []float32)
		ch <- []float32{0.1, 0.1}
		return false
	})
	assert.True(t, found)
	time.Sleep(100 * time.Millisecond)
	cancel()
}

func TestListRecordingFiles(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "test_recordings_*")
	defer os.RemoveAll(tmpDir)
	os.WriteFile(tmpDir+"/rec1.wav", []byte("data"), 0644)

	state := &types.AppState{}
	cfg := &config.Config{StorageLocation: tmpDir}
	router := setupTestRouter(state, cfg)

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

func TestUpdateAudioConfig(t *testing.T) {
	state := &types.AppState{}
	cfg := &config.Config{}
	router := setupTestRouter(state, cfg)

	body := map[string]interface{}{"chL": 1, "chR": 2, "boost": 2.5}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("PATCH", "/api/audio/config", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int32(1), state.ChLeft)
	assert.Equal(t, 2.5, state.Boost)
}

func TestPushRecordingToCloud(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "test_push_*")
	defer os.RemoveAll(tmpDir)
	os.WriteFile(tmpDir+"/rec.wav", []byte("data"), 0644)

	state := &types.AppState{}
	cfg := &config.Config{StorageLocation: tmpDir, CloudDriveLocation: tmpDir + "/cloud"}
	os.Mkdir(cfg.CloudDriveLocation, 0755)

	router := setupTestRouter(state, cfg)
	body := map[string]interface{}{"source": "rec.wav", "target": "pushed.wav"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/recordings/push", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSubtitlesHandler(t *testing.T) {
	state := &types.AppState{GeminiEnabled: true}
	mockTranslator := &MockTranslator{}
	state.Translator = mockTranslator
	router := setupTestRouter(state, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "/subtitles/English", nil)
	w := httptest.NewRecorder()
	go router.ServeHTTP(w, req)

	time.Sleep(100 * time.Millisecond)
	if mockTranslator.subtitleChan != nil {
		mockTranslator.subtitleChan <- "Hello"
	}
	time.Sleep(100 * time.Millisecond)
	cancel()
}

func TestNewWSHandler(t *testing.T) {
	state := &types.AppState{}
	cfg := &config.Config{AdminPassword: "pass"}
	server := httptest.NewServer(setupTestRouter(state, cfg))
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws?pass=pass"
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)
	found := false
	state.Clients.Range(func(key, value interface{}) bool {
		found = true
		return false
	})
	assert.True(t, found)
}

type MockTranslator struct {
	types.Translator
	subtitleChan chan string
}

func (m *MockTranslator) SetEnabled(enabled bool) {}
func (m *MockTranslator) GetChannel(lang string) chan []float32 { return nil }
func (m *MockTranslator) ListSessions() []types.SessionInfo {
	return []types.SessionInfo{{Language: "English"}}
}
func (m *MockTranslator) GetSubtitles(lang string) (chan string, func()) {
	m.subtitleChan = make(chan string, 10)
	return m.subtitleChan, func() { close(m.subtitleChan) }
}
func (m *MockTranslator) StopSession(lang string, subs bool) {}
func (m *MockTranslator) CloseAll() {}

var cfg = &config.Config{}
