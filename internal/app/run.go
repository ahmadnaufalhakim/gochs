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
	b.ColorToMove = chess.White

	inpPairs := []struct {
		coord     string
		color     chess.PieceColor
		pieceType chess.PieceType
	}{
		{coord: "a3", color: chess.White, pieceType: chess.Pawn},
		{coord: "b2", color: chess.White, pieceType: chess.Pawn},
		{coord: "b3", color: chess.Black, pieceType: chess.Knight},
		{coord: "c2", color: chess.White, pieceType: chess.Pawn},
		{coord: "d4", color: chess.Black, pieceType: chess.Knight},
		{coord: "e2", color: chess.Black, pieceType: chess.Pawn},

		{coord: "e5", color: chess.White, pieceType: chess.Pawn},
		{coord: "f1", color: chess.White, pieceType: chess.Rook},
		{coord: "f5", color: chess.White, pieceType: chess.King},
		{coord: "g2", color: chess.White, pieceType: chess.Pawn},
		{coord: "g3", color: chess.White, pieceType: chess.Pawn},
		{coord: "h2", color: chess.White, pieceType: chess.Pawn},

		{coord: "e7", color: chess.Black, pieceType: chess.King},

		{coord: "a7", color: chess.White, pieceType: chess.Pawn},
		{coord: "b8", color: chess.Black, pieceType: chess.Queen},
	}
	// inpPairs := []struct {
	// 	coord     string
	// 	color     chess.PieceColor
	// 	pieceType chess.PieceType
	// }{
	// 	{coord: "c4", color: chess.Black, pieceType: chess.King},
	// 	{coord: "c5", color: chess.Black, pieceType: chess.Pawn},
	// }

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

	r.Print(b)

	fmt.Println(chess.GenerateLegalMoves(b))
	err := b.MakeMove(chess.NewMove(chess.Square(37), chess.Square(28), chess.QuietMove))
	fmt.Println(err)
	r.Print(b)
	fmt.Println(b.Validate())
	fmt.Println(b.IsColorInCheck(b.ColorToMove))
	fmt.Println(b.ColorToMove.String())

	fmt.Println(chess.GenerateLegalMoves(b))
	err = b.MakeMove(chess.NewMove(chess.Square(17), chess.Square(34), chess.QuietMove))
	fmt.Println(err)
	r.Print(b)
	fmt.Println(b.Validate())
	fmt.Println(b.IsColorInCheck(b.ColorToMove))
	fmt.Println(b.ColorToMove.String())

	fmt.Println(chess.GenerateLegalMoves(b))
	err = b.MakeMove(chess.NewMove(chess.Square(28), chess.Square(35), chess.QuietMove))
	fmt.Println(err)
	r.Print(b)
	fmt.Println(b.Validate())
	fmt.Println(b.IsColorInCheck(b.ColorToMove))
	fmt.Println(b.ColorToMove.String())

	fmt.Println(chess.GenerateLegalMoves(b))
	err = b.MakeMove(chess.NewMove(chess.Square(52), chess.Square(45), chess.QuietMove))
	fmt.Println(err)
	r.Print(b)
	fmt.Println(b.Validate())
	fmt.Println(b.IsColorInCheck(b.ColorToMove))
	fmt.Println(b.ColorToMove.String())
}
