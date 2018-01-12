package snakes

import (
	"io"

	"github.com/gorilla/websocket"
)

// WebSocketViewer is a WebSocket based ViewerClient implementation.
type WebSocketViewer struct {
	c *websocket.Conn
}

// NewWebSocketViewer creates a new WebSocketViewer around the given
// WebSocket connection.
func NewWebSocketViewer(conn *websocket.Conn) *WebSocketViewer {
	return &WebSocketViewer{
		c: conn,
	}
}

var _ ViewerClient = (*WebSocketViewer)(nil)

// Run keeps the underlying WebSocket connection alive.
// It returns when the underlying WebSocket connection closes.
func (v *WebSocketViewer) Run() error {
	for {
		var val interface{}

		// Keep connection alive
		if err := v.c.ReadJSON(&val); err != nil {
			if err == io.ErrUnexpectedEOF {
				err = nil
			}
			return err
		}
	}
}

// SendMessage sends the message to the client.
func (v *WebSocketViewer) SendMessage(msg *Message) error {
	return v.c.WriteJSON(msg)
}
