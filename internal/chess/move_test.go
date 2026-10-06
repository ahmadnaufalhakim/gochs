package chess

import (
	"slices"
	"testing"
)

func square(t *testing.T, coordinate string) Square {
	t.Helper()

	square, err := ParseCoordinate(coordinate)
	if err != nil {
		t.Fatalf("ParseCoordinate(%q) returned an error: %v", coordinate, err)
	}

	return square
}

func boardWithKings() Board {
	var board Board
	board.SetPieceAt(e1, Piece{Color: White, Type: King})
	board.SetPieceAt(e8, Piece{Color: Black, Type: King})
	board.ColorToMove = White
	return board
}

func movesContain(moves []Move, want Move) bool {
	return slices.Contains(moves, want)
}

func requireMove(t *testing.T, moves []Move, want Move) {
	t.Helper()
	if !movesContain(moves, want) {
		t.Fatalf("moves %v do not contain %v", moves, want)
	}
}

func requireNoMove(t *testing.T, moves []Move, unwanted Move) {
	t.Helper()
	if movesContain(moves, unwanted) {
		t.Fatalf("moves %v unexpectedly contain %v", moves, unwanted)
	}
}

func requireMakeMove(t *testing.T, board *Board, move Move) {
	t.Helper()
	if err := board.MakeMove(move); err != nil {
		t.Fatalf("MakeMove(%v) returned an error: %v", move, err)
	}
}

func TestMoveEncodingAndString(t *testing.T) {
	from := square(t, "e2")
	to := square(t, "e4")

	tests := []struct {
		flag MoveFlag
		want string
	}{
		{KingSideCastle, "e2-e4 (O-O)"},
		{QueenSideCastle, "e2-e4 (O-O-O)"},
		{QuietMove, "e2-e4"},
		{Capture, "e2xe4"},
		{DoublePawnPush, "e2-e4 (double push)"},
		{EnPassant, "e2xe4 e.p."},
		{PromoteKnight, "e2-e4=N"},
		{PromoteBishop, "e2-e4=B"},
		{PromoteRook, "e2-e4=R"},
		{PromoteQueen, "e2-e4=Q"},
		{PromoteCaptureKnight, "e2xe4=N"},
		{PromoteCaptureBishop, "e2xe4=B"},
		{PromoteCaptureRook, "e2xe4=R"},
		{PromoteCaptureQueen, "e2xe4=Q"},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			move := NewMove(from, to, test.flag)
			if got := move.From(); got != from {
				t.Errorf("From() = %s, want %s", got, from)
			}
			if got := move.To(); got != to {
				t.Errorf("To() = %s, want %s", got, to)
			}
			if got := move.Flag(); got != test.flag {
				t.Errorf("Flag() = %d, want %d", got, test.flag)
			}
			if got := move.String(); got != test.want {
				t.Errorf("String() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestMovePrint(t *testing.T) {
	move := NewMove(e2, e4, DoublePawnPush)
	if got, want := captureStdout(t, move.Print), "e2-e4 (double push)\n"; got != want {
		t.Errorf("Print() = %q, want %q", got, want)
	}
}

func TestPawnPseudoLegalMoves(t *testing.T) {
	t.Run("white push, capture, and promotion", func(t *testing.T) {
		var board Board
		board.SetPieceAt(a1, Piece{Color: White, Type: King})
		board.SetPieceAt(a8, Piece{Color: Black, Type: King})
		board.ColorToMove = White
		board.SetPieceAt(e2, Piece{Color: White, Type: Pawn})
		board.SetPieceAt(d3, Piece{Color: Black, Type: Knight})
		board.SetPieceAt(e7, Piece{Color: White, Type: Pawn})
		board.SetPieceAt(f8, Piece{Color: Black, Type: Rook})

		moves := GeneratePawnPseudoLegalMoves(board, White)
		requireMove(t, moves, NewMove(e2, e3, QuietMove))
		requireMove(t, moves, NewMove(e2, e4, DoublePawnPush))
		requireMove(t, moves, NewMove(e2, d3, Capture))
		for _, flag := range []MoveFlag{PromoteKnight, PromoteBishop, PromoteRook, PromoteQueen} {
			requireMove(t, moves, NewMove(e7, e8, flag))
		}
		for _, flag := range []MoveFlag{PromoteCaptureKnight, PromoteCaptureBishop, PromoteCaptureRook, PromoteCaptureQueen} {
			requireMove(t, moves, NewMove(e7, f8, flag))
		}
	})

	t.Run("blocked pawn cannot push but can capture", func(t *testing.T) {
		board := boardWithKings()
		board.SetPieceAt(e2, Piece{Color: White, Type: Pawn})
		board.SetPieceAt(e3, Piece{Color: Black, Type: Knight})
		board.SetPieceAt(d3, Piece{Color: Black, Type: Bishop})

		moves := GeneratePawnPseudoLegalMoves(board, White)
		requireNoMove(t, moves, NewMove(e2, e3, QuietMove))
		requireNoMove(t, moves, NewMove(e2, e4, DoublePawnPush))
		requireMove(t, moves, NewMove(e2, d3, Capture))
	})

	t.Run("black direction", func(t *testing.T) {
		board := boardWithKings()
		board.SetPieceAt(e7, Piece{Color: Black, Type: Pawn})
		board.SetPieceAt(f6, Piece{Color: White, Type: Bishop})

		moves := GeneratePawnPseudoLegalMoves(board, Black)
		requireMove(t, moves, NewMove(e7, e6, QuietMove))
		requireMove(t, moves, NewMove(e7, e5, DoublePawnPush))
		requireMove(t, moves, NewMove(e7, f6, Capture))
	})
}

func TestPawnAttacksDoNotWrapFiles(t *testing.T) {
	board := boardWithKings()
	board.SetPieceAt(a2, Piece{Color: White, Type: Pawn})
	board.SetPieceAt(h7, Piece{Color: Black, Type: Pawn})

	if got, want := GeneratePawnAttacks(board, White), b3.Mask(); got != want {
		t.Errorf("white pawn attacks = %#x, want %#x", got, want)
	}
	if got, want := GeneratePawnAttacks(board, Black), g6.Mask(); got != want {
		t.Errorf("black pawn attacks = %#x, want %#x", got, want)
	}
}

func TestJumpAndSlidingPseudoLegalMoves(t *testing.T) {
	board := boardWithKings()
	board.SetPieceAt(d4, Piece{Color: White, Type: Knight})
	board.SetPieceAt(f5, Piece{Color: White, Type: Pawn})
	board.SetPieceAt(c2, Piece{Color: Black, Type: Bishop})

	knightMoves := GenerateKnightPseudoLegalMoves(board, White)
	requireMove(t, knightMoves, NewMove(d4, c2, Capture))
	requireNoMove(t, knightMoves, NewMove(d4, f5, QuietMove))

	board.Clear()
	board.SetPieceAt(e1, Piece{Color: White, Type: King})
	board.SetPieceAt(e8, Piece{Color: Black, Type: King})
	board.SetPieceAt(d4, Piece{Color: White, Type: Rook})
	board.SetPieceAt(d6, Piece{Color: Black, Type: Knight})
	board.SetPieceAt(d2, Piece{Color: White, Type: Pawn})

	rookMoves := GenerateRookPseudoLegalMoves(board, White)
	requireMove(t, rookMoves, NewMove(d4, d6, Capture))
	requireNoMove(t, rookMoves, NewMove(d4, d7, QuietMove))
	requireNoMove(t, rookMoves, NewMove(d4, d2, QuietMove))
}

func TestAttackAndCheckDetection(t *testing.T) {
	tests := []struct {
		name       string
		whiteKing  Square
		blackPiece Piece
		square     Square
	}{
		{"pawn", e4, Piece{Color: Black, Type: Pawn}, d5},
		{"knight", e1, Piece{Color: Black, Type: Knight}, f3},
		{"bishop", e1, Piece{Color: Black, Type: Bishop}, b4},
		{"rook", e1, Piece{Color: Black, Type: Rook}, e8},
		{"queen", e1, Piece{Color: Black, Type: Queen}, e7},
		{"king", e1, Piece{Color: Black, Type: King}, e2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var board Board
			board.SetPieceAt(test.whiteKing, Piece{Color: White, Type: King})
			board.SetPieceAt(a8, Piece{Color: Black, Type: King})
			board.SetPieceAt(test.square, test.blackPiece)
			if !board.IsColorInCheck(White) {
				t.Fatal("White is not reported in check")
			}
		})
	}

	board := boardWithKings()
	board.SetPieceAt(e8, Piece{Color: Black, Type: Rook})
	board.SetPieceAt(e4, Piece{Color: White, Type: Pawn})
	if board.IsColorInCheck(White) {
		t.Fatal("blocked rook incorrectly checks White")
	}
}

func TestGenerateLegalMovesFiltersKingSafety(t *testing.T) {
	board := boardWithKings()
	board.SetPieceAt(e2, Piece{Color: White, Type: Rook})
	board.SetPieceAt(e8, Piece{Color: Black, Type: Rook})

	pseudo := GeneratePseudoLegalMoves(board, White)
	requireMove(t, pseudo, NewMove(e2, d2, QuietMove))

	legal := GenerateLegalMoves(board)
	requireNoMove(t, legal, NewMove(e2, d2, QuietMove))
	requireMove(t, legal, NewMove(e2, e8, Capture))

	before := board
	_ = GenerateLegalMoves(board)
	if board != before {
		t.Fatal("GenerateLegalMoves mutated its input board")
	}
}

func TestMakeMoveHandlesDoublePushPromotionAndEnPassant(t *testing.T) {
	t.Run("double push creates target", func(t *testing.T) {
		var board Board
		board.Reset()
		requireMakeMove(t, &board, NewMove(e2, e4, DoublePawnPush))
		if got, want := board.EnPassantTarget, e3.Mask(); got != want {
			t.Errorf("EnPassantTarget = %#x, want %#x", got, want)
		}
		if got := board.PieceAt(e4); got != (Piece{Color: White, Type: Pawn}) {
			t.Errorf("piece at e4 = %#v, want white pawn", got)
		}
	})

	t.Run("promotion changes pawn type", func(t *testing.T) {
		var board Board
		board.SetPieceAt(a1, Piece{Color: White, Type: King})
		board.SetPieceAt(a8, Piece{Color: Black, Type: King})
		board.SetPieceAt(e7, Piece{Color: White, Type: Pawn})
		board.ColorToMove = White

		requireMakeMove(t, &board, NewMove(e7, e8, PromoteQueen))
		if got := board.PieceAt(e8); got != (Piece{Color: White, Type: Queen}) {
			t.Errorf("piece at e8 = %#v, want white queen", got)
		}
	})

	t.Run("en passant removes bypassed pawn", func(t *testing.T) {
		var board Board
		board.SetPieceAt(e1, Piece{Color: White, Type: King})
		board.SetPieceAt(e8, Piece{Color: Black, Type: King})
		board.SetPieceAt(d5, Piece{Color: White, Type: Pawn})
		board.SetPieceAt(e7, Piece{Color: Black, Type: Pawn})
		board.ColorToMove = Black

		requireMakeMove(t, &board, NewMove(e7, e5, DoublePawnPush))
		requireMakeMove(t, &board, NewMove(d5, e6, EnPassant))
		if got := board.PieceAt(e6); got != (Piece{Color: White, Type: Pawn}) {
			t.Errorf("piece at e6 = %#v, want white pawn", got)
		}
		if got := board.PieceAt(e5); got.Type != PieceNone {
			t.Errorf("piece at e5 = %#v, want empty", got)
		}
		if board.EnPassantTarget != 0 {
			t.Errorf("EnPassantTarget = %#x, want zero", board.EnPassantTarget)
		}
	})

	t.Run("en passant target expires after a reply", func(t *testing.T) {
		var board Board
		board.Reset()
		requireMakeMove(t, &board, NewMove(e2, e4, DoublePawnPush))
		requireMakeMove(t, &board, NewMove(a7, a6, QuietMove))
		if board.EnPassantTarget != 0 {
			t.Errorf("EnPassantTarget = %#x, want zero", board.EnPassantTarget)
		}
	})
}

func TestMakeMoveRejectsIllegalMoveWithoutMutation(t *testing.T) {
	var board Board
	board.Reset()
	before := board

	err := board.MakeMove(NewMove(e2, e5, QuietMove))
	if err == nil {
		t.Fatal("MakeMove() returned no error for an illegal move")
	}
	if board != before {
		t.Fatal("MakeMove() mutated the board after rejecting a move")
	}
}

func TestInitialPositionPerft(t *testing.T) {
	var board Board
	board.Reset()

	tests := []struct {
		depth int
		want  int
	}{
		{1, 20},
		{2, 400},
		{3, 8902},
		{4, 197281},
		{5, 4865609},
	}

	for _, test := range tests {
		if got := perft(board, test.depth); got != test.want {
			t.Errorf("perft(%d) = %d, want %d", test.depth, got, test.want)
		}
	}
}

func TestEnPassantThatExposesKingIsNotLegal(t *testing.T) {
	var board Board
	board.SetPieceAt(h5, Piece{Color: White, Type: King})
	board.SetPieceAt(e8, Piece{Color: Black, Type: King})
	board.SetPieceAt(g5, Piece{Color: White, Type: Pawn})
	board.SetPieceAt(f7, Piece{Color: Black, Type: Pawn})
	board.SetPieceAt(a5, Piece{Color: Black, Type: Rook})
	board.ColorToMove = Black

	requireMakeMove(t, &board, NewMove(f7, f5, DoublePawnPush))
	enPassant := NewMove(g5, f6, EnPassant)
	requireMove(t, GeneratePawnPseudoLegalMoves(board, White), enPassant)
	requireNoMove(t, GenerateLegalMoves(board), enPassant)
}

func standardCastlingBoard(color PieceColor, side CastlingSide) Board {
	var board Board
	board.SetPieceAt(e1, Piece{Color: White, Type: King})
	board.SetPieceAt(e8, Piece{Color: Black, Type: King})

	right := defaultStartingCastlingRights[color][side]
	board.SetPieceAt(right.RookFrom, Piece{Color: color, Type: Rook})
	board.CastlingRights[color][side] = right
	board.ColorToMove = color

	return board
}

func TestGenerateLegalMovesIncludesStandardCastles(t *testing.T) {
	for _, color := range []PieceColor{White, Black} {
		for _, side := range []CastlingSide{KingSide, QueenSide} {
			board := standardCastlingBoard(color, side)
			kingTo, _ := castlingDestinations(color, side)
			want := NewMove(defaultStartingCastlingRights[color][side].KingFrom, kingTo, MoveFlag(side))

			requireMove(t, GenerateLegalMoves(board), want)
		}
	}
}

func TestCastlingRejectsBlockedAndAttackedKingPaths(t *testing.T) {
	tests := []struct {
		name  string
		board Board
		move  Move
	}{
		{
			name: "king-side blocker",
			board: func() Board {
				board := standardCastlingBoard(White, KingSide)
				board.SetPieceAt(f1, Piece{Color: White, Type: Bishop})
				return board
			}(),
			move: NewMove(e1, g1, KingSideCastle),
		},
		{
			name: "queen-side blocker",
			board: func() Board {
				board := standardCastlingBoard(White, QueenSide)
				board.SetPieceAt(b1, Piece{Color: White, Type: Knight})
				return board
			}(),
			move: NewMove(e1, c1, QueenSideCastle),
		},
		{
			name: "king currently in check",
			board: func() Board {
				board := standardCastlingBoard(White, KingSide)
				board.ClearSquare(e8)
				board.SetPieceAt(a8, Piece{Color: Black, Type: King})
				board.SetPieceAt(e8, Piece{Color: Black, Type: Rook})
				return board
			}(),
			move: NewMove(e1, g1, KingSideCastle),
		},
		{
			name: "attacked transit square",
			board: func() Board {
				board := standardCastlingBoard(White, KingSide)
				board.ClearSquare(e8)
				board.SetPieceAt(a8, Piece{Color: Black, Type: King})
				board.SetPieceAt(f8, Piece{Color: Black, Type: Rook})
				return board
			}(),
			move: NewMove(e1, g1, KingSideCastle),
		},
		{
			name: "attacked destination square",
			board: func() Board {
				board := standardCastlingBoard(White, KingSide)
				board.ClearSquare(e8)
				board.SetPieceAt(a8, Piece{Color: Black, Type: King})
				board.SetPieceAt(g8, Piece{Color: Black, Type: Rook})
				return board
			}(),
			move: NewMove(e1, g1, KingSideCastle),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireNoMove(t, GenerateLegalMoves(test.board), test.move)
		})
	}
}

func TestMakeMoveAppliesCastling(t *testing.T) {
	for _, color := range []PieceColor{White, Black} {
		for _, side := range []CastlingSide{KingSide, QueenSide} {
			board := standardCastlingBoard(color, side)
			right := defaultStartingCastlingRights[color][side]
			kingTo, rookTo := castlingDestinations(color, side)
			castle := NewMove(right.KingFrom, kingTo, MoveFlag(side))

			requireMakeMove(t, &board, castle)
			if got := board.PieceAt(kingTo); got != (Piece{Color: color, Type: King}) {
				t.Errorf("king at %s = %#v, want %s king", kingTo, got, color)
			}
			if got := board.PieceAt(rookTo); got != (Piece{Color: color, Type: Rook}) {
				t.Errorf("rook at %s = %#v, want %s rook", rookTo, got, color)
			}
			if board.PieceAt(right.KingFrom).Type != PieceNone || board.PieceAt(right.RookFrom).Type != PieceNone {
				t.Error("castling left a piece on an origin square")
			}
			if board.ColorToMove != color.Opponent() {
				t.Errorf("ColorToMove = %s, want %s", board.ColorToMove, color.Opponent())
			}
			for castlingSide := range CastlingSideCount {
				if board.CastlingRights[color][castlingSide].Available {
					t.Errorf("%s castling right %d remains available", color, castlingSide)
				}
			}
		}
	}
}

func TestCastlingRightsAreRevokedByKingRookMovesAndRookCapture(t *testing.T) {
	t.Run("king move", func(t *testing.T) {
		var board Board
		board.Reset()
		requireMakeMove(t, &board, NewMove(e2, e4, DoublePawnPush))
		requireMakeMove(t, &board, NewMove(a7, a6, QuietMove))
		requireMakeMove(t, &board, NewMove(e1, e2, QuietMove))

		for side := range CastlingSideCount {
			if board.CastlingRights[White][side].Available {
				t.Errorf("white castling right %d remains available after king move", side)
			}
		}
	})

	t.Run("rook move", func(t *testing.T) {
		board := standardCastlingBoard(White, KingSide)
		board.CastlingRights[White][QueenSide] = defaultStartingCastlingRights[White][QueenSide]
		board.SetPieceAt(a1, Piece{Color: White, Type: Rook})
		requireMakeMove(t, &board, NewMove(h1, h2, QuietMove))

		if board.CastlingRights[White][KingSide].Available {
			t.Error("white king-side castling remains available after h1 rook moves")
		}
		if !board.CastlingRights[White][QueenSide].Available {
			t.Error("white queen-side castling was revoked by the h1 rook move")
		}
	})

	t.Run("rook capture", func(t *testing.T) {
		var board Board
		board.SetPieceAt(e1, Piece{Color: White, Type: King})
		board.SetPieceAt(a8, Piece{Color: Black, Type: King})
		board.SetPieceAt(h1, Piece{Color: White, Type: Rook})
		board.SetPieceAt(h8, Piece{Color: Black, Type: Rook})
		board.CastlingRights[White][KingSide] = defaultStartingCastlingRights[White][KingSide]
		board.ColorToMove = Black

		requireMakeMove(t, &board, NewMove(h8, h1, Capture))
		if board.CastlingRights[White][KingSide].Available {
			t.Error("white king-side castling remains available after the h1 rook is captured")
		}
	})
}

func perft(board Board, depth int) int {
	if depth == 0 {
		return 1
	}

	nodes := 0
	for _, move := range GenerateLegalMoves(board) {
		next := board
		next.applyMove(move)
		nodes += perft(next, depth-1)
	}

	return nodes
}
