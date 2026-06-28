package audioengine

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"abel/src/backend/lib/telemetry"
	"context"
	"fmt"
	"log/slog"
	"time"

	pa "github.com/gordonklaus/portaudio"
)

func StartAudioEngine(streamer AudioStreamer, appState *state.AppState, cfg *config.Config, deviceID int, recordChan chan<- []float32, playbackChan chan<- []float32) error {
	if streamer == nil {
		streamer = &PADriver{}
	}

	if q := appState.QuitAudio; q != nil {
		close(q)
		appState.QuitAudio = nil
		time.Sleep(100 * time.Millisecond)
	}
	quit := make(chan bool)
	appState.QuitAudio = quit
	appState.Engine().SetRunning(true)

	devices := appState.Devices

	if deviceID >= len(devices) {
		return fmt.Errorf("invalid device")
	}
	dev := devices[deviceID]

	// Engine GoRoutine
	logger := slog.With("component", "audio")
	go func() {
		logger.Info("Audio engine started", slog.String("device", dev.Name))
		defer logger.Info("Audio engine stopped")
		defer func() { appState.Engine().SetRunning(false) }()
		defer func() {
			if appState.Translator != nil {
				appState.Translator.CloseAll()
			}
		}()

		var in []float32
		var stream PortAudioStream
		var err error
		openedSampleRate := cfg.SampleRate
		openedChannels := dev.MaxInputChannels

		logger.Info("Opening stream",
			slog.String("audio.device", dev.Name),
			slog.Int("audio.channels", openedChannels),
			slog.Int("audio.sample_rate", openedSampleRate),
		)

		// Try 1: Configured sample rate and MaxInputChannels
		in = make([]float32, cfg.BufferSize*openedChannels)
		stream, err = streamer.OpenStream(pa.StreamParameters{
			Input:      pa.StreamDeviceParameters{Device: dev, Channels: openedChannels, Latency: dev.DefaultLowInputLatency},
			SampleRate: float64(openedSampleRate), FramesPerBuffer: cfg.BufferSize,
		}, in)

		// Try 2: If configured sample rate failed, fallback directly to DefaultSampleRate of the device
		if err != nil {
			logger.Warn("Failed to open stream at requested sample rate, falling back to device default",
				slog.Int("audio.requested_rate", cfg.SampleRate),
				slog.Float64("audio.default_rate", dev.DefaultSampleRate),
				slog.Any("audio.error", err),
			)
			openedSampleRate = int(dev.DefaultSampleRate)
			if openedSampleRate <= 0 {
				openedSampleRate = 44100 // Safe default fallback
			}
			in = make([]float32, cfg.BufferSize*openedChannels)
			stream, err = streamer.OpenStream(pa.StreamParameters{
				Input:      pa.StreamDeviceParameters{Device: dev, Channels: openedChannels, Latency: dev.DefaultLowInputLatency},
				SampleRate: float64(openedSampleRate), FramesPerBuffer: cfg.BufferSize,
			}, in)
		}

		// Try 3: If that still failed, try fallback to 2 channels
		if err != nil && openedChannels > 2 {
			logger.Warn("Failed to open stream with max channels, trying fallback with 2 channels",
				slog.Int("audio.requested_channels", openedChannels),
				slog.Any("audio.error", err),
			)
			openedChannels = 2
			in = make([]float32, cfg.BufferSize*openedChannels)
			stream, err = streamer.OpenStream(pa.StreamParameters{
				Input:      pa.StreamDeviceParameters{Device: dev, Channels: openedChannels, Latency: dev.DefaultLowInputLatency},
				SampleRate: float64(openedSampleRate), FramesPerBuffer: cfg.BufferSize,
			}, in)
		}

		if err != nil {
			logger.Error("Error opening stream: all configurations failed", slog.Any("audio.error", err))
			return
		}

		// Update state with the actually opened sample rate!
		state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
			s.SetSampleRate(int32(openedSampleRate))
		})

		stream.Start()
		defer stream.Stop()
		defer stream.Close()

		for {
			select {
			case <-quit:
				return
			default:
			}

			startTime := time.Now()

			if err := stream.Read(); err != nil {
				continue
			}

			// Read current interface config once per loop
			conf := appState.Config()
			chL := int(conf.ChL())
			chR := int(conf.ChR())
			boost := float32(conf.Boost())

			if boost == 0 {
				boost = 1.0
			}

			stereoChunk := make([]float32, cfg.BufferSize*2)
			for i := 0; i < cfg.BufferSize; i++ {
				idxL := (i * openedChannels) + chL
				idxR := (i * openedChannels) + chR

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

			if telemetry.AudioLoopLatency != nil {
				telemetry.AudioLoopLatency.Record(context.Background(), float64(time.Since(startTime).Nanoseconds())/1e6)
			}
		}
	}()

	return nil
}
