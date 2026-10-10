package chess

import "testing"

func requireUCI(t *testing.T, board *Board, uci string) Move {
	t.Helper()
	for _, move := range GenerateLegalMoves(*board) {
		if move.UCI() == uci {
			return move
		}
	}
	t.Fatalf("could not find legal UCI move %q", uci)
	return Move(0)
}

func requireSAN(t *testing.T, board *Board, uci, want string) {
	t.Helper()
	move := requireUCI(t, board, uci)
	got, err := board.SAN(move)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("SAN(%s) = %q, want %q", uci, got, want)
	}
	requireMakeMove(t, board, move)
}

func TestSAN(t *testing.T) {
	t.Run("ordinary moves, capture, and en passant", func(t *testing.T) {
		var board Board
		board.Reset()
		requireSAN(t, &board, "e2e4", "e4")
		requireSAN(t, &board, "a7a6", "a6")
		requireSAN(t, &board, "e4e5", "e5")
		requireSAN(t, &board, "d7d5", "d5")
		requireSAN(t, &board, "e5d6", "exd6")
	})

	t.Run("disambiguation", func(t *testing.T) {
		board := Board{ColorToMove: White, FullmoveNumber: 1}
		board.SetPieceAt(square(t, "a1"), Piece{Color: White, Type: King})
		board.SetPieceAt(square(t, "h8"), Piece{Color: Black, Type: King})
		board.SetPieceAt(square(t, "b1"), Piece{Color: White, Type: Knight})
		board.SetPieceAt(square(t, "d1"), Piece{Color: White, Type: Knight})
		requireSAN(t, &board, "b1c3", "Nbc3")
	})

	t.Run("castling", func(t *testing.T) {
		board := Board{ColorToMove: White, FullmoveNumber: 1}
		board.SetPieceAt(square(t, "e1"), Piece{Color: White, Type: King})
		board.SetPieceAt(square(t, "h1"), Piece{Color: White, Type: Rook})
		board.SetPieceAt(square(t, "e8"), Piece{Color: Black, Type: King})
		board.CastlingRights[White][KingSide] = CastlingRight{KingFrom: square(t, "e1"), RookFrom: square(t, "h1"), Available: true}
		requireSAN(t, &board, "e1g1", "O-O")
	})

	t.Run("checkmate", func(t *testing.T) {
		var board Board
		board.Reset()
		requireSAN(t, &board, "f2f3", "f3")
		requireSAN(t, &board, "e7e5", "e5")
		requireSAN(t, &board, "g2g4", "g4")
		requireSAN(t, &board, "d8h4", "Qh4#")
	})
}
