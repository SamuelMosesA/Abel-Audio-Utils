package audioengine

import (
	"abel/src/backend/lib/audioengine/audio_processing"
	"abel/src/backend/lib/state"
	"bytes"
	"encoding/binary"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWriteAudio(t *testing.T) {
	buf := new(bytes.Buffer)
	chunk := []float32{0.5, -0.5, 1.0, -1.0, 0.0, 0.0}

	n, err := WriteAudio(buf, chunk)
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	// Verify the conversion
	// float32 * 32767
	// 0.5 * 32767 = 16383.5 -> 16383
	// -0.5 * 32767 = -16383.5 -> -16383
	// 1.0 * 32767 = 32767
	// -1.0 * 32767 = -32767

	expected := []int16{16383, -16383, 32767, -32767, 0, 0}
	actual := make([]int16, 6)
	err = binary.Read(buf, binary.LittleEndian, &actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestStartStorageWorker(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) {
		s.SetRecording(true)
	})
	f, err := os.CreateTemp("", "test_worker_*.raw")
	assert.NoError(t, err)
	defer os.Remove(f.Name())
	appState.Engine().SetFile(f)

	recordChan := make(chan []float32, 1)
	StartStorageWorker(appState, recordChan)

	chunk := []float32{0.5, -0.5}
	recordChan <- chunk

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	samples := appState.Engine().SamplesWrote()
	assert.Equal(t, int64(1), samples)

	// Stop recording and check
	state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) {
		s.SetRecording(false)
	})

	recordChan <- chunk
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int64(1), appState.Engine().SamplesWrote()) // Should not have increased

	close(recordChan)
}

func TestWriteAudioEdgeCasesAndClamping(t *testing.T) {
	buf := new(bytes.Buffer)

	// Empty slice
	n, err := WriteAudio(buf, []float32{})
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, 0, buf.Len())

	// Odd length slice (last sample ignored in stereo pairs)
	n, err = WriteAudio(buf, []float32{0.0, 0.0, 0.5})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, 4, buf.Len())

	// Values requiring clamping (> 1.0, < -1.0)
	buf.Reset()
	n, err = WriteAudio(buf, []float32{2.5, -3.0})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	actual := make([]int16, 2)
	err = binary.Read(buf, binary.LittleEndian, &actual)
	assert.NoError(t, err)
	assert.Equal(t, int16(32767), actual[0])
	assert.Equal(t, int16(-32767), actual[1])
}

func TestStartStorageWorkerConcurrentStop(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) {
		s.SetRecording(true)
	})

	f, err := os.CreateTemp("", "test_concurrent_*.wav")
	assert.NoError(t, err)
	defer os.Remove(f.Name())

	err = audio_processing.WritePlaceholderWavHeader(f, 2, 44100)
	assert.NoError(t, err)
	appState.Engine().SetFile(f)

	recordChan := make(chan []float32, 200)
	StartStorageWorker(appState, recordChan)

	stopSignal := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			select {
			case <-stopSignal:
				return
			case recordChan <- []float32{0.1, -0.1}:
			}
			time.Sleep(100 * time.Microsecond)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	// Concurrently take file and finalize as handlers_recordings does
	state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) {
		s.SetRecording(false)
	})
	file, samplesWrote := appState.Engine().TakeFile()
	close(stopSignal)
	wg.Wait()

	assert.NotNil(t, file)
	err = audio_processing.FinalizeWavHeaderWithSamples(file, 2, samplesWrote, 44100)
	assert.NoError(t, err)
	err = file.Close()
	assert.NoError(t, err)

	// Further writes to recordChan must not panic or error
	recordChan <- []float32{0.5, -0.5}
	time.Sleep(10 * time.Millisecond)
	close(recordChan)
}
