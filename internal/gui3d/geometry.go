package gui3d

import (
	"math"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
)

const SquareSize float32 = 1

type Vec3 struct {
	X float32
	Y float32
	Z float32
}

func SquareCenter(square chess.Square) Vec3 {
	return Vec3{
		// White's perspective should match a standard board diagram: a1 is at
		// the lower-left and h1 at the lower-right.
		X: (3.5 - float32(square.File())) * SquareSize,
		Y: 0,
		Z: (float32(square.Rank()) - 3.5) * SquareSize,
	}
}

func SquareAtWorld(position Vec3) (chess.Square, bool) {
	file := int(math.Floor(float64(4 - position.X/SquareSize)))
	rank := int(math.Floor(float64(position.Z/SquareSize + 4)))
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return 0, false
	}
	square, err := chess.ParseCoordinate(string(rune('a'+file)) + string(rune('1'+rank)))
	return square, err == nil
}

func CameraPosition(color chess.PieceColor) Vec3 {
	if color == chess.Black {
		return Vec3{X: 0, Y: 5.8, Z: 11.5}
	}
	return Vec3{X: 0, Y: 5.8, Z: -11.5}
}
