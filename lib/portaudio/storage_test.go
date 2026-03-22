package portaudio

import (
	"behringerRecorder/lib/types"
	"bytes"
	"encoding/binary"
	"os"
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

	WritePlaceholderHeader(f)

	// Seek back and check size
	info, err := f.Stat()
	assert.NoError(t, err)
	assert.Equal(t, int64(44), info.Size())

	data := make([]byte, 44)
	f.Seek(0, 0)
	f.Read(data)
	for _, b := range data {
		assert.Equal(t, byte(0), b)
	}
}

func TestFinalizeWavHeader(t *testing.T) {
	f, err := os.CreateTemp("", "test_finalize_*.wav")
	assert.NoError(t, err)
	defer os.Remove(f.Name())

	// Write 44 bytes of placeholder + 8 bytes of "data" (2 samples)
	f.Write(make([]byte, 44))
	f.Write([]byte{0x01, 0x00, 0x01, 0x00, 0x02, 0x00, 0x02, 0x00}) // 2 stereo samples (int16)

	FinalizeWavHeader(f, 2, 2, 48000)
	// FinalizeWavHeader closes the file

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
	state := &types.AppState{
		IsRecording: true,
	}
	f, err := os.CreateTemp("", "test_worker_*.raw")
	assert.NoError(t, err)
	defer os.Remove(f.Name())
	state.File = f

	recordChan := make(chan []float32, 1)
	StartStorageWorker(state, recordChan)

	chunk := []float32{0.5, -0.5}
	recordChan <- chunk

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	state.Mu.RLock()
	samples := state.SamplesWrote
	state.Mu.RUnlock()
	assert.Equal(t, int64(1), samples)

	// Stop recording and check
	state.Mu.Lock()
	state.IsRecording = false
	state.Mu.Unlock()

	recordChan <- chunk
	time.Sleep(100 * time.Millisecond)
	
	state.Mu.RLock()
	assert.Equal(t, int64(1), state.SamplesWrote) // Should not have increased
	state.Mu.RUnlock()

	close(recordChan)
}

