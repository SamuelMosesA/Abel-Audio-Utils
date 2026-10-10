package state

import (
	"sync"

	"github.com/gorilla/websocket"
)

// WSClient wraps a websocket connection with a mutex for thread-safe writes.
type WSClient struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
	Type string // "admin" or "listener"
}

func (c *WSClient) WriteJSON(v interface{}) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return c.Conn.WriteJSON(v)
}

func (c *WSClient) WriteMessage(messageType int, data []byte) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return c.Conn.WriteMessage(messageType, data)
}

func (c *WSClient) Close() error {
	return c.Conn.Close()
}
