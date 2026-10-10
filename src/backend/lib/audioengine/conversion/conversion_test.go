package conversion_test

import (
	"bytes"
	"testing"

	"abel/src/backend/lib/audioengine/conversion"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFloat32ToPCM16_Clamping(t *testing.T) {
	tests := []struct {
		name     string
		input    []float32
		expected []byte
	}{
		{
			name:     "clamps values and produces little endian bytes",
			input:    []float32{-2.0, 0.0, 2.0},
			expected: []byte{1, 128, 0, 0, 255, 127}, // -32767 -> 0x8001, 0 -> 0x0000, 32767 -> 0x7fff
		},
		{
			name:     "empty slice returns empty bytes",
			input:    []float32{},
			expected: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := conversion.Float32ToPCM16(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestDownsampleStereoToMonoPCM24k(t *testing.T) {
	// Stereo 48kHz input to 24kHz mono
	stereo48k := []float32{0.5, 0.5, 0.5, 0.5} // 2 stereo frames at 48k -> 1 mono frame at 24k
	downsampled := conversion.DownsampleStereoToMonoPCM24k(stereo48k, 48000)
	require.Len(t, downsampled, 2) // 1 frame * 2 bytes/sample

	// Same rate stereo 24k to mono 24k
	stereo24k := []float32{0.0, 0.0}
	downsampled24k := conversion.DownsampleStereoToMonoPCM24k(stereo24k, 24000)
	assert.Equal(t, []byte{0, 0}, downsampled24k)
}

func TestWavHeaderGeneration(t *testing.T) {
	var buf bytes.Buffer
	err := conversion.WritePlaceholderWavHeader(&buf, 2, 48000)
	require.NoError(t, err)
	assert.Equal(t, 44, buf.Len())
	assert.Equal(t, "RIFF", string(buf.Bytes()[0:4]))
	assert.Equal(t, "WAVE", string(buf.Bytes()[8:12]))
	assert.Equal(t, "fmt ", string(buf.Bytes()[12:16]))
	assert.Equal(t, "data", string(buf.Bytes()[36:40]))
}

func TestExtractStereoChunk(t *testing.T) {
	t.Run("normal extraction and gain scaling", func(t *testing.T) {
		in := []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
		out := conversion.ExtractStereoChunk(in, 2, 4, 1, 2, 2.0)
		require.Len(t, out, 4)
		assert.InDelta(t, float32(0.4), out[0], 1e-6)
		assert.InDelta(t, float32(0.6), out[1], 1e-6)
		assert.InDelta(t, float32(1.0), out[2], 1e-6)
		assert.InDelta(t, float32(1.0), out[3], 1e-6)
	})

	t.Run("clamping negative values", func(t *testing.T) {
		in := []float32{-0.8, -0.9}
		out := conversion.ExtractStereoChunk(in, 1, 2, 0, 1, 2.0)
		require.Len(t, out, 2)
		assert.Equal(t, float32(-1.0), out[0])
		assert.Equal(t, float32(-1.0), out[1])
	})

	t.Run("non-positive boost defaults to unity gain", func(t *testing.T) {
		in := []float32{0.3, 0.4}
		outZero := conversion.ExtractStereoChunk(in, 1, 2, 0, 1, 0.0)
		assert.InDelta(t, float32(0.3), outZero[0], 1e-6)
		assert.InDelta(t, float32(0.4), outZero[1], 1e-6)

		outNeg := conversion.ExtractStereoChunk(in, 1, 2, 0, 1, -2.5)
		assert.InDelta(t, float32(0.3), outNeg[0], 1e-6)
		assert.InDelta(t, float32(0.4), outNeg[1], 1e-6)
	})

	t.Run("out of bounds channel indices safely yield zero", func(t *testing.T) {
		in := []float32{0.5}
		out := conversion.ExtractStereoChunk(in, 1, 2, 5, -1, 1.0)
		require.Len(t, out, 2)
		assert.Equal(t, float32(0.0), out[0])
		assert.Equal(t, float32(0.0), out[1])
	})
}
