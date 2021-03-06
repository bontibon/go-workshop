package snakes

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// WebSocketBot is the client-side for the WebSocket server.
// It provides a helpful abstraction over the game rounds and turns.
type WebSocketBot struct {
	c    *websocket.Conn
	name string

	rounds chan *BotRound

	mu  sync.Mutex
	err error
}

// NewWebSocketBot establishes a new bot connection to the given server address
// and uses botName as the bot's identifier.
//
// nil and an error is returned if there was a problem establishing the connection.
func NewWebSocketBot(addr, botName string) (*WebSocketBot, error) {
	var dialer websocket.Dialer

	headers := make(http.Header)
	headers.Set("X-Snake-Name", botName)

	conn, _, err := dialer.Dial(addr, headers)
	if err != nil {
		return nil, err
	}

	bot := &WebSocketBot{
		c:    conn,
		name: botName,

		rounds: make(chan *BotRound),
	}
	go bot.reader()

	return bot, nil
}

// reader is the background reader for the bot. It is spawn from NewWebSocketBot and is alive
// until the connection is closed.
func (w *WebSocketBot) reader() {
	defer close(w.rounds)
	var currentRound *BotRound

	for {
		var msg Message
		if err := w.c.ReadJSON(&msg); err != nil {
			if err != io.ErrUnexpectedEOF {
				w.mu.Lock()
				if currentRound != nil {
					close(currentRound.winner)
					if !currentRound.died {
						close(currentRound.turns)
					}
				}
				w.err = err
				w.mu.Unlock()
			}
			break
		}

		switch {
		case msg.WaitingMessage != nil:
		case msg.RoundPreparation != nil:
		case msg.RoundStateMessage != nil:
			if currentRound == nil {
				currentRound = &BotRound{
					w: w,

					turns:  make(chan *BotTurn),
					winner: make(chan string, 1),
				}
				w.rounds <- currentRound
			}
			if currentRound.died {
				break
			}

			for _, player := range msg.RoundStateMessage.Players {
				if player.Name == w.name {
					if len(player.Pieces) > 0 {
						currentRound.turns <- &BotTurn{
							RoundStateMessage: msg.RoundStateMessage,

							r: currentRound,
						}
					} else {
						currentRound.died = true
						close(currentRound.turns)
					}
					break
				}
			}
		case msg.RoundOverMessage != nil:
			if msg.RoundOverMessage.Winner != nil {
				currentRound.winner <- *msg.RoundOverMessage.Winner
			}
			close(currentRound.winner)
			if !currentRound.died {
				close(currentRound.turns)
			}
			currentRound = nil
		default:
		}
	}
}

// Close closes the connection to the server.
func (w *WebSocketBot) Close() error {
	return w.c.Close()
}

// Err returns the error that caused the connection to close.
func (w *WebSocketBot) Err() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}

// Rounds returns a channel of BotRounds. A BotRound is sent on the channel when a
// new rounds begins.
//
// The returned channel is closed when the connection to the server is closed.
func (w *WebSocketBot) Rounds() <-chan *BotRound {
	return w.rounds
}

// BotRound represents a game round that the bot is participating in.
type BotRound struct {
	w *WebSocketBot

	died   bool
	turns  chan *BotTurn
	winner chan string
}

// Turns returns a channel of *BotTurns. A BotTurn is sent on the channel when the
// next tick of the game is received from the server.
//
// The returned channel is closed when the round is over, your snake dies, or the
// connection to the server is closed.
func (b *BotRound) Turns() <-chan *BotTurn {
	return b.turns
}

// Winner returns a channel on which the round winner will be sent. The channel is closed
// without a name sent if no player won the round (i.e. the remaining players died at
// the same time).
func (b *BotRound) Winner() <-chan string {
	return b.winner
}

// BotTurn represents a turn in a BotRound. It contains a snapshot of the current
// game arena state.
type BotTurn struct {
	*RoundStateMessage
	r     *BotRound
	moved int32
}

// Move tells the server to move your bot in the given direction at the end of
// the turn. Calling this function multiple times per turn will have no effect.
func (t *BotTurn) Move(direction Direction) error {
	if !atomic.CompareAndSwapInt32(&t.moved, 0, 1) {
		return errors.New("already moved")
	}

	msg := ClientMessage{
		DirectionClientMessage: &DirectionClientMessage{
			Direction: direction,
		},
	}
	return t.r.w.c.WriteJSON(&msg)
}
