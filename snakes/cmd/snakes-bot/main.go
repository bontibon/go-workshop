package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/bontibon/refresh-go-workshop/snakes"
	snakeswebsocket "github.com/bontibon/refresh-go-workshop/snakes/websocket"
	"github.com/gorilla/websocket"
)

func SendDirection(c *websocket.Conn, direction snakes.Direction) error {
	msg := snakeswebsocket.ClientMessage{
		DirectionClientMessage: &snakeswebsocket.DirectionClientMessage{
			Direction: direction,
		},
	}
	return c.WriteJSON(&msg)
}

func main() {
	defaultName, _ := os.Hostname()

	addr := flag.String("addr", "ws://127.0.0.1:8080/ws", "server address")
	name := flag.String("name", defaultName, "bot name")
	flag.Parse()

	var dialer websocket.Dialer

	headers := make(http.Header)
	headers.Set("X-Snake-Name", *name)

	conn, _, err := dialer.Dial(*addr, headers)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	log.Println("Connected to the server")
	for {
		var msg snakes.Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("read JSON error: %s", err)
			break
		}

		switch {
		case msg.WaitingMessage != nil:
			log.Printf("Waiting for round to start: %#v\n", *msg.WaitingMessage)
		case msg.RoundPreparation != nil:
			log.Printf("Round starting soon")
		case msg.RoundStateMessage != nil:
			log.Printf("Round state updated")
			SendDirection(conn, snakes.DirectionEast)
		case msg.RoundOverMessage != nil:
			log.Printf("Round is over: %v", (*msg.RoundOverMessage).Winner)
		default:
			log.Printf("unknown message")
		}
	}
	log.Println("Disonnected")
}
