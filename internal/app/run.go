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
	b.Clear()

	inpPairs := []struct {
		coord     string
		color     chess.PieceColor
		pieceType chess.PieceType
	}{
		{coord: "a3", color: chess.White, pieceType: chess.Pawn},
		{coord: "b2", color: chess.White, pieceType: chess.Pawn},
		{coord: "b3", color: chess.White, pieceType: chess.Knight},
		{coord: "c2", color: chess.White, pieceType: chess.Pawn},
		{coord: "e2", color: chess.White, pieceType: chess.Pawn},
		{coord: "e5", color: chess.White, pieceType: chess.Pawn},
		{coord: "g2", color: chess.White, pieceType: chess.Pawn},
		{coord: "g3", color: chess.White, pieceType: chess.Pawn},
		{coord: "h2", color: chess.White, pieceType: chess.Pawn},
	}
	for _, pair := range inpPairs {
		square, err := chess.ParseCoordinate(pair.coord)
		if err != nil {
			panic(err)
		}
		b.SetPieceAt(square, chess.Piece{
			Color: pair.color,
			Type:  pair.pieceType,
		})
	}

	// b.SetPieceAt()
	// b.ColorToMove = chess.Black

	r.Print(b)

	fmt.Println("Validating the board..")
	fmt.Println(b.Validate())
	pawnMoves := chess.GeneratePawnMoves(b)
	pawnMoves.Print()
}
