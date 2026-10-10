package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
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

// GetLocalIP attempts to determine the primary outbound IP address of the host machine.
func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if udpAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok && !udpAddr.IP.IsLoopback() {
			return udpAddr.IP.String()
		}
	}

	// Fallback: scan local non-loopback network interfaces
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
					return ip.String()
				}
			}
		}
	}

	return "127.0.0.1"
}

// ParseDarwinSSID extracts the Wi-Fi SSID from macOS ipconfig getsummary output.
func ParseDarwinSSID(out string) string {
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "SSID : ") {
			parts := strings.SplitN(trimmed, "SSID : ", 2)
			if len(parts) == 2 {
				ssid := strings.TrimSpace(parts[1])
				if ssid != "" {
					return ssid
				}
			}
		}
	}
	return ""
}

// GetWiFiSSID queries the active Wi-Fi SSID using platform utilities.
// On macOS: runs `ipconfig getsummary en0` with en1 fallback and airport fallback.
// On Linux: runs `nmcli -t -f active,ssid dev wifi`.
func GetWiFiSSID() string {
	if runtime.GOOS == "darwin" {
		// Try ipconfig getsummary on en0 (standard Wi-Fi interface)
		for _, iface := range []string{"en0", "en1"} {
			cmd := exec.Command("ipconfig", "getsummary", iface)
			if out, err := cmd.Output(); err == nil {
				if ssid := ParseDarwinSSID(string(out)); ssid != "" {
					return ssid
				}
			}
		}

		// Fallback to airport command if present
		cmd := exec.Command("/System/Library/PrivateFrameworks/Apple80211.framework/Resources/airport", "-I")
		if out, err := cmd.Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "SSID: ") {
					return strings.TrimPrefix(line, "SSID: ")
				}
			}
		}
	} else if runtime.GOOS == "linux" {
		cmd := exec.Command("nmcli", "-t", "-f", "active,ssid", "dev", "wifi")
		if out, err := cmd.Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "yes:") {
					return strings.TrimPrefix(line, "yes:")
				}
			}
		}
	}
	return "N/A"
}

// ChangeLogHandler streams real-time SSE updates to connected clients.
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

// SystemConnectionResponse describes the externally reachable server connection endpoints and Wi-Fi SSID.
type SystemConnectionResponse struct {
	ServerURL       string `json:"serverUrl"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	DisplayEndpoint string `json:"displayEndpoint"`
	SSID            string `json:"ssid"`
}

// ResolveExternalHost returns the most suitable public/LAN IP or hostname for external clients.
func ResolveExternalHost(c *gin.Context) string {
	if c != nil && c.Request != nil && c.Request.Host != "" {
		host := c.Request.Host
		if strings.Contains(host, ":") {
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
		}
		// If client reached us via a non-loopback host (e.g. 192.168.x.x or church.local), use it
		if host != "" && host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return host
		}
	}
	return GetLocalIP()
}

// GetSystemConnection returns connection details for mobile and remote clients.
func GetSystemConnection(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		port := "8080"
		if cfg != nil && cfg.Port != "" {
			port = cfg.Port
		}

		host := ResolveExternalHost(c)
		displayEndpoint := fmt.Sprintf("%s:%s", host, port)
		serverURL := fmt.Sprintf("http://%s", displayEndpoint)

		c.JSON(http.StatusOK, SystemConnectionResponse{
			ServerURL:       serverURL,
			Host:            host,
			Port:            port,
			DisplayEndpoint: displayEndpoint,
			SSID:            GetWiFiSSID(),
		})
	}
}
