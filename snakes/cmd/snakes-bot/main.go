package main

import (
	"log"

	"github.com/bontibon/go-workshop/snakes"
)

func main() {
	// Server address
	const addr = "ws://127.0.0.1:8080/ws"

	// Bot name
	// The first character of your name will be displayed on you bot's head.
	// Emojis are supported! https://emojipedia.org
	// TODO: change me!
	const name = "BotName"

	bot, err := snakes.NewWebSocketBot(addr, name)
	if err != nil {
		log.Fatal(err)
	}
	defer bot.Close()
	log.Printf("Connected to the server; waiting for a new round")

	for round := range bot.Rounds() {
		log.Println("New round started")

		for turn := range round.Turns() {
			//
			//
			//
			// TODO: create your bot logic here!
			//
			//
			// Control your bot with turn.Move. Example:
			turn.Move(snakes.DirectionEast)
		}

		if winner := round.Winner(); winner != nil {
			log.Printf("%s won the round\n", *winner)
		} else {
			log.Println("Round over, and there was no winner")
		}
	}

	if err := bot.Err(); err != nil {
		log.Fatal(err)
	}
}
