package snakes

import (
	"io"

	"github.com/gorilla/websocket"
)

type WebSocketViewer struct {
	c *websocket.Conn
}

func NewWebSocketViewer(conn *websocket.Conn) (*WebSocketViewer, error) {
	return &WebSocketViewer{
		c: conn,
	}, nil
}

var _ ViewerClient = (*WebSocketViewer)(nil)

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
