package audio_processing

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	mp3BitrateKbps        = "128k"
	mp3InputBufferSize    = 512
	mp3ListenerBufferSize = 64
	mp3ChunkReadSize      = 4096
)

var (
	ErrBroadcasterClosed = errors.New("audio broadcaster is closed")
	ErrStreamNotRunning  = errors.New("audio stream is not running")
	ErrInvalidStreamName = errors.New("invalid stream name")
	streamNamePattern    = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
)

type mp3Stream struct {
	language   string
	sampleRate int
	input      chan []float32
	ctx        context.Context
	cancel     context.CancelFunc
	done       chan struct{}

	mu        sync.RWMutex
	listeners map[chan []byte]struct{}
}

// LiveAudioBroadcaster manages in-memory FFmpeg MP3 encoding pipelines for live audio channels.
// Audio is piped into FFmpeg via stdin (float32le PCM) and read from stdout as continuous MP3 frames,
// with zero disk I/O and zero temporary files.
type LiveAudioBroadcaster struct {
	mu         sync.RWMutex
	ffmpegPath string
	streams    map[string]*mp3Stream
	closed     bool
}

// NewLiveAudioBroadcaster creates a broadcaster using the system ffmpeg binary.
func NewLiveAudioBroadcaster() (*LiveAudioBroadcaster, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg is required for live MP3 audio streaming: %w", err)
	}
	return NewLiveAudioBroadcasterWithPath(ffmpegPath), nil
}

// NewLiveAudioBroadcasterWithPath creates a broadcaster with an explicit ffmpeg binary path.
func NewLiveAudioBroadcasterWithPath(ffmpegPath string) *LiveAudioBroadcaster {
	return &LiveAudioBroadcaster{
		ffmpegPath: ffmpegPath,
		streams:    make(map[string]*mp3Stream),
	}
}

// EnsureStream ensures an in-memory FFmpeg MP3 encoder is running for the given language channel.
func (b *LiveAudioBroadcaster) EnsureStream(language string, sampleRate int, source <-chan []float32) error {
	if !streamNamePattern.MatchString(language) {
		return ErrInvalidStreamName
	}
	if sampleRate <= 0 {
		sampleRate = DefaultWavSampleRate
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrBroadcasterClosed
	}

	if current, ok := b.streams[language]; ok {
		if current.sampleRate == sampleRate {
			b.mu.Unlock()
			return nil
		}
		// Sample rate changed; restart encoder
		delete(b.streams, language)
		current.cancel()
		b.mu.Unlock()
		select {
		case <-current.done:
		case <-time.After(2 * time.Second):
		}
		return b.EnsureStream(language, sampleRate, source)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, b.ffmpegPath,
		"-f", "f32le",
		"-ar", strconv.Itoa(sampleRate),
		"-ac", "2",
		"-i", "pipe:0",
		"-f", "mp3",
		"-b:a", mp3BitrateKbps,
		"-flush_packets", "1",
		"pipe:1",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		b.mu.Unlock()
		return fmt.Errorf("open ffmpeg stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		b.mu.Unlock()
		return fmt.Errorf("open ffmpeg stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		b.mu.Unlock()
		return fmt.Errorf("start in-memory ffmpeg mp3 stream: %w", err)
	}

	stream := &mp3Stream{
		language:   language,
		sampleRate: sampleRate,
		input:      make(chan []float32, mp3InputBufferSize),
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
		listeners:  make(map[chan []byte]struct{}),
	}
	b.streams[language] = stream
	b.mu.Unlock()

	go b.runStream(language, stream, cmd, stdin, stdout)

	if source != nil {
		go func() {
			for {
				select {
				case chunk, ok := <-source:
					if !ok {
						b.stopStream(language, stream)
						return
					}
					_ = b.Publish(language, sampleRate, chunk)
				case <-stream.done:
					return
				}
			}
		}()
	}

	return nil
}

// Publish enqueues a stereo float32 PCM chunk to the language's active MP3 encoder.
func (b *LiveAudioBroadcaster) Publish(language string, sampleRate int, chunk []float32) error {
	if len(chunk) == 0 {
		return nil
	}
	if err := b.EnsureStream(language, sampleRate, nil); err != nil {
		return err
	}

	b.mu.RLock()
	stream := b.streams[language]
	b.mu.RUnlock()

	if stream == nil {
		return ErrStreamNotRunning
	}

	copyOfChunk := append([]float32(nil), chunk...)
	select {
	case stream.input <- copyOfChunk:
	default:
		// Drop chunk if encoder queue is backed up
	}
	return nil
}

// Subscribe registers a listener to receive real-time MP3 bytes for the specified language.
// It ensures the stream is running, allocates a bounded listener channel, and returns an unsubscribe func.
func (b *LiveAudioBroadcaster) Subscribe(language string, sampleRate int, source <-chan []float32) (<-chan []byte, func(), error) {
	if err := b.EnsureStream(language, sampleRate, source); err != nil {
		return nil, nil, err
	}

	b.mu.RLock()
	stream := b.streams[language]
	b.mu.RUnlock()

	if stream == nil {
		return nil, nil, ErrStreamNotRunning
	}

	ch := make(chan []byte, mp3ListenerBufferSize)

	stream.mu.Lock()
	stream.listeners[ch] = struct{}{}
	stream.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			stream.mu.Lock()
			_, exists := stream.listeners[ch]
			if exists {
				delete(stream.listeners, ch)
			}
			stream.mu.Unlock()

			// Only close the channel if it was still in the listener map
			// (prevents double-close panic if the stream runner already closed all listeners)
			if exists {
				close(ch)
			}
		})
	}

	return ch, unsubscribe, nil
}

// runStream coordinates writing PCM to stdin and fanning out encoded MP3 from stdout.
func (b *LiveAudioBroadcaster) runStream(language string, stream *mp3Stream, cmd *exec.Cmd, stdin io.WriteCloser, stdout io.ReadCloser) {
	logger := slog.With("component", "mp3_stream", "stream.language", language)
	logger.Info("In-memory MP3 streaming pipeline started")

	// Goroutine 1: Read MP3 frames from stdout and fan out to listeners
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, mp3ChunkReadSize)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])

				stream.mu.RLock()
				for listener := range stream.listeners {
					select {
					case listener <- chunk:
					default:
						// Non-blocking drop for slow listener to prevent blocking broadcaster
					}
				}
				stream.mu.RUnlock()
			}
			if err != nil {
				break
			}
		}
	}()

	// Main loop: Write PCM chunks to stdin
	rawBuf := make([]byte, 0, 4096*4)
	stopping := false
	for !stopping {
		select {
		case chunk, ok := <-stream.input:
			if !ok {
				stopping = true
				break
			}
			needed := len(chunk) * 4
			if cap(rawBuf) < needed {
				rawBuf = make([]byte, needed)
			} else {
				rawBuf = rawBuf[:needed]
			}
			for i, sample := range chunk {
				binary.LittleEndian.PutUint32(rawBuf[i*4:], math.Float32bits(sample))
			}
			if _, err := stdin.Write(rawBuf); err != nil {
				stopping = true
			}
		case <-stream.ctx.Done():
			stopping = true
		}
	}

	_ = stdin.Close()
	_ = cmd.Wait()
	wg.Wait()
	close(stream.done)

	// Clean up listeners: collect and clear map under lock, then close once
	stream.mu.Lock()
	listenersToClose := make([]chan []byte, 0, len(stream.listeners))
	for ch := range stream.listeners {
		listenersToClose = append(listenersToClose, ch)
	}
	stream.listeners = make(map[chan []byte]struct{})
	stream.mu.Unlock()

	for _, ch := range listenersToClose {
		close(ch)
	}

	b.mu.Lock()
	if current, ok := b.streams[language]; ok && current == stream {
		delete(b.streams, language)
	}
	b.mu.Unlock()

	logger.Info("In-memory MP3 streaming pipeline stopped")
}

// stopStream terminates an active stream cleanly.
func (b *LiveAudioBroadcaster) stopStream(language string, expected *mp3Stream) {
	b.mu.Lock()
	current, ok := b.streams[language]
	if !ok || (expected != nil && current != expected) {
		b.mu.Unlock()
		return
	}
	delete(b.streams, language)
	current.cancel()
	b.mu.Unlock()

	select {
	case <-current.done:
	case <-time.After(2 * time.Second):
	}
}

// Close terminates all active streams and closes the broadcaster.
func (b *LiveAudioBroadcaster) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	streams := make([]*mp3Stream, 0, len(b.streams))
	for _, s := range b.streams {
		streams = append(streams, s)
	}
	b.streams = make(map[string]*mp3Stream)
	b.mu.Unlock()

	for _, s := range streams {
		s.cancel()
		select {
		case <-s.done:
		case <-time.After(2 * time.Second):
		}
	}
	return nil
}
