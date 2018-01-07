package websocket

import (
	"errors"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"unicode/utf8"

	"github.com/bontibon/refresh-go-workshop/snakes"
	"github.com/gorilla/websocket"
)

type ServerConn struct {
	c    *websocket.Conn
	name string

	direction int32
}

var _ snakes.Client = (*ServerConn)(nil)

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

func NewServerConn(conn *websocket.Conn, r *http.Request) (*ServerConn, error) {
	snakeName := r.Header.Get("X-Snake-Name")
	if !validBotName(snakeName) {
		return nil, errors.New("invalid snake name")
	}

	c := &ServerConn{
		c:    conn,
		name: snakeName,
	}

	return c, nil
}

func (s *ServerConn) Run() error {
	for {
		var msg ClientMessage
		err := s.c.ReadJSON(&msg)
		if err != nil {
			if err != io.ErrUnexpectedEOF {
				log.Println(err)
			}
			break
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

func (s *ServerConn) ID() string {
	return s.name
}

func (s *ServerConn) Direction() snakes.Direction {
	return snakes.Direction(atomic.LoadInt32(&s.direction))
}

// SendMessage sends the message to the client.
func (s *ServerConn) SendMessage(msg *snakes.Message) error {
	return s.c.WriteJSON(msg)
}
