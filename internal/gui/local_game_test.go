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
	game.uciInput = "e2e4"
	game.submitUCI()

	if got := game.board.PieceAt(guiSquare(t, "e4")); got != (chess.Piece{Color: chess.White, Type: chess.Pawn}) {
		t.Errorf("piece at e4 = %#v, want white pawn", got)
	}
	if game.board.ColorToMove != chess.Black {
		t.Errorf("ColorToMove = %s, want Black", game.board.ColorToMove)
	}
	if !game.hasLastMove || game.lastMove.From() != guiSquare(t, "e2") || game.lastMove.To() != guiSquare(t, "e4") {
		t.Errorf("last move = %v, want e2e4", game.lastMove)
	}
	if len(game.history) != 1 || game.history[0] != "e4" {
		t.Errorf("history = %v, want [e4]", game.history)
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

func TestLocalGameClickingAnotherSourceChangesSelection(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}

	game.handleMouse(tcell.NewEventMouse(layout.x+8, layout.y+6, tcell.Button1, tcell.ModNone), 80, 24)
	game.handleMouse(tcell.NewEventMouse(layout.x+6, layout.y+6, tcell.Button1, tcell.ModNone), 80, 24)
	if game.selectedSource == nil || *game.selectedSource != guiSquare(t, "d2") {
		t.Errorf("selected source = %v, want d2", game.selectedSource)
	}
}

func TestLocalGameOutsideBoardClickClearsSelection(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}

	game.selectSource(guiSquare(t, "e2"))
	game.handleMouse(tcell.NewEventMouse(layout.x-1, layout.y, tcell.Button1, tcell.ModNone), 80, 24)
	if game.selectedSource != nil {
		t.Errorf("outside click left source selected: %s", *game.selectedSource)
	}
}

func TestBrightenRGBCapsAllChannels(t *testing.T) {
	got := brightenRGB(chess.RGB{R: 240, G: 250, B: 20}, 31)
	if got != (chess.RGB{R: 255, G: 255, B: 51}) {
		t.Errorf("brightenRGB() = %#v, want capped brighter color", got)
	}
}

func TestPromotionOptionAt(t *testing.T) {
	layout := boardLayout{x: 10, y: 4}
	game := newLocalGameState(chess.WOOD)
	target := guiSquare(t, "e8")
	game.promotionChoices = []chess.Move{
		chess.NewMove(guiSquare(t, "e7"), target, chess.PromoteQueen),
	}
	popupX, targetY := promotionScreenPosition(target, layout)
	for index, want := range promotionOptions {
		got, ok := game.promotionOptionAt(popupX+1, targetY-index-1, layout)
		if !ok || got != want {
			t.Errorf("promotionOptionAt(%d, %d) = %q, %t; want %q, true", popupX+1, targetY-index-1, got, ok, want)
		}
	}

	if _, ok := game.promotionOptionAt(popupX+squareWidth, targetY-1, layout); ok {
		t.Error("promotionOptionAt accepted a different file")
	}
}

func TestPromotionPieceType(t *testing.T) {
	if got := promotionPieceType('n'); got != chess.Knight {
		t.Errorf("promotionPieceType('n') = %s, want Knight", got)
	}
	if got := promotionPieceType('q'); got != chess.Queen {
		t.Errorf("promotionPieceType('q') = %s, want Queen", got)
	}
}

func TestSquareStyleHighlightsLegalMovesWhileInCheck(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.board.Clear()
	game.board.SetPieceAt(guiSquare(t, "e1"), chess.Piece{Color: chess.White, Type: chess.King})
	game.board.SetPieceAt(guiSquare(t, "a8"), chess.Piece{Color: chess.Black, Type: chess.King})
	game.board.SetPieceAt(guiSquare(t, "e8"), chess.Piece{Color: chess.Black, Type: chess.Rook})
	game.board.ColorToMove = chess.White
	legalMoves := chess.GenerateLegalMoves(game.board)
	if !game.board.IsColorInCheck(chess.White) {
		t.Fatal("white king should be in check")
	}

	game.hoveredSquare = new(chess.Square)
	*game.hoveredSquare = guiSquare(t, "e1")
	_, color := squareStyle(game, guiSquare(t, "e1"), legalMoves)
	if want := darkenRGB(chess.RGB{R: 205, G: 50, B: 50}, 71); color != want {
		t.Errorf("hover color while checked = %#v, want %#v", color, want)
	}

	game.hoveredSquare = nil
	game.selectSource(guiSquare(t, "e1"))
	_, color = squareStyle(game, guiSquare(t, "d1"), legalMoves)
	want := squareColor(game.theme, guiSquare(t, "d1"))
	want.G = 255
	if color != want {
		t.Errorf("destination color while checked = %#v, want %#v", color, want)
	}
}

func TestMoveHistoryWindowUsesFullmoves(t *testing.T) {
	history := make([]string, 14)
	start, end := moveHistoryWindow(history)
	if start != 1 || end != 7 {
		t.Errorf("moveHistoryWindow(14 moves) = %d, %d; want 1, 7", start, end)
	}
}

func squareColor(theme chess.ColorTheme, square chess.Square) chess.RGB {
	light, dark := theme.SquareColors()
	if square.IsDark() {
		return dark
	}
	return light
}

func TestLocalGamePromotionInput(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.board.Clear()
	game.board.SetPieceAt(guiSquare(t, "a1"), chess.Piece{Color: chess.White, Type: chess.King})
	game.board.SetPieceAt(guiSquare(t, "a8"), chess.Piece{Color: chess.Black, Type: chess.King})
	game.board.SetPieceAt(guiSquare(t, "e7"), chess.Piece{Color: chess.White, Type: chess.Pawn})
	game.board.ColorToMove = chess.White
	game.uciInput = "e7e8q"

	game.submitUCI()
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
