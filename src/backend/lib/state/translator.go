package state

type SessionInfo struct {
	Language  string `json:"language"`
	Listeners int    `json:"listeners"`
	Subtitles bool   `json:"subtitles"`
}

type Translator interface {
	GetChannel(language string) chan []float32
	GetSubtitles(language string) (chan string, func())
	OnNewAudioChunk(chunk []float32)
	CloseAll()
	ListSessions() []SessionInfo
	StopSession(language string, subtitles bool)
	SetEnabled(enabled bool)
	SetOnStateChange(fn func())
	GetListenerCount(language string) int
}
