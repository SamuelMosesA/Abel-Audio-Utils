package portaudio

import (
	"behringerRecorder/lib/types"
	"encoding/binary"
)

func StartStorageWorker(state *types.AppState, recordChan <-chan []float32) {
	go func() {
		for chunk := range recordChan {
			state.Mu.Lock()
			if state.IsRecording {
				for i := 0; i < len(chunk); i += 2 {
					sL, sR := chunk[i], chunk[i+1]
					iL, iR := int16(sL*32767), int16(sR*32767)
					binary.Write(state.File, binary.LittleEndian, iL)
					binary.Write(state.File, binary.LittleEndian, iR)
				}
				state.SamplesWrote += int64(len(chunk) / 2)
			}
			state.Mu.Unlock()
		}
	}()
}
