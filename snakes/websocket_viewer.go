package snakes

import (
	"io"

	"github.com/gorilla/websocket"
)

type ViewerConn struct {
	c *websocket.Conn
}

func NewViewerConn(conn *websocket.Conn) (*ViewerConn, error) {
	return &ViewerConn{
		c: conn,
	}, nil
}

func (v *ViewerConn) Run() error {
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
func (v *ViewerConn) SendMessage(msg *Message) error {
	return v.c.WriteJSON(msg)
}
