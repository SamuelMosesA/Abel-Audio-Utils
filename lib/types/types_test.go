package types

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUpdateState(t *testing.T) {
	state := &AppState{}
	
	t.Run("Update and Broadcast", func(t *testing.T) {
		ch := make(chan StateChange, 1)
		state.BroadcastHub.Store(ch, true)
		
		section := "test-section"
		sessionID := "test-session"
		
		state.UpdateState(sessionID, section, func() {
			state.IsRecording = true
		})
		
		assert.True(t, state.IsRecording)
		
		select {
		case change := <-ch:
			assert.Equal(t, section, change.Section)
			assert.Equal(t, sessionID, change.SessionID)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for broadcast")
		}
	})
}

func TestAppStateConcurrency(t *testing.T) {
	state := &AppState{}
	const workers = 50
	const iterations = 100
	
	var wg sync.WaitGroup
	wg.Add(workers)
	
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				state.UpdateState("session", "section", func() {
					state.SamplesWrote++
				})
			}
		}(i)
	}
	
	wg.Wait()
	assert.Equal(t, int64(workers*iterations), state.SamplesWrote)
}

func TestBroadcastHubRobustness(t *testing.T) {
	state := &AppState{}
	chFull := make(chan StateChange, 1)
	chFull <- StateChange{Section: "full"} // Fill it
	
	state.BroadcastHub.Store(chFull, true)
	
	// This should not block even if chFull is full
	done := make(chan bool)
	go func() {
		state.UpdateState("session", "section", func() {})
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("UpdateState blocked on full channel")
	}
}
