package gui

import (
	"strings"
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
		square, ok := squareAtScreenPosition(test.x, test.y, layout, chess.White)
		if ok != test.ok {
			t.Errorf("squareAtScreenPosition(%d, %d) ok = %t, want %t", test.x, test.y, ok, test.ok)
			continue
		}
		if ok && square.String() != test.want {
			t.Errorf("squareAtScreenPosition(%d, %d) = %s, want %s", test.x, test.y, square, test.want)
		}
	}
}

func TestSquareAtScreenPositionFromBlackPerspective(t *testing.T) {
	layout := boardLayout{x: 10, y: 2}
	for _, test := range []struct {
		x    int
		y    int
		want string
	}{
		{10, 2, "h1"},
		{25, 9, "a8"},
	} {
		square, ok := squareAtScreenPosition(test.x, test.y, layout, chess.Black)
		if !ok || square.String() != test.want {
			t.Errorf("black squareAtScreenPosition(%d, %d) = %s, %t; want %s, true", test.x, test.y, square, ok, test.want)
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
	popupX, targetY := promotionScreenPosition(target, layout, chess.White)
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

func TestPromotionOutsideClickCancelsSelection(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}
	target := guiSquare(t, "e8")
	game.promotionChoices = []chess.Move{
		chess.NewMove(guiSquare(t, "e7"), target, chess.PromoteQueen),
	}

	game.handleMouse(tcell.NewEventMouse(layout.x, layout.y+7, tcell.Button1, tcell.ModNone), 80, 24)
	if game.promotionChoices != nil {
		t.Errorf("outside click left promotion choices active: %v", game.promotionChoices)
	}
	if game.message != "Promotion cancelled" {
		t.Errorf("message = %q, want promotion cancellation", game.message)
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
	_, color := squareStyle(game, game.board, guiSquare(t, "e1"), legalMoves)
	if want := darkenRGB(chess.RGB{R: 205, G: 50, B: 50}, 71); color != want {
		t.Errorf("hover color while checked = %#v, want %#v", color, want)
	}

	game.hoveredSquare = nil
	game.selectSource(guiSquare(t, "e1"))
	_, color = squareStyle(game, game.board, guiSquare(t, "d1"), legalMoves)
	want := squareColor(game.theme, guiSquare(t, "d1"))
	want.G = 255
	if color != want {
		t.Errorf("destination color while checked = %#v, want %#v", color, want)
	}
}

func TestMoveHistoryWindowUsesFullmoves(t *testing.T) {
	history := make([]string, 14)
	start, end := moveHistoryWindow(history)
	if start != 3 || end != 7 {
		t.Errorf("moveHistoryWindow(14 moves) = %d, %d; want 3, 7", start, end)
	}
}

func TestFormatMoveHistoryRowUsesFixedWidth(t *testing.T) {
	row := formatMoveHistoryRow(5949, "Nb1xd2#", "exf8=Q#")
	if want := "5949. Nb1xd2#   exf8=Q# "; row != want {
		t.Errorf("formatMoveHistoryRow() = %q, want %q", row, want)
	}
	if len(row) != moveHistoryRowWidth {
		t.Errorf("row width = %d, want %d", len(row), moveHistoryRowWidth)
	}

	row = formatMoveHistoryRow(1, "e4", "")
	if len(row) != moveHistoryRowWidth {
		t.Errorf("single-move row width = %d, want %d", len(row), moveHistoryRowWidth)
	}
}

func TestResignationConfirmationAndCancellation(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}
	buttonX, buttonY := resignButtonPosition(layout)

	game.handleMouse(tcell.NewEventMouse(buttonX, buttonY, tcell.Button1, tcell.ModNone), 80, 24)
	if !game.resignPending || game.resignedBy != nil {
		t.Errorf("first resign click = pending %t, resigned %v; want pending confirmation", game.resignPending, game.resignedBy)
	}
	game.handleKey(tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone), 80, 24)
	if game.resignPending {
		t.Error("Escape did not cancel resignation confirmation")
	}

	game.handleMouse(tcell.NewEventMouse(buttonX, buttonY, tcell.Button1, tcell.ModNone), 80, 24)
	game.handleMouse(tcell.NewEventMouse(layout.x, layout.y, tcell.Button1, tcell.ModNone), 80, 24)
	if game.resignPending {
		t.Error("outside click did not cancel resignation confirmation")
	}
}

func TestResignationEndsLocalGame(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}
	buttonX, buttonY := resignButtonPosition(layout)
	game.handleMouse(tcell.NewEventMouse(buttonX, buttonY, tcell.Button1, tcell.ModNone), 80, 24)
	game.handleMouse(tcell.NewEventMouse(buttonX, buttonY, tcell.Button1, tcell.ModNone), 80, 24)

	if game.resignedBy == nil || *game.resignedBy != chess.White {
		t.Errorf("resignedBy = %v, want White", game.resignedBy)
	}
	if got := gameTitle(game); got != "White resigned · Black wins 1-0" {
		t.Errorf("gameTitle() = %q, want resignation result", got)
	}

	game.uciInput = "e2e4"
	game.submitUCI()
	game.handleMouse(tcell.NewEventMouse(layout.x+8, layout.y+6, tcell.Button1, tcell.ModNone), 80, 24)
	if game.board.PieceAt(guiSquare(t, "e2")) != (chess.Piece{Color: chess.White, Type: chess.Pawn}) {
		t.Error("resigned game accepted a move")
	}
	if len(game.history) != 0 {
		t.Errorf("resigned game recorded history: %v", game.history)
	}
}

func TestResignButtonLabel(t *testing.T) {
	if got := resignButtonLabel(false); got != "[🏳️ ] Resign" {
		t.Errorf("resignButtonLabel(false) = %q", got)
	}
	if got := resignButtonLabel(true); got != "[🏳️ ] Are you sure?" {
		t.Errorf("resignButtonLabel(true) = %q", got)
	}
}

func TestResignButtonHover(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}
	buttonX, buttonY := resignButtonPosition(layout)

	game.handleMouse(tcell.NewEventMouse(buttonX, buttonY, tcell.ButtonNone, tcell.ModNone), 80, 24)
	if !game.hoveredResign {
		t.Error("hovering resign button did not set hoveredResign")
	}
	game.handleMouse(tcell.NewEventMouse(layout.x, layout.y, tcell.ButtonNone, tcell.ModNone), 80, 24)
	if game.hoveredResign {
		t.Error("moving away from resign button did not clear hoveredResign")
	}
}

func TestManualAndAutomaticBoardFlip(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}
	buttonX, buttonY := flipButtonPosition(layout)

	game.handleMouse(tcell.NewEventMouse(buttonX, buttonY, tcell.ButtonNone, tcell.ModNone), 80, 24)
	if !game.hoveredFlip {
		t.Error("hovering flip button did not set hoveredFlip")
	}
	game.handleMouse(tcell.NewEventMouse(buttonX, buttonY, tcell.Button1, tcell.ModNone), 80, 24)
	if game.perspective != chess.Black {
		t.Errorf("manual flip perspective = %s, want Black", game.perspective)
	}

	game = newLocalGameState(chess.WOOD)
	game.autoFlip = true
	game.uciInput = "e2e4"
	game.submitUCI()
	if game.perspective != chess.Black {
		t.Errorf("perspective after White move = %s, want Black", game.perspective)
	}
	game.uciInput = "e7e5"
	game.submitUCI()
	if game.perspective != chess.White {
		t.Errorf("perspective after Black move = %s, want White", game.perspective)
	}
}

func TestLocalGameAutomaticDraws(t *testing.T) {
	t.Run("threefold repetition", func(t *testing.T) {
		game := newLocalGameState(chess.WOOD)
		for _, uci := range []string{
			"g1f3", "g8f6", "f3g1", "f6g8",
			"g1f3", "g8f6", "f3g1", "f6g8",
		} {
			game.uciInput = uci
			game.submitUCI()
		}
		if game.drawReason != "Draw by threefold repetition" {
			t.Errorf("drawReason = %q, want threefold repetition", game.drawReason)
		}
	})

	t.Run("fifty move rule", func(t *testing.T) {
		game := newLocalGameState(chess.WOOD)
		game.board.HalfmoveClock = 99
		game.uciInput = "g1f3"
		game.submitUCI()
		if game.drawReason != "Draw by fifty-move rule" {
			t.Errorf("drawReason = %q, want fifty-move rule", game.drawReason)
		}
	})

	t.Run("insufficient material", func(t *testing.T) {
		game := newLocalGameState(chess.WOOD)
		game.board.Clear()
		game.board.SetPieceAt(guiSquare(t, "e1"), chess.Piece{Color: chess.White, Type: chess.King})
		game.board.SetPieceAt(guiSquare(t, "e8"), chess.Piece{Color: chess.Black, Type: chess.King})
		game.board.SetPieceAt(guiSquare(t, "b1"), chess.Piece{Color: chess.White, Type: chess.Knight})
		game.board.ColorToMove = chess.White
		game.positionCounts = map[uint64]uint{game.board.PositionKey(): 1}
		game.uciInput = "b1c3"
		game.submitUCI()
		if game.drawReason != "Draw by insufficient material" {
			t.Errorf("drawReason = %q, want insufficient material", game.drawReason)
		}

		game.uciInput = "e8e7"
		game.submitUCI()
		if game.board.PieceAt(guiSquare(t, "e8")) != (chess.Piece{Color: chess.Black, Type: chess.King}) {
			t.Error("automatic draw accepted a later move")
		}
	})
}

func TestLocalGameCheckmateAndStalemateAreTerminal(t *testing.T) {
	t.Run("checkmate", func(t *testing.T) {
		game := newLocalGameState(chess.WOOD)
		for _, uci := range []string{"f2f3", "e7e5", "g2g4", "d8h4"} {
			game.uciInput = uci
			game.submitUCI()
		}
		if game.terminalTitle != "Checkmate · Black wins 0-1" {
			t.Errorf("terminalTitle = %q, want checkmate", game.terminalTitle)
		}
		assertTerminalGameBlocksInput(t, &game)
		assertTerminalControlsHidden(t, game)
	})

	t.Run("stalemate", func(t *testing.T) {
		game := newLocalGameState(chess.WOOD)
		game.board.Clear()
		game.board.SetPieceAt(guiSquare(t, "c6"), chess.Piece{Color: chess.White, Type: chess.King})
		game.board.SetPieceAt(guiSquare(t, "d7"), chess.Piece{Color: chess.White, Type: chess.Queen})
		game.board.SetPieceAt(guiSquare(t, "a8"), chess.Piece{Color: chess.Black, Type: chess.King})
		game.board.ColorToMove = chess.White
		game.positionCounts = map[uint64]uint{game.board.PositionKey(): 1}
		game.uciInput = "d7c7"
		game.submitUCI()
		if game.terminalTitle != "Stalemate · ½-½" {
			t.Errorf("terminalTitle = %q, want stalemate", game.terminalTitle)
		}
		assertTerminalGameBlocksInput(t, &game)
		assertTerminalControlsHidden(t, game)
	})
}

func TestTerminalReplayNavigation(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	for _, uci := range []string{"f2f3", "e7e5", "g2g4", "d8h4"} {
		game.uciInput = uci
		game.submitUCI()
	}
	if len(game.positions) != 5 || game.viewPly != 4 {
		t.Fatalf("replay snapshots = %d at ply %d, want 5 at ply 4", len(game.positions), game.viewPly)
	}

	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}
	game.handleMouse(tcell.NewEventMouse(layout.x, layout.y, tcell.WheelUp, tcell.ModNone), 80, 24)
	if game.viewPly != 3 {
		t.Errorf("viewPly after board wheel up = %d, want 3", game.viewPly)
	}
	board := game.boardForDisplay()
	if got := board.PieceAt(guiSquare(t, "g4")); got != (chess.Piece{Color: chess.White, Type: chess.Pawn}) {
		t.Errorf("replay board piece at g4 = %#v, want white pawn", got)
	}
	if got := gameTitle(game); got != "Viewing 2. g4" {
		t.Errorf("gameTitle() = %q, want viewing title", got)
	}

	historyX := layout.x + 20
	game.handleMouse(tcell.NewEventMouse(historyX+moveNumberWidth+2, layout.y+2, tcell.Button1, tcell.ModNone), 80, 24)
	if game.viewPly != 1 {
		t.Errorf("White SAN click selected ply %d, want 1", game.viewPly)
	}
	game.handleMouse(tcell.NewEventMouse(historyX+moveNumberWidth+2+sanWidth+2, layout.y+2, tcell.Button1, tcell.ModNone), 80, 24)
	if game.viewPly != 2 {
		t.Errorf("Black SAN click selected ply %d, want 2", game.viewPly)
	}
}

func TestReplayHistoryViewportAnchoring(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.terminalTitle = "Stalemate · ½-½"
	game.history = make([]string, 20)
	game.moves = make([]chess.Move, 20)
	game.positions = make([]chess.Board, 21)

	game.viewPly = 12
	game.moveReplay(-1)
	if game.viewPly != 11 || game.historyScroll != 4 {
		t.Errorf("backward replay = ply %d, scroll %d; want ply 11, scroll 4", game.viewPly, game.historyScroll)
	}
	game.moveReplay(1)
	if game.viewPly != 12 || game.historyScroll != 3 {
		t.Errorf("forward replay = ply %d, scroll %d; want ply 12, scroll 3", game.viewPly, game.historyScroll)
	}
	game.viewPly = 1
	game.syncHistoryToView(-1)
	if game.historyScroll != 0 {
		t.Errorf("first move history scroll = %d, want 0", game.historyScroll)
	}
	game.viewPly = len(game.moves)
	game.syncHistoryToView(1)
	if game.historyScroll != game.historyMaxScroll() {
		t.Errorf("final move history scroll = %d, want %d", game.historyScroll, game.historyMaxScroll())
	}

	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}
	game.viewPly = 12
	game.historyScroll = 3
	viewPly := game.viewPly
	historyX := layout.x + 20
	game.handleMouse(tcell.NewEventMouse(historyX, layout.y+2, tcell.WheelDown, tcell.ModNone), 80, 24)
	if game.viewPly != viewPly {
		t.Errorf("history wheel changed view ply to %d, want %d", game.viewPly, viewPly)
	}
	if game.historyScroll != 4 {
		t.Errorf("history wheel scroll = %d, want 4", game.historyScroll)
	}
}

func assertTerminalGameBlocksInput(t *testing.T, game *localGameState) {
	t.Helper()
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("local board layout is unavailable")
	}
	buttonX, buttonY := resignButtonPosition(layout)
	before := game.board

	game.uciInput = "a2a3"
	game.submitUCI()
	game.handleMouse(tcell.NewEventMouse(buttonX, buttonY, tcell.Button1, tcell.ModNone), 80, 24)
	if game.resignedBy != nil {
		t.Error("terminal game accepted resignation")
	}
	if game.board != before {
		t.Error("terminal game accepted a move")
	}
}

func assertTerminalControlsHidden(t *testing.T, game localGameState) {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)

	drawLocalGame(screen, game)
	screen.Show()
	contents, width, height := screen.GetContents()
	text := screenText(contents, width, height)
	for _, label := range []string{"Your move (UCI):", "Resign"} {
		if strings.Contains(text, label) {
			t.Errorf("terminal game rendered %q", label)
		}
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
