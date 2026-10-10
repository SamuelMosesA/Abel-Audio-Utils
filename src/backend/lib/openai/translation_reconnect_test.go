package openai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRealtime stands in for the OpenAI realtime endpoint. Each accepted
// socket is handed to onConn with its 1-based dial index.
type fakeRealtime struct {
	srv    *httptest.Server
	dials  atomic.Int32
	onConn func(n int, c *websocket.Conn)
	reject atomic.Int32 // number of upcoming dials to fail with HTTP 503
}

func newFakeRealtime(t *testing.T, onConn func(n int, c *websocket.Conn)) *fakeRealtime {
	f := &fakeRealtime{onConn: onConn}
	up := websocket.Upgrader{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(f.dials.Add(1))
		if f.reject.Load() > 0 {
			f.reject.Add(-1)
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		f.onConn(n, c)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeRealtime) wsURL() string {
	return "ws" + strings.TrimPrefix(f.srv.URL, "http")
}

func newTestManager(t *testing.T, endpoint string) *TranslationManager {
	cfg := &config.Config{SampleRate: 48000, OpenAITranslateModel: "test-model"}
	appState := state.NewAppState("", "")
	m, err := NewTranslationManager(cfg, appState, "test-key", "test-model", "", "en", 100, 1000, 100)
	require.NoError(t, err)
	m.endpoint = endpoint
	m.SetEnabled(true)
	return m
}

// readEvents drains a socket into typed events until it closes.
func readEvents(c *websocket.Conn, out chan<- map[string]interface{}) {
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		var ev map[string]interface{}
		if json.Unmarshal(msg, &ev) == nil {
			out <- ev
		}
	}
}

func waitFor(t *testing.T, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// The session survives OpenAI's session_expired: the same session object
// re-dials, audio pushed while disconnected is replayed on the new socket, and
// subtitles keep flowing to existing subscribers.
func TestTranslationSessionReconnectsAfterExpiry(t *testing.T) {
	var mu sync.Mutex
	replayedAudio := 0
	firstConnReady := make(chan struct{})
	replayConnReady := make(chan struct{})

	var fake *fakeRealtime
	fake = newFakeRealtime(t, func(n int, c *websocket.Conn) {
		events := make(chan map[string]interface{}, 100)
		go readEvents(c, events)
		switch n {
		case 1:
			ev := <-events
			assert.Equal(t, "session.update", ev["type"])
			close(firstConnReady)
			// Emulate the 60-minute cap: error event, then the server hangs up.
			// Reject the next dial so the session sits in backoff: that is the
			// window in which audio must be buffered, not dropped.
			fake.reject.Store(1)
			c.WriteJSON(map[string]interface{}{
				"type":  "error",
				"error": map[string]interface{}{"type": "invalid_request_error", "code": "session_expired"},
			})
			return
		case 3:
			ev := <-events
			assert.Equal(t, "session.update", ev["type"])
			close(replayConnReady)
			c.WriteJSON(map[string]interface{}{"type": "session.output_transcript.delta", "delta": "bonjour"})
			for ev := range events {
				if ev["type"] == "session.input_audio_buffer.append" {
					mu.Lock()
					replayedAudio++
					mu.Unlock()
				}
			}
		default:
			for range events {
			}
		}
	})

	m := newTestManager(t, fake.wsURL())
	subs, cleanup := m.GetSubtitles("fr")
	defer cleanup()

	audioOut := m.GetChannel("fr")
	require.NotNil(t, audioOut)
	<-firstConnReady

	// Wait until the rejected re-dial has happened: the session is now in backoff.
	waitFor(t, func() bool { return fake.dials.Load() == 2 }, "rejected re-dial")
	chunk := make([]float32, 960) // 10 ms of 48 kHz stereo
	for i := 0; i < 5; i++ {
		m.OnNewAudioChunk(chunk)
	}

	<-replayConnReady
	waitFor(t, func() bool { mu.Lock(); defer mu.Unlock(); return replayedAudio >= 5 }, "buffered audio replayed on the new socket")

	select {
	case msg := <-subs:
		assert.Contains(t, msg, "bonjour")
	case <-time.After(2 * time.Second):
		t.Fatal("subtitle from reconnected session never reached the subscriber")
	}

	// Same session object, still registered; channel handed out earlier is still the live one.
	v, ok := m.Sessions.Load("fr")
	require.True(t, ok, "session must survive the reconnect")
	assert.Equal(t, audioOut, v.AudioOut)
	assert.Equal(t, int32(3), fake.dials.Load())

	m.StopSession("fr", true)
	waitFor(t, func() bool { _, ok := m.Sessions.Load("fr"); return !ok }, "session removed after StopSession")
	_, open := <-audioOut
	assert.False(t, open, "AudioOut closes when the session is stopped")
}

// A failed dial is retried with backoff instead of ending the session.
func TestTranslationSessionRetriesFailedDial(t *testing.T) {
	connected := make(chan struct{})
	fake := newFakeRealtime(t, func(n int, c *websocket.Conn) {
		close(connected)
		events := make(chan map[string]interface{}, 100)
		readEvents(c, events)
	})
	fake.reject.Store(1)

	m := newTestManager(t, fake.wsURL())
	require.NotNil(t, m.GetChannel("de"))

	select {
	case <-connected:
	case <-time.After(5 * time.Second):
		t.Fatal("never connected after a rejected dial")
	}
	assert.Equal(t, int32(2), fake.dials.Load())
	m.StopSession("de", true)
	waitFor(t, func() bool { _, ok := m.Sessions.Load("de"); return !ok }, "session removed")
}

// StopSession ends the loop: no reconnect after cancellation.
func TestTranslationSessionStopDoesNotReconnect(t *testing.T) {
	fake := newFakeRealtime(t, func(n int, c *websocket.Conn) {
		events := make(chan map[string]interface{}, 100)
		readEvents(c, events)
	})
	m := newTestManager(t, fake.wsURL())
	require.NotNil(t, m.GetChannel("es"))
	waitFor(t, func() bool { return fake.dials.Load() == 1 }, "first dial")
	m.StopSession("es", true)
	waitFor(t, func() bool { _, ok := m.Sessions.Load("es"); return !ok }, "session removed")
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), fake.dials.Load(), "no re-dial after StopSession")
}

// Subtitle-only viewers get a session even though nobody called GetChannel.
func TestTranslationAutoStartsForSubtitleSubscribers(t *testing.T) {
	fake := newFakeRealtime(t, func(n int, c *websocket.Conn) {
		events := make(chan map[string]interface{}, 100)
		readEvents(c, events)
	})
	m := newTestManager(t, fake.wsURL())
	_, cleanup := m.GetSubtitles("it")
	defer cleanup()

	_, ok := m.Sessions.Load("it")
	assert.False(t, ok, "no session before audio flows")

	m.OnNewAudioChunk(make([]float32, 960))
	waitFor(t, func() bool { _, ok := m.Sessions.Load("it"); return ok }, "auto-started session")
	waitFor(t, func() bool { return fake.dials.Load() == 1 }, "dial")
	m.StopSession("it", true)
}
