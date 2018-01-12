package snakes

import (
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

// WebSocketClient is a WebSocket based Client.
type WebSocketClient struct {
	c    *websocket.Conn
	name string

	direction int32
}

var _ Client = (*WebSocketClient)(nil)

func validBotName(name string) bool {
	if len(name) == 0 {
		return false
	}
	if !utf8.ValidString(name) {
		return false
	}
	if utf8.RuneCountInString(name) > 16 {
		return false
	}
	return true
}

// NewWebSocketClient creates a new WebSocketClient from the given WebSocket connection.
//
// nil and an error is returned if the HTTP request does not contain
// a valid name in the X-Snake-Name header.
func NewWebSocketClient(conn *websocket.Conn, r *http.Request) (*WebSocketClient, error) {
	snakeName := r.Header.Get("X-Snake-Name")
	if !validBotName(snakeName) {
		return nil, errors.New("invalid snake name")
	}

	c := &WebSocketClient{
		c:    conn,
		name: snakeName,
	}

	return c, nil
}

// Run continuously reads client messages from the WebSocket connection.
// An error is returned on the first error reading from the connection.
func (s *WebSocketClient) Run() error {
	for {
		var msg ClientMessage
		err := s.c.ReadJSON(&msg)
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}

		switch {
		case msg.DirectionClientMessage != nil:
			atomic.StoreInt32(&s.direction, int32(msg.DirectionClientMessage.Direction))
		default:
			return errors.New("invalid client message")
		}
	}

	return nil
}

// ID returns the client's name as provided by the X-Snake-Name header
// when the WebSocket connection was established.
func (s *WebSocketClient) ID() string {
	return s.name
}

// Direction returns the direction in which the client wishes to move their
// snake.
func (s *WebSocketClient) Direction() Direction {
	return Direction(atomic.LoadInt32(&s.direction))
}

// SendMessage sends the message to the client.
func (s *WebSocketClient) SendMessage(msg *Message) error {
	return s.c.WriteJSON(msg)
}
