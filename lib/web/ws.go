package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/state"
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewWSHandler(appState *state.AppState, cfg *config.Config) gin.HandlerFunc {
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
		fmt.Printf("[CLIENT] New admin connected (ID: %p, Session: %s).\n", wsClient, sessionID)

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
					fmt.Printf("[CLIENT] Admin disconnected (ID: %p). Session lock released.\n", wsClient)
					wsClient.Close()
					return
				}
			}
		}()
	}
}
