package audio_processing

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertStereoFloat32ToPCM16LE(t *testing.T) {
	t.Run("empty or insufficient samples returns nil", func(t *testing.T) {
		assert.Nil(t, ConvertStereoFloat32ToPCM16LE(nil))
		assert.Nil(t, ConvertStereoFloat32ToPCM16LE([]float32{}))
		assert.Nil(t, ConvertStereoFloat32ToPCM16LE([]float32{0.5})) // single sample, not a pair
	})

	t.Run("clamps values beyond [-1.0, 1.0] and encodes little-endian", func(t *testing.T) {
		input := []float32{
			1.5, -2.0, // clamped to 1.0, -1.0
			0.0, 0.5, // 0, 16383
			-0.5, 1.0, // -16383, 32767
		}
		buf := ConvertStereoFloat32ToPCM16LE(input)
		require.Len(t, buf, 3*4) // 3 pairs * 4 bytes = 12 bytes

		// Sample 0: L=1.0 -> 32767
		s0L := int16(binary.LittleEndian.Uint16(buf[0:2]))
		assert.Equal(t, int16(32767), s0L)

		// Sample 0: R=-1.0 -> -32767
		s0R := int16(binary.LittleEndian.Uint16(buf[2:4]))
		assert.Equal(t, int16(-32767), s0R)

		// Sample 1: L=0.0 -> 0
		s1L := int16(binary.LittleEndian.Uint16(buf[4:6]))
		assert.Equal(t, int16(0), s1L)

		// Sample 1: R=0.5 -> 16383
		s1R := int16(binary.LittleEndian.Uint16(buf[6:8]))
		assert.Equal(t, int16(16383), s1R)
	})
}

func TestFloat32ToPCM16(t *testing.T) {
	chunk := []float32{1.5, -1.5, 0.0}
	pcm := Float32ToPCM16(chunk)
	require.Len(t, pcm, 6)

	v0 := int16(binary.LittleEndian.Uint16(pcm[0:2]))
	v1 := int16(binary.LittleEndian.Uint16(pcm[2:4]))
	v2 := int16(binary.LittleEndian.Uint16(pcm[4:6]))

	assert.Equal(t, int16(32767), v0)
	assert.Equal(t, int16(-32767), v1)
	assert.Equal(t, int16(0), v2)
}

func TestExtractStereoChunk(t *testing.T) {
	in := []float32{
		0.1, 0.2, 0.3, 0.4, // Frame 0: 4 channels
		0.5, 0.6, 0.7, 0.8, // Frame 1: 4 channels
	}
	out := ExtractStereoChunk(in, 2, 4, 1, 3, 2.0)
	require.Len(t, out, 4)

	// chL=1, boost=2.0 -> 0.2 * 2.0 = 0.4
	// chR=3, boost=2.0 -> 0.4 * 2.0 = 0.8
	assert.InDelta(t, 0.4, out[0], 1e-4)
	assert.InDelta(t, 0.8, out[1], 1e-4)
	// Frame 1
	// chL=1 -> 0.6 * 2.0 = 1.2 -> clamped to 1.0
	// chR=3 -> 0.8 * 2.0 = 1.6 -> clamped to 1.0
	assert.InDelta(t, 1.0, out[2], 1e-4)
	assert.InDelta(t, 1.0, out[3], 1e-4)
}

func TestWavHeader(t *testing.T) {
	header := GenerateWavHeader(2, 1000, 48000)
	assert.Equal(t, "RIFF", string(header[0:4]))
	assert.Equal(t, "WAVE", string(header[8:12]))
	assert.Equal(t, "fmt ", string(header[12:16]))
	assert.Equal(t, "data", string(header[36:40]))

	// Channels: 2
	assert.Equal(t, uint16(2), binary.LittleEndian.Uint16(header[22:24]))
	// Sample rate: 48000
	assert.Equal(t, uint32(48000), binary.LittleEndian.Uint32(header[24:28]))
	// Data size: 1000
	assert.Equal(t, uint32(1000), binary.LittleEndian.Uint32(header[40:44]))

	var buf bytes.Buffer
	err := WritePlaceholderWavHeader(&buf, 2, 48000)
	require.NoError(t, err)
	assert.Equal(t, WavHeaderSize, buf.Len())

	// Test seek and finalize on file with byte count
	tmpFile := filepath.Join(t.TempDir(), "test.wav")
	f, err := os.Create(tmpFile)
	require.NoError(t, err)
	defer f.Close()

	err = WritePlaceholderWavHeader(f, 2, 48000)
	require.NoError(t, err)
	_, err = f.Write([]byte("12345678"))
	require.NoError(t, err)

	err = FinalizeWavHeader(f, 2, 8, 48000)
	require.NoError(t, err)

	f.Seek(0, io.SeekStart)
	var finalized [WavHeaderSize]byte
	_, err = io.ReadFull(f, finalized[:])
	require.NoError(t, err)
	assert.Equal(t, uint32(8), binary.LittleEndian.Uint32(finalized[40:44]))

	// Test seek and finalize with sample count
	err = FinalizeWavHeaderWithSamples(f, 2, 2, 48000) // 2 samples * 2 ch * 2 bytes = 8 bytes
	require.NoError(t, err)

	f.Seek(0, io.SeekStart)
	_, err = io.ReadFull(f, finalized[:])
	require.NoError(t, err)
	assert.Equal(t, uint32(8), binary.LittleEndian.Uint32(finalized[40:44]))
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
	assert.Equal(t, uint16(4), binary.LittleEndian.Uint16(h[32:34]))         // BlockAlign
	assert.Equal(t, uint16(16), binary.LittleEndian.Uint16(h[34:36]))        // BitsPerSample
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

	// Negative sample bytes clamps to 0
	err = FinalizeWavHeader(f, 2, -10, 44100)
	assert.NoError(t, err)
	header := make([]byte, 44)
	f.Seek(0, io.SeekStart)
	f.Read(header)
	assert.Equal(t, uint32(0), binary.LittleEndian.Uint32(header[40:44]))
	assert.Equal(t, uint32(36), binary.LittleEndian.Uint32(header[4:8]))

	// > 4GB clamps to max RIFF size (0xFFFFFFFF - 36)
	err = FinalizeWavHeader(f, 2, 5000000000, 44100)
	assert.NoError(t, err)
	f.Seek(0, io.SeekStart)
	f.Read(header)
	maxData := uint32(0xFFFFFFFF - 36)
	assert.Equal(t, maxData, binary.LittleEndian.Uint32(header[40:44]))
	assert.Equal(t, uint32(0xFFFFFFFF), binary.LittleEndian.Uint32(header[4:8]))
}

func TestDownsampleStereoToMonoPCM24k(t *testing.T) {
	// Simple stereo signal at 48000 Hz: 2 frames = 4 float32 samples
	input := []float32{0.5, 0.5, -0.5, -0.5}
	downsampled := DownsampleStereoToMonoPCM24k(input, 48000)
	require.NotEmpty(t, downsampled)
	assert.Equal(t, 2, len(downsampled)) // 48k -> 24k reduces 2 frames to 1 frame (1 mono sample = 2 bytes)

	// Pass through when already 24k
	down24 := DownsampleStereoToMonoPCM24k([]float32{0.2, 0.4}, 24000)
	require.Len(t, down24, 2)
}

func TestDecodeAudioDelta(t *testing.T) {
	// Encode 1 sample of 16-bit PCM (e.g. 16384) in base64
	pcm := make([]byte, 2)
	binary.LittleEndian.PutUint16(pcm, uint16(16384))
	encoded := base64.StdEncoding.EncodeToString(pcm)

	floats, err := DecodeAudioDelta(encoded, 48000)
	require.NoError(t, err)
	require.NotEmpty(t, floats)
	// Upsampled to 48000 stereo has 4 samples (2 frames of L+R)
	assert.True(t, len(floats) >= 2)
}
