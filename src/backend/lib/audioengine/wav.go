package audioengine

import (
	"encoding/binary"
	"io"
	"os"
)

const (
	wavHeaderSize        = 44
	wavBitsPerSample     = 16
	wavBytesPerSample    = wavBitsPerSample / 8
	defaultWavSampleRate = 44100
)

// WritePlaceholderHeader writes a valid WAV header with an empty data chunk.
// The final file and data sizes are unknown until recording finishes, so
// FinalizeWavHeader seeks back and updates those fields after capture stops.
// WAV header structure is always 44 bytes for standard PCM audio:
//   - RIFF header (12 bytes)
//   - fmt chunk (24 bytes)
//   - data chunk header (8 bytes)
func WritePlaceholderHeader(f *os.File, ch uint16, sampleRate int) error {
	if f == nil {
		return nil
	}
	return writeWavHeader(f, ch, 0, sampleRate)
}

func FinalizeWavHeader(f *os.File, ch uint16, s int64, sampleRate int) error {
	if f == nil {
		return nil
	}
	if s < 0 {
		s = 0
	}
	if ch == 0 {
		ch = 2
	}
	totalBytes := s * int64(ch) * wavBytesPerSample
	const maxWavDataSize = uint32(0xFFFFFFFF - 36)
	var dataSize uint32
	if totalBytes > int64(maxWavDataSize) {
		dataSize = maxWavDataSize
	} else {
		dataSize = uint32(totalBytes)
	}
	return writeWavHeader(f, ch, dataSize, sampleRate)
}

// GenerateWavHeader constructs the 44-byte standard PCM WAV header in memory.
func GenerateWavHeader(ch uint16, dataSize uint32, sampleRate int) [wavHeaderSize]byte {
	if ch == 0 {
		ch = 2
	}
	if sampleRate <= 0 {
		sampleRate = defaultWavSampleRate
	}

	byteRate := uint32(sampleRate) * uint32(ch) * wavBytesPerSample
	blockAlign := uint16(ch * wavBytesPerSample)

	var header [wavHeaderSize]byte

	// 0..3 "RIFF"
	copy(header[0:4], "RIFF")
	// 4..7 FileSize - 8 = 36 + dataSize
	binary.LittleEndian.PutUint32(header[4:8], 36+dataSize)
	// 8..11 "WAVE"
	copy(header[8:12], "WAVE")
	// 12..15 "fmt "
	copy(header[12:16], "fmt ")
	// 16..19 Subchunk1Size = 16
	binary.LittleEndian.PutUint32(header[16:20], 16)
	// 20..21 AudioFormat = 1 (PCM)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	// 22..23 NumChannels
	binary.LittleEndian.PutUint16(header[22:24], ch)
	// 24..27 SampleRate
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	// 28..31 ByteRate
	binary.LittleEndian.PutUint32(header[28:32], byteRate)
	// 32..33 BlockAlign
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)
	// 34..35 BitsPerSample = 16
	binary.LittleEndian.PutUint16(header[34:36], uint16(wavBitsPerSample))
	// 36..39 "data"
	copy(header[36:40], "data")
	// 40..43 DataSize
	binary.LittleEndian.PutUint32(header[40:44], dataSize)

	return header
}

func writeWavHeader(f *os.File, ch uint16, dataSize uint32, sampleRate int) error {
	header := GenerateWavHeader(ch, dataSize, sampleRate)

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err := f.Write(header[:])
	return err
}
