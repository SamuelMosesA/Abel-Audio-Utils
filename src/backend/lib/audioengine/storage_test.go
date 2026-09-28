package audioengine

import (
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

func TestWritePlaceholderHeader(t *testing.T) {
	// Create a temp file
	f, err := os.CreateTemp("", "test_header_*.wav")
	assert.NoError(t, err)
	defer os.Remove(f.Name())
	defer f.Close()

	err = WritePlaceholderHeader(f, 2, 44100)
	assert.NoError(t, err)

	// Seek back and check size
	info, err := f.Stat()
	assert.NoError(t, err)
	assert.Equal(t, int64(44), info.Size())

	data := make([]byte, 44)
	f.Seek(0, 0)
	f.Read(data)
	assert.Equal(t, "RIFF", string(data[0:4]))
	assert.Equal(t, uint32(36), binary.LittleEndian.Uint32(data[4:8]))
	assert.Equal(t, "WAVE", string(data[8:12]))
	assert.Equal(t, "fmt ", string(data[12:16]))
	assert.Equal(t, uint16(1), binary.LittleEndian.Uint16(data[20:22]))
	assert.Equal(t, uint16(2), binary.LittleEndian.Uint16(data[22:24]))
	assert.Equal(t, uint32(44100), binary.LittleEndian.Uint32(data[24:28]))
	assert.Equal(t, uint16(16), binary.LittleEndian.Uint16(data[34:36]))
	assert.Equal(t, "data", string(data[36:40]))
	assert.Equal(t, uint32(0), binary.LittleEndian.Uint32(data[40:44]))
}

func TestFinalizeWavHeader(t *testing.T) {
	f, err := os.CreateTemp("", "test_finalize_*.wav")
	assert.NoError(t, err)
	defer os.Remove(f.Name())

	// Write 44 bytes of placeholder + 8 bytes of "data" (2 samples)
	f.Write(make([]byte, 44))
	f.Write([]byte{0x01, 0x00, 0x01, 0x00, 0x02, 0x00, 0x02, 0x00}) // 2 stereo samples (int16)

	err = FinalizeWavHeader(f, 2, 2, 48000)
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	f2, err := os.Open(f.Name())
	assert.NoError(t, err)
	defer f2.Close()

	header := make([]byte, 44)
	f2.Read(header)

	assert.Equal(t, "RIFF", string(header[0:4]))
	// File size - 8 = 44 + 8 - 8 = 44
	assert.Equal(t, uint32(44), binary.LittleEndian.Uint32(header[4:8]))
	assert.Equal(t, "WAVE", string(header[8:12]))
	assert.Equal(t, "fmt ", string(header[12:16]))
	assert.Equal(t, uint32(16), binary.LittleEndian.Uint32(header[16:20]))
	assert.Equal(t, uint16(1), binary.LittleEndian.Uint16(header[20:22]))
	assert.Equal(t, uint16(2), binary.LittleEndian.Uint16(header[22:24]))
	assert.Equal(t, uint32(48000), binary.LittleEndian.Uint32(header[24:28]))
	assert.Equal(t, "data", string(header[36:40]))
	assert.Equal(t, uint32(8), binary.LittleEndian.Uint32(header[40:44]))
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

	err = WritePlaceholderHeader(f, 2, 44100)
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
	err = FinalizeWavHeader(file, 2, samplesWrote, 44100)
	assert.NoError(t, err)
	err = file.Close()
	assert.NoError(t, err)

	// Further writes to recordChan must not panic or error
	recordChan <- []float32{0.5, -0.5}
	time.Sleep(10 * time.Millisecond)
	close(recordChan)
}

func TestGenerateWavHeaderByteLayout(t *testing.T) {
	// Test stereo 48000Hz with 1000 bytes data
	h := GenerateWavHeader(2, 1000, 48000)
	assert.Equal(t, "RIFF", string(h[0:4]))
	assert.Equal(t, uint32(1036), binary.LittleEndian.Uint32(h[4:8]))
	assert.Equal(t, "WAVE", string(h[8:12]))
	assert.Equal(t, "fmt ", string(h[12:16]))
	assert.Equal(t, uint32(16), binary.LittleEndian.Uint32(h[16:20]))
	assert.Equal(t, uint16(1), binary.LittleEndian.Uint16(h[20:22]))
	assert.Equal(t, uint16(2), binary.LittleEndian.Uint16(h[22:24]))
	assert.Equal(t, uint32(48000), binary.LittleEndian.Uint32(h[24:28]))
	assert.Equal(t, uint32(48000*2*2), binary.LittleEndian.Uint32(h[28:32])) // ByteRate
	assert.Equal(t, uint16(4), binary.LittleEndian.Uint16(h[32:34]))          // BlockAlign
	assert.Equal(t, uint16(16), binary.LittleEndian.Uint16(h[34:36]))         // BitsPerSample
	assert.Equal(t, "data", string(h[36:40]))
	assert.Equal(t, uint32(1000), binary.LittleEndian.Uint32(h[40:44]))

	// Test mono fallback (ch=1, sampleRate<=0 -> fallback to 44100)
	hMono := GenerateWavHeader(1, 500, 0)
	assert.Equal(t, uint16(1), binary.LittleEndian.Uint16(hMono[22:24]))
	assert.Equal(t, uint32(44100), binary.LittleEndian.Uint32(hMono[24:28]))
	assert.Equal(t, uint32(44100*1*2), binary.LittleEndian.Uint32(hMono[28:32]))
	assert.Equal(t, uint16(2), binary.LittleEndian.Uint16(hMono[32:34]))
	assert.Equal(t, uint32(500), binary.LittleEndian.Uint32(hMono[40:44]))
}

func TestFinalizeWavHeaderEdgeCases(t *testing.T) {
	// Nil file
	err := FinalizeWavHeader(nil, 2, 100, 44100)
	assert.NoError(t, err)

	f, err := os.CreateTemp("", "test_edge_*.wav")
	assert.NoError(t, err)
	defer os.Remove(f.Name())
	defer f.Close()

	// Negative samples
	err = FinalizeWavHeader(f, 2, -10, 44100)
	assert.NoError(t, err)

	data := make([]byte, 44)
	f.Seek(0, 0)
	f.Read(data)
	assert.Equal(t, uint32(0), binary.LittleEndian.Uint32(data[40:44]))

	// Huge sample count (clamped to max uint32 - 36)
	err = FinalizeWavHeader(f, 2, 5000000000, 44100)
	assert.NoError(t, err)

	f.Seek(0, 0)
	f.Read(data)
	assert.Equal(t, uint32(0xFFFFFFFF-36), binary.LittleEndian.Uint32(data[40:44]))
}
