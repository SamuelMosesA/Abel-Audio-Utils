package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func setupTestRouter(stateObj *state.AppState, cfg *config.Config) *gin.Engine {
	return setupTestRouterWithProcessor(stateObj, cfg, NewRecordingProcessor(cfg))
}

func setupTestRouterWithProcessor(stateObj *state.AppState, cfg *config.Config, processor *RecordingProcessor) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	sessionSecret, _ := config.ResolveSessionSecret(cfg)
	store := cookie.NewStore(sessionSecret)
	r.Use(sessions.Sessions("abel_session", store))

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
		api.POST("/auth/session", LoginHandler(cfg, stateObj))
		api.DELETE("/auth/session", LogoutHandler(stateObj))
		api.POST("/auth/logout", LogoutHandler(stateObj))
		RegisterAdminRoutes(api, stateObj, cfg, processor)
		api.GET("/recordings", GetRecordingStatus(stateObj))
		api.GET("/ai/streams", GetAIStreamsStatus(stateObj, cfg))
		api.GET("/system/connection", GetSystemConnection(cfg))
	}
	r.GET("/stream", StreamHandler())
	r.GET("/subtitles/:lang", SubtitlesHandler(stateObj, cfg))
	r.GET("/ws", NewWSHandler(stateObj, cfg))

	return r
}

type MockTranslator struct {
	state.Translator
	mu           sync.Mutex
	subtitleChan chan string
}

func (m *MockTranslator) SetEnabled(enabled bool)               {}
func (m *MockTranslator) GetChannel(lang string) chan []float32 { return nil }
func (m *MockTranslator) ListSessions() []state.SessionInfo {
	return []state.SessionInfo{{Language: "en"}}
}
func (m *MockTranslator) GetSubtitles(lang string) (chan string, func()) {
	m.mu.Lock()
	m.subtitleChan = make(chan string, 10)
	channel := m.subtitleChan
	m.mu.Unlock()
	return channel, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.subtitleChan != nil {
			close(m.subtitleChan)
			m.subtitleChan = nil
		}
	}
}
func (m *MockTranslator) SendSubtitle(value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.subtitleChan != nil {
		m.subtitleChan <- value
	}
}
func (m *MockTranslator) StopSession(lang string, subs bool) {}
func (m *MockTranslator) CloseAll()                          {}
func (m *MockTranslator) PushAudio(samples []float32)        {}
func (m *MockTranslator) SetOnStateChange(fn func())         {}
func (m *MockTranslator) GetListenerCount(lang string) int   { return 0 }

var testCfg = &config.Config{}
