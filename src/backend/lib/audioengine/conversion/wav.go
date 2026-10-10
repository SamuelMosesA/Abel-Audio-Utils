package conversion

import (
	"encoding/binary"
	"fmt"
	"io"
)

// GenerateWavHeader constructs the 44-byte standard PCM WAV header in memory.
func GenerateWavHeader(ch uint16, dataSize uint32, sampleRate int) [WavHeaderSize]byte {
	if ch == 0 {
		ch = 2
	}
	if sampleRate <= 0 {
		sampleRate = DefaultWavSampleRate
	}

	byteRate := uint32(sampleRate) * uint32(ch) * WavBytesPerSample
	blockAlign := uint16(ch * WavBytesPerSample)

	var header [WavHeaderSize]byte

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
	binary.LittleEndian.PutUint16(header[34:36], uint16(WavBitsPerSample))
	// 36..39 "data"
	copy(header[36:40], "data")
	// 40..43 DataSize
	binary.LittleEndian.PutUint32(header[40:44], dataSize)

	return header
}

// WritePlaceholderWavHeader writes an initial 44-byte WAV header with 0 data size.
func WritePlaceholderWavHeader(w io.Writer, ch uint16, sampleRate int) error {
	return WriteWavHeader(w, ch, 0, sampleRate)
}

// FinalizeWavHeader seeks back to the beginning and rewrites the WAV header with total bytes recorded.
func FinalizeWavHeader(ws io.WriteSeeker, ch uint16, dataBytes int64, sampleRate int) error {
	if _, err := ws.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek beginning of WAV file: %w", err)
	}
	if dataBytes < 0 {
		dataBytes = 0
	}
	const maxWavDataSize = uint32(0xFFFFFFFF - 36)
	var dataSize uint32
	if dataBytes > int64(maxWavDataSize) {
		dataSize = maxWavDataSize
	} else {
		dataSize = uint32(dataBytes)
	}
	return WriteWavHeader(ws, ch, dataSize, sampleRate)
}

// WriteWavHeader writes a standard 44-byte RIFF/WAVE header to the given writer.
func WriteWavHeader(w io.Writer, ch uint16, dataSize uint32, sampleRate int) error {
	header := GenerateWavHeader(ch, dataSize, sampleRate)
	_, err := w.Write(header[:])
	return err
}
