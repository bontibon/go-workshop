package snakes

import (
	"encoding"
	"encoding/binary"
	"errors"
	"hash/crc64"
	"math/rand"
)

// Direction is a direction in which a snake can be moved.
type Direction int

var (
	_ encoding.TextMarshaler   = (*Direction)(nil)
	_ encoding.TextUnmarshaler = (*Direction)(nil)
)

// Valid directions.
const (
	DirectionNorth Direction = iota
	DirectionEast
	DirectionSouth
	DirectionWest
)

func (d Direction) MarshalText() ([]byte, error) {
	switch d {
	case DirectionNorth:
		return []byte("north"), nil
	case DirectionEast:
		return []byte("east"), nil
	case DirectionSouth:
		return []byte("south"), nil
	case DirectionWest:
		return []byte("west"), nil
	}
	return nil, errors.New("invalid direction")
}

func (d *Direction) UnmarshalText(b []byte) error {
	switch string(b) {
	case "north":
		*d = DirectionNorth
	case "east":
		*d = DirectionEast
	case "south":
		*d = DirectionSouth
	case "west":
		*d = DirectionWest
	default:
		return errors.New("invalid direction")
	}
	return nil
}

// Location represents a 2D location.
type Location struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// IsInsideBounds returns if the location is inside of the given width and height.
func (l Location) IsInsideBounds(width, height int) bool {
	return l.X >= 0 && l.X < width && l.Y >= 0 && l.Y < height
}

// NextLocation returns the next location moving in the given direction.
// The function panics on an invalid direction.
func NextLocation(base Location, direction Direction) Location {
	switch direction {
	case DirectionNorth:
		base.Y--
	case DirectionEast:
		base.X++
	case DirectionSouth:
		base.Y++
	case DirectionWest:
		base.X--
	default:
		panic("unknown direction")
	}
	return base
}

// locationPair is a pair of two locations.
type locationPair [2]Location

// Swap returns a copy of the location pair with the items swapped.
func (l locationPair) Swap() locationPair {
	return locationPair{l[1], l[0]}
}

type Snake struct {
	Alive  bool
	Length int
	// Pieces[0] is the head of the snake
	Pieces []Location
}

func (s *Snake) HasPieceAt(x, y int) bool {
	for _, piece := range s.Pieces {
		if piece.X == x && piece.Y == y {
			return true
		}
	}
	return false
}

type Apple struct {
	Location `json:"location"`
}

type StateConfig struct {
	Width, Height      int
	SnakeCount         int
	InitialSnakeLength int
}

type State struct {
	Width, Height int
	Snakes        []*Snake
	Apple         *Apple
}

func NewState(cfg StateConfig) *State {
	if cfg.SnakeCount < 2 {
		panic("snakeCount < 2")
	}

	s := &State{
		Width:  cfg.Width,
		Height: cfg.Height,

		Snakes: make([]*Snake, cfg.SnakeCount),
	}

	if cfg.SnakeCount > s.Width {
		panic("snakeCount > s.Width")
	}

	xSpaceBetween := s.Width / cfg.SnakeCount
	xInitialSpace := xSpaceBetween / 2

	yLine := s.Height / 2

	for i := range s.Snakes {
		s.Snakes[i] = &Snake{
			Alive:  true,
			Length: cfg.InitialSnakeLength,
		}
		s.Snakes[i].Pieces = make([]Location, 1, s.Snakes[i].Length)
		s.Snakes[i].Pieces[0] = Location{
			X: xInitialSpace + xSpaceBetween*i,
			Y: yLine,
		}
	}

	s.Apple = &Apple{
		Location: GenerateAppleLocation(s.Width, s.Height, s.Snakes),
	}

	return s
}

func GenerateAppleLocation(width, height int, snakes []*Snake) Location {
	h := crc64.New(crc64.MakeTable(crc64.ISO))

	for _, snake := range snakes {
		var b [16]byte
		var x, y uint64
		if snake.Alive {
			x, y = uint64(snake.Pieces[0].X), uint64(snake.Pieces[0].Y)
		}
		binary.BigEndian.PutUint64(b[:8], x)
		binary.BigEndian.PutUint64(b[8:], y)
		h.Write(b[:])
	}

	rng := rand.New(rand.NewSource(int64(h.Sum64())))
	for {
		x := rng.Intn(width)
		y := rng.Intn(height)

		ok := true
		for _, snake := range snakes {
			if snake.HasPieceAt(x, y) {
				ok = false
				break
			}
		}

		if ok {
			return Location{
				X: x,
				Y: y,
			}
		}
	}
}

func (s *State) clone() (newState *State, maxLength int) {
	newState = &State{
		Width:  s.Width,
		Height: s.Height,

		Snakes: make([]*Snake, len(s.Snakes)),
	}

	if s.Apple != nil {
		newState.Apple = &Apple{
			Location: s.Apple.Location,
		}
	}

	for i, snake := range s.Snakes {
		newState.Snakes[i] = &Snake{
			Alive:  snake.Alive,
			Length: snake.Length,
			Pieces: make([]Location, len(snake.Pieces), cap(snake.Pieces)),
		}
		if snake.Length > maxLength {
			maxLength = snake.Length
		}
		copy(newState.Snakes[i].Pieces, snake.Pieces)
	}

	return
}

func (s *State) Next(snakeDirections []Direction) *State {
	if len(snakeDirections) != len(s.Snakes) {
		panic("len(snakeDirections) != len(s.Snakes)")
	}

	next, maxLength := s.clone()

	tails := make(map[Location]int, len(next.Snakes)*maxLength)
	headLocations := make(map[locationPair]int, len(next.Snakes))
	nextHeadLocations := make(map[Location]int, len(next.Snakes))
	repositionApple := false

	for snakeNo, snake := range next.Snakes {
		if !snake.Alive {
			continue
		}
		nextLocation := NextLocation(snake.Pieces[0], snakeDirections[snakeNo])
		if nextLocation == next.Apple.Location {
			snake.Length++
			repositionApple = true
		}
		if snake.Length > len(snake.Pieces) {
			snake.Pieces = append(snake.Pieces, Location{})
		}
		for i := len(snake.Pieces) - 1; i >= 1; i-- {
			snake.Pieces[i] = snake.Pieces[i-1]
			tails[snake.Pieces[i]] = snakeNo
		}
		locPair := locationPair{snake.Pieces[0], nextLocation}
		snake.Pieces[0] = nextLocation
		if !nextLocation.IsInsideBounds(next.Width, next.Height) {
			// collided with wall
			snake.Alive = false
		} else if otherSnakeNo, ok := nextHeadLocations[nextLocation]; ok {
			// two snakes tried to go to the same location
			snake.Alive = false
			next.Snakes[otherSnakeNo].Alive = false
		} else if otherSnakeNo, ok := headLocations[locPair.Swap()]; ok {
			// two snake heads "swapped" locations
			snake.Alive = false
			next.Snakes[otherSnakeNo].Alive = false
		} else {
			headLocations[locPair] = snakeNo
			nextHeadLocations[nextLocation] = snakeNo
		}
	}

	// Tail collisions
	for loc, snakeNo := range nextHeadLocations {
		if _, ok := tails[loc]; ok {
			next.Snakes[snakeNo].Alive = false
		}
	}

	if repositionApple {
		next.Apple.Location = GenerateAppleLocation(next.Width, next.Height, next.Snakes)
	}

	return next
}

// IsComplete returns if the game is completed and which snake number is the winner.
// -1 is returned as the snake winner if no snakes are left alive.
func (s *State) IsCompleted() (bool, int) {
	alive := -1
	nonAlive := true
	for snakeNo, snake := range s.Snakes {
		if snake.Alive {
			if alive >= 0 {
				return false, 0
			}
			nonAlive = false
			alive = snakeNo
		}
	}

	return alive >= 0 || nonAlive, alive
}
