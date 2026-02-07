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

	devices := state.Devices

	if deviceID >= len(devices) {
		return fmt.Errorf("invalid device")
	}
	dev := devices[deviceID]

	// Engine GoRoutine
	go func() {
		log.Printf("[AUDIO] Started: %s", dev.Name)
		defer log.Println("[AUDIO] Stopped")

		// Input buffer provided to PortAudio. When the stream is opened we pass
		// this slice to PortAudio. Each call to `stream.Read()` fills `in` with
		// `FramesPerBuffer` frames of audio. Samples are float32 values in the
		// range [-1.0, 1.0], laid out as interleaved channels per frame:
		//   [frame0_ch0, frame0_ch1, ..., frame0_chN, frame1_ch0, frame1_ch1, ...]
		// The total length is `cfg.BufferSize * dev.MaxInputChannels`.
		//
		// We later read specific input channel indexes (chL / chR) out of this
		// interleaved buffer and copy them into a separate stereo buffer
		// (`stereoChunk`) with layout [L,R,L,R,...] for consumers.
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

		chL, chR := state.ChLeft, state.ChRight
		boost := float32(state.Boost)
		if boost == 0 {
			boost = 1.0
		}

		for {
			select {
			case <-quit:
				return
			default:
			}

			// `stream.Read()` blocks (or returns quickly depending on callback/RT
			// behavior) and fills the `in` slice with the next `cfg.BufferSize`
			// frames of audio, in the interleaved layout described above. An
			// error can indicate an underrun/overrun or device error; when it
			// happens we skip this buffer and continue.
			if err := stream.Read(); err != nil {
				continue
			}

			stereoChunk := make([]float32, cfg.BufferSize*2)
			for i := 0; i < cfg.BufferSize; i++ {
				// Compute index into the interleaved `in` buffer for this
				// frame `i` and the chosen channel indexes `chL`/`chR`.
				// Formula: index = (frameIndex * numChannels) + channelIndex
				idxL := (i * dev.MaxInputChannels) + chL
				idxR := (i * dev.MaxInputChannels) + chR

				// Read samples if available (some devices may report fewer
				// channels than MaxInputChannels; guard against out-of-range).
				var sL, sR float32
				if idxL < len(in) {
					sL = in[idxL]
				}
				if idxR < len(in) {
					sR = in[idxR]
				}

				// Apply gain/boost, then clamp to the valid float sample range
				// expected by downstream consumers ([-1.0, 1.0]). This keeps the
				// audio safe for playback and prevents extreme values when
				// serializing or writing to files.
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
