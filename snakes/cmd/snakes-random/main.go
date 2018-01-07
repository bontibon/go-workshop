package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/bontibon/refresh-go-workshop/snakes"
)

func printState(s *snakes.State) {
	r := make([]rune, s.Height*s.Width)
	for i := range r {
		r[i] = '.'
	}

	for snakeNo, snake := range s.Snakes {
		if !snake.Alive {
			continue
		}
		for _, piece := range snake.Pieces {
			r[piece.Y*s.Width+piece.X] = rune('0' + snakeNo)
		}
	}

	for row := 0; row < s.Height; row++ {
		fmt.Fprintf(os.Stderr, "%s\n", string(r[s.Width*row:s.Width*(row+1)]))
	}
}

func clear() {
	for i := 0; i < 120; i++ {
		fmt.Println()
	}
}

func main() {
	width := flag.Int("width", 20, "board width")
	height := flag.Int("height", 20, "board height")
	snakeCount := flag.Int("snakes", 3, "snake count")
	seed := flag.Int64("seed", time.Now().Unix(), "random number seed")
	interval := flag.Duration("interval", time.Millisecond*150, "update interval")
	flag.Parse()

	cfg := snakes.StateConfig{
		Width:              *width,
		Height:             *height,
		SnakeCount:         *snakeCount,
		InitialSnakeLength: 5,
	}
	s := snakes.NewState(cfg)

	rng := rand.New(rand.NewSource(*seed))

	next := make([]snakes.Direction, *snakeCount)

	for {
		clear()

		for i := range next {
			next[i] = snakes.Direction(rng.Intn(int(snakes.DirectionWest) + 1))
		}
		s = s.Next(next)
		printState(s)
		if isCompleted, winner := s.IsCompleted(); isCompleted {
			fmt.Fprintf(os.Stderr, "game completed. winner: %d\n", winner)
			break
		}
		time.Sleep(*interval)
	}
}
