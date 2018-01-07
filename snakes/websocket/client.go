package websocket

import (
	"github.com/bontibon/refresh-go-workshop/snakes"
)

type DirectionClientMessage struct {
	Direction snakes.Direction `json:"direction"`
}

type ClientMessage struct {
	DirectionClientMessage *DirectionClientMessage `json:"direction"`
}
