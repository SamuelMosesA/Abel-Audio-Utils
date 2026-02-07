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

	Clients       map[*websocket.Conn]bool
	PrimaryClient *websocket.Conn // Client with primary control
	QuitAudio     chan bool

	// Communication channels
	RecordChan   chan []float32
	PlaybackChan chan []float32

	// Audio Devices cache
	Devices []*pa.DeviceInfo
}

type AudioDevice struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	In   int    `json:"inputs"`
}
