package openai

// DefaultPendingAudioMaxBytes is ~15 seconds of 24 kHz mono 16-bit PCM (24000 * 2 bytes * 15 sec).
const DefaultPendingAudioMaxBytes = 24000 * 2 * 15

// PendingAudioBuffer is a bounded FIFO buffer that holds audio chunks during WebSocket disconnects,
// evicting the oldest chunks if the maximum capacity is exceeded so client streams never stall.
type PendingAudioBuffer struct {
	chunks   [][]byte
	bytes    int
	maxBytes int
}

// NewPendingAudioBuffer creates a new buffer with the specified byte capacity limit.
func NewPendingAudioBuffer(maxBytes int) *PendingAudioBuffer {
	if maxBytes <= 0 {
		maxBytes = DefaultPendingAudioMaxBytes
	}
	return &PendingAudioBuffer{
		maxBytes: maxBytes,
	}
}

// Push enqueues a new audio chunk, dropping oldest chunks if capacity is exceeded.
func (p *PendingAudioBuffer) Push(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	p.chunks = append(p.chunks, chunk)
	p.bytes += len(chunk)
	for p.bytes > p.maxBytes && len(p.chunks) > 0 {
		p.bytes -= len(p.chunks[0])
		p.chunks = p.chunks[1:]
	}
}

// Pop dequeues the oldest audio chunk from the buffer.
func (p *PendingAudioBuffer) Pop() []byte {
	if len(p.chunks) == 0 {
		return nil
	}
	chunk := p.chunks[0]
	p.bytes -= len(chunk)
	p.chunks = p.chunks[1:]
	return chunk
}

// Len returns the number of chunks currently buffered.
func (p *PendingAudioBuffer) Len() int {
	return len(p.chunks)
}

// Bytes returns the total byte count of buffered audio.
func (p *PendingAudioBuffer) Bytes() int {
	return p.bytes
}

// Chunks returns the raw chunk slice for iteration.
func (p *PendingAudioBuffer) Chunks() [][]byte {
	return p.chunks
}

// Clear flushes all buffered audio chunks.
func (p *PendingAudioBuffer) Clear() {
	p.chunks = nil
	p.bytes = 0
}
