package snakes

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type WaitingMessage struct {
	CurrentPlayers  int `json:"current_players"`
	RequiredPlayers int `json:"required_players"`
}

type RoundPreparationMessage struct {
}

type RoundStateMessagePlayer struct {
	Name   string     `json:"name"`
	Pieces []Location `json:"pieces"`
}

type RoundStateMessage struct {
	Width  int `json:"width"`
	Height int `json:"height"`

	Players []*RoundStateMessagePlayer `json:"players"`
}

func roundStateMessageFromState(clients []Client, s *State) *RoundStateMessage {
	m := &RoundStateMessage{
		Width:  s.Width,
		Height: s.Height,

		Players: make([]*RoundStateMessagePlayer, len(clients)),
	}

	for i, client := range clients {
		p := &RoundStateMessagePlayer{
			Name: client.ID(),
		}
		if snake := s.Snakes[i]; snake.Alive {
			p.Pieces = make([]Location, len(snake.Pieces))
			copy(p.Pieces, snake.Pieces)
		}
		m.Players[i] = p
	}

	return m
}

type RoundOverMessage struct {
	Winner *int `json:"winner"`
}

type Message struct {
	// Waiting for enough players to connect
	WaitingMessage *WaitingMessage `json:"waiting,omitempty"`
	// Round preparation
	RoundPreparation *RoundPreparationMessage `json:"round_preparation,omitempty"`
	// Complete state of the game round
	RoundStateMessage *RoundStateMessage `json:"round_state,omitempty"`
	// Round has been completed, with an optional winner
	RoundOverMessage *RoundOverMessage `json:"round_over,omitempty"`
}

type ViewerClient interface {
	// SendMessage sends the message to the client.
	SendMessage(*Message) error
}

type Client interface {
	// ID returns a unique ID for the client. This ID must be unique
	// between all Client instances that are added to a server.
	ID() string

	// Direction returns the direction in which the client wishes to go.
	// This function must return quickly, or else the server timing.
	Direction() Direction

	ViewerClient
}

type Server struct {
	broadcastMu sync.Mutex
	viewers     []ViewerClient
	lastMessage *Message

	clientsMu sync.Mutex
	clients   []Client

	isStopped uint32
	stopped   chan struct{}

	clientsUpdated chan struct{}

	// Time to wait before and in-between games for clients to reconnect
	queueWait time.Duration

	tickInterval time.Duration
}

func NewServer() *Server {
	return &Server{
		clientsUpdated: make(chan struct{}, 1),

		stopped: make(chan struct{}),

		queueWait: time.Second * 1,

		tickInterval: time.Millisecond * 250,
	}
}

func (s *Server) signalClientsUpdated() {
	select {
	case s.clientsUpdated <- struct{}{}:
	default:
	}
}

func (s *Server) clearClientsUpdated() {
	select {
	case <-s.clientsUpdated:
	default:
	}
}

func (s *Server) Stop() {
	if atomic.CompareAndSwapUint32(&s.isStopped, 0, 1) {
		close(s.stopped)
	}
}

func (s *Server) broadcast(msg *Message, clients ...Client) {
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()

	s.lastMessage = msg

	for _, viewer := range s.viewers {
		viewer.SendMessage(msg)
	}

	for _, client := range clients {
		client.SendMessage(msg)
	}
}

// Blocks the caller.
func (s *Server) Run() {
	const minClients = 2

	for {
		// Need at least two players connect to start the game
		s.clientsMu.Lock()
		s.clearClientsUpdated()

		if len(s.clients) < minClients {
			s.broadcast(&Message{
				WaitingMessage: &WaitingMessage{
					CurrentPlayers:  len(s.clients),
					RequiredPlayers: minClients,
				},
			}, s.clients...)
			s.clientsMu.Unlock()

			select {
			case <-s.clientsUpdated:
				continue
			case <-s.stopped:
				break
			}
		}

		// At least two players have connected. Now wait for any other
		// potential players to join.
		s.broadcast(&Message{
			RoundPreparation: &RoundPreparationMessage{},
		}, s.clients...)
		s.clientsMu.Unlock()

		select {
		case <-time.After(s.queueWait):
		case <-s.stopped:
			break
		}

		s.clientsMu.Lock()
		if len(s.clients) < minClients {
			s.clientsMu.Unlock()
			continue
		}
		roundClients := make([]Client, len(s.clients))
		copy(roundClients, s.clients)
		s.clientsMu.Unlock()

		const size = 50 // TODO: base off of client count

		cfg := StateConfig{
			Width:              size,
			Height:             size / 2,
			SnakeCount:         len(roundClients),
			InitialSnakeLength: 5,
		}
		gameState := NewState(cfg)
		s.broadcast(&Message{
			RoundStateMessage: roundStateMessageFromState(roundClients, gameState),
		}, roundClients...)

		// TODO: "randomly" spawn food (would be cool if this was deterministic)

		directions := make([]Direction, len(roundClients))
		ticker := time.NewTicker(s.tickInterval)
		for {
			<-ticker.C

			for i, client := range roundClients {
				directions[i] = client.Direction()
			}

			gameState = gameState.Next(directions)
			s.broadcast(&Message{
				RoundStateMessage: roundStateMessageFromState(roundClients, gameState),
			}, roundClients...)

			if completed, winner := gameState.IsCompleted(); completed {
				rom := &RoundOverMessage{}
				if winner >= 0 {
					rom.Winner = new(int)
					*rom.Winner = winner
				}
				s.broadcast(&Message{
					RoundOverMessage: rom,
				}, roundClients...)
				time.Sleep(s.queueWait)
				break
			}
		}
		ticker.Stop()
	}
}

func (s *Server) AddClient(c Client) error {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	id := c.ID()
	for _, client := range s.clients {
		if client.ID() == id {
			return errors.New("duplicate client ID")
		}
	}

	s.clients = append(s.clients, c)
	s.signalClientsUpdated()
	return nil
}

func (s *Server) RemoveClient(c Client) bool {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	id := c.ID()

	for i, client := range s.clients {
		if client.ID() == id {
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
			s.signalClientsUpdated()
			return true
		}
	}
	return false
}

func (s *Server) AddViewer(v ViewerClient) error {
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()

	s.viewers = append(s.viewers, v)
	v.SendMessage(s.lastMessage)
	return nil
}

func (s *Server) RemoveViewer(v ViewerClient) error {
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()

	for i, viewer := range s.viewers {
		if v == viewer {
			s.viewers = append(s.viewers[:i], s.viewers[i+1:]...)
			break
		}
	}

	return nil
}
