package app

import (
	"fmt"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
)

func Run() {
	fmt.Println("Hello from gochs!")

	var r chess.Renderer
	r.Theme = chess.WOOD
	r.Perspective = chess.White

	var b chess.Board
	b.Reset()

	square, err := chess.ParseCoordinate("a3")
	if err != nil {
		panic(err)
	}
	b.SetPieceAt(square, chess.Piece{
		Color: chess.White,
		Type:  chess.Pawn,
	})
	// b.SetPieceAt()
	// b.ColorToMove = chess.Black

	r.Print(b)
}
