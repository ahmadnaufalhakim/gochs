package gui

import (
	"strings"
	"testing"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/engine"
	"github.com/ahmadnaufalhakim/gochs/internal/sound"
	"github.com/gdamore/tcell/v2"
)

func TestClassifyMove(t *testing.T) {
	var board chess.Board
	board.Reset()
	move, _ := board.ParseUCI("e2e4")
	best, _ := board.ParseUCI("d2d4")
	tests := []struct {
		loss int
		want moveClassification
	}{
		{0, moveBest},
		{11, moveExcellent},
		{31, moveOkay},
		{81, moveInaccuracy},
		{181, moveMistake},
		{351, moveBlunder},
	}
	for _, test := range tests {
		if got := classifyMove(move, best, test.loss); got != test.want {
			t.Errorf("classifyMove(%d) = %s, want %s", test.loss, got, test.want)
		}
	}
	if got := classifyMove(move, move, 1000); got != moveBest {
		t.Errorf("best move classification = %s", got)
	}
}

func TestGreatMoveRequiresOnlyWinningPrimaryVariation(t *testing.T) {
	var board chess.Board
	board.Reset()
	whiteMove, _ := board.ParseUCI("e2e4")
	whiteAnalysis := engine.Analysis{
		BestMove: "e2e4",
		Lines: []engine.PrincipalVariation{
			{Rank: 1, Depth: gameReviewDepth, Score: engine.Score{Centipawns: 150}},
			{Rank: 2, Depth: gameReviewDepth, Score: engine.Score{Centipawns: 50}},
		},
	}
	if !isGreatMove(board, whiteMove, whiteAnalysis, gameReviewDepth) {
		t.Fatal("white only-winning move was not Great")
	}
	whiteAnalysis.Lines[1].Score.Centipawns = 51
	if isGreatMove(board, whiteMove, whiteAnalysis, gameReviewDepth) {
		t.Fatal("winning alternative was classified as Great")
	}
	whiteAnalysis.Lines[1].Score.Centipawns = 50
	whiteAnalysis.Lines[1].Depth = gameReviewDepth - 1
	if isGreatMove(board, whiteMove, whiteAnalysis, gameReviewDepth) {
		t.Fatal("mixed-depth variations were classified as Great")
	}

	if err := board.MakeMove(whiteMove); err != nil {
		t.Fatal(err)
	}
	blackMove, _ := board.ParseUCI("e7e5")
	blackAnalysis := engine.Analysis{
		BestMove: "e7e5",
		Lines: []engine.PrincipalVariation{
			{Rank: 1, Depth: gameReviewDepth, Score: engine.Score{Centipawns: 150}},
			{Rank: 2, Depth: gameReviewDepth, Score: engine.Score{Centipawns: 50}},
		},
	}
	if !isGreatMove(board, blackMove, blackAnalysis, gameReviewDepth) {
		t.Fatal("black only-winning move was not Great")
	}
	blackAnalysis.Lines = blackAnalysis.Lines[:1]
	if isGreatMove(board, blackMove, blackAnalysis, gameReviewDepth) {
		t.Fatal("single-PV analysis was classified as Great")
	}
}

func TestBuildMoveReviewsNormalizesBlackToMoveScores(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.uciInput = "e2e4"
	game.submitUCI()
	reviews, err := buildMoveReviews(game.positions, game.moves, []engine.Analysis{
		{Score: engine.Score{}, BestMove: "d2d4", PV: []string{"d2d4"}},
		{Score: engine.Score{Centipawns: 100}, BestMove: "e7e5", PV: []string{"e7e5"}},
	})
	if err != nil {
		t.Fatalf("buildMoveReviews: %v", err)
	}
	if got := reviews[0].classification; got != moveInaccuracy {
		t.Errorf("classification = %s, want %s", got, moveInaccuracy)
	}
	if got := reviews[0].after.centipawns; got != -100 {
		t.Errorf("post-move white evaluation = %d, want -100", got)
	}
}

func TestGameReviewVariationStartsAtBestMoveAndResumesGame(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.uciInput = "e2e4"
	game.submitUCI()
	best, err := game.positions[0].ParseUCI("d2d4")
	if err != nil {
		t.Fatalf("parse best move: %v", err)
	}
	variationBoard := game.positions[0]
	if err := variationBoard.MakeMove(best); err != nil {
		t.Fatal(err)
	}
	continuation, err := variationBoard.ParseUCI("d7d5")
	if err != nil {
		t.Fatal(err)
	}
	var events []sound.Event
	game.sounds = recordingPlayer(&events)
	game.terminalTitle = "Draw · ½-½"
	game.viewPly = 1
	game.review = gameReview{active: true, moves: []moveReview{{classification: moveExcellent, bestMove: best, bestLine: []chess.Move{best, continuation}}}}
	realPosition := game.positions[1]
	game.startVariation()
	if game.review.variation == nil || len(game.review.variation.positions) != 3 {
		t.Fatal("variation was not created")
	}
	if game.boardForDisplay() != game.review.variation.positions[1] {
		t.Fatal("best move was not displayed immediately")
	}
	game.moveReplay(1)
	if game.boardForDisplay() != game.review.variation.positions[2] {
		t.Fatal("variation position was not displayed")
	}
	game.moveReplay(-1)
	game.moveReplay(-1)
	if game.review.variation != nil {
		t.Fatal("scrolling before the best move did not leave variation view")
	}
	if game.positions[1] != realPosition {
		t.Fatal("variation changed the real game position")
	}
	if game.boardForDisplay() != realPosition {
		t.Fatal("game position was not restored after variation")
	}
	assertSoundEvents(t, events, []sound.Event{sound.Move, sound.Move, sound.Move, sound.Move})
}

func TestFinishedGameReviewButtonStartsReview(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.terminalTitle = "Draw · ½-½"
	layout, ok := localBoardLayout(80, 24)
	if !ok {
		t.Fatal("expected board layout")
	}
	x, y := gameReviewButtonPosition(layout)
	game.handleMouse(tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone), 80, 24)
	if !game.review.active || !game.review.requested || !game.review.analyzing {
		t.Fatalf("review state = %#v", game.review)
	}
}

func TestGameReviewReusesCompletedAnalysisForSameGame(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.uciInput = "e2e4"
	game.submitUCI()
	game.terminalTitle = "Draw · ½-½"
	game.review = gameReview{active: true, complete: true, moves: []moveReview{{classification: moveBest}}}
	game.leaveReview()
	game.startReview()
	if !game.review.active || game.review.analyzing || game.review.requested || !game.review.complete {
		t.Fatalf("cached review state = %#v", game.review)
	}

	newGame := newLocalGameState(chess.WOOD)
	newGame.terminalTitle = "Draw · ½-½"
	newGame.startReview()
	if !newGame.review.analyzing || !newGame.review.requested || newGame.review.complete {
		t.Fatalf("new game review state = %#v", newGame.review)
	}
}

func TestGameReviewDiscardsResultFromCancelledGeneration(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.terminalTitle = "Draw · ½-½"
	game.startReview()
	staleGeneration := game.reviewGeneration
	game.leaveReview()
	game.startReview()
	if game.reviewGeneration == staleGeneration {
		t.Fatal("new review did not use a new generation")
	}
	game.applyReviewResult(reviewResult{generation: staleGeneration, progress: 99, total: 99, done: true, moves: []moveReview{{classification: moveBlunder}}})
	if !game.review.analyzing || game.review.progress != 0 || game.review.complete {
		t.Fatalf("stale result changed new review: %#v", game.review)
	}
}

func TestGameReviewPanelShowsVerdictBestMoveAndEvaluation(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.uciInput = "e2e4"
	game.submitUCI()
	best, err := game.positions[0].ParseUCI("d2d4")
	if err != nil {
		t.Fatal(err)
	}
	game.terminalTitle = "Draw · ½-½"
	game.viewPly = 1
	game.review = gameReview{active: true, moves: []moveReview{{
		classification: moveInaccuracy,
		before:         positionEvaluation{},
		after:          positionEvaluation{centipawns: -90},
		bestMove:       best,
		bestSAN:        "d4",
		bestLine:       []chess.Move{best},
	}}}
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
	for _, want := range []string{"e4 is inaccuracy.", "The best move was [d4]", "Eval: +0.00 -> -0.90"} {
		if !strings.Contains(text, want) {
			t.Errorf("review panel does not show %q", want)
		}
	}

	game.handleKey(keyEvent(tcell.KeyEsc, 0), 80, 24)
	if game.review.active {
		t.Fatal("Esc did not return from Game Review to exports")
	}
}

func TestGreatMoveReviewDoesNotShowRedundantBestMove(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	game.uciInput = "e2e4"
	game.submitUCI()
	game.terminalTitle = "Draw · ½-½"
	game.viewPly = 1
	game.review = gameReview{active: true, moves: []moveReview{{classification: moveGreat}}}
	if got := game.reviewVerdict(); got != "e4 is a great move." {
		t.Errorf("reviewVerdict() = %q", got)
	}
	if game.startVariation(); game.review.variation != nil {
		t.Fatal("Great move exposed a redundant variation")
	}
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)
	drawLocalGame(screen, game)
	screen.Show()
	contents, width, height := screen.GetContents()
	if !strings.Contains(screenText(contents, width, height), "It was the only winning move.") {
		t.Fatal("Great move explanation was not rendered")
	}
}
