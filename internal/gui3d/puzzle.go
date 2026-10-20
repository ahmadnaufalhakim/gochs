package gui3d

import (
	"fmt"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
)

type puzzleSession struct {
	puzzle      entity.Puzzle
	board       chess.Board
	solverColor chess.PieceColor
	nextMove    int
	recorded    bool
}

func newPuzzleSession(puzzle entity.Puzzle) (puzzleSession, error) {
	if len(puzzle.Moves) < 2 {
		return puzzleSession{}, fmt.Errorf("puzzle %s has fewer than two moves", puzzle.ID)
	}
	board, err := chess.ParseFEN(puzzle.FEN)
	if err != nil {
		return puzzleSession{}, err
	}
	if err := board.MakeMove(puzzle.Moves[0]); err != nil {
		return puzzleSession{}, fmt.Errorf("apply puzzle opening move: %w", err)
	}
	return puzzleSession{
		puzzle:      puzzle,
		board:       board,
		solverColor: board.ColorToMove,
		nextMove:    1,
	}, nil
}

// submit applies a solver move and all expected replies. The caller must only
// pass legal moves from the current puzzle position.
func (p *puzzleSession) submit(move chess.Move, record PuzzleAttemptRecorder) (correct, solved bool, err error) {
	if p.nextMove >= len(p.puzzle.Moves) {
		return false, true, nil
	}
	if move != p.puzzle.Moves[p.nextMove] {
		p.record(entity.PuzzleFailed, record)
		return false, false, nil
	}
	if err := p.board.MakeMove(move); err != nil {
		return false, false, err
	}
	p.nextMove++
	for p.nextMove < len(p.puzzle.Moves) && p.board.ColorToMove != p.solverColor {
		if err := p.board.MakeMove(p.puzzle.Moves[p.nextMove]); err != nil {
			return false, false, err
		}
		p.nextMove++
	}
	if p.nextMove == len(p.puzzle.Moves) {
		p.record(entity.PuzzleSolved, record)
		return true, true, nil
	}
	return true, false, nil
}

func (p *puzzleSession) record(outcome entity.PuzzleOutcome, record PuzzleAttemptRecorder) {
	if p.recorded || record == nil {
		return
	}
	_, _ = record(p.puzzle, outcome)
	p.recorded = true
}
