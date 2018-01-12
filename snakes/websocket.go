package snakes

// ClientMessage is a message sent from a WebSocketClient to
// a WebSocket server.
type ClientMessage struct {
	DirectionClientMessage *DirectionClientMessage `json:"direction"`
}

// DirectionClientMessage contains the direction that the client
// wishes to move their snake on the game board.
type DirectionClientMessage struct {
	Direction Direction `json:"direction"`
}
