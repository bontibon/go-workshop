package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"

	"github.com/bontibon/go-workshop/snakes"
)

func main() {
	// Server address
	const addr = "ws://131.162.18.57:8000/ws"

	// Bot name
	// The first character of your name will be displayed on you bot's head.
	// Emojis are supported! https://emojipedia.org
	// TODO: change me!
	name := "Bot-" + RandomName()

	bot, err := snakes.NewWebSocketBot(addr, name)
	if err != nil {
		log.Fatal(err)
	}
	defer bot.Close()
	log.Printf("Connected to the server as %s; waiting for a new round", name)

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
			// turn.Move(snakes.DirectionNorth)

			// Find your bot's location
			var loc snakes.Location
			for _, player := range turn.Players {
				if player.Name == name {
					loc = player.Pieces[0]
					break
				}
			}

			// Target the apple
			// TODO: adapt so you do not move your bot into a location where another player already is
			if loc.X < turn.Apple.X {
				turn.Move(snakes.DirectionEast)
			} else if loc.X > turn.Apple.X {
				turn.Move(snakes.DirectionWest)
			} else if loc.Y < turn.Apple.Y {
				turn.Move(snakes.DirectionSouth)
			} else {
				turn.Move(snakes.DirectionNorth)
			}
		}

		if winner, someoneWon := <-round.Winner(); someoneWon {
			log.Printf("%s won the round\n", winner)
		} else {
			log.Println("Round over, and there was no winner")
		}
	}

	if err := bot.Err(); err != nil {
		log.Fatal(err)
	}
}

func RandomName() string {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
