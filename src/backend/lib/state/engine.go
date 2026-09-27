package state

import (
	"os"
	"sync/atomic"
)

type EngineState struct {
	samplesWrote atomic.Int64
	file         *os.File // Guarded by the fact that only one engine thread exists
	isRunning    atomic.Bool
}

func (e *EngineState) SamplesWrote() int64 {
	return e.samplesWrote.Load()
}

func (e *EngineState) AddSamples(n int64) {
	e.samplesWrote.Add(n)
}

func (e *EngineState) ResetSamples() {
	e.samplesWrote.Store(0)
}

func (e *EngineState) File() *os.File {
	return e.file
}

func (e *EngineState) SetFile(f *os.File) {
	e.file = f
}

func (e *EngineState) IsRunning() bool {
	return e.isRunning.Load()
}

func (e *EngineState) SetRunning(b bool) {
	e.isRunning.Store(b)
}
