package state

import (
	"sync"

	pa "github.com/gordonklaus/portaudio"
	"github.com/gorilla/websocket"
)

type StateChange struct {
	SessionID string      `json:"sessionId"`
	Section   string      `json:"section"`
	Details   interface{} `json:"details,omitempty"`
}

type SessionInfo struct {
	Language  string `json:"language"`
	Listeners int    `json:"listeners"`
	Subtitles bool   `json:"subtitles"`
}

type Translator interface {
	GetChannel(language string) chan []float32
	GetSubtitles(language string) (chan string, func())
	PushAudio(chunk []float32)
	CloseAll()
	ListSessions() []SessionInfo
	StopSession(language string, subtitles bool)
	SetEnabled(enabled bool)
	SetOnStateChange(fn func())
}

// WSClient wraps a websocket connection with a mutex for thread-safe writes.
type WSClient struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
	Type string // "admin" or "listener"
}

func (c *WSClient) WriteJSON(v interface{}) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return c.Conn.WriteJSON(v)
}

func (c *WSClient) WriteMessage(messageType int, data []byte) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return c.Conn.WriteMessage(messageType, data)
}

func (c *WSClient) Close() error {
	return c.Conn.Close()
}

type RecordIntent struct {
	isRecording bool
}

func (i RecordIntent) IsRecording() bool { return i.isRecording }
func (i *RecordIntent) SetRecording(b bool) { i.isRecording = b }

type StaticLocations struct {
	storage    string
	cloudDrive string
}

func (l StaticLocations) Storage() string    { return l.storage }
func (l StaticLocations) CloudDrive() string { return l.cloudDrive }

type AppState struct {
	mu sync.RWMutex

	config  configState
	engine  EngineState
	intent  RecordIntent
	static  StaticLocations
	
	// Channels and specialized maps remain here for now
	Clients         sync.Map // map[*WSClient]bool
	AdminClient     *WSClient
	MasterSessionID string
	QuitAudio       chan bool

	RecordChan   chan []float32
	PlaybackChan chan []float32

	StreamChannels sync.Map // map[chan []float32]bool
	BroadcastHub   sync.Map // map[chan StateChange]bool

	Devices []*pa.DeviceInfo
	Translator Translator
}

func NewAppState(storage, cloud string) *AppState {
	return &AppState{
		static: StaticLocations{
			storage:    storage,
			cloudDrive: cloud,
		},
		RecordChan:   make(chan []float32, 100),
		PlaybackChan: make(chan []float32, 100),
	}
}

// Getters

func (s *AppState) Config() InterfaceConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.interfaceCfg
}

func (s *AppState) AI() AIConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.aiCfg
}

func (s *AppState) IsRecording() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.intent.isRecording
}

func (s *AppState) Locations() StaticLocations {
	// Static locations don't change after init, no lock needed if initialized properly
	return s.static
}

func (s *AppState) Engine() *EngineState {
	return &s.engine
}

func (s *AppState) Broadcast(sec Section) {
	change := StateChange{
		Section: sec.String(),
	}

	s.BroadcastHub.Range(func(key, value interface{}) bool {
		ch := key.(chan StateChange)
		select {
		case ch <- change:
		default:
		}
		return true
	})
}
