package state

import (
	"os"
	"sync/atomic"
)

type EngineState struct {
	samplesWrote atomic.Int64
	file         *os.File // Guarded by the fact that only one engine thread exists
	isRunning    atomic.Bool
	restarting   atomic.Bool
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

// BeginRestart marks an engine restart as in progress. It returns false if one is already running.
func (e *EngineState) BeginRestart() bool {
	return e.restarting.CompareAndSwap(false, true)
}

func (e *EngineState) EndRestart() {
	e.restarting.Store(false)
}

func (e *EngineState) IsRestarting() bool {
	return e.restarting.Load()
}
