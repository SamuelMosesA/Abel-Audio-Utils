package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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

func TestGetLocalIP(t *testing.T) {
	ip := GetLocalIP()
	t.Logf("GetLocalIP returned: %s", ip)
	assert.NotEmpty(t, ip)
	// Must not be empty and must be a valid IP address
	parsed := net.ParseIP(ip)
	require.NotNil(t, parsed)
}

func TestInterfaceScoringAndFilters(t *testing.T) {
	assert.True(t, isVirtualInterface("docker0"))
	assert.True(t, isVirtualInterface("br-76a26541b31e"))
	assert.True(t, isVirtualInterface("veth12345"))
	assert.True(t, isVirtualInterface("utun3"))
	assert.True(t, isVirtualInterface("awdl0"))
	assert.False(t, isVirtualInterface("wlan0"))
	assert.False(t, isVirtualInterface("en0"))
	assert.False(t, isVirtualInterface("eth0"))

	assert.True(t, isPhysicalInterface("wlan0"))
	assert.True(t, isPhysicalInterface("en0"))
	assert.True(t, isPhysicalInterface("eth0"))
	assert.True(t, isPhysicalInterface("eno1"))
	assert.False(t, isPhysicalInterface("lo"))
	assert.False(t, isPhysicalInterface("docker0"))

	// Loopback should return negative score
	loIface := net.Interface{Name: "lo", Flags: net.FlagUp | net.FlagLoopback}
	assert.Equal(t, -1, scoreIPv4(net.ParseIP("127.0.0.1"), loIface))

	// Physical Wi-Fi on 192.168.1.x should have high score
	wlanIface := net.Interface{Name: "wlan0", Flags: net.FlagUp}
	wlanScore := scoreIPv4(net.ParseIP("192.168.1.19"), wlanIface)
	assert.Greater(t, wlanScore, 100)

	// Docker bridge should score lower than physical LAN
	dockerIface := net.Interface{Name: "docker0", Flags: net.FlagUp}
	dockerScore := scoreIPv4(net.ParseIP("172.17.0.1"), dockerIface)
	assert.Less(t, dockerScore, wlanScore)

	// Link-local address should score lower than standard LAN
	llScore := scoreIPv4(net.ParseIP("169.254.10.20"), wlanIface)
	assert.Less(t, llScore, wlanScore)
	assert.Greater(t, llScore, 0)
}

func TestResolveExternalHost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Direct LAN access should retain the client Host header
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Host = "192.168.1.100:8080"
	assert.Equal(t, "192.168.1.100", ResolveExternalHost(c))

	// Localhost access should resolve to the non-loopback local LAN IP
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request, _ = http.NewRequest("GET", "/", nil)
	c2.Request.Host = "localhost:8080"
	resolved := ResolveExternalHost(c2)
	assert.NotEmpty(t, resolved)
	assert.NotEqual(t, "localhost", resolved)
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
}
