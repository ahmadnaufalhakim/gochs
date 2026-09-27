package chess

import "testing"

func TestSquareString(t *testing.T) {
	tests := []struct {
		square Square
		want   string
	}{
		{a1, "a1"},
		{e4, "e4"},
		{h8, "h8"},
	}

	for _, test := range tests {
		if got := test.square.String(); got != test.want {
			t.Errorf("Square(%d).String() = %q, want %q", test.square, got, test.want)
		}
	}
}

func TestSquareMask(t *testing.T) {
	tests := []struct {
		square Square
		want   Bitboard
	}{
		{a1, 0x0000000000000001},
		{e4, 0x0000000010000000},
		{h8, 0x8000000000000000},
	}

	for _, test := range tests {
		if got := test.square.Mask(); got != test.want {
			t.Errorf("%s.Mask() = %#016x, want %#016x", test.square, got, test.want)
		}
	}
}

func TestParseCoordinate(t *testing.T) {
	tests := []struct {
		coordinate string
		want       Square
	}{
		{"a1", a1},
		{"e4", e4},
		{"h8", h8},
	}

	for _, test := range tests {
		got, err := ParseCoordinate(test.coordinate)
		if err != nil {
			t.Errorf("ParseCoordinate(%q) returned an error: %v", test.coordinate, err)
			continue
		}
		if got != test.want {
			t.Errorf("ParseCoordinate(%q) = %d, want %d", test.coordinate, got, test.want)
		}
	}
}

func TestParseCoordinateRejectsInvalidCoordinates(t *testing.T) {
	for _, coordinate := range []string{"", "a", "a0", "a9", "i1", "A1", "a11"} {
		if _, err := ParseCoordinate(coordinate); err == nil {
			t.Errorf("ParseCoordinate(%q) returned no error", coordinate)
		}
	}
}

func TestBoardReset(t *testing.T) {
	board := Board{WhitePawn: ^Bitboard(0)}
	got := board.Reset()

	if got != &board {
		t.Fatal("Reset() did not return the receiver")
	}

	want := Board{
		WhitePawn:   0x000000000000ff00,
		WhiteKnight: 0x0000000000000042,
		WhiteBishop: 0x0000000000000024,
		WhiteRook:   0x0000000000000081,
		WhiteQueen:  0x0000000000000008,
		WhiteKing:   0x0000000000000010,
		BlackPawn:   0x00ff000000000000,
		BlackKnight: 0x4200000000000000,
		BlackBishop: 0x2400000000000000,
		BlackRook:   0x8100000000000000,
		BlackQueen:  0x0800000000000000,
		BlackKing:   0x1000000000000000,
	}

	if board != want {
		t.Errorf("Reset() = %#v, want %#v", board, want)
	}
}
