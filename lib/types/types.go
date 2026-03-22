package types

import (
	"os"
	"sync"

	pa "github.com/gordonklaus/portaudio"
	"github.com/gorilla/websocket"
)

type StateChange struct {
	SessionID string      `json:"sessionId"`
	Section   string      `json:"section"`
	Details   interface{} `json:"details,omitempty"`
}

type AppState struct {
	Mu sync.RWMutex

	IsRecording bool
	IsRunning   bool // Engine status
	DeviceID    int32
	ChLeft      int32
	ChRight     int32
	Boost       float64
	GeminiEnabled bool

	File         *os.File
	SamplesWrote int64

	Clients       sync.Map // map[*WSClient]bool
	AdminClient   *WSClient
	MasterSessionID string
	QuitAudio     chan bool

	// Communication channels
	RecordChan   chan []float32
	PlaybackChan chan []float32

	StorageLocation    string
	CloudDriveLocation string

	// Live streaming channels
	StreamChannels sync.Map // map[chan []float32]bool

	// Change Log (SSE)
	BroadcastHub sync.Map // map[chan StateChange]bool

	// Audio Devices cache
	Devices []*pa.DeviceInfo

	// Translation
	Translator Translator
}

// UpdateState locks the state, executes the update, and broadcasts the change.
func (s *AppState) UpdateState(sessionID string, section string, updateFn func()) {
	s.Mu.Lock()
	updateFn()
	s.Mu.Unlock()

	change := StateChange{
		SessionID: sessionID,
		Section:   section,
	}

	s.BroadcastHub.Range(func(key, value interface{}) bool {
		ch := key.(chan StateChange)
		select {
		case ch <- change:
		default:
			// Buffer full or client slow, skip or handle as needed
		}
		return true
	})
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
}

func (s *AppState) GetBoost() float64 {
	return s.Boost
}

func (s *AppState) SetBoost(b float64) {
	s.Boost = b
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

type AudioDevice struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	In   int    `json:"inputs"`
}
