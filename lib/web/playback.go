package web

import (
	"behringerRecorder/lib/types"
	"encoding/binary"
	"math"
	"time"

	"github.com/gorilla/websocket"
)

// CalculatePeakMeters finds the peak (maximum absolute) volume for left and right channels.
// Used to display VU meters in the UI showing real-time recording levels.
//
// Parameters:
//
//	buffer: float32 array of stereo samples interleaved [L, R, L, R, ...]
//
// Returns:
//
//	maxVolL: Peak absolute value for left channel [0.0 to 1.0]
//	maxVolR: Peak absolute value for right channel [0.0 to 1.0]
func CalculatePeakMeters(buffer []float32) (float32, float32) {
	var maxVolL, maxVolR float32
	for i := 0; i < len(buffer); i += 2 {
		sL := float32(math.Abs(float64(buffer[i])))
		sR := float32(math.Abs(float64(buffer[i+1])))
		if sL > maxVolL {
			maxVolL = sL
		}
		if sR > maxVolR {
			maxVolR = sR
		}
	}
	return maxVolL, maxVolR
}

// StartAudioBroadcaster starts a goroutine that continuously broadcasts audio data to connected WebSocket clients.
// It encodes audio chunks into a binary protocol and sends them to the frontend UI.
//
// Binary Protocol Format (Little Endian):
//
//	Offset  Size  Field       Type      Description
//	------  ----  -----       ----      -----------
//	0       4     maxL        float32   Peak level for left channel [0.0 to 1.0]
//	4       4     maxR        float32   Peak level for right channel [0.0 to 1.0]
//	8+      4*N   audioData   float32[] Stereo audio samples, interleaved [L, R, L, R, ...]
//
// Total Packet Size: 8 + (chunk length * 4) bytes
//
// Example:
//
//	Input chunk: [0.5, -0.3, 0.1, 0.2]
//	maxL = 0.5, maxR = 0.3
//	Binary packet (hex):
//	  Bytes 0-3:   00 00 00 3f (0.5 as float32)
//	  Bytes 4-7:   9a 99 99 3e (0.3 as float32)
//	  Bytes 8-11:  00 00 00 3f (0.5 as float32 audio sample)
//	  Bytes 12-15: cd cc 9e be (-0.3 as float32 audio sample)
//	  Bytes 16-19: cd cc cc 3d (0.1 as float32 audio sample)
//	  Bytes 20-23: cd cc 23 3e (0.2 as float32 audio sample)
func StartAudioBroadcaster(state *types.AppState, playbackChan <-chan []float32) {
	go func() {
		for chunk := range playbackChan {
			maxL, maxR := CalculatePeakMeters(chunk)

			state.Mu.RLock()
			if len(state.Clients) == 0 {
				state.Mu.RUnlock()
				continue
			}

			// Build binary packet: [maxL (4B)] [maxR (4B)] [audio samples (4B each)]
			packetSize := 8 + (len(chunk) * 4)
			packetBuf := make([]byte, packetSize)

			// Write meter peaks (float32 bits in little endian)
			binary.LittleEndian.PutUint32(packetBuf[0:], math.Float32bits(maxL))
			binary.LittleEndian.PutUint32(packetBuf[4:], math.Float32bits(maxR))

			// Write audio samples as float32 in little endian
			for i, v := range chunk {
				binary.LittleEndian.PutUint32(packetBuf[8+i*4:], math.Float32bits(v))
			}

			for c := range state.Clients {
				c.Conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
				if err := c.WriteMessage(websocket.BinaryMessage, packetBuf); err != nil {
					c.Close()
					// We can't delete here while RLock holds.
					// The handler loop in server.go handles cleanup on error/close.
				}
			}
			state.Mu.RUnlock()

			// Re-acquire lock to delete closed clients if needed
			// For now, let's just keep it simple as it was a POC
		}
	}()
}
