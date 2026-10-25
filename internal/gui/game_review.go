package gui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/engine"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

const gameReviewDepth = 16

type moveClassification uint8

const (
	moveBest moveClassification = iota
	moveExcellent
	moveOkay
	moveInaccuracy
	moveMistake
	moveBlunder
)

func (c moveClassification) String() string {
	return [...]string{"Best", "Excellent", "Okay", "Inaccuracy", "Mistake", "Blunder"}[c]
}

func (c moveClassification) marker() string {
	switch c {
	case moveBest:
		return ""
	case moveExcellent:
		return ""
	case moveInaccuracy:
		return "?!"
	case moveMistake:
		return "?"
	case moveBlunder:
		return "??"
	default:
		return ""
	}
}

type positionEvaluation struct {
	centipawns int
	mate       int
	isMate     bool
}

type moveReview struct {
	classification moveClassification
	before         positionEvaluation
	after          positionEvaluation
	bestMove       chess.Move
	bestSAN        string
	bestLine       []chess.Move
}

type reviewVariation struct {
	sourcePly int
	positions []chess.Board
	viewPly   int
}

type gameReview struct {
	active          bool
	analyzing       bool
	requested       bool
	progress, total int
	moves           []moveReview
	variation       *reviewVariation
	hoveredStart    bool
	hoveredBestMove bool
	message         string
}

type reviewResult struct {
	target   *localGameState
	progress int
	total    int
	moves    []moveReview
	err      error
	done     bool
}

func startGameReview(ctx context.Context, target *localGameState, stockfishPath string, results chan<- reviewResult) {
	positions := append([]chess.Board(nil), target.positions...)
	moves := append([]chess.Move(nil), target.moves...)
	go func() {
		client, err := engine.Start(context.Background(), engine.Config{Path: stockfishPath, Difficulty: 8})
		if err != nil {
			sendReviewResult(ctx, results, reviewResult{target: target, err: err, done: true})
			return
		}
		defer client.Close()

		analyses := make([]engine.Analysis, len(positions))
		for index, position := range positions {
			if analysis, terminal := terminalPositionAnalysis(position); terminal {
				analyses[index] = analysis
			} else {
				fen, err := position.FEN()
				if err == nil {
					analyses[index], err = client.Analyze(ctx, fen, gameReviewDepth)
				}
				if err != nil {
					sendReviewResult(ctx, results, reviewResult{target: target, err: err, done: true})
					return
				}
			}
			sendReviewResult(ctx, results, reviewResult{target: target, progress: index + 1, total: len(positions)})
		}

		reviews, err := buildMoveReviews(positions, moves, analyses)
		sendReviewResult(ctx, results, reviewResult{target: target, progress: len(positions), total: len(positions), moves: reviews, err: err, done: true})
	}()
}

func terminalPositionAnalysis(position chess.Board) (engine.Analysis, bool) {
	if len(chess.GenerateLegalMoves(position)) != 0 {
		return engine.Analysis{}, false
	}
	if position.IsColorInCheck(position.ColorToMove) {
		return engine.Analysis{Depth: 1, Score: engine.Score{Mate: -1, IsMate: true}}, true
	}
	return engine.Analysis{Depth: 1}, true
}

func beginGameReview(target *localGameState, stockfishPath string, results chan<- reviewResult) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	startGameReview(ctx, target, stockfishPath, results)
	return cancel
}

func sendReviewResult(ctx context.Context, results chan<- reviewResult, result reviewResult) {
	if !result.done {
		select {
		case results <- result:
		default:
		}
		return
	}
	select {
	case results <- result:
	case <-ctx.Done():
	}
}

func buildMoveReviews(positions []chess.Board, moves []chess.Move, analyses []engine.Analysis) ([]moveReview, error) {
	if len(positions) != len(moves)+1 || len(analyses) != len(positions) {
		return nil, fmt.Errorf("review positions and analyses do not match moves")
	}
	reviews := make([]moveReview, len(moves))
	for index, move := range moves {
		before := evaluationForWhite(positions[index], analyses[index].Score)
		after := evaluationForWhite(positions[index+1], analyses[index+1].Score)
		bestMove, err := positions[index].ParseUCI(analyses[index].BestMove)
		if err != nil {
			return nil, fmt.Errorf("parse Stockfish best move: %w", err)
		}
		bestSAN, err := positions[index].SAN(bestMove)
		if err != nil {
			return nil, err
		}
		line, err := principalVariation(positions[index], analyses[index].PV)
		if err != nil {
			return nil, err
		}
		loss := evaluationLoss(positions[index].ColorToMove, before, after)
		reviews[index] = moveReview{
			classification: classifyMove(move, bestMove, loss),
			before:         before,
			after:          after,
			bestMove:       bestMove,
			bestSAN:        bestSAN,
			bestLine:       line,
		}
	}
	return reviews, nil
}

func evaluationForWhite(board chess.Board, score engine.Score) positionEvaluation {
	evaluation := positionEvaluation{centipawns: score.Centipawns, mate: score.Mate, isMate: score.IsMate}
	if board.ColorToMove == chess.Black {
		evaluation.centipawns = -evaluation.centipawns
		evaluation.mate = -evaluation.mate
	}
	return evaluation
}

func evaluationLoss(mover chess.PieceColor, before, after positionEvaluation) int {
	beforeValue := evaluationValue(before)
	afterValue := evaluationValue(after)
	if mover == chess.Black {
		beforeValue = -beforeValue
		afterValue = -afterValue
	}
	return max(0, beforeValue-afterValue)
}

func evaluationValue(evaluation positionEvaluation) int {
	if evaluation.isMate {
		if evaluation.mate > 0 {
			return 100000 - evaluation.mate
		}
		return -100000 - evaluation.mate
	}
	return evaluation.centipawns
}

func classifyMove(move, bestMove chess.Move, loss int) moveClassification {
	if move == bestMove || loss <= 10 {
		return moveBest
	}
	if loss <= 30 {
		return moveExcellent
	}
	if loss <= 80 {
		return moveOkay
	}
	if loss <= 180 {
		return moveInaccuracy
	}
	if loss <= 350 {
		return moveMistake
	}
	return moveBlunder
}

func principalVariation(board chess.Board, uciMoves []string) ([]chess.Move, error) {
	line := make([]chess.Move, 0, min(8, len(uciMoves)))
	for _, uci := range uciMoves {
		if len(line) == 8 {
			break
		}
		move, err := board.ParseUCI(uci)
		if err != nil {
			return nil, fmt.Errorf("parse Stockfish variation: %w", err)
		}
		line = append(line, move)
		if err := board.MakeMove(move); err != nil {
			return nil, err
		}
	}
	return line, nil
}

func formatEvaluation(evaluation positionEvaluation) string {
	if evaluation.isMate {
		return fmt.Sprintf("#%d", evaluation.mate)
	}
	return fmt.Sprintf("%+.2f", float64(evaluation.centipawns)/100)
}

func (g *localGameState) startReview() {
	if g.review.analyzing {
		return
	}
	g.review = gameReview{active: true, analyzing: true, requested: true, total: len(g.positions)}
}

func (g *localGameState) applyReviewResult(result reviewResult) {
	if !g.review.analyzing {
		return
	}
	g.review.progress = result.progress
	g.review.total = result.total
	if !result.done {
		return
	}
	g.review.analyzing = false
	if result.err != nil {
		g.review.message = "Game Review failed: " + result.err.Error()
		return
	}
	g.review.moves = result.moves
}

func (g *localGameState) leaveReview() {
	g.review = gameReview{}
}

func (g *localGameState) selectedMoveReview() (*moveReview, bool) {
	if !g.review.active || g.review.analyzing || g.viewPly < 1 || g.viewPly > len(g.review.moves) {
		return nil, false
	}
	return &g.review.moves[g.viewPly-1], true
}

func (g *localGameState) startVariation() {
	review, ok := g.selectedMoveReview()
	if !ok || review.classification == moveBest || len(review.bestLine) == 0 {
		return
	}
	board := g.positions[g.viewPly-1]
	positions := []chess.Board{board}
	for _, move := range review.bestLine {
		if err := board.MakeMove(move); err != nil {
			return
		}
		positions = append(positions, board)
	}
	g.review.variation = &reviewVariation{sourcePly: g.viewPly, positions: positions}
}

func (g localGameState) reviewVerdict() string {
	review, ok := g.selectedMoveReview()
	if !ok {
		return "Select a move to inspect its review."
	}
	if review.classification == moveBest {
		return fmt.Sprintf("%s is the best move.", g.history[g.viewPly-1])
	}
	if review.classification == moveOkay {
		return fmt.Sprintf("%s is an okay move.", g.history[g.viewPly-1])
	}
	return fmt.Sprintf("%s is %s.", g.history[g.viewPly-1], strings.ToLower(review.classification.String()))
}

func (g localGameState) annotatedSAN(index int) string {
	if !g.review.active || g.review.analyzing || index < 0 || index >= len(g.history) || index >= len(g.review.moves) {
		return g.history[index]
	}
	marker := g.review.moves[index].classification.marker()
	if marker == "" || runewidth.StringWidth(g.history[index]+marker) > sanWidth {
		return g.history[index]
	}
	return g.history[index] + marker
}

func gameReviewButtonPosition(layout boardLayout) (int, int) {
	return layout.x - 2, layout.y + 12
}

func gameReviewButtonLabel() string { return "[🔎] Game Review" }

func gameReviewButtonAt(x, y int, layout boardLayout) bool {
	buttonX, buttonY := gameReviewButtonPosition(layout)
	return y == buttonY && x >= buttonX && x < buttonX+runewidth.StringWidth(gameReviewButtonLabel())
}

func drawGameReviewButton(screen tcell.Screen, game localGameState, layout boardLayout) {
	x, y := gameReviewButtonPosition(layout)
	style := backgroundStyle
	if game.review.hoveredStart {
		style = selectedStyle
	}
	drawString(screen, x, y, gameReviewButtonLabel(), style)
}

func (g *localGameState) handleGameReviewMouse(event *tcell.EventMouse, layout boardLayout, panelY int) {
	x, y := event.Position()
	if event.Buttons() == tcell.ButtonNone {
		g.hoveredFlip = flipButtonAt(x, y, layout)
		g.review.hoveredBestMove = bestMoveAt(x, y, layout, panelY, *g)
		return
	}
	buttons := event.Buttons()
	if buttons == tcell.Button1 {
		if flipButtonAt(x, y, layout) {
			g.flipBoard()
			return
		}
		if bestMoveAt(x, y, layout, panelY, *g) {
			g.startVariation()
			return
		}
		if ply, ok := g.historyPlyAt(x, y, layout); ok {
			g.review.variation = nil
			g.viewPly = ply
			g.syncHistoryAfterClick(y - (layout.y + 1))
		}
		return
	}
	if buttons&tcell.WheelUp != 0 || buttons&tcell.WheelDown != 0 {
		if _, ok := squareAtScreenPosition(x, y, layout, g.perspective); ok {
			direction := 1
			if buttons&tcell.WheelUp != 0 {
				direction = -1
			}
			g.moveReplay(direction)
			return
		}
		if historyAreaAt(x, y, layout) {
			if buttons&tcell.WheelUp != 0 {
				g.historyScroll = max(0, g.historyScroll-1)
			} else {
				g.historyScroll = min(g.historyMaxScroll(), g.historyScroll+1)
			}
		}
	}
}

func drawGameReviewPanel(screen tcell.Screen, game localGameState, layout boardLayout, y int, compact bool) {
	x := layout.x - 2
	if game.review.analyzing {
		drawString(screen, x, y, fmt.Sprintf("Game Review: analyzing %d/%d positions", game.review.progress, game.review.total), mutedStyle)
		drawString(screen, x, y+reviewHintOffset(compact), ",/.: sound volume  Esc: return to exports", mutedStyle)
		return
	}
	if game.review.message != "" {
		drawString(screen, x, y, game.review.message, mutedStyle)
		drawString(screen, x, y+reviewHintOffset(compact), "Esc: return to exports", mutedStyle)
		return
	}
	drawString(screen, x, y, game.reviewVerdict(), mutedStyle)
	review, ok := game.selectedMoveReview()
	if ok {
		drawString(screen, x, y+2, "Eval: "+formatEvaluation(review.before)+" -> "+formatEvaluation(review.after), mutedStyle)
		if review.classification != moveBest {
			prefix := "The best move was "
			drawString(screen, x, y+1, prefix, mutedStyle)
			style := backgroundStyle
			if game.review.hoveredBestMove {
				style = selectedStyle
			}
			drawString(screen, x+runewidth.StringWidth(prefix), y+1, "["+review.bestSAN+"]", style)
		}
	}
	if game.review.variation != nil && !compact {
		drawString(screen, x, y+4, "Variation: scroll board to navigate", mutedStyle)
	}
	drawString(screen, x, y+reviewHintOffset(compact), ",/.: sound volume  Esc: return to exports", mutedStyle)
}

func reviewHintOffset(compact bool) int {
	if compact {
		return 3
	}
	return 6
}

func bestMoveAt(x, y int, layout boardLayout, panelY int, game localGameState) bool {
	review, ok := game.selectedMoveReview()
	if !ok || review.classification == moveBest || y != panelY+1 {
		return false
	}
	prefixWidth := runewidth.StringWidth("The best move was ")
	startX := layout.x - 2 + prefixWidth
	width := runewidth.StringWidth("[" + review.bestSAN + "]")
	return x >= startX && x < startX+width
}
