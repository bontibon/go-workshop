package snakes

// Client is a client connected to a Server.
// A client controls a single snake in the game arena.
type Client interface {
	// ID returns a unique ID for the client. This ID must be unique
	// between all Client instances that are added to a server.
	ID() string

	// Direction returns the direction in which the client wishes their snake
	// to go. This function should return quickly.
	Direction() Direction

	ViewerClient
}

// ViewerClient is a client that is broadcast every message that the server
// broadcasts to regular clients.
// A ViewerClient does not control a snake in the arena.
type ViewerClient interface {
	// SendMessage sends the message to the client.
	SendMessage(*Message) error
}

// Message is the message that is broadcast to clients when the server's
// state changes.
//
// Message will only have one non-nil field.
type Message struct {
	WaitingMessage    *WaitingMessage          `json:"waiting,omitempty"`
	RoundPreparation  *RoundPreparationMessage `json:"round_preparation,omitempty"`
	RoundStateMessage *RoundStateMessage       `json:"round_state,omitempty"`
	RoundOverMessage  *RoundOverMessage        `json:"round_over,omitempty"`
}

// WaitingMessage is broadcast when the server is waiting for the minimum
// number of clients to connect before the round can start.
type WaitingMessage struct {
	CurrentPlayers  int `json:"current_players"`
	RequiredPlayers int `json:"required_players"`
}

// RoundPreparationMessage is broadcast when the minimum number of clients
// have connected to the server and the round is about to begin.
//
// A client may receive a WaitingMessage after a RoundPreparationMessage, if enough
// clients disconnect before the round starts.
type RoundPreparationMessage struct {
}

// RoundStateMessage is broadcast while the round is active. It contains a snapshot
// of the state of the game arena.
type RoundStateMessage struct {
	Width  int `json:"width"`
	Height int `json:"height"`

	Players []*RoundStateMessagePlayer `json:"players"`

	Apple Apple `json:"apple"`

	// Number of seconds remaining in the round. nil if there is no time limit.
	SecondsRemaining *int `json:"seconds_remaining,omitempty"`
}

// RoundStateMessagePlayer is a player in the round.
type RoundStateMessagePlayer struct {
	Name   string     `json:"name"`
	Pieces []Location `json:"pieces"`
}

// IsAt returns if any of the player's pieces is at the given location.
func (p *RoundStateMessagePlayer) IsAt(l Location) bool {
	for _, piece := range p.Pieces {
		if piece == l {
			return true
		}
	}
	return false
}

// roundStateMessageFromState creates a RoundStateMessage from the given clients
// and game state.
func roundStateMessageFromState(clients []Client, s *State) *RoundStateMessage {
	m := &RoundStateMessage{
		Width:  s.Width,
		Height: s.Height,

		Players: make([]*RoundStateMessagePlayer, len(clients)),

		Apple: s.Apple,
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

// RoundOverMessage is broadcast when the round is over.
// If there was no winner (e.g. all remaining snakes died at the same time), Winner
// will be nil.
type RoundOverMessage struct {
	Winner *string `json:"winner"`
}
