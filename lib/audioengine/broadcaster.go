package audioengine

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/state"
	"encoding/binary"
	"log"
	"math"

	"github.com/gorilla/websocket"
)

// CalculatePeakMeters finds the peak (maximum absolute) volume for left and right channels.
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
func StartAudioBroadcaster(appState *state.AppState, cfg *config.Config, playbackChan <-chan []float32) {
	go func() {
		count := 0
		totalAIPush := 0

		for chunk := range playbackChan {
			// Calculate peak meters for the chunk
			maxL, maxR := CalculatePeakMeters(chunk)

			// Build binary packet for WS
			packetSize := 8 + (len(chunk) * 4)
			packetBuf := make([]byte, packetSize)
			binary.LittleEndian.PutUint32(packetBuf[0:], math.Float32bits(maxL))
			binary.LittleEndian.PutUint32(packetBuf[4:], math.Float32bits(maxR))
			for i, v := range chunk {
				binary.LittleEndian.PutUint32(packetBuf[8+i*4:], math.Float32bits(v))
			}

			// Broadcast to all WS clients (Non-blocking)
			go func(p []byte) {
				appState.Clients.Range(func(key, value interface{}) bool {
					client := key.(*state.WSClient)
					// Use thread-safe wrapper
					client.WriteMessage(websocket.BinaryMessage, p)
					return true
				})
			}(packetBuf)

			// Fan out to HTTP stream channels
			appState.StreamChannels.Range(func(key, value interface{}) bool {
				ch := key.(chan []float32)
				select {
				case ch <- chunk:
				default:
				}
				return true
			})

			// Push to AI for translation
			if appState.Translator != nil {
				appState.Translator.PushAudio(chunk)
				totalAIPush++
			}

			count++
			if count%100 == 0 {
				log.Printf("[BROADCAST] Processed %d raw chunks (AI Pushes: %d)", count, totalAIPush)
			}
		}
	}()
}
