package gui

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/engine"
)

type engineResult struct {
	move string
	err  error
}

func (g *localGameState) startStockfish(path string, color stockfishColor, difficulty int) error {
	computerColor := chess.Black
	if color == stockfishBlack || (color == stockfishRandom && rand.Intn(2) == 0) {
		computerColor = chess.White
	}
	client, err := engine.Start(context.Background(), engine.Config{Path: path, Difficulty: difficulty})
	if err != nil {
		return err
	}

	g.computerColor = new(chess.PieceColor)
	*g.computerColor = computerColor
	g.perspective = computerColor.Opponent()
	g.stockfish = client
	g.engineResults = make(chan engineResult, 1)
	g.engineDifficulty = difficulty
	g.engineRandom = rand.Intn
	g.startEngineTurn()
	return nil
}

func (g localGameState) isComputerTurn() bool {
	return g.computerColor != nil && *g.computerColor == g.board.ColorToMove
}

func (g *localGameState) startEngineTurn() {
	if g.stockfish == nil || !g.isComputerTurn() || g.engineThinking || g.hasTerminalResult() {
		return
	}
	g.engineThinking = true
	moves := make([]string, len(g.moves))
	for index, move := range g.moves {
		moves[index] = move.UCI()
	}
	search := engine.Search{
		WhiteTime:      g.clockRemaining[chess.White],
		BlackTime:      g.clockRemaining[chess.Black],
		WhiteIncrement: g.timeControl.increment,
		BlackIncrement: g.timeControl.increment,
	}
	policy, err := engine.DifficultyPolicy(g.engineDifficulty)
	if err != nil {
		g.engineThinking = false
		g.message = err.Error()
		return
	}
	search.MaxMoveTime = policy.MoveTime
	if !g.timeControl.isTimed() {
		search.MoveTime = policy.MoveTime
	}

	go func() {
		move, err := g.stockfish.BestMove(context.Background(), moves, search)
		g.engineResults <- engineResult{move: move, err: err}
	}()
}

func (g *localGameState) applyEngineResult(result engineResult) {
	g.engineThinking = false
	if g.hasTerminalResult() {
		return
	}
	if result.err != nil {
		g.message = fmt.Sprintf("Stockfish error: %v", result.err)
		return
	}
	legalMoves := chess.GenerateLegalMoves(g.board)
	legalUCIs := make([]string, len(legalMoves))
	for index, legalMove := range legalMoves {
		legalUCIs[index] = legalMove.UCI()
	}
	policy, err := engine.DifficultyPolicy(g.engineDifficulty)
	if err != nil {
		g.message = err.Error()
		return
	}
	randomInt := g.engineRandom
	if randomInt == nil {
		randomInt = rand.Intn
	}
	move, err := g.board.ParseUCI(engine.SelectMove(policy, result.move, legalUCIs, randomInt))
	if err != nil {
		g.message = fmt.Sprintf("Stockfish returned invalid move: %v", err)
		return
	}
	g.makeMove(move)
}

func (g *localGameState) closeStockfish() {
	if g.stockfish != nil {
		_ = g.stockfish.Close()
		g.stockfish = nil
	}
}

func (g localGameState) stockfishStatus() string {
	if g.engineThinking {
		return "Stockfish is thinking..."
	}
	return ""
}
