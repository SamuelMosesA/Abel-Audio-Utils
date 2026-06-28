package web

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"abel/src/backend/lib/telemetry"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewWSHandler(appState *state.AppState, cfg *config.Config) gin.HandlerFunc {
	logger := slog.With("component", "client")
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionID, _ := session.Get("session_id").(string)
		auth := session.Get("authenticated")

		if auth != true {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized session"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		wsClient := &state.WSClient{Conn: conn, Type: "admin"}
		appState.Clients.Store(wsClient, true)

		// Set the active admin client (multiple can now connect, but let's keep track of 'the' admin if needed)
		appState.AdminClient = wsClient
		clientLogger := logger.With(
			slog.String("client.id", fmt.Sprintf("%p", wsClient)),
			slog.String("client.session_id", sessionID),
		)
		clientLogger.Info("New admin connected")

		// Listen for client messages (mostly for disconnect)
		go func() {
			for {
				_, _, err := wsClient.Conn.ReadMessage()
				if err != nil {
					// Client disconnected
					appState.Clients.Delete(wsClient)
					if appState.AdminClient == wsClient {
						appState.AdminClient = nil
					}
					clientLogger.Info("Admin disconnected, session lock released")
					wsClient.Close()
					if telemetry.DroppedConnections != nil {
						telemetry.DroppedConnections.Add(context.Background(), 1)
					}
					return
				}
			}
		}()
	}
}
