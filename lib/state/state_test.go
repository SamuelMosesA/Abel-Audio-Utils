package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUpdateBroadcast(t *testing.T) {
	appState := NewAppState("", "")
	
	t.Run("Update and Broadcast", func(t *testing.T) {
		ch := make(chan StateChange, 1)
		appState.BroadcastHub.Store(ch, true)
		
		section := SectionRecording
		
		Update[RecordIntent](appState, section, func(s *RecordIntent) {
			s.SetRecording(true)
		})
		
		assert.True(t, appState.IsRecording())
		
		select {
		case change := <-ch:
			assert.Equal(t, "recording", change.Section)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for broadcast")
		}
	})
}

func TestBroadcastHubRobustness(t *testing.T) {
	appState := NewAppState("", "")
	chFull := make(chan StateChange, 1)
	chFull <- StateChange{Section: "full"} // Fill it
	
	appState.BroadcastHub.Store(chFull, true)
	
	// This should not block even if chFull is full
	done := make(chan bool)
	go func() {
		Update[RecordIntent](appState, SectionRecording, func(s *RecordIntent) {})
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Update blocked on full channel")
	}
}
