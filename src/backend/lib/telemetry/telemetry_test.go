package telemetry

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionLogFilterHandler_DowngradesSessionError(t *testing.T) {
	var buf bytes.Buffer
	innerHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	filterHandler := NewSessionLogFilterHandler(innerHandler)
	logger := slog.New(filterHandler)

	// Simulate exact log call made by github.com/gin-contrib/sessions
	logger.Error("[sessions] ERROR!", "err", errors.New("securecookie: the value is not valid"))

	output := buf.String()
	assert.Contains(t, output, "level=WARN")
	assert.Contains(t, output, "[sessions]")
	assert.Contains(t, output, "securecookie: the value is not valid")
	assert.NotContains(t, output, "level=ERROR")
}

func TestSessionLogFilterHandler_PreservesNormalErrors(t *testing.T) {
	var buf bytes.Buffer
	innerHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	filterHandler := NewSessionLogFilterHandler(innerHandler)
	logger := slog.New(filterHandler)

	// Normal errors should remain level=ERROR
	logger.Error("Database connection failed", "err", errors.New("timeout"))

	output := buf.String()
	assert.Contains(t, output, "level=ERROR")
	assert.Contains(t, output, "Database connection failed")
	assert.NotContains(t, output, "level=WARN")
}
