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
		{coord: "b3", color: chess.Black, pieceType: chess.Knight},
		{coord: "c2", color: chess.White, pieceType: chess.Pawn},
		{coord: "d4", color: chess.Black, pieceType: chess.Knight},
		{coord: "e2", color: chess.White, pieceType: chess.Pawn},
		{coord: "e5", color: chess.White, pieceType: chess.Pawn},
		{coord: "g2", color: chess.White, pieceType: chess.Pawn},
		{coord: "g3", color: chess.White, pieceType: chess.Pawn},
		{coord: "h2", color: chess.White, pieceType: chess.Pawn},
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
	b.ColorToMove = chess.White

	r.Print(b)

	fmt.Println("Validating the board..")
	fmt.Println(b.Validate())
	fmt.Println("pawn moves:")
	pawnMoves := chess.GeneratePawnMoves(b)
	pawnMoves.Print()
	fmt.Println("pawn capture moves:")
	pawnCaptureMoves := chess.GeneratePawnCaptureMoves(b)
	pawnCaptureMoves.Print()
	fmt.Println("knight moves:")
	knightMoves := chess.GenerateKnightMoves(b)
	knightMoves.Print()
	fmt.Println("bishop moves:")
	bishopMoves := chess.GenerateBishopMoves(b)
	bishopMoves.Print()
	fmt.Println("rook moves:")
	rookMoves := chess.GenerateRookMoves(b)
	rookMoves.Print()
	fmt.Println("queen moves:")
	queenMoves := chess.GenerateQueenMoves(b)
	queenMoves.Print()
	fmt.Println("king moves:")
	kingMoves := chess.GenerateKingMoves(b)
	kingMoves.Print()
}
