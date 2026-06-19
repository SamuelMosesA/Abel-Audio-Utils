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
	dataSize := uint32(s * int64(ch) * wavBytesPerSample)
	return writeWavHeader(f, ch, dataSize, sampleRate)
}

func writeWavHeader(f *os.File, ch uint16, dataSize uint32, sampleRate int) error {
	if ch == 0 {
		ch = 2
	}
	if sampleRate <= 0 {
		sampleRate = defaultWavSampleRate
	}

	byteRate := uint32(sampleRate) * uint32(ch) * wavBytesPerSample
	blockAlign := uint16(ch * wavBytesPerSample)

	// Seek to start and write the complete WAV header
	// WAV File Format (Little Endian):
	//   Offset  Size  Field          Description
	//   ------  ----  -----          -----------
	//   0       4     "RIFF"         Chunk ID (marks this as RIFF file)
	//   4       4     FileSize-8     File size minus 8 bytes (for RIFF header itself)
	//   8       4     "WAVE"         Format identifier (always "WAVE" for audio)
	//   12      4     "fmt "         Subchunk1 ID (format chunk, note the space)
	//   16      4     16             Subchunk1 Size (16 bytes for PCM)
	//   20      2     1              Audio Format (1 = PCM, others = compressed)
	//   22      2     Channels       Number of audio channels (1=mono, 2=stereo)
	//   24      4     SampleRate     Sample rate in Hz (e.g., 48000, 44100)
	//   28      4     ByteRate       SampleRate * Channels * BytesPerSample
	//   32      2     BlockAlign     Channels * BytesPerSample (frame size)
	//   34      2     BitsPerSample  Bits per sample (16 for int16)
	//   36      4     "data"         Data chunk ID (marks audio data section)
	//   40      4     DataSize       Number of bytes of audio data
	//   44      ...   Audio Data     Raw PCM samples follow
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := f.Write([]byte{'R', 'I', 'F', 'F'}); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(36+dataSize)); err != nil {
		return err
	}
	if _, err := f.Write([]byte{'W', 'A', 'V', 'E'}); err != nil {
		return err
	}
	if _, err := f.Write([]byte{'f', 'm', 't', ' '}); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, ch); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(wavBitsPerSample)); err != nil {
		return err
	}
	if _, err := f.Write([]byte{'d', 'a', 't', 'a'}); err != nil {
		return err
	}
	return binary.Write(f, binary.LittleEndian, dataSize)
}
