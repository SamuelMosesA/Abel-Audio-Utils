package types

import (
	"os"
	"sync"

	pa "github.com/gordonklaus/portaudio"
	"github.com/gorilla/websocket"
)

type AppState struct {
	Mu          sync.RWMutex
	IsRecording bool
	IsRunning   bool // Engine status
	DeviceID    int  // Currently connected device ID
	ChLeft      int
	ChRight     int
	Boost       float64

	File         *os.File
	SamplesWrote int64

	Clients       map[*WSClient]bool
	PrimaryClient *WSClient // Client with primary control
	QuitAudio     chan bool

	// Communication channels
	RecordChan   chan []float32
	PlaybackChan chan []float32

	StorageLocation    string
	CloudDriveLocation string

	// Audio Devices cache
	Devices []*pa.DeviceInfo
}

// WSClient wraps a websocket connection with a mutex for thread-safe writes.
type WSClient struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
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
