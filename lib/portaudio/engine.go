package portaudio

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/types"
	"fmt"
	"log"
	"time"

	pa "github.com/gordonklaus/portaudio"
)

func StartAudioEngine(state *types.AppState, cfg *config.Config, deviceID int, recordChan chan<- []float32, playbackChan chan<- []float32) error {
	state.Mu.Lock()
	if state.QuitAudio != nil {
		close(state.QuitAudio)
		state.QuitAudio = nil
		time.Sleep(100 * time.Millisecond)
	}
	quit := make(chan bool)
	state.QuitAudio = quit
	state.IsRunning = true
	state.Mu.Unlock()

	devices, _ := pa.Devices()
	if deviceID >= len(devices) {
		return fmt.Errorf("invalid device")
	}
	dev := devices[deviceID]

	// Engine GoRoutine
	go func() {
		log.Printf("[AUDIO] Started: %s", dev.Name)
		defer log.Println("[AUDIO] Stopped")

		in := make([]float32, cfg.BufferSize*dev.MaxInputChannels)
		stream, err := pa.OpenStream(pa.StreamParameters{
			Input:      pa.StreamDeviceParameters{Device: dev, Channels: dev.MaxInputChannels, Latency: dev.DefaultLowInputLatency},
			SampleRate: float64(cfg.SampleRate), FramesPerBuffer: cfg.BufferSize,
		}, in)
		if err != nil {
			log.Println(err)
			return
		}
		stream.Start()
		defer stream.Stop()
		defer stream.Close()

		for {
			select {
			case <-quit:
				return
			default:
			}

			if err := stream.Read(); err != nil {
				continue
			}

			state.Mu.RLock()
			chL, chR := state.ChLeft, state.ChRight
			boost := float32(state.Boost)
			if boost == 0 {
				boost = 1.0
			}
			state.Mu.RUnlock()

			stereoChunk := make([]float32, cfg.BufferSize*2)
			for i := 0; i < cfg.BufferSize; i++ {
				idxL := (i * dev.MaxInputChannels) + chL
				idxR := (i * dev.MaxInputChannels) + chR

				var sL, sR float32
				if idxL < len(in) {
					sL = in[idxL]
				}
				if idxR < len(in) {
					sR = in[idxR]
				}

				sL *= boost
				sR *= boost
				if sL > 1.0 {
					sL = 1.0
				} else if sL < -1.0 {
					sL = -1.0
				}
				if sR > 1.0 {
					sR = 1.0
				} else if sR < -1.0 {
					sR = -1.0
				}

				stereoChunk[i*2] = sL
				stereoChunk[i*2+1] = sR
			}

			// Fan-out to consumers
			select {
			case recordChan <- stereoChunk:
			default:
			}
			select {
			case playbackChan <- stereoChunk:
			default:
			}
		}
	}()

	return nil
}
