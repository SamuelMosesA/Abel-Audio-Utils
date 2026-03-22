package portaudio

import pa "github.com/gordonklaus/portaudio"

// AudioStreamer defines the interface for PortAudio stream operations.
type AudioStreamer interface {
	OpenStream(params pa.StreamParameters, args ...interface{}) (PortAudioStream, error)
}

// PortAudioStream defines the interface for a single PortAudio stream.
type PortAudioStream interface {
	Start() error
	Stop() error
	Close() error
	Read() error
}

// PADriver is the real implementation of AudioStreamer using PortAudio.
type PADriver struct{}

func (d *PADriver) OpenStream(params pa.StreamParameters, args ...interface{}) (PortAudioStream, error) {
	stream, err := pa.OpenStream(params, args...)
	if err != nil {
		return nil, err
	}
	return stream, nil
}
