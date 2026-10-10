package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GetLocalIP returns the machine's primary non-loopback IPv4 address on the local network.
func GetLocalIP() string {
	// 1. Ask OS routing table for the local address used for outbound traffic
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if udpAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok && !udpAddr.IP.IsLoopback() {
			if ip4 := udpAddr.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}

	// 2. Fallback: inspect network interface addresses for first non-loopback IPv4
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ip4 := ipNet.IP.To4(); ip4 != nil {
					return ip4.String()
				}
			}
		}
	}

	return "127.0.0.1"
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

// SystemConnectionResponse describes the externally reachable server connection endpoints.
type SystemConnectionResponse struct {
	ServerURL       string `json:"serverUrl"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	DisplayEndpoint string `json:"displayEndpoint"`
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
		})
	}
}
