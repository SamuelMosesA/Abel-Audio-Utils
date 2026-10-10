package state

import (
	"os"
	"sync/atomic"
)

// EngineState encapsulates the audio engine runtime file and sample counters
// using lock-free atomic primitives for high-performance recording loops.
type EngineState struct {
	file         atomic.Pointer[os.File]
	samplesWrote atomic.Int64
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
	return e.file.Load()
}

func (e *EngineState) SetFile(f *os.File) {
	e.file.Store(f)
}

// WriteWithFile executes a write function against the active recording file in a lock-free manner.
// If no file is active, it safely no-ops and returns (0, nil).
// If err is nil and n > 0, samplesWrote is atomically incremented by n.
func (e *EngineState) WriteWithFile(fn func(f *os.File) (int, error)) (int, error) {
	f := e.file.Load()
	if f == nil {
		return 0, nil
	}
	n, err := fn(f)
	if err == nil && n > 0 {
		e.samplesWrote.Add(int64(n))
	}
	return n, err
}

// TakeFile atomically detaches the active file pointer and returns the file and final samples count.
// After TakeFile returns, subsequent chunk writes safely see a nil file without acquiring locks.
func (e *EngineState) TakeFile() (*os.File, int64) {
	f := e.file.Swap(nil)
	samples := e.samplesWrote.Load()
	return f, samples
}

func (e *EngineState) IsRunning() bool {
	return e.isRunning.Load()
}

func (e *EngineState) SetRunning(b bool) {
	e.isRunning.Store(b)
}
