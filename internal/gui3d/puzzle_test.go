package gui3d

import (
	"testing"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
)

func TestPuzzleSessionSolvesAndRecordsOnce(t *testing.T) {
	puzzle := testPuzzle(t)
	session, err := newPuzzleSession(puzzle)
	if err != nil {
		t.Fatal(err)
	}
	var outcomes []entity.PuzzleOutcome
	record := func(_ entity.Puzzle, outcome entity.PuzzleOutcome) (entity.TrainingResult, error) {
		outcomes = append(outcomes, outcome)
		return entity.TrainingResult{}, nil
	}
	move := mustUCI(t, session.board, "e2e4")
	correct, solved, err := session.submit(move, record)
	if err != nil || !correct || !solved {
		t.Fatalf("submit = (%t, %t, %v), want (true, true, nil)", correct, solved, err)
	}
	if len(outcomes) != 1 || outcomes[0] != entity.PuzzleSolved {
		t.Fatalf("outcomes = %v, want [solved]", outcomes)
	}
}

func TestPuzzleSessionWrongLegalMoveRecordsFailure(t *testing.T) {
	session, err := newPuzzleSession(testPuzzle(t))
	if err != nil {
		t.Fatal(err)
	}
	var outcomes []entity.PuzzleOutcome
	record := func(_ entity.Puzzle, outcome entity.PuzzleOutcome) (entity.TrainingResult, error) {
		outcomes = append(outcomes, outcome)
		return entity.TrainingResult{}, nil
	}
	correct, solved, err := session.submit(mustUCI(t, session.board, "d2d4"), record)
	if err != nil || correct || solved {
		t.Fatalf("submit = (%t, %t, %v), want (false, false, nil)", correct, solved, err)
	}
	if len(outcomes) != 1 || outcomes[0] != entity.PuzzleFailed {
		t.Fatalf("outcomes = %v, want [failed]", outcomes)
	}
}

func testPuzzle(t *testing.T) entity.Puzzle {
	t.Helper()
	board, err := chess.ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	opening := mustUCI(t, board, "e7e5")
	if err := board.MakeMove(opening); err != nil {
		t.Fatal(err)
	}
	return entity.Puzzle{ID: "test", FEN: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 1", Moves: []chess.Move{opening, mustUCI(t, board, "e2e4")}}
}

func mustUCI(t *testing.T, board chess.Board, uci string) chess.Move {
	t.Helper()
	move, err := board.ParseUCI(uci)
	if err != nil {
		t.Fatal(err)
	}
	return move
}
