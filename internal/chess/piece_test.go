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

func TestPieceTypeString(t *testing.T) {
	tests := []struct {
		pieceType PieceType
		want      string
	}{
		{Pawn, "pawn"},
		{Knight, "knight"},
		{Bishop, "bishop"},
		{Rook, "rook"},
		{Queen, "queen"},
		{King, "king"},
		{PieceNone, "unknown"},
		{PieceTypeCount, "unknown"},
	}

	for _, test := range tests {
		if got := test.pieceType.String(); got != test.want {
			t.Errorf("PieceType(%d).String() = %q, want %q", test.pieceType, got, test.want)
		}
	}
}

func TestPieceColorMethods(t *testing.T) {
	if White.Opponent() != Black || Black.Opponent() != White {
		t.Fatal("Opponent() did not return the opposing color")
	}

	tests := []struct {
		color PieceColor
		want  string
	}{
		{White, "White"},
		{Black, "Black"},
		{PieceColorCount, "White"},
	}

	for _, test := range tests {
		if got := test.color.String(); got != test.want {
			t.Errorf("PieceColor(%d).String() = %q, want %q", test.color, got, test.want)
		}
	}
}
