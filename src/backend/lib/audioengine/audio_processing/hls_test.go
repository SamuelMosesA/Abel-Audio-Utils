package audio_processing

import (
	"context"
	"errors"
	"math"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHLSPublisherProducesPlaylistAndMPEGTransportStream(t *testing.T) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}

	publisher := NewHLSPublisherWithRoot(ffmpegPath, t.TempDir())
	t.Cleanup(func() { require.NoError(t, publisher.Close()) })

	const sampleRate = 48000
	const framesPerChunk = 1024
	for chunkIndex := 0; chunkIndex < 120; chunkIndex++ {
		chunk := make([]float32, framesPerChunk*2)
		for frame := 0; frame < framesPerChunk; frame++ {
			sampleIndex := chunkIndex*framesPerChunk + frame
			sample := float32(math.Sin(2 * math.Pi * 440 * float64(sampleIndex) / sampleRate))
			chunk[frame*2] = sample
			chunk[frame*2+1] = sample
		}
		require.NoError(t, publisher.Publish("default", sampleRate, chunk))
	}

	playlist, err := publisher.WaitForPlaylist(context.Background(), "default", 10*time.Second)
	require.NoError(t, err)
	assert.Contains(t, string(playlist), "#EXTM3U")
	assert.Contains(t, string(playlist), "#EXT-X-TARGETDURATION")

	segmentName := regexp.MustCompile(`segment-[0-9]+\.ts`).FindString(string(playlist))
	require.NotEmpty(t, segmentName)
	segment, err := publisher.ReadSegment("default", segmentName)
	require.NoError(t, err)
	require.NotEmpty(t, segment)
	assert.Equal(t, byte(0x47), segment[0], "MPEG-TS packets start with a sync byte")

	ffprobePath, err := exec.LookPath("ffprobe")
	require.NoError(t, err)
	probe := exec.Command(ffprobePath, "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=codec_name", "-of", "default=noprint_wrappers=1:nokey=1", filepath.Join(publisher.Root(), "default", segmentName))
	codec, err := probe.Output()
	require.NoError(t, err)
	assert.Contains(t, string(codec), "aac")
}

func TestHLSPublisherRejectsPathTraversal(t *testing.T) {
	publisher := NewHLSPublisherWithRoot("ffmpeg", t.TempDir())

	_, err := publisher.ReadSegment("default", "../index.m3u8")
	assert.True(t, errors.Is(err, ErrInvalidStreamName))
	assert.ErrorIs(t, publisher.EnsureStream("../default", 48000, nil), ErrInvalidStreamName)
}

func TestFloat32ToPCM16ClampsSamples(t *testing.T) {
	pcm := Float32ToPCM16([]float32{-2, 0, 2})
	assert.Equal(t, []byte{1, 128, 0, 0, 255, 127}, pcm)
}
