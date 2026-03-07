package types

import (
	"math"
	"os"
	"sync"
	"sync/atomic"

	pa "github.com/gordonklaus/portaudio"
	"github.com/gorilla/websocket"
)

type AppState struct {
	Mu          sync.RWMutex
	IsRecording atomic.Bool
	IsRunning   atomic.Bool // Engine status
	DeviceID    atomic.Int32
	ChLeft      atomic.Int32
	ChRight     atomic.Int32
	Boost       atomic.Uint64 // Storing float64 bits

	File         *os.File
	SamplesWrote atomic.Int64

	Clients       sync.Map // map[*WSClient]bool
	AdminClient   atomic.Pointer[WSClient]
	QuitAudio     chan bool

	// Communication channels
	RecordChan   chan []float32
	PlaybackChan chan []float32

	StorageLocation    string
	CloudDriveLocation string

	// Live streaming channels
	StreamChannels sync.Map // map[chan []float32]bool

	// Audio Devices cache
	Devices []*pa.DeviceInfo
}

func (s *AppState) GetBoost() float64 {
	return math.Float64frombits(s.Boost.Load())
}

func (s *AppState) SetBoost(b float64) {
	s.Boost.Store(math.Float64bits(b))
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
