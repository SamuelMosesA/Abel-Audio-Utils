package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func setupTestRouter(stateObj *state.AppState, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	store := cookie.NewStore([]byte("secret"))
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
		RegisterAdminRoutes(api, stateObj, cfg)
		api.GET("/recordings", GetRecordingStatus(stateObj))
		api.GET("/ai/streams", GetAIStreamsStatus(stateObj))
		api.GET("/system/connection", GetSystemConnection(cfg))
	}
	r.GET("/stream", StreamHandler(stateObj, cfg))
	r.GET("/subtitles/:lang", SubtitlesHandler(stateObj, cfg))
	r.GET("/ws", NewWSHandler(stateObj, cfg))
	
	return r
}

type MockTranslator struct {
	state.Translator
	subtitleChan chan string
}

func (m *MockTranslator) SetEnabled(enabled bool) {}
func (m *MockTranslator) GetChannel(lang string) chan []float32 { return nil }
func (m *MockTranslator) ListSessions() []state.SessionInfo {
	return []state.SessionInfo{{Language: "en"}}
}
func (m *MockTranslator) GetSubtitles(lang string) (chan string, func()) {
	m.subtitleChan = make(chan string, 10)
	return m.subtitleChan, func() { if m.subtitleChan != nil { close(m.subtitleChan); m.subtitleChan = nil } }
}
func (m *MockTranslator) StopSession(lang string, subs bool) {}
func (m *MockTranslator) CloseAll() {}
func (m *MockTranslator) PushAudio(samples []float32) {}
func (m *MockTranslator) SetOnStateChange(fn func()) {}

var testCfg = &config.Config{}
