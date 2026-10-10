package state

import (
	"sync"
	"time"

	pa "github.com/gordonklaus/portaudio"
)

type StateChange struct {
	SessionID string      `json:"sessionId"`
	Section   string      `json:"section"`
	Details   interface{} `json:"details,omitempty"`
}

// RecordIntent represents active recording intent and target output parameters.
type RecordIntent struct {
	isRecording bool
}

func (i RecordIntent) IsRecording() bool    { return i.isRecording }
func (i *RecordIntent) SetRecording(b bool) { i.isRecording = b }

type StaticLocations struct {
	storage    string
	cloudDrive string
}

func (l StaticLocations) Storage() string    { return l.storage }
func (l StaticLocations) CloudDrive() string { return l.cloudDrive }

// AppState manages thread-safe synchronized sections of audio, system, and recording state.
type AppState struct {
	mu sync.RWMutex

	config configState
	engine EngineState
	intent RecordIntent
	static StaticLocations

	// Channels and specialized maps remain here for now
	Clients         sync.Map // map[*WSClient]bool
	AdminClient     *WSClient
	MasterSessionID string
	QuitAudio       chan bool
	DoneAudio       chan struct{}

	RecordChan   chan []float32
	PlaybackChan chan []float32

	StreamChannels sync.Map // map[chan []float32]bool
	BroadcastHub   sync.Map // map[chan StateChange]bool

	RevokedSessions sync.Map // map[string]time.Time

	Devices    []*pa.DeviceInfo
	Translator Translator
}

// NewAppState initializes thread-safe global state for Abel Audio Utils.
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

// RevokeSession records sessionID as revoked at the current timestamp.
func (s *AppState) RevokeSession(sessionID string) {
	if sessionID != "" {
		s.RevokedSessions.Store(sessionID, time.Now())
	}
}

// IsSessionRevoked checks if a sessionID has been revoked on the server.
func (s *AppState) IsSessionRevoked(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	_, revoked := s.RevokedSessions.Load(sessionID)
	return revoked
}

// Getters

func (s *AppState) Config() AudioEngineUIConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.interfaceCfg
}

func (s *AppState) AI() AIConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg := s.config.aiCfg
	cfg.blockedLanguages = cfg.BlockedLanguages()
	return cfg
}

func (s *AppState) IsLanguageBlocked(lang string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.aiCfg.IsBlocked(lang)
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
