package gui

import (
	"testing"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/gdamore/tcell/v2"
)

func guiSquare(t *testing.T, coordinate string) chess.Square {
	t.Helper()

	square, err := chess.ParseCoordinate(coordinate)
	if err != nil {
		t.Fatal(err)
	}

	return square
}

func TestSquareAtScreenPosition(t *testing.T) {
	layout := boardLayout{x: 10, y: 2}
	tests := []struct {
		x    int
		y    int
		want string
		ok   bool
	}{
		{10, 2, "a8", true},
		{25, 9, "h1", true},
		{9, 2, "", false},
		{26, 9, "", false},
	}

	for _, test := range tests {
		square, ok := squareAtScreenPosition(test.x, test.y, layout)
		if ok != test.ok {
			t.Errorf("squareAtScreenPosition(%d, %d) ok = %t, want %t", test.x, test.y, ok, test.ok)
			continue
		}
		if ok && square.String() != test.want {
			t.Errorf("squareAtScreenPosition(%d, %d) = %s, want %s", test.x, test.y, square, test.want)
		}
	}
}

func TestLocalGameCoordinateInputMakesMove(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.input = "e2e4"
	game.submitInput()

	if got := game.board.PieceAt(guiSquare(t, "e4")); got != (chess.Piece{Color: chess.White, Type: chess.Pawn}) {
		t.Errorf("piece at e4 = %#v, want white pawn", got)
	}
	if game.board.ColorToMove != chess.Black {
		t.Errorf("ColorToMove = %s, want Black", game.board.ColorToMove)
	}
	if !game.hasLastMove || game.lastMove.From() != guiSquare(t, "e2") || game.lastMove.To() != guiSquare(t, "e4") {
		t.Errorf("last move = %v, want e2-e4", game.lastMove)
	}
}

func TestLocalGameInvalidDestinationClearsSelection(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}

	game.handleMouse(tcell.NewEventMouse(layout.x+8, layout.y+6, tcell.Button1, tcell.ModNone), 80, 24)
	if game.selectedSource == nil || *game.selectedSource != guiSquare(t, "e2") {
		t.Fatalf("selected source = %v, want e2", game.selectedSource)
	}

	game.handleMouse(tcell.NewEventMouse(layout.x+8, layout.y+3, tcell.Button1, tcell.ModNone), 80, 24)
	if game.selectedSource != nil {
		t.Errorf("invalid destination left source selected: %s", *game.selectedSource)
	}
	if game.board.PieceAt(guiSquare(t, "e2")) != (chess.Piece{Color: chess.White, Type: chess.Pawn}) {
		t.Error("invalid destination mutated the board")
	}
}

func TestAdjustRGBAddsGreenWithoutOverflow(t *testing.T) {
	got := adjustRGB(chess.RGB{R: 10, G: 240, B: 20}, 0, 31, 0)
	if got != (chess.RGB{R: 10, G: 255, B: 20}) {
		t.Errorf("adjustRGB() = %#v, want green capped at 255", got)
	}
}

func TestLocalGamePromotionInput(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.board.Clear()
	game.board.SetPieceAt(guiSquare(t, "a1"), chess.Piece{Color: chess.White, Type: chess.King})
	game.board.SetPieceAt(guiSquare(t, "a8"), chess.Piece{Color: chess.Black, Type: chess.King})
	game.board.SetPieceAt(guiSquare(t, "e7"), chess.Piece{Color: chess.White, Type: chess.Pawn})
	game.board.ColorToMove = chess.White
	game.input = "e7e8q"

	game.submitInput()
	if got := game.board.PieceAt(guiSquare(t, "e8")); got != (chess.Piece{Color: chess.White, Type: chess.Queen}) {
		t.Errorf("piece at e8 = %#v, want white queen", got)
	}
}

func TestCastlingUsesRookAsMouseTarget(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.board.Clear()
	game.board.SetPieceAt(guiSquare(t, "e1"), chess.Piece{Color: chess.White, Type: chess.King})
	game.board.SetPieceAt(guiSquare(t, "h1"), chess.Piece{Color: chess.White, Type: chess.Rook})
	game.board.SetPieceAt(guiSquare(t, "e8"), chess.Piece{Color: chess.Black, Type: chess.King})
	game.board.CastlingRights[chess.White][chess.KingSide] = chess.CastlingRight{
		KingFrom:  guiSquare(t, "e1"),
		RookFrom:  guiSquare(t, "h1"),
		Available: true,
	}
	game.selectSource(guiSquare(t, "e1"))

	candidates := game.movesForTarget(guiSquare(t, "h1"), chess.GenerateLegalMoves(game.board))
	if len(candidates) != 1 || candidates[0].Flag() != chess.KingSideCastle {
		t.Errorf("rook-targeted castling candidates = %v, want king-side castle", candidates)
	}
}
