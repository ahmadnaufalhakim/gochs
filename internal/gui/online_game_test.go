package gui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	lichessapi "github.com/ahmadnaufalhakim/gochs/internal/lichess"
	"github.com/gdamore/tcell/v2"
)

func TestOnlineTimeControlExcludesUnlimitedAndCustom(t *testing.T) {
	state := menuState{page: timeControlMenu, onlineGame: true, timeControlSelected: onlineTimeControlStart + onlineTimeControlCount - 1}
	state.moveTimeControlSelection(1)
	if state.timeControlSelected != onlineTimeControlStart {
		t.Fatalf("online selection = %d, want wrap to first supported preset", state.timeControlSelected)
	}
	state.timeControlSelected = 0
	if state.activateTimeControl() {
		t.Fatal("Bullet must not start an online seek")
	}
}

func TestOnlineGameRebuildsServerMovesAndPerspective(t *testing.T) {
	full := lichessapi.GameFullEvent{
		Game:       lichessapi.Game{ID: "game", White: lichessapi.Player{ID: "opponent", Name: "Opponent", Rating: 1600}, Black: lichessapi.Player{ID: "me", Name: "Me", Rating: 1700}},
		InitialFEN: "startpos",
		State:      lichessapi.GameState{Moves: "e2e4 e7e5", WhiteTime: 60000, BlackTime: 59000, Status: "started"},
	}
	game, err := newOnlineGame(full, "me", "Me", chess.WOOD)
	if err != nil {
		t.Fatal(err)
	}
	if game.me != chess.Black || game.perspective != chess.Black {
		t.Errorf("perspective = %v/%v, want black", game.me, game.perspective)
	}
	if len(game.moves) != 2 || game.board.ColorToMove != chess.White {
		t.Errorf("rebuilt game = moves:%d turn:%v", len(game.moves), game.board.ColorToMove)
	}
	e4, _ := chess.ParseCoordinate("e4")
	if piece := game.board.PieceAt(e4); piece.Type != chess.Pawn || piece.Color != chess.White {
		t.Errorf("e4 piece = %#v, want white pawn", piece)
	}
	e5, _ := chess.ParseCoordinate("e5")
	if piece := game.board.PieceAt(e5); piece.Type != chess.Pawn || piece.Color != chess.Black {
		t.Errorf("e5 piece = %#v, want black pawn", piece)
	}
	if got := playerLine(game.black, chess.Black); got != "○ Me (1700)" {
		t.Errorf("player line = %q", got)
	}
}

func TestOnlineGameAppliesServerMoveAndReconcilesClock(t *testing.T) {
	full := lichessapi.GameFullEvent{Game: lichessapi.Game{ID: "game", White: lichessapi.Player{ID: "me"}, Black: lichessapi.Player{ID: "opponent"}}, InitialFEN: "startpos", State: lichessapi.GameState{Moves: "e2e4", WhiteTime: 59000, BlackTime: 60000, Status: "started"}}
	game, err := newOnlineGame(full, "me", "Me", chess.WOOD)
	if err != nil {
		t.Fatal(err)
	}
	game.apply(onlineResult{event: lichessapi.GameStateEvent{State: lichessapi.GameState{Moves: "e2e4 e7e5", WhiteTime: 59000, BlackTime: 59500, Status: "started"}}})
	if len(game.moves) != 2 || game.board.ColorToMove != chess.White {
		t.Fatalf("server move was not applied: moves:%d turn:%s", len(game.moves), game.board.ColorToMove)
	}
	game.advanceClock(game.clockUpdated.Add(500 * time.Millisecond))
	if got := game.clockRemaining[chess.White]; got != 58500*time.Millisecond {
		t.Errorf("white clock = %s, want 58.5s", got)
	}
}

func TestOnlineMoveFailureAllowsRetry(t *testing.T) {
	game := onlineGameState{pendingMove: true}
	game.apply(onlineResult{action: "move", err: context.DeadlineExceeded})
	if game.pendingMove {
		t.Fatal("failed move must clear pending state")
	}
}

func TestOnlineIncomingOfferDisablesOfferButton(t *testing.T) {
	game := onlineGameState{me: chess.White, state: lichessapi.GameState{Status: "started", BlackDraw: true}}
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("expected board layout")
	}
	x, y := onlineActionButtonPosition(layout, "draw")
	game.handleMouse(tcell.NewEventMouse(x, y, tcell.ButtonNone, tcell.ModNone), 80, 24, nil, nil)
	if game.hoveredDraw {
		t.Fatal("incoming draw must disable draw-offer button")
	}
}

func TestOnlineGameKeepsStreamingWhileCreated(t *testing.T) {
	game := onlineGameState{state: lichessapi.GameState{Status: "created"}}
	if !game.isInProgress() || game.isFinished() {
		t.Fatal("created game must remain active until Lichess reports a terminal status")
	}
	game.state.Status = "mate"
	if game.isInProgress() || !game.isFinished() {
		t.Fatal("mate game must be terminal")
	}
}

func TestOnlineGameUsesMaterialIndicator(t *testing.T) {
	full := lichessapi.GameFullEvent{Game: lichessapi.Game{ID: "game", White: lichessapi.Player{ID: "me"}, Black: lichessapi.Player{ID: "opponent"}}, InitialFEN: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPP1/RNBQKBNR w KQkq - 0 1", State: lichessapi.GameState{Status: "started"}}
	game, err := newOnlineGame(full, "me", "Me", chess.WOOD)
	if err != nil {
		t.Fatal(err)
	}
	if got := materialIndicator(game.board, chess.Black); got != "♙ +1" {
		t.Errorf("material indicator = %q, want pawn advantage", got)
	}
}

func TestOnlineButtonsHoverAndCancelResign(t *testing.T) {
	game := onlineGameState{localGameState: newLocalGameState(chess.WOOD), state: lichessapi.GameState{Status: "started"}}
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("expected board layout")
	}
	resignX, resignY := resignButtonPosition(layout)
	game.handleMouse(tcell.NewEventMouse(resignX, resignY, tcell.ButtonNone, tcell.ModNone), 80, 24, nil, nil)
	if !game.hoveredResign {
		t.Fatal("resign button was not highlighted")
	}
	game.handleMouse(tcell.NewEventMouse(resignX, resignY, tcell.Button1, tcell.ModNone), 80, 24, nil, nil)
	if !game.resignPending {
		t.Fatal("first resign click did not request confirmation")
	}
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)
	drawOnlineGame(screen, game)
	screen.Show()
	cells, screenWidth, screenHeight := screen.GetContents()
	if !strings.Contains(screenText(cells, screenWidth, screenHeight), "Are you sure?") {
		t.Fatal("online resign confirmation was not rendered")
	}
	game.handleMouse(tcell.NewEventMouse(layout.x, layout.y, tcell.Button1, tcell.ModNone), 80, 24, nil, nil)
	if game.resignPending {
		t.Fatal("clicking elsewhere did not cancel resign confirmation")
	}
}

func TestOnlineGameHighlightsHoverablePiece(t *testing.T) {
	game := onlineGameState{
		localGameState: newLocalGameState(chess.WOOD),
		me:             chess.White,
		state:          lichessapi.GameState{Status: "started"},
	}
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("expected board layout")
	}
	game.handleMouse(tcell.NewEventMouse(layout.x+4*squareWidth, layout.y+6, tcell.ButtonNone, tcell.ModNone), 80, 24, nil, nil)
	e2, _ := chess.ParseCoordinate("e2")
	if game.hoveredSquare == nil || *game.hoveredSquare != e2 {
		t.Errorf("hovered square = %v, want e2", game.hoveredSquare)
	}
}

func TestFinishedOnlineGameReviewControls(t *testing.T) {
	game := onlineGameState{localGameState: newLocalGameState(chess.WOOD), state: lichessapi.GameState{Status: "mate"}}
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("expected board layout")
	}
	flipX, flipY := flipButtonPosition(layout)
	game.handleMouse(tcell.NewEventMouse(flipX, flipY, tcell.ButtonNone, tcell.ModNone), 80, 24, nil, nil)
	if !game.hoveredFlip {
		t.Fatal("finished game flip button was not highlighted")
	}
	game.handleMouse(tcell.NewEventMouse(flipX, flipY, tcell.Button1, tcell.ModNone), 80, 24, nil, nil)
	if game.perspective != chess.Black {
		t.Errorf("perspective = %s, want Black after flip", game.perspective)
	}
	saveX, saveY := onlineSavePGNButtonPosition(layout, false)
	game.handleMouse(tcell.NewEventMouse(saveX, saveY, tcell.ButtonNone, tcell.ModNone), 80, 24, nil, nil)
	if !game.hoveredSavePGN {
		t.Fatal("finished game PGN button was not highlighted")
	}
	clocksX, clocksY := onlineSavePGNButtonPosition(layout, true)
	game.handleMouse(tcell.NewEventMouse(clocksX, clocksY, tcell.ButtonNone, tcell.ModNone), 80, 24, nil, nil)
	if !game.hoveredSaveClocks {
		t.Fatal("finished game PGN clocks button was not highlighted")
	}
}

func TestOnlinePGNButtonsDoNotOverlapPlayerRows(t *testing.T) {
	layout, ok := localBoardLayout(80, 20)
	if !ok {
		t.Fatal("expected board layout")
	}
	_, plainY := onlineSavePGNButtonPosition(layout, false)
	_, clocksY := onlineSavePGNButtonPosition(layout, true)
	if plainY <= layout.y+11 || clocksY <= layout.y+11 {
		t.Fatalf("PGN buttons overlap online player rows: %d, %d", plainY, clocksY)
	}
	if clocksY >= 20 {
		t.Fatalf("PGN clocks button is outside the minimum terminal: %d", clocksY)
	}
}

func TestOnlineTerminalTitle(t *testing.T) {
	tests := []struct {
		state lichessapi.GameState
		want  string
	}{
		{state: lichessapi.GameState{Status: "mate", Winner: "white"}, want: "Checkmate · White wins 1-0"},
		{state: lichessapi.GameState{Status: "resign", Winner: "black"}, want: "White resigned · Black wins 0-1"},
		{state: lichessapi.GameState{Status: "stalemate"}, want: "Stalemate · ½-½"},
		{state: lichessapi.GameState{Status: "draw"}, want: "Draw · ½-½"},
	}
	for _, test := range tests {
		game := onlineGameState{state: test.state}
		if got := game.onlineTerminalTitle(); got != test.want {
			t.Errorf("terminal title = %q, want %q", got, test.want)
		}
	}
}

func TestFinishedOnlineGameScrollsBoardHistory(t *testing.T) {
	full := lichessapi.GameFullEvent{Game: lichessapi.Game{ID: "game", White: lichessapi.Player{ID: "me"}, Black: lichessapi.Player{ID: "opponent"}}, InitialFEN: "startpos", State: lichessapi.GameState{Moves: "e2e4 e7e5", Status: "mate", Winner: "white"}}
	game, err := newOnlineGame(full, "me", "Me", chess.WOOD)
	if err != nil {
		t.Fatal(err)
	}
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("expected board layout")
	}
	game.handleMouse(tcell.NewEventMouse(layout.x, layout.y, tcell.WheelUp, tcell.ModNone), 80, 24, nil, nil)
	if game.viewPly != 1 {
		t.Errorf("view ply = %d, want 1 after scrolling up", game.viewPly)
	}
}

func TestOnlineOfferButtonLabelsAnimateAndDisable(t *testing.T) {
	game := onlineGameState{pendingDraw: true, pendingTakeback: true, state: lichessapi.GameState{Status: "started"}}
	if got := game.drawOfferLabel(); got != "[🤝] Offering draw." {
		t.Errorf("draw label = %q", got)
	}
	if got := game.takebackLabel(); got != "[🔙] Requesting takeback." {
		t.Errorf("takeback label = %q", got)
	}
	game.offerAnimation = 2
	if got := game.drawOfferLabel(); got != "[🤝] Offering draw..." {
		t.Errorf("draw animation label = %q", got)
	}
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("expected board layout")
	}
	x, y := onlineActionButtonPosition(layout, "draw")
	game.handleMouse(tcell.NewEventMouse(x, y, tcell.ButtonNone, tcell.ModNone), 80, 24, nil, nil)
	if game.hoveredDraw {
		t.Fatal("pending draw button must not highlight")
	}
}

func TestOnlineGameRejectsGameWithoutConnectedAccount(t *testing.T) {
	_, err := newOnlineGame(lichessapi.GameFullEvent{Game: lichessapi.Game{White: lichessapi.Player{ID: "white"}, Black: lichessapi.Player{ID: "black"}}, InitialFEN: "startpos"}, "me", "Me", chess.WOOD)
	if err == nil {
		t.Fatal("expected account mismatch error")
	}
}

func TestOnlineGameKeepsManualPerspectiveOnReconnect(t *testing.T) {
	full := lichessapi.GameFullEvent{Game: lichessapi.Game{ID: "game", White: lichessapi.Player{ID: "me"}, Black: lichessapi.Player{ID: "opponent"}}, InitialFEN: "startpos", State: lichessapi.GameState{Status: "started"}}
	game, err := newOnlineGame(full, "me", "Me", chess.WOOD)
	if err != nil {
		t.Fatal(err)
	}
	game.flipBoard()
	game.apply(onlineResult{event: full, accountID: "me", accountName: "Me"})
	if game.perspective != chess.Black {
		t.Errorf("perspective = %s, want Black after reconnect", game.perspective)
	}
}

func TestOnlineGameRendersPlayersAndServerClocks(t *testing.T) {
	game, err := newOnlineGame(lichessapi.GameFullEvent{Game: lichessapi.Game{ID: "game", White: lichessapi.Player{Name: "White", Rating: 1500}, Black: lichessapi.Player{ID: "me", Name: "Black", Rating: 1600}}, InitialFEN: "startpos", State: lichessapi.GameState{WhiteTime: 61000, BlackTime: 59000, Status: "started"}}, "me", "Black", chess.WOOD)
	if err != nil {
		t.Fatal(err)
	}
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 30)
	drawOnlineGame(screen, *game)
	screen.Show()
	cells, width, height := screen.GetContents()
	text := screenText(cells, width, height)
	for _, want := range []string{"○ Black (1600)", "● White (1500)", "01:01", "00:59", "[🤝] Draw offer", "[🔙] Takeback request"} {
		if !strings.Contains(text, want) {
			t.Errorf("online board missing %q", want)
		}
	}
	if !strings.Contains(text, "Move (UCI)") {
		t.Error("online board must render a UCI input box while active")
	}
}

func TestOnlineGameShowsRatingChangesAfterFinish(t *testing.T) {
	changes := lichessapi.RatingChanges{White: 0, Black: -5}
	game := onlineGameState{
		state:         lichessapi.GameState{Status: "resign"},
		ratingChanges: &changes,
	}
	if got := game.playerLine(lichessapi.Player{Name: "White", Rating: 1600}, chess.White); got != "● White (1600) +0 -> 1600" {
		t.Errorf("white rating line = %q", got)
	}
	if got := game.playerLine(lichessapi.Player{Name: "Black", Rating: 1600}, chess.Black); got != "○ Black (1600) -5 -> 1595" {
		t.Errorf("black rating line = %q", got)
	}
}

func TestOnlinePGNFallbackSavesMetadataAndRendersMessage(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("XDG_DATA_HOME", directory)
	started := time.Date(2026, time.October, 24, 12, 34, 56, 0, time.UTC)
	changes := lichessapi.RatingChanges{White: 7, Black: -7}
	game := onlineGameState{
		localGameState: newLocalGameState(chess.WOOD),
		gameID:         "abc123",
		createdAt:      started.UnixMilli(),
		white:          lichessapi.Player{Name: "White", Rating: 1600},
		black:          lichessapi.Player{Name: "Black", Rating: 1700},
		state:          lichessapi.GameState{Status: "timeout", Winner: "white"},
		seek:           lichessapi.Seek{TimeSeconds: 600, IncrementSeconds: 5, Rated: true},
		ratingChanges:  &changes,
	}
	game.history = []string{"e4", "e5"}
	if game.pgnFileDate().Location() != time.Local {
		t.Fatalf("filename time zone = %s, want local", game.pgnFileDate().Location())
	}
	if game.pgnDate().Location() != time.UTC {
		t.Fatalf("PGN time zone = %s, want UTC", game.pgnDate().Location())
	}
	game.apply(onlineResult{action: "save PGN", includeClocks: false, err: context.DeadlineExceeded})
	if !strings.Contains(game.exportMessage, "Lichess export unavailable") {
		t.Fatalf("export message = %q", game.exportMessage)
	}
	path := filepath.Join(directory, "gochs", "games", game.pgnFileDate().Format("20060102-150405.000000000")+"-plain.pgn")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"[Event \"rated rapid game\"]", "[Site \"https://lichess.org/game/abc123\"]", "[GameId \"abc123\"]",
		"[UTCDate \"2026.10.24\"]", "[UTCTime \"12:34:56\"]", "[WhiteElo \"1600\"]", "[BlackElo \"1700\"]",
		"[WhiteRatingDiff \"+7\"]", "[BlackRatingDiff \"-7\"]", "[TimeControl \"600+5\"]", "[Termination \"Time forfeit\"]",
	} {
		if !strings.Contains(string(contents), want) {
			t.Errorf("PGN missing %q:\n%s", want, contents)
		}
	}
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 30)
	drawOnlineGame(screen, game)
	screen.Show()
	cells, width, height := screen.GetContents()
	if !strings.Contains(screenText(cells, width, height), "Lichess export unavailable") {
		t.Fatal("fallback export message was not rendered")
	}
}

func TestLichessMenuStartsOnlinePlay(t *testing.T) {
	state := menuState{page: lichessMenu, lichessAccount: &entity.LichessAccount{Username: "me"}, lichessSelected: 3}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != onlineGameTypeMenu || !state.onlineGame || state.onlineRated {
		t.Errorf("online activation = %#v", state)
	}
	state.handleKey(keyEvent(tcell.KeyDown, 0))
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != timeControlMenu || !state.onlineRated {
		t.Errorf("rated game type = %#v", state)
	}
}

func TestLocalGameClearsOnlineTimeControlMode(t *testing.T) {
	state := menuState{onlineGame: true}
	state.activatePlayItem(localGame)
	if state.onlineGame || state.timeControlSelected != 0 {
		t.Errorf("local game state = %#v", state)
	}
}

func TestLichessMenuResumesActiveOnlineGame(t *testing.T) {
	state := menuState{page: lichessMenu, lichessAccount: &entity.LichessAccount{Username: "me"}, lichessSelected: 3, onlineActive: true}
	if !state.handleKey(keyEvent(tcell.KeyEnter, 0)) || state.result != OnlineGame {
		t.Errorf("online resume state = %#v", state)
	}
}
