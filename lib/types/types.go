package types

import (
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

type AppState struct {
	Mu          sync.RWMutex
	IsRecording bool
	IsRunning   bool // Engine status
	ChLeft      int
	ChRight     int
	Boost       float64

	FileL        *os.File
	FileR        *os.File
	FileStereo   *os.File
	SamplesWrote int64

	Clients   map[*websocket.Conn]bool
	QuitAudio chan bool

	// Communication channels
	RecordChan   chan []float32
	PlaybackChan chan []float32
}

type AudioDevice struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	In   int    `json:"inputs"`
}
