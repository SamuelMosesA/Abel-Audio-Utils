package state

import (
	"os"
	"sync"
	"sync/atomic"
)

type EngineState struct {
	fileMu       sync.Mutex
	file         *os.File
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
	e.fileMu.Lock()
	defer e.fileMu.Unlock()
	return e.file
}

func (e *EngineState) SetFile(f *os.File) {
	e.fileMu.Lock()
	defer e.fileMu.Unlock()
	e.file = f
}

// WriteWithFile executes a write function against the active recording file under a mutex lock.
// It increments samplesWrote by n if err is nil. If no file is active, it returns (0, nil).
func (e *EngineState) WriteWithFile(fn func(f *os.File) (int, error)) (int, error) {
	e.fileMu.Lock()
	defer e.fileMu.Unlock()
	if e.file == nil {
		return 0, nil
	}
	n, err := fn(e.file)
	if err == nil && n > 0 {
		e.samplesWrote.Add(int64(n))
	}
	return n, err
}

// TakeFile safely detaches the active file pointer and returns the file and final samples count.
// After TakeFile returns, subsequent WriteWithFile calls will safely see a nil file.
func (e *EngineState) TakeFile() (*os.File, int64) {
	e.fileMu.Lock()
	defer e.fileMu.Unlock()
	f := e.file
	e.file = nil
	samples := e.samplesWrote.Load()
	return f, samples
}

func (e *EngineState) IsRunning() bool {
	return e.isRunning.Load()
}

func (e *EngineState) SetRunning(b bool) {
	e.isRunning.Store(b)
}
