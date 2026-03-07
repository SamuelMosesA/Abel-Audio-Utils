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
	if q := state.QuitAudio; q != nil {
		close(q)
		state.QuitAudio = nil
		time.Sleep(100 * time.Millisecond)
	}
	quit := make(chan bool)
	state.QuitAudio = quit
	state.IsRunning.Store(true)
	state.Mu.Unlock()

	devices := state.Devices

	if deviceID >= len(devices) {
		return fmt.Errorf("invalid device")
	}
	dev := devices[deviceID]

	// Engine GoRoutine
	go func() {
		log.Printf("[AUDIO] Started: %s", dev.Name)
		defer log.Println("[AUDIO] Stopped")
		defer state.IsRunning.Store(false)
		defer func() {
			if state.Translator != nil {
				state.Translator.CloseAll()
			}
		}()

		in := make([]float32, cfg.BufferSize*dev.MaxInputChannels)
		log.Printf("[AUDIO] Opening stream: %s (%d channels, %d Hz)", dev.Name, dev.MaxInputChannels, cfg.SampleRate)
		
		stream, err := pa.OpenStream(pa.StreamParameters{
			Input:      pa.StreamDeviceParameters{Device: dev, Channels: dev.MaxInputChannels, Latency: dev.DefaultLowInputLatency},
			SampleRate: float64(cfg.SampleRate), FramesPerBuffer: cfg.BufferSize,
		}, in)
		
		if err != nil {
			log.Printf("[AUDIO] Failed to open stream with %d channels: %v. Trying with 2 channels...", dev.MaxInputChannels, err)
			
			// Fallback to 2 channels if possible
			channels := 2
			if dev.MaxInputChannels < 2 {
				channels = dev.MaxInputChannels
			}
			
			if channels > 0 {
				in = make([]float32, cfg.BufferSize*channels)
				stream, err = pa.OpenStream(pa.StreamParameters{
					Input:      pa.StreamDeviceParameters{Device: dev, Channels: channels, Latency: dev.DefaultLowInputLatency},
					SampleRate: float64(cfg.SampleRate), FramesPerBuffer: cfg.BufferSize,
				}, in)
			}
		}

		if err != nil {
			log.Printf("[AUDIO] Error opening stream: %v", err)
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

			chL := int(state.ChLeft.Load())
			chR := int(state.ChRight.Load())
			boost := float32(state.GetBoost())
			if boost == 0 {
				boost = 1.0
			}

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
