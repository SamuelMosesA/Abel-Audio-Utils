package audioengine

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

const (
	hlsPlaylistName = "index.m3u8"
	hlsInputBuffer  = 512
)

var (
	ErrHLSNotReady       = errors.New("HLS stream is not ready")
	ErrInvalidStreamName = errors.New("invalid stream name")
	hlsStreamNamePattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	hlsSegmentPattern    = regexp.MustCompile(`^segment-[0-9]+\.ts$`)
)

type hlsStream struct {
	input      chan []float32
	dir        string
	sampleRate int
	ctx        context.Context
	cancel     context.CancelFunc
	done       chan struct{}
}

// HLSPublisher packages stereo float32 PCM as short AAC/HLS segments. One
// encoder is shared by every listener of a language stream.
type HLSPublisher struct {
	mu         sync.RWMutex
	root       string
	ffmpegPath string
	streams    map[string]*hlsStream
	closed     bool
}

func NewHLSPublisher() (*HLSPublisher, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg is required for HLS audio streaming: %w", err)
	}

	root, err := os.MkdirTemp("", "abel-hls-")
	if err != nil {
		return nil, fmt.Errorf("create HLS workspace: %w", err)
	}

	return newHLSPublisher(ffmpegPath, root), nil
}

func newHLSPublisher(ffmpegPath, root string) *HLSPublisher {
	return &HLSPublisher{
		root:       root,
		ffmpegPath: ffmpegPath,
		streams:    make(map[string]*hlsStream),
	}
}

func (p *HLSPublisher) EnsureStream(language string, sampleRate int, source <-chan []float32) error {
	if !hlsStreamNamePattern.MatchString(language) {
		return ErrInvalidStreamName
	}
	if sampleRate <= 0 {
		sampleRate = defaultWavSampleRate
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errors.New("HLS publisher is closed")
	}
	if current, ok := p.streams[language]; ok {
		if current.sampleRate == sampleRate {
			p.mu.Unlock()
			return nil
		}
		// A device can restart at a different sample rate. Start a clean HLS
		// timeline so the encoder and playlist never describe the wrong rate.
		delete(p.streams, language)
		current.cancel()
		p.mu.Unlock()
		select {
		case <-current.done:
		case <-time.After(2 * time.Second):
		}
		return p.EnsureStream(language, sampleRate, source)
	}

	dir := filepath.Join(p.root, language)
	if err := os.RemoveAll(dir); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("reset HLS stream directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("create HLS stream directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	playlistPath := filepath.Join(dir, hlsPlaylistName)
	segmentPattern := filepath.Join(dir, "segment-%09d.ts")
	cmd := exec.CommandContext(ctx, p.ffmpegPath,
		"-hide_banner", "-loglevel", "error", "-nostdin",
		"-fflags", "+nobuffer", "-analyzeduration", "0", "-probesize", "32",
		"-f", "s16le", "-ar", fmt.Sprint(sampleRate), "-ac", "2", "-i", "pipe:0",
		"-vn", "-c:a", "aac", "-b:a", "128k",
		"-flush_packets", "1",
		"-f", "hls", "-hls_time", "1", "-hls_list_size", "6", "-hls_delete_threshold", "6",
		"-hls_flags", "delete_segments+omit_endlist+independent_segments+temp_file",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		p.mu.Unlock()
		return fmt.Errorf("open ffmpeg input: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		p.mu.Unlock()
		return fmt.Errorf("open ffmpeg error stream: %w", err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		p.mu.Unlock()
		return fmt.Errorf("start ffmpeg HLS encoder: %w", err)
	}

	stream := &hlsStream{
		input:      make(chan []float32, hlsInputBuffer),
		dir:        dir,
		sampleRate: sampleRate,
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
	}
	p.streams[language] = stream
	p.mu.Unlock()

	go p.runEncoder(language, stream, cmd, stdin, stderr)
	if source != nil {
		go func() {
			for {
				select {
				case chunk, ok := <-source:
					if !ok {
						p.stopStream(language, stream)
						return
					}
					_ = p.Publish(language, sampleRate, chunk)
				case <-stream.done:
					return
				}
			}
		}()
	}

	return nil
}

func (p *HLSPublisher) runEncoder(language string, stream *hlsStream, cmd *exec.Cmd, stdin io.WriteCloser, stderr io.Reader) {
	logger := slog.With("component", "hls", "stream.language", language)
	errorText := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(stderr)
		errorText <- string(data)
	}()

	stopping := false
	for !stopping {
		select {
		case chunk := <-stream.input:
			pcm := float32ToPCM16(chunk)
			if _, err := stdin.Write(pcm); err != nil {
				if stream.ctx.Err() == nil {
					logger.Warn("HLS encoder input closed", slog.Any("error", err))
				}
				stopping = true
			}
		case <-stream.ctx.Done():
			stopping = true
		}
	}
	_ = stdin.Close()
	err := cmd.Wait()
	message := <-errorText
	if err != nil && stream.ctx.Err() == nil {
		logger.Error("HLS encoder stopped", slog.Any("error", err), slog.String("ffmpeg.stderr", message))
	}
	close(stream.done)

	p.mu.Lock()
	if current, ok := p.streams[language]; ok && current == stream {
		delete(p.streams, language)
	}
	p.mu.Unlock()
}

func float32ToPCM16(chunk []float32) []byte {
	pcm := make([]byte, len(chunk)*2)
	for i, sample := range chunk {
		if sample > 1 {
			sample = 1
		} else if sample < -1 {
			sample = -1
		}
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(int16(sample*32767)))
	}
	return pcm
}

func (p *HLSPublisher) Publish(language string, sampleRate int, chunk []float32) error {
	if len(chunk) == 0 {
		return nil
	}
	if err := p.EnsureStream(language, sampleRate, nil); err != nil {
		return err
	}

	p.mu.RLock()
	stream := p.streams[language]
	p.mu.RUnlock()
	if stream == nil {
		return ErrHLSNotReady
	}

	copyOfChunk := append([]float32(nil), chunk...)
	select {
	case stream.input <- copyOfChunk:
	default:
		// Keep live playback near the head instead of blocking the audio engine.
	}
	return nil
}

func (p *HLSPublisher) WaitForPlaylist(ctx context.Context, language string, timeout time.Duration) ([]byte, error) {
	if !hlsStreamNamePattern.MatchString(language) {
		return nil, ErrInvalidStreamName
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		data, err := p.readStreamFile(language, hlsPlaylistName)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, ErrHLSNotReady
		case <-ticker.C:
		}
	}
}

func (p *HLSPublisher) ReadSegment(language, name string) ([]byte, error) {
	if !hlsStreamNamePattern.MatchString(language) || !hlsSegmentPattern.MatchString(name) {
		return nil, ErrInvalidStreamName
	}
	return p.readStreamFile(language, name)
}

func (p *HLSPublisher) readStreamFile(language, name string) ([]byte, error) {
	p.mu.RLock()
	stream := p.streams[language]
	p.mu.RUnlock()
	if stream == nil {
		return nil, ErrHLSNotReady
	}
	return os.ReadFile(filepath.Join(stream.dir, name))
}

func (p *HLSPublisher) stopStream(language string, expected *hlsStream) {
	p.mu.RLock()
	stream := p.streams[language]
	p.mu.RUnlock()
	if stream == nil || stream != expected {
		return
	}
	stream.cancel()
	select {
	case <-stream.done:
	case <-time.After(2 * time.Second):
	}
}

func (p *HLSPublisher) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	streams := make([]*hlsStream, 0, len(p.streams))
	for _, stream := range p.streams {
		streams = append(streams, stream)
		stream.cancel()
	}
	p.mu.Unlock()

	for _, stream := range streams {
		select {
		case <-stream.done:
		case <-time.After(2 * time.Second):
		}
	}
	return os.RemoveAll(p.root)
}
