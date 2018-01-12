package snakes

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// Server is a snakes game server implementation.
//
// The server manages the game lifecycle from client queuing to broadcasting game state
// to connected clients.
//
// Clients are added to the server to take part in a game round.
// The servers waits until ServerConfig.MinimumClients are added, then waits
// ServerConfig.PreRoundWait, then, if ServerConfig.MinimumClients are still connected,
// starts the round.
//
// The server polls the round clients for their direction every ServerConfig.RoundTick,
// calculates the next game state, then broadcasts the state.
//
// Once the round is over, the winner (or lack of winner) is broadcast, the server waits
// ServerConfig.PostRoundWait, then the queue process is restarted.
type Server struct {
	config ServerConfig

	broadcastMu sync.Mutex
	viewers     []ViewerClient
	lastMessage *Message

	clientsMu sync.Mutex
	clients   []Client

	isStopped uint32
	stopped   chan struct{}

	clientsUpdated chan struct{}
}

// ServerConfig contains configuration variables for Server.
type ServerConfig struct {
	// Minimum number of clients needed to start a round.
	MinimumClients int
	// Amount of time to wait after enough clients joining and the beginning of
	// the round.
	PreRoundWait time.Duration
	// Amount of time to wait after a round before preparing for another round.
	PostRoundWait time.Duration
	// Duration of a round tick. This should be large enough for clients to
	// receive the current state, process it, then send a response.
	RoundTick time.Duration
}

// NewServer creates a new server with the given configuration.
func NewServer(config ServerConfig) *Server {
	return &Server{
		config:         config,
		clientsUpdated: make(chan struct{}, 1),
		stopped:        make(chan struct{}),
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

// Stop requests that the server stop after the current round.
func (s *Server) Stop() {
	if atomic.CompareAndSwapUint32(&s.isStopped, 0, 1) {
		close(s.stopped)
	}
}

// broadcast broadcasts msg too all of the server's viewers and the given clients.
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

// Run runs the game loop.
// The function returns after s.Stop is called.
func (s *Server) Run() {
	for {
		// Need at least two players connect to start the game
		s.clientsMu.Lock()
		s.clearClientsUpdated()

		if len(s.clients) < s.config.MinimumClients {
			s.broadcast(&Message{
				WaitingMessage: &WaitingMessage{
					CurrentPlayers:  len(s.clients),
					RequiredPlayers: s.config.MinimumClients,
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
		case <-time.After(s.config.PreRoundWait):
		case <-s.stopped:
			break
		}

		s.clientsMu.Lock()
		if len(s.clients) < s.config.MinimumClients {
			// A client left while waiting for the round to begin.
			s.clientsMu.Unlock()
			continue
		}
		roundClients := make([]Client, len(s.clients))
		copy(roundClients, s.clients)
		s.clientsMu.Unlock()

		// Shuffle clients so no one is consistently starting from the same location
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := 0; i < len(roundClients); i++ {
			idx := rng.Intn(len(roundClients))
			roundClients[i], roundClients[idx] = roundClients[idx], roundClients[i]
		}

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

		directions := make([]Direction, len(roundClients))
		ticker := time.NewTicker(s.config.RoundTick)
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
				time.Sleep(s.config.PostRoundWait)
				break
			}
		}
		ticker.Stop()
	}
}

// AddClient adds the client to the server.
// An error is returned if the client's name is not unique to the server.
func (s *Server) AddClient(c Client) error {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	id := c.ID()
	for _, client := range s.clients {
		if client.ID() == id {
			return errors.New("duplicate client ID")
		}
	}

	// send message in case we're in the middle of a round
	c.SendMessage(&Message{
		WaitingMessage: &WaitingMessage{
			CurrentPlayers:  1,
			RequiredPlayers: s.config.MinimumClients,
		},
	})

	s.clients = append(s.clients, c)
	s.signalClientsUpdated()
	return nil
}

// RemoveClient removes the client from the server.
// The client is not removed from the active round.
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

// AddViewer adds a viewer client to the server.
func (s *Server) AddViewer(v ViewerClient) error {
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()

	s.viewers = append(s.viewers, v)
	v.SendMessage(s.lastMessage)
	return nil
}

// RemoveViewer removes the given viewer from the server.
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
