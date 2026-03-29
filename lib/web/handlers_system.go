package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/state"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func GetWiFiSSID() string {
	if runtime.GOOS == "darwin" {
		// macOS
		cmd := exec.Command("/System/Library/PrivateFrameworks/Apple80211.framework/Resources/airport", "-I")
		out, err := cmd.Output()
		if err != nil {
			return "N/A"
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "SSID: ") {
				return strings.TrimPrefix(line, "SSID: ")
			}
		}
	} else if runtime.GOOS == "linux" {
		// Linux
		cmd := exec.Command("nmcli", "-t", "-f", "active,ssid", "dev", "wifi")
		out, err := cmd.Output()
		if err != nil {
			return "N/A"
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "yes:") {
				return strings.TrimPrefix(line, "yes:")
			}
		}
	}
	return "N/A"
}

// @Summary Get system change log
// @Description Real-time SSE stream of state changes across the console
// @Tags System
// @Produce text/event-stream
// @Success 200 {object} string "SSE Stream"
// @Security CookieAuth
// @Security BasicAuth
// @Router /api/system/changelog [get]
func ChangeLogHandler(appState *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		ch := make(chan state.StateChange, 100)
		appState.BroadcastHub.Store(ch, true)
		defer appState.BroadcastHub.Delete(ch)

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
			return
		}

		for {
			select {
			case change, ok := <-ch:
				if !ok {
					return
				}
				payload, _ := json.Marshal(change)
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(payload))
				flusher.Flush()
			case <-c.Request.Context().Done():
				return
			case <-time.After(30 * time.Second):
				fmt.Fprintf(c.Writer, ": keep-alive\n\n")
				flusher.Flush()
			}
		}
	}
}

func GetSystemConnection(cfg *config.Config) gin.HandlerFunc {
	// @Summary Get system connection info
	// @Description Returns local IP and WiFi SSID
	// @Tags System
	// @Produce json
	// @Success 200 {object} object "Connection Status"
	// @Security CookieAuth
	// @Security BasicAuth
	// @Router /api/system/connection [get]
	return func(c *gin.Context) {
		status := struct {
			ServerURL string `json:"serverUrl"`
			SSID      string `json:"ssid"`
		}{
			ServerURL: fmt.Sprintf("http://%s:%s", GetLocalIP(), cfg.Port),
			SSID:      GetWiFiSSID(),
		}
		c.JSON(http.StatusOK, status)
	}
}
