package gui

import (
	"strings"
	"testing"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	"github.com/gdamore/tcell/v2"
)

func TestPuzzleRevealsInitialMoveBeforeSolverTurn(t *testing.T) {
	now := time.Now()
	puzzle := testPuzzle(t, "e2e4 e7e5 g1f3 b8c6")
	state, err := newPuzzleState(puzzle, chess.WOOD, now)
	if err != nil {
		t.Fatal(err)
	}
	if state.solverColor != chess.Black || state.title() != "Black to move" {
		t.Errorf("solver state = %s, %q", state.solverColor, state.title())
	}
	state.advance(now.Add(puzzleInitialDelay - time.Millisecond))
	if len(state.moves) != 0 || state.phase != puzzlePreview {
		t.Fatalf("preview state changed early: %#v", state)
	}
	state.advance(now.Add(puzzleInitialDelay))
	if state.phase != puzzleSolverTurn || len(state.moves) != 1 || state.moves[0].UCI() != "e2e4" {
		t.Errorf("initial reveal = phase %d, moves %v", state.phase, state.moves)
	}
}

func TestPuzzleCorrectMoveWaitsForOpponentReply(t *testing.T) {
	now := time.Now()
	state := revealedPuzzle(t, now, "e2e4 e7e5 g1f3 b8c6")
	state.submitMove(state.puzzle.Moves[1], now)
	if state.phase != puzzleOpponentReply || state.message != "Best move! Keep going.." {
		t.Fatalf("correct move state = %#v", state)
	}
	if got := state.board.PieceAt(guiSquare(t, "e5")); got != (chess.Piece{Color: chess.Black, Type: chess.Pawn}) {
		t.Errorf("piece at e5 = %#v", got)
	}
	state.advance(now.Add(puzzleReplyDelay - time.Millisecond))
	if len(state.moves) != 2 {
		t.Errorf("moves before reply = %d, want 2", len(state.moves))
	}
	state.advance(now.Add(puzzleReplyDelay))
	if state.phase != puzzleSolverTurn || len(state.moves) != 3 || state.moves[2].UCI() != "g1f3" {
		t.Errorf("reply state = phase %d, moves %v", state.phase, state.moves)
	}
}

func TestPuzzleWrongMoveRollsBack(t *testing.T) {
	now := time.Now()
	state := revealedPuzzle(t, now, "e2e4 e7e5")
	wrongMove, err := state.board.ParseUCI("c7c5")
	if err != nil {
		t.Fatal(err)
	}
	state.submitMove(wrongMove, now)
	if state.phase != puzzleWrongMove || state.message != "That's not the move! Try again.." {
		t.Fatalf("wrong move state = %#v", state)
	}
	state.advance(now.Add(puzzleWrongDelay))
	if state.phase != puzzleSolverTurn || state.board.PieceAt(guiSquare(t, "c7")).Type != chess.Pawn || state.board.PieceAt(guiSquare(t, "c5")).Type != chess.PieceNone {
		t.Errorf("rolled back board = %#v", state.board)
	}
	if len(state.moves) != 1 {
		t.Errorf("wrong move was recorded: %v", state.moves)
	}
}

func TestPuzzleAcceptsAlternativeFinalMate(t *testing.T) {
	const fen = "1k6/2Q5/2K5/8/8/8/8/8 b - - 0 1"
	board, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	moves, err := chess.ParseUCIMoves(board, "b8a8 c7a7")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	state, err := newPuzzleState(entity.Puzzle{
		ID:     "multiple-mates",
		FEN:    fen,
		Moves:  moves,
		Themes: []entity.Theme{{ID: "mate", DisplayName: "Checkmate"}},
	}, chess.WOOD, now)
	if err != nil {
		t.Fatal(err)
	}
	state.advance(now.Add(puzzleInitialDelay))
	alternative, err := state.board.ParseUCI("c7b7")
	if err != nil {
		t.Fatal(err)
	}
	state.submitMove(alternative, now)
	if state.phase != puzzleSolved {
		t.Fatalf("alternative mate phase = %d, want solved", state.phase)
	}
	if state.moves[len(state.moves)-1] != alternative {
		t.Errorf("recorded move = %s, want %s", state.moves[len(state.moves)-1], alternative)
	}
}

func TestPuzzleRejectsAlternativeMateWithoutMateTheme(t *testing.T) {
	const fen = "1k6/2Q5/2K5/8/8/8/8/8 b - - 0 1"
	board, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	moves, err := chess.ParseUCIMoves(board, "b8a8 c7a7")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	state, err := newPuzzleState(entity.Puzzle{ID: "non-mate", FEN: fen, Moves: moves}, chess.WOOD, now)
	if err != nil {
		t.Fatal(err)
	}
	state.advance(now.Add(puzzleInitialDelay))
	alternative, err := state.board.ParseUCI("c7b7")
	if err != nil {
		t.Fatal(err)
	}
	state.submitMove(alternative, now)
	if state.phase != puzzleWrongMove {
		t.Errorf("non-mate alternative phase = %d, want wrong move", state.phase)
	}
}

func TestPuzzleHelpDeadlineStartsAtLoad(t *testing.T) {
	now := time.Now()
	state := revealedPuzzle(t, now, "e2e4 e7e5")
	state.helpAvailableAt = time.Now().Add(-time.Millisecond)
	if !state.helpAvailable() {
		t.Fatal("help is unavailable after the fixed load deadline")
	}
	state.submitMove(state.puzzle.Moves[1], now)
	if state.helpAvailable() {
		t.Fatal("help is available while the opponent reply is pending")
	}
	state.advance(now.Add(puzzleReplyDelay))
	if state.phase != puzzleSolved {
		t.Errorf("two-ply puzzle phase = %d, want solved", state.phase)
	}
}

func TestPuzzleHintsAddBlueSourceThenDestination(t *testing.T) {
	now := time.Now()
	state := revealedPuzzle(t, now, "e2e4 e7e5")
	state.hintCount = 1
	state.hintMoveIndex = state.nextSolutionMove
	_, sourceColor := puzzleSquareStyle(state, state.board, guiSquare(t, "e7"), nil)
	_, destinationColor := puzzleSquareStyle(state, state.board, guiSquare(t, "e5"), nil)
	if sourceColor.B != 255 || destinationColor.B == 255 {
		t.Fatalf("first hint colors = source %#v, destination %#v", sourceColor, destinationColor)
	}
	state.hintCount = 2
	_, destinationColor = puzzleSquareStyle(state, state.board, guiSquare(t, "e5"), nil)
	if destinationColor.B != 255 {
		t.Errorf("second hint destination color = %#v, want blue 255", destinationColor)
	}
	state.nextSolutionMove++
	_, sourceColor = puzzleSquareStyle(state, state.board, guiSquare(t, "e7"), nil)
	if sourceColor.B == 255 {
		t.Error("hint remained visible after the expected move advanced")
	}
}

func TestPuzzleCheckSquareUsesLocalGameRed(t *testing.T) {
	board, err := chess.ParseFEN("4k3/4R3/8/8/8/8/8/K7 b - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	_, color := puzzleSquareStyle(puzzleState{theme: chess.WOOD}, board, guiSquare(t, "e8"), nil)
	if color != (chess.RGB{R: 205, G: 50, B: 50}) {
		t.Errorf("check color = %#v, want local-game red", color)
	}
}

func TestPuzzleViewSolutionPreservesReplayLine(t *testing.T) {
	now := time.Now()
	state := revealedPuzzle(t, now, "e2e4 e7e5 g1f3 b8c6")
	state.viewSolution()
	if state.phase != puzzleSolutionViewed || len(state.moves) != 4 || state.viewPly != 2 {
		t.Fatalf("solution state = phase %d, moves %d, ply %d", state.phase, len(state.moves), state.viewPly)
	}
	if move, ok := state.viewedMove(); !ok || move.UCI() != "e7e5" {
		t.Errorf("shown solution move = %v, %t", move, ok)
	}
}

func TestDrawCompletedPuzzleShowsThemesAndContinue(t *testing.T) {
	now := time.Now()
	state := revealedPuzzle(t, now, "e2e4 e7e5")
	state.submitMove(state.puzzle.Moves[1], now)
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)
	drawPuzzle(screen, state)
	screen.Show()
	contents, width, height := screen.GetContents()
	text := screenText(contents, width, height)
	for _, want := range []string{"Success!", "┏━ Puzzle themes", "Fork", "[➡️] Continue training"} {
		if !strings.Contains(text, want) {
			t.Errorf("completed puzzle rendering does not contain %q", want)
		}
	}
}

func revealedPuzzle(t *testing.T, now time.Time, sequence string) puzzleState {
	t.Helper()
	state, err := newPuzzleState(testPuzzle(t, sequence), chess.WOOD, now)
	if err != nil {
		t.Fatal(err)
	}
	state.advance(now.Add(puzzleInitialDelay))
	return state
}

func testPuzzle(t *testing.T, sequence string) entity.Puzzle {
	t.Helper()
	const fen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	board, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	moves, err := chess.ParseUCIMoves(board, sequence)
	if err != nil {
		t.Fatal(err)
	}
	return entity.Puzzle{
		ID:     "test",
		FEN:    fen,
		Moves:  moves,
		Rating: 1500,
		Themes: []entity.Theme{{ID: "fork", DisplayName: "Fork"}},
	}
}
