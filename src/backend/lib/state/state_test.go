package state

import (
	"os"
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

func TestEngineStateOperations(t *testing.T) {
	appState := NewAppState("", "")
	engine := appState.Engine()

	// Initial state
	assert.Nil(t, engine.File())
	assert.Equal(t, int64(0), engine.SamplesWrote())
	assert.False(t, engine.IsRunning())

	// Set running
	engine.SetRunning(true)
	assert.True(t, engine.IsRunning())

	// WriteWithFile with no file
	n, err := engine.WriteWithFile(func(f *os.File) (int, error) {
		return 10, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, int64(0), engine.SamplesWrote())

	// Set file and write
	tmpFile, err := os.CreateTemp("", "test_engine_state_*.raw")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	engine.SetFile(tmpFile)
	assert.Equal(t, tmpFile, engine.File())

	n, err = engine.WriteWithFile(func(f *os.File) (int, error) {
		_, writeErr := f.Write([]byte{1, 2, 3, 4})
		return 2, writeErr
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, int64(2), engine.SamplesWrote())

	// TakeFile
	takenFile, samples := engine.TakeFile()
	assert.Equal(t, tmpFile, takenFile)
	assert.Equal(t, int64(2), samples)
	assert.Nil(t, engine.File())

	// Subsequent WriteWithFile should safely no-op
	n, err = engine.WriteWithFile(func(f *os.File) (int, error) {
		return 5, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, int64(2), engine.SamplesWrote())

	// Reset samples
	engine.ResetSamples()
	assert.Equal(t, int64(0), engine.SamplesWrote())
}

func TestAIConfigBlockedLanguages(t *testing.T) {
	appState := NewAppState("", "")

	// Default state: not blocked
	assert.False(t, appState.IsLanguageBlocked("es"))
	assert.False(t, appState.AI().IsBlocked("es"))
	assert.Empty(t, appState.AI().BlockedLanguages())

	// Block "es"
	err := Update[AIConfig](appState, SectionAI, func(s *AIConfig) {
		s.SetBlocked("es", true)
	})
	assert.NoError(t, err)

	assert.True(t, appState.IsLanguageBlocked("es"))
	assert.True(t, appState.IsLanguageBlocked("ES")) // Case insensitive
	assert.True(t, appState.AI().IsBlocked("es"))
	assert.False(t, appState.IsLanguageBlocked("fr"))
	assert.True(t, appState.AI().BlockedLanguages()["es"])

	// Unblock "es"
	err = Update[AIConfig](appState, SectionAI, func(s *AIConfig) {
		s.SetBlocked("es", false)
	})
	assert.NoError(t, err)

	assert.False(t, appState.IsLanguageBlocked("es"))
	assert.False(t, appState.AI().IsBlocked("es"))
	assert.Empty(t, appState.AI().BlockedLanguages())
}
