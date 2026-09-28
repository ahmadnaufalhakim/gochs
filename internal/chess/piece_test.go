package chess

import "testing"

func TestPieceLabel(t *testing.T) {
	tests := []struct {
		piece Piece
		want  string
	}{
		{Piece{Color: White, Type: PieceNone}, " "},
		{Piece{Color: White, Type: Pawn}, "♙"},
		{Piece{Color: White, Type: King}, "♔"},
		{Piece{Color: Black, Type: Pawn}, "♙"},
		{Piece{Color: Black, Type: King}, "♚"},
	}

	for _, test := range tests {
		if got := test.piece.Label(); got != test.want {
			t.Errorf("%v %v label = %q, want %q", test.piece.Color, test.piece.Type, got, test.want)
		}
	}
}
