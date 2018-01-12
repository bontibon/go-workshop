package snakes

type DirectionClientMessage struct {
	Direction Direction `json:"direction"`
}

type ClientMessage struct {
	DirectionClientMessage *DirectionClientMessage `json:"direction"`
}
