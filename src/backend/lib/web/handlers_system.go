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

// isVirtualInterface returns true if the interface name indicates a virtual, container, or tunnel interface.
func isVirtualInterface(name string) bool {
	name = strings.ToLower(name)
	virtualPrefixes := []string{
		"docker", "br-", "veth", "virbr", "dummy", // Linux containers / bridges
		"tun", "tap", "tailscale", "wg", // VPNs & tunnels
		"utun", "awdl", "llw", "bridge", "gif", "stf", // macOS virtual & internal
		"vmnet", "vboxnet", // Virtualization
	}
	for _, p := range virtualPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// isPhysicalInterface returns true if the interface name matches common physical LAN/WLAN interfaces.
func isPhysicalInterface(name string) bool {
	name = strings.ToLower(name)
	physicalPrefixes := []string{
		"en", "eth", "wlan", "wl", "eno", "ens", "enp",
	}
	for _, p := range physicalPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// scoreIPv4 calculates a preference score for an IPv4 candidate address.
// Higher scores represent better candidates for mobile/LAN client connectivity.
// Negative scores represent unusable addresses (e.g., loopback, unspecified).
func scoreIPv4(ip net.IP, iface net.Interface) int {
	ip4 := ip.To4()
	if ip4 == nil {
		return -1
	}
	if ip4.IsLoopback() || ip4[0] == 127 || ip4.IsUnspecified() || ip4[0] == 0 {
		return -1
	}

	isUp := (iface.Flags & net.FlagUp) != 0
	isPhys := isPhysicalInterface(iface.Name)
	isVirt := isVirtualInterface(iface.Name)
	isLinkLocal := ip4.IsLinkLocalUnicast() || (ip4[0] == 169 && ip4[1] == 254)

	baseScore := 0
	if isPhys && !isVirt {
		baseScore += 50
	} else if !isVirt {
		baseScore += 30
	}

	if isUp {
		baseScore += 20
	}

	// Prefer standard private subnets commonly used for routers and LANs
	if ip4[0] == 192 && ip4[1] == 168 {
		return baseScore + 40
	}
	if ip4[0] == 10 {
		return baseScore + 35
	}
	if ip4[0] == 172 && (ip4[1] >= 16 && ip4[1] <= 31) && !isVirt {
		return baseScore + 30
	}
	if !isLinkLocal {
		return baseScore + 20
	}
	// Link-local fallback (still preferred over loopback)
	return baseScore + 5
}

// GetLocalIP returns the machine's primary non-loopback IPv4 address on the local network.
// It prioritizes physical LAN interfaces and avoids loopback (127.0.0.1) as much as possible.
func GetLocalIP() string {
	// 1. First attempt: Ask OS routing table for the local outbound address to well-known targets
	for _, target := range []string{"8.8.8.8:80", "1.1.1.1:80"} {
		conn, err := net.DialTimeout("udp", target, 200*time.Millisecond)
		if err == nil {
			localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
			conn.Close()
			if ok && localAddr.IP != nil {
				ip4 := localAddr.IP.To4()
				if ip4 != nil && !ip4.IsLoopback() && ip4[0] != 127 && !(ip4[0] == 169 && ip4[1] == 254) {
					return ip4.String()
				}
			}
		}
	}

	// 2. Second attempt: Rank all available network interfaces to find the best physical LAN IPv4
	ifaces, err := net.Interfaces()
	if err == nil {
		bestIP := ""
		bestScore := -1

		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 {
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
				score := scoreIPv4(ip, iface)
				if score > bestScore {
					bestScore = score
					bestIP = ip.To4().String()
				}
			}
		}

		if bestIP != "" {
			return bestIP
		}
	}

	// 3. Absolute last resort: loopback address only if no non-loopback interface exists
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

		notify := c.Request.Context().Done()

		for {
			select {
			case <-notify:
				return
			case change, ok := <-ch:
				if !ok {
					return
				}
				data, err := json.Marshal(change)
				if err != nil {
					continue
				}
				fmt.Fprintf(c.Writer, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}

// SystemConnectionResponse provides connection details for clients.
type SystemConnectionResponse struct {
	ServerURL       string `json:"serverUrl"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	DisplayEndpoint string `json:"displayEndpoint"`
}

// ResolveExternalHost determines the host address to display to clients.
func ResolveExternalHost(c *gin.Context) string {
	if c != nil && c.Request != nil && c.Request.Host != "" {
		host := c.Request.Host
		if strings.Contains(host, ":") {
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
		}
		// If client reached us via a non-loopback host (e.g. 192.168.x.x or church.local), use it
		if host != "" && host != "localhost" && host != "127.0.0.1" && host != "::1" && !strings.HasPrefix(host, "127.") {
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
