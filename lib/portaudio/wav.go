package portaudio

import (
	"encoding/binary"
	"os"
)

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
	dataSize := uint32(s * int64(ch) * 2)
	byteRate := uint32(uint32(sampleRate) * uint32(ch) * 2)
	blockAlign := uint16(ch * 2)

	f.Seek(0, 0)
	f.Write([]byte{'R', 'I', 'F', 'F'})
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	f.Write([]byte{'W', 'A', 'V', 'E'})
	f.Write([]byte{'f', 'm', 't', ' '})
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, ch)
	binary.Write(f, binary.LittleEndian, uint32(sampleRate))
	binary.Write(f, binary.LittleEndian, byteRate)
	binary.Write(f, binary.LittleEndian, blockAlign)
	binary.Write(f, binary.LittleEndian, uint16(16))
	f.Write([]byte{'d', 'a', 't', 'a'})
	binary.Write(f, binary.LittleEndian, dataSize)
	f.Close()
}
