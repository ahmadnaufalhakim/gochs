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
		CastlingRights: defaultStartingCastlingRights,
		ColorToMove:    White,
		FullmoveNumber: 1,
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

	t.Run("valid Chess960 castling right", func(t *testing.T) {
		var board Board
		board.SetPieceAt(f1, Piece{Color: White, Type: King})
		board.SetPieceAt(h1, Piece{Color: White, Type: Rook})
		board.SetPieceAt(e8, Piece{Color: Black, Type: King})
		board.CastlingRights[White][KingSide] = CastlingRight{
			KingFrom:  f1,
			RookFrom:  h1,
			Available: true,
		}
		board.FullmoveNumber = 1
		if err := board.Validate(); err != nil {
			t.Errorf("Validate() returned an error: %v", err)
		}
	})

	t.Run("castling right without rook", func(t *testing.T) {
		board := boardWithKings()
		board.CastlingRights[White][KingSide] = defaultStartingCastlingRights[White][KingSide]
		if err := board.Validate(); err == nil {
			t.Fatal("Validate() returned no error for a castling right without its rook")
		}
	})

	t.Run("castling right with rook on wrong side", func(t *testing.T) {
		board := boardWithKings()
		board.SetPieceAt(d1, Piece{Color: White, Type: Rook})
		board.CastlingRights[White][KingSide] = CastlingRight{
			KingFrom:  e1,
			RookFrom:  d1,
			Available: true,
		}
		if err := board.Validate(); err == nil {
			t.Fatal("Validate() returned no error for a king-side rook left of its king")
		}
	})

	t.Run("overlapping pieces", func(t *testing.T) {
		board := boardWithKings()
		board.Pieces[White][Queen] = e1.Mask()
		if err := board.Validate(); err == nil {
			t.Fatal("Validate() returned no error for overlapping pieces")
		}
	})

	t.Run("invalid color to move", func(t *testing.T) {
		board := boardWithKings()
		board.ColorToMove = PieceColorCount
		if err := board.Validate(); err == nil {
			t.Fatal("Validate() returned no error for an invalid color to move")
		}
	})
}

func TestBoardFEN(t *testing.T) {
	tests := []struct {
		name  string
		board Board
		want  string
	}{
		{
			name: "initial position",
			board: func() Board {
				var board Board
				board.Reset()
				return board
			}(),
			want: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name: "position state",
			board: func() Board {
				board := boardWithKings()
				board.ColorToMove = White
				board.SetPieceAt(h6, Piece{Color: Black, Type: Knight})
				board.SetPieceAt(d5, Piece{Color: Black, Type: Pawn})
				board.SetPieceAt(e5, Piece{Color: White, Type: Pawn})
				board.EnPassantTarget = d6.Mask()
				board.HalfmoveClock = 17
				board.FullmoveNumber = 42
				return board
			}(),
			want: "4k3/8/7n/3pP3/8/8/8/4K3 w - d6 17 42",
		},
		{
			name: "Chess960 castling rights",
			board: func() Board {
				board := castlingBoard(White, KingSide, f1, h1)
				board.FullmoveNumber = 1
				return board
			}(),
			want: "4k3/8/8/8/8/8/8/5K1R w H - 0 1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.board.FEN()
			if err != nil {
				t.Fatalf("FEN() returned an error: %v", err)
			}
			if got != test.want {
				t.Errorf("FEN() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBoardFENRejectsInvalidState(t *testing.T) {
	tests := []struct {
		name  string
		board Board
	}{
		{
			name:  "missing kings",
			board: Board{FullmoveNumber: 1},
		},
		{
			name: "multiple en-passant targets",
			board: func() Board {
				board := boardWithKings()
				board.EnPassantTarget = e3.Mask() | e6.Mask()
				return board
			}(),
		},
		{
			name: "zero fullmove number",
			board: func() Board {
				board := boardWithKings()
				board.FullmoveNumber = 0
				return board
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.board.FEN(); err == nil {
				t.Fatal("FEN() returned no error")
			}
		})
	}

}
