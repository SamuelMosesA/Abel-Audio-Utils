package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeLogHandler(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{}
	router := setupTestRouter(appState, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "/api/system/changelog", nil)
	req.Header.Set("X-Test-Auth", "true")

	w := httptest.NewRecorder()
	stateObj := appState
	go router.ServeHTTP(w, req)

	time.Sleep(100 * time.Millisecond)
	state.Update[state.RecordIntent](stateObj, state.SectionRecording, func(s *state.RecordIntent) {})
	time.Sleep(100 * time.Millisecond)
	cancel()
}

func TestGetSystemConnection(t *testing.T) {
	cfg := &config.Config{Port: "8080"}
	router := setupTestRouter(state.NewAppState("", ""), cfg)

	req, _ := http.NewRequest("GET", "/api/system/connection", nil)
	req.Header.Set("X-Test-Auth", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp SystemConnectionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Contains(t, resp.ServerURL, "8080")
	assert.Equal(t, "8080", resp.Port)
	assert.NotEmpty(t, resp.Host)
	assert.Equal(t, resp.Host+":8080", resp.DisplayEndpoint)
	assert.NotEmpty(t, resp.SSID)
}

func TestParseDarwinSSID(t *testing.T) {
	sampleOutput := `
en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	ether 3c:22:fb:11:22:33
	inet 192.168.1.100 netmask 0xffffff00 broadcast 192.168.1.255
	BSSID : a0:b1:c2:d3:e4:f5
	SSID : Church_Auditorium_5G
	Channel : 36
`
	ssid := ParseDarwinSSID(sampleOutput)
	assert.Equal(t, "Church_Auditorium_5G", ssid)

	assert.Empty(t, ParseDarwinSSID("no ssid here"))
}
