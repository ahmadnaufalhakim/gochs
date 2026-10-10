package chess

import "testing"

func TestPositionKey(t *testing.T) {
	t.Run("ignores move counters", func(t *testing.T) {
		var board Board
		board.Reset()
		key := board.PositionKey()
		board.HalfmoveClock = 42
		board.FullmoveNumber = 99
		if got := board.PositionKey(); got != key {
			t.Errorf("PositionKey() = %#x, want %#x", got, key)
		}
	})

	t.Run("includes side to move and castling rights", func(t *testing.T) {
		var board Board
		board.Reset()
		key := board.PositionKey()

		board.ColorToMove = Black
		if got := board.PositionKey(); got == key {
			t.Error("PositionKey() did not include side to move")
		}

		board.Reset()
		board.CastlingRights[White][KingSide].Available = false
		if got := board.PositionKey(); got == key {
			t.Error("PositionKey() did not include castling rights")
		}
	})

	t.Run("includes normalized en passant rights", func(t *testing.T) {
		board := Board{ColorToMove: Black, FullmoveNumber: 1}
		board.SetPieceAt(e1, Piece{Color: White, Type: King})
		board.SetPieceAt(e8, Piece{Color: Black, Type: King})
		board.SetPieceAt(e4, Piece{Color: White, Type: Pawn})
		board.SetPieceAt(d4, Piece{Color: Black, Type: Pawn})
		board.EnPassantTarget = e3.Mask()

		withoutEnPassant := board
		withoutEnPassant.EnPassantTarget = 0
		if board.PositionKey() == withoutEnPassant.PositionKey() {
			t.Error("PositionKey() did not include a usable en passant right")
		}
	})
}

func TestHasInsufficientMaterial(t *testing.T) {
	tests := []struct {
		name   string
		pieces []struct {
			square Square
			piece  Piece
		}
		want bool
	}{
		{name: "kings only", want: true},
		{
			name: "single bishop",
			pieces: []struct {
				square Square
				piece  Piece
			}{{c1, Piece{Color: White, Type: Bishop}}},
			want: true,
		},
		{
			name: "single knight",
			pieces: []struct {
				square Square
				piece  Piece
			}{{b1, Piece{Color: White, Type: Knight}}},
			want: true,
		},
		{
			name: "same colored bishops",
			pieces: []struct {
				square Square
				piece  Piece
			}{
				{c1, Piece{Color: White, Type: Bishop}},
				{f8, Piece{Color: Black, Type: Bishop}},
			},
			want: true,
		},
		{
			name: "opposite colored bishops",
			pieces: []struct {
				square Square
				piece  Piece
			}{
				{c1, Piece{Color: White, Type: Bishop}},
				{c8, Piece{Color: Black, Type: Bishop}},
			},
			want: false,
		},
		{
			name: "two knights",
			pieces: []struct {
				square Square
				piece  Piece
			}{
				{b1, Piece{Color: White, Type: Knight}},
				{g1, Piece{Color: White, Type: Knight}},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			board := Board{ColorToMove: White, FullmoveNumber: 1}
			board.SetPieceAt(e1, Piece{Color: White, Type: King})
			board.SetPieceAt(e8, Piece{Color: Black, Type: King})
			for _, placement := range test.pieces {
				board.SetPieceAt(placement.square, placement.piece)
			}
			if got := board.HasInsufficientMaterial(); got != test.want {
				t.Errorf("HasInsufficientMaterial() = %t, want %t", got, test.want)
			}
		})
	}
}
