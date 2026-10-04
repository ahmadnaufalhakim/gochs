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
	board := Board{Pieces: [PieceColorCount][PieceTypeCount]Bitboard{
		White: {Pawn: ^Bitboard(0)},
	}}
	board.Reset()

	want := Board{
		Pieces: [PieceColorCount][PieceTypeCount]Bitboard{
			White: {
				Pawn:   0x000000000000ff00,
				Knight: 0x0000000000000042,
				Bishop: 0x0000000000000024,
				Rook:   0x0000000000000081,
				Queen:  0x0000000000000008,
				King:   0x0000000000000010,
			},
			Black: {
				Pawn:   0x00ff000000000000,
				Knight: 0x4200000000000000,
				Bishop: 0x2400000000000000,
				Rook:   0x8100000000000000,
				Queen:  0x0800000000000000,
				King:   0x1000000000000000,
			},
		},
		ColorToMove: White,
	}

	if board != want {
		t.Errorf("Reset() = %#v, want %#v", board, want)
	}
}

func TestSquareProperties(t *testing.T) {
	for square := a1; square <= h8; square++ {
		coordinate := square.String()
		parsed, err := ParseCoordinate(coordinate)
		if err != nil {
			t.Errorf("ParseCoordinate(%q) returned an error: %v", coordinate, err)
			continue
		}
		if parsed != square {
			t.Errorf("ParseCoordinate(%q) = %s, want %s", coordinate, parsed, square)
		}
		if square.File() > 7 || square.Rank() > 7 {
			t.Errorf("%s has invalid file/rank %d/%d", square, square.File(), square.Rank())
		}
		if square.IsDark() == square.IsLight() {
			t.Errorf("%s must be either dark or light", square)
		}
	}
}

func TestBoardMutationAndQueries(t *testing.T) {
	var board Board
	board.SetPieceAt(c3, Piece{Color: White, Type: Knight})
	board.SetPieceAt(c6, Piece{Color: Black, Type: Rook})

	if got := board.PieceAt(c3); got != (Piece{Color: White, Type: Knight}) {
		t.Errorf("PieceAt(c3) = %#v, want white knight", got)
	}
	if !board.IsSquareOccupied(c6) || !board.IsSquareOccupiedByColor(c6, Black) {
		t.Error("c6 should be occupied by Black")
	}
	if !board.IsSquareOccupiedByPiece(c3, Piece{Color: White, Type: Knight}) {
		t.Error("c3 should be occupied by a white knight")
	}
	if board.IsSquareOccupiedByPieceType(c3, Rook) {
		t.Error("c3 should not be occupied by a rook")
	}

	wantOccupied := c3.Mask() | c6.Mask()
	if got := board.Occupied(); got != wantOccupied {
		t.Errorf("Occupied() = %#x, want %#x", got, wantOccupied)
	}
	if got := board.OccupiedByColor(White); got != c3.Mask() {
		t.Errorf("OccupiedByColor(White) = %#x, want %#x", got, c3.Mask())
	}
	if got := board.OccupiedByPieceType(Rook); got != c6.Mask() {
		t.Errorf("OccupiedByPieceType(Rook) = %#x, want %#x", got, c6.Mask())
	}

	board.SetPieceAt(c3, Piece{Color: Black, Type: Queen})
	if got := board.PieceAt(c3); got != (Piece{Color: Black, Type: Queen}) {
		t.Errorf("SetPieceAt() replacement = %#v, want black queen", got)
	}
	board.ClearSquare(c3)
	if got := board.PieceAt(c3); got.Type != PieceNone {
		t.Errorf("ClearSquare(c3) = %#v, want empty square", got)
	}

	board.EnPassantTarget = e3.Mask()
	board.Clear()
	if board.Occupied() != 0 || board.EnPassantTarget != 0 {
		t.Errorf("Clear() left state behind: %#v", board)
	}
}

func TestBoardValidate(t *testing.T) {
	t.Run("initial position", func(t *testing.T) {
		var board Board
		board.Reset()
		if err := board.Validate(); err != nil {
			t.Errorf("Validate() returned an error: %v", err)
		}
	})

	t.Run("pawn on promotion rank", func(t *testing.T) {
		board := boardWithKings()
		board.SetPieceAt(a8, Piece{Color: White, Type: Pawn})
		if err := board.Validate(); err == nil {
			t.Fatal("Validate() returned no error for a pawn on rank 8")
		}
	})

	t.Run("side not to move in check", func(t *testing.T) {
		board := boardWithKings()
		board.SetPieceAt(e7, Piece{Color: White, Type: Rook})
		if err := board.Validate(); err == nil {
			t.Fatal("Validate() returned no error when Black is in check before White moves")
		}
	})

	t.Run("valid en passant target", func(t *testing.T) {
		board := boardWithKings()
		board.SetPieceAt(e4, Piece{Color: White, Type: Pawn})
		board.ColorToMove = Black
		board.EnPassantTarget = e3.Mask()
		if err := board.Validate(); err != nil {
			t.Errorf("Validate() returned an error: %v", err)
		}
	})

	t.Run("invalid en passant target", func(t *testing.T) {
		board := boardWithKings()
		board.EnPassantTarget = e3.Mask() | e6.Mask()
		if err := board.Validate(); err == nil {
			t.Fatal("Validate() returned no error for a multi-square en-passant target")
		}
	})

	t.Run("occupied en passant source", func(t *testing.T) {
		board := boardWithKings()
		board.SetPieceAt(e4, Piece{Color: White, Type: Pawn})
		board.SetPieceAt(e2, Piece{Color: White, Type: Knight})
		board.ColorToMove = Black
		board.EnPassantTarget = e3.Mask()
		if err := board.Validate(); err == nil {
			t.Fatal("Validate() returned no error for an occupied double-push source")
		}
	})
}
