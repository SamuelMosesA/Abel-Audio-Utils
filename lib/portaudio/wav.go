package portaudio

import (
	"encoding/binary"
	"os"
)

// WritePlaceholderHeader writes a 44-byte placeholder WAV header at the start of the file.
// This is done because the WAV header contains the total file size and data size,
// which are unknown until recording finishes. By writing a placeholder first,
// we reserve space and can seek back to fill in the correct values later.
// WAV header structure is always 44 bytes for standard PCM audio:
//   - RIFF header (12 bytes)
//   - fmt chunk (24 bytes)
//   - data chunk header (8 bytes)
func WritePlaceholderHeader(f *os.File) {
	if f == nil {
		return
	}
	f.Seek(0, 0)
	f.Write(make([]byte, 44))
}

func FinalizeWavHeader(f *os.File, ch uint16, s int64, sampleRate int) {
	if f == nil {
		return
	}
	// Calculate sizes in bytes
	// Each sample is int16 (2 bytes), stereo has 2 channels
	dataSize := uint32(s * int64(ch) * 2)                   // Total audio data in bytes
	byteRate := uint32(uint32(sampleRate) * uint32(ch) * 2) // Bytes per second (SampleRate * Channels * 2)
	blockAlign := uint16(ch * 2)                            // Bytes per sample frame (Channels * 2)

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
	f.Seek(0, 0)
	f.Write([]byte{'R', 'I', 'F', 'F'})
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	f.Write([]byte{'W', 'A', 'V', 'E'})
	f.Write([]byte{'f', 'm', 't', ' '})
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1)) // PCM format
	binary.Write(f, binary.LittleEndian, ch)
	binary.Write(f, binary.LittleEndian, uint32(sampleRate))
	binary.Write(f, binary.LittleEndian, byteRate)
	binary.Write(f, binary.LittleEndian, blockAlign)
	binary.Write(f, binary.LittleEndian, uint16(16)) // 16-bit samples
	f.Write([]byte{'d', 'a', 't', 'a'})
	binary.Write(f, binary.LittleEndian, dataSize)
	f.Close()
}
