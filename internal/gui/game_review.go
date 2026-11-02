package gui

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/engine"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

const (
	gameReviewDepth             = 16
	moveVerificationDepth       = 20
	greatWinningScore           = 150
	greatNotWinningScore        = 50
	brilliantTrivialScore       = 700
	brilliantAlternativeMaxLoss = 125
)

type moveClassification uint8

const (
	moveBrilliant moveClassification = iota
	moveGreat
	moveBest
	moveExcellent
	moveOkay
	moveInaccuracy
	moveMistake
	moveBlunder
)

func (c moveClassification) String() string {
	return [...]string{"Brilliant", "Great", "Best", "Excellent", "Okay", "Inaccuracy", "Mistake", "Blunder"}[c]
}

func (c moveClassification) marker() string {
	switch c {
	case moveBrilliant:
		return "!!"
	case moveGreat:
		return "!"
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

type brilliantVerification struct {
	before engine.Analysis
	after  engine.Analysis
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
	playedBest     bool
	equivalentBest bool
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
	complete        bool
	requested       bool
	progress, total int
	moves           []moveReview
	variation       *reviewVariation
	hoveredStart    bool
	hoveredBestMove bool
	message         string
}

type reviewResult struct {
	target     *localGameState
	generation uint64
	progress   int
	total      int
	moves      []moveReview
	err        error
	done       bool
}

func startGameReview(ctx context.Context, target *localGameState, generation uint64, stockfishPath string, results chan<- reviewResult) {
	positions := append([]chess.Board(nil), target.positions...)
	moves := append([]chess.Move(nil), target.moves...)
	go func() {
		client, err := engine.Start(ctx, engine.Config{Path: stockfishPath, Difficulty: 8, MultiPV: 2})
		if err != nil {
			sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, err: err, done: true})
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
					sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, err: err, done: true})
					return
				}
			}
			sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, progress: index + 1, total: len(positions)})
		}

		greatAnalyses, verifiedGreatAnalyses, greatCandidateCount, err := verifyGreatMoves(ctx, client, positions, moves, analyses, results, target, generation)
		if err != nil {
			sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, err: err, done: true})
			return
		}
		brilliantAnalyses, brilliantCandidateCount, err := verifyBrilliantMoves(ctx, client, positions, moves, analyses, verifiedGreatAnalyses, results, target, generation, greatCandidateCount)
		if err != nil {
			sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, err: err, done: true})
			return
		}
		reviews, err := buildMoveReviewsWithClassifications(positions, moves, analyses, greatAnalyses, brilliantAnalyses)
		total := len(positions) + greatCandidateCount + brilliantCandidateCount
		sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, progress: total, total: total, moves: reviews, err: err, done: true})
	}()
}

func verifyGreatMoves(ctx context.Context, client *engine.Client, positions []chess.Board, moves []chess.Move, analyses []engine.Analysis, results chan<- reviewResult, target *localGameState, generation uint64) (map[int]engine.Analysis, map[int]engine.Analysis, int, error) {
	candidates := make([]int, 0)
	for index, move := range moves {
		if isGreatMove(positions[index], move, analyses[index], gameReviewDepth) {
			candidates = append(candidates, index)
		}
	}
	confirmed := make(map[int]engine.Analysis, len(candidates))
	verified := make(map[int]engine.Analysis, len(candidates))
	sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, progress: len(positions), total: len(positions) + len(candidates)})
	for index, candidate := range candidates {
		fen, err := positions[candidate].FEN()
		if err != nil {
			return nil, nil, 0, err
		}
		analysis, err := client.Analyze(ctx, fen, moveVerificationDepth)
		if err != nil {
			return nil, nil, 0, err
		}
		verified[candidate] = analysis
		if isGreatMove(positions[candidate], moves[candidate], analysis, moveVerificationDepth) {
			confirmed[candidate] = analysis
		}
		sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, progress: len(positions) + index + 1, total: len(positions) + len(candidates)})
	}
	return confirmed, verified, len(candidates), nil
}

func verifyBrilliantMoves(ctx context.Context, client *engine.Client, positions []chess.Board, moves []chess.Move, analyses []engine.Analysis, verifiedGreatAnalyses map[int]engine.Analysis, results chan<- reviewResult, target *localGameState, generation uint64, completed int) (map[int]brilliantVerification, int, error) {
	candidates := make([]int, 0)
	for index, move := range moves {
		if isBrilliantMove(positions[index], move, analyses[index], positions[index+1], analyses[index+1], gameReviewDepth) {
			candidates = append(candidates, index)
		}
	}
	steps := brilliantVerificationSteps(candidates, verifiedGreatAnalyses)
	confirmed := make(map[int]brilliantVerification, len(candidates))
	base := len(positions) + completed
	sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, progress: base, total: base + steps})
	progress := 0
	for _, candidate := range candidates {
		before, ok := verifiedGreatAnalyses[candidate]
		if !ok {
			fen, err := positions[candidate].FEN()
			if err != nil {
				return nil, 0, err
			}
			before, err = client.Analyze(ctx, fen, moveVerificationDepth)
			if err != nil {
				return nil, 0, err
			}
			progress++
			sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, progress: base + progress, total: base + steps})
		}
		after, terminal := terminalPositionAnalysis(positions[candidate+1])
		if !terminal {
			fen, err := positions[candidate+1].FEN()
			if err != nil {
				return nil, 0, err
			}
			after, err = client.Analyze(ctx, fen, moveVerificationDepth)
			if err != nil {
				return nil, 0, err
			}
		}
		progress++
		sendReviewResult(ctx, results, reviewResult{target: target, generation: generation, progress: base + progress, total: base + steps})
		if isBrilliantMove(positions[candidate], moves[candidate], before, positions[candidate+1], after, moveVerificationDepth) {
			confirmed[candidate] = brilliantVerification{before: before, after: after}
		}
	}
	return confirmed, steps, nil
}

func brilliantVerificationSteps(candidates []int, verifiedAnalyses map[int]engine.Analysis) int {
	steps := len(candidates) * 2
	for _, candidate := range candidates {
		if _, ok := verifiedAnalyses[candidate]; ok {
			steps--
		}
	}
	return steps
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
	startGameReview(ctx, target, target.reviewGeneration, stockfishPath, results)
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
	return buildMoveReviewsWithClassifications(positions, moves, analyses, nil, nil)
}

func buildMoveReviewsWithClassifications(positions []chess.Board, moves []chess.Move, analyses []engine.Analysis, greatAnalyses map[int]engine.Analysis, brilliantAnalyses map[int]brilliantVerification) ([]moveReview, error) {
	if len(positions) != len(moves)+1 || len(analyses) != len(positions) {
		return nil, fmt.Errorf("review positions and analyses do not match moves")
	}
	reviews := make([]moveReview, len(moves))
	for index, move := range moves {
		analysis := analyses[index]
		if verified, ok := greatAnalyses[index]; ok {
			analysis = verified
		}
		afterAnalysis := analyses[index+1]
		if verified, ok := brilliantAnalyses[index]; ok {
			analysis = verified.before
			afterAnalysis = verified.after
		}
		before := evaluationForWhite(positions[index], analysis.Score)
		after := evaluationForWhite(positions[index+1], afterAnalysis.Score)
		bestMove, err := positions[index].ParseUCI(analysis.BestMove)
		if err != nil {
			return nil, fmt.Errorf("parse Stockfish best move: %w", err)
		}
		bestSAN, err := positions[index].SAN(bestMove)
		if err != nil {
			return nil, err
		}
		line, err := principalVariation(positions[index], analysis.PV)
		if err != nil {
			return nil, err
		}
		classification := classifyMove(move, bestMove, positions[index].ColorToMove, before, after)
		if _, ok := brilliantAnalyses[index]; ok {
			classification = moveBrilliant
		} else if _, ok := greatAnalyses[index]; ok {
			classification = moveGreat
		}
		reviews[index] = moveReview{
			classification: classification,
			before:         before,
			after:          after,
			playedBest:     move == bestMove,
			equivalentBest: isEquivalentBrilliantMove(classification, positions[index].ColorToMove, before, after, move == bestMove),
			bestMove:       bestMove,
			bestSAN:        bestSAN,
			bestLine:       line,
		}
	}
	return reviews, nil
}

func isEquivalentBrilliantMove(classification moveClassification, mover chess.PieceColor, before, after positionEvaluation, playedBest bool) bool {
	if classification != moveBrilliant || playedBest {
		return false
	}
	beforeScore := scoreForEvaluation(mover, before)
	afterScore := scoreForEvaluation(mover, after)
	return beforeScore >= greatWinningScore && afterScore >= greatWinningScore && beforeScore-afterScore <= brilliantAlternativeMaxLoss
}

func isBrilliantMove(before chess.Board, played chess.Move, beforeAnalysis engine.Analysis, afterBoard chess.Board, afterAnalysis engine.Analysis, depth int) bool {
	if before.IsColorInCheck(before.ColorToMove) || !analysisAtDepth(afterAnalysis, afterBoard, depth) {
		return false
	}
	mover := before.ColorToMove
	if scoreForColor(afterBoard, afterAnalysis.Score, mover) < 0 {
		return false
	}
	if _, secondary, ok := primaryAndSecondary(beforeAnalysis); ok && secondary.Depth == depth {
		if scoreForColor(before, secondary.Score, mover) >= brilliantTrivialScore {
			return false
		}
	} else if scoreForColor(afterBoard, afterAnalysis.Score, mover) >= brilliantTrivialScore {
		return false
	}
	return chess.IsBrilliantSacrifice(before, played)
}

func analysisAtDepth(analysis engine.Analysis, board chess.Board, depth int) bool {
	if analysis.Depth == depth {
		return true
	}
	_, terminal := terminalPositionAnalysis(board)
	return terminal
}

func isGreatMove(board chess.Board, played chess.Move, analysis engine.Analysis, depth int) bool {
	if len(chess.GenerateLegalMoves(board)) < 2 || board.IsColorInCheck(board.ColorToMove) {
		return false
	}
	bestMove, err := board.ParseUCI(analysis.BestMove)
	if err != nil || played != bestMove {
		return false
	}
	primary, secondary, ok := primaryAndSecondary(analysis)
	if !ok {
		return false
	}
	if primary.Depth != depth || secondary.Depth != depth {
		return false
	}
	return scoreForMover(board, primary.Score) >= greatWinningScore && scoreForMover(board, secondary.Score) <= greatNotWinningScore
}

func primaryAndSecondary(analysis engine.Analysis) (engine.PrincipalVariation, engine.PrincipalVariation, bool) {
	var primary, secondary engine.PrincipalVariation
	for _, line := range analysis.Lines {
		switch line.Rank {
		case 1:
			primary = line
		case 2:
			secondary = line
		}
	}
	if primary.Rank != 1 || secondary.Rank != 2 {
		return engine.PrincipalVariation{}, engine.PrincipalVariation{}, false
	}
	return primary, secondary, true
}

func scoreForMover(board chess.Board, score engine.Score) int {
	return scoreForColor(board, score, board.ColorToMove)
}

func scoreForColor(board chess.Board, score engine.Score, color chess.PieceColor) int {
	evaluation := evaluationValue(evaluationForWhite(board, score))
	if color == chess.Black {
		return -evaluation
	}
	return evaluation
}

func evaluationForWhite(board chess.Board, score engine.Score) positionEvaluation {
	evaluation := positionEvaluation{centipawns: score.Centipawns, mate: score.Mate, isMate: score.IsMate}
	if board.ColorToMove == chess.Black {
		evaluation.centipawns = -evaluation.centipawns
		evaluation.mate = -evaluation.mate
	}
	return evaluation
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

func classifyMove(move, bestMove chess.Move, mover chess.PieceColor, before, after positionEvaluation) moveClassification {
	if move == bestMove {
		return moveBest
	}
	if before.isMate || after.isMate {
		return classifyMateMove(mover, before, after)
	}

	beforeScore := scoreForEvaluation(mover, before)
	afterScore := scoreForEvaluation(mover, after)
	pointLoss := max(0, expectedPoints(beforeScore)-expectedPoints(afterScore))
	classification := classifyExpectedPointLoss(pointLoss)
	if afterScore >= 300 && beforeScore-afterScore <= 300 && classification > moveOkay {
		return moveOkay
	}
	return classification
}

func classifyMateMove(mover chess.PieceColor, before, after positionEvaluation) moveClassification {
	afterScore := scoreForEvaluation(mover, after)
	if before.isMate && after.isMate {
		beforeMate := mateForMover(mover, before)
		afterMate := mateForMover(mover, after)
		if beforeMate > 0 && afterMate < 0 {
			if afterMate < -3 {
				return moveMistake
			}
			return moveBlunder
		}
		mateLoss := afterMate - beforeMate
		if mateLoss < 0 || mateLoss == 0 && afterMate < 0 {
			return moveBest
		}
		if mateLoss < 2 {
			return moveExcellent
		}
		if mateLoss < 7 {
			return moveOkay
		}
		return moveInaccuracy
	}
	if before.isMate {
		switch {
		case afterScore >= 800:
			return moveExcellent
		case afterScore >= 400:
			return moveOkay
		case afterScore >= 200:
			return moveInaccuracy
		case afterScore >= 0:
			return moveMistake
		default:
			return moveBlunder
		}
	}
	if mateForMover(mover, after) > 0 {
		return moveBest
	}
	if mateForMover(mover, after) >= -2 {
		return moveBlunder
	}
	if mateForMover(mover, after) >= -5 {
		return moveMistake
	}
	return moveInaccuracy
}

func mateForMover(color chess.PieceColor, evaluation positionEvaluation) int {
	if color == chess.Black {
		return -evaluation.mate
	}
	return evaluation.mate
}

func classifyExpectedPointLoss(pointLoss float64) moveClassification {
	if pointLoss < 0.01 {
		return moveBest
	}
	if pointLoss < 0.045 {
		return moveExcellent
	}
	if pointLoss < 0.08 {
		return moveOkay
	}
	if pointLoss < 0.12 {
		return moveInaccuracy
	}
	if pointLoss < 0.22 {
		return moveMistake
	}
	return moveBlunder
}

func scoreForEvaluation(color chess.PieceColor, evaluation positionEvaluation) int {
	score := evaluationValue(evaluation)
	if color == chess.Black {
		return -score
	}
	return score
}

func expectedPoints(score int) float64 {
	return 1 / (1 + math.Exp(-0.0035*float64(score)))
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
	if g.review.complete && len(g.review.moves) == len(g.moves) {
		g.review.active = true
		g.review.variation = nil
		g.review.hoveredBestMove = false
		return
	}
	g.reviewGeneration++
	g.review = gameReview{active: true, analyzing: true, requested: true, total: len(g.positions)}
}

func (g *localGameState) applyReviewResult(result reviewResult) {
	if result.generation != g.reviewGeneration || !g.review.analyzing {
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
	g.review.complete = true
}

func (g *localGameState) invalidateReview() {
	if g.review.analyzing {
		g.reviewCancelRequested = true
	}
	g.reviewGeneration++
	g.review = gameReview{}
}

func (g *localGameState) leaveReview() {
	if !g.review.complete {
		g.review = gameReview{}
		return
	}
	g.review.active = false
	g.review.variation = nil
	g.review.hoveredBestMove = false
}

func (g *localGameState) selectedMoveReview() (*moveReview, bool) {
	if !g.review.active || g.review.analyzing || g.viewPly < 1 || g.viewPly > len(g.review.moves) {
		return nil, false
	}
	return &g.review.moves[g.viewPly-1], true
}

func (g *localGameState) startVariation() {
	review, ok := g.selectedMoveReview()
	if !ok || reviewIsBestMove(*review) || len(review.bestLine) == 0 {
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
	g.review.variation = &reviewVariation{sourcePly: g.viewPly, positions: positions, viewPly: 1}
	g.playReplayMove(review.bestLine[0])
}

func (g localGameState) reviewVerdict() string {
	review, ok := g.selectedMoveReview()
	if !ok {
		return "Select a move to inspect its review."
	}
	if review.classification == moveBest {
		return fmt.Sprintf("%s is the best move.", g.history[g.viewPly-1])
	}
	if review.classification == moveBrilliant {
		return fmt.Sprintf("%s is a brilliant move.", g.history[g.viewPly-1])
	}
	if review.classification == moveGreat {
		return fmt.Sprintf("%s is a great move.", g.history[g.viewPly-1])
	}
	if review.classification == moveOkay {
		return fmt.Sprintf("%s is an okay move.", g.history[g.viewPly-1])
	}
	return fmt.Sprintf("%s is %s.", g.history[g.viewPly-1], strings.ToLower(review.classification.String()))
}

func isBestMoveClassification(classification moveClassification) bool {
	return classification == moveGreat || classification == moveBest
}

func reviewIsBestMove(review moveReview) bool {
	return isBestMoveClassification(review.classification) || review.classification == moveBrilliant && review.playedBest
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
		if review.classification == moveGreat {
			drawString(screen, x, y+1, "It was the only winning move.", mutedStyle)
		} else if !reviewIsBestMove(*review) {
			prefix := reviewBestMovePrefix(*review)
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

func reviewBestMovePrefix(review moveReview) string {
	if review.classification == moveBrilliant {
		if review.equivalentBest {
			return "Another winning move was "
		}
		return "Engine preferred "
	}
	return "The best move was "
}

func reviewHintOffset(compact bool) int {
	if compact {
		return 3
	}
	return 6
}

func bestMoveAt(x, y int, layout boardLayout, panelY int, game localGameState) bool {
	review, ok := game.selectedMoveReview()
	if !ok || reviewIsBestMove(*review) || y != panelY+1 {
		return false
	}
	prefixWidth := runewidth.StringWidth(reviewBestMovePrefix(*review))
	startX := layout.x - 2 + prefixWidth
	width := runewidth.StringWidth("[" + review.bestSAN + "]")
	return x >= startX && x < startX+width
}
