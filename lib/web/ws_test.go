package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/state"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestNewWSHandler(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{
		Credentials: map[string]string{"admin": "password"},
	}
	router := setupTestRouter(appState, cfg)
	server := httptest.NewServer(router)
	defer server.Close()

	// 1. Login to get a session cookie
	loginBody := map[string]string{"username": "admin", "password": "password"}
	jsonBody, _ := json.Marshal(loginBody)
	resp, err := http.Post(server.URL+"/api/auth/session", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	cookie := resp.Header.Get("Set-Cookie")
	assert.NotEmpty(t, cookie)

	// 2. Dial WebSocket with session cookie
	wsURL := "ws" + server.URL[4:] + "/ws"
	header := http.Header{"Cookie": {cookie}}
	dialer := websocket.Dialer{}
	conn, wsResp, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	assert.Equal(t, http.StatusSwitchingProtocols, wsResp.StatusCode)
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)
	found := false
	appState.Clients.Range(func(key, value interface{}) bool {
		found = true
		return false
	})
	assert.True(t, found)
}
