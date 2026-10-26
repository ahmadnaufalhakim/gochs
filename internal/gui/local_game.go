package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/engine"
	"github.com/ahmadnaufalhakim/gochs/internal/pgn"
	"github.com/ahmadnaufalhakim/gochs/internal/sound"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

const (
	squareWidth         = 2
	moveNumberWidth     = 4
	sanWidth            = 8
	moveHistoryRowWidth = moveNumberWidth + 2 + sanWidth*2 + 2
	moveHistoryBoxWidth = moveHistoryRowWidth + 2
	uciInputBoxWidth    = 21
)

type localGameState struct {
	board               chess.Board
	theme               chess.ColorTheme
	perspective         chess.PieceColor
	autoFlip            bool
	selectedSource      *chess.Square
	hoveredSquare       *chess.Square
	promotionChoices    []chess.Move
	hoveredPromotion    rune
	uciInput            string
	message             string
	history             []string
	clockHistory        []time.Duration
	moves               []chess.Move
	positions           []chess.Board
	viewPly             int
	historyScroll       int
	lastMove            chess.Move
	hasLastMove         bool
	positionCounts      map[uint64]uint
	drawReason          string
	terminalTitle       string
	timedOutBy          *chess.PieceColor
	timeControl         timeControl
	clockRemaining      [chess.PieceColorCount]time.Duration
	turnStarted         time.Time
	startedAt           time.Time
	resignPending       bool
	hoveredResign       bool
	hoveredFlip         bool
	hoveredSavePGN      bool
	hoveredSaveClocks   bool
	exportMessage       string
	resignedBy          *chess.PieceColor
	computerColor       *chess.PieceColor
	stockfish           *engine.Client
	engineResults       chan engineResult
	engineDifficulty    int
	engineThinking      bool
	engineRandom        func(int) int
	sounds              sound.Player
	suppressSoundEvents bool
	adjustSoundVolume   func(int) int
	soundStarted        bool
	soundEnded          bool
	lowTimePlayed       [chess.PieceColorCount]bool
	review              gameReview
}

type boardLayout struct {
	x int
	y int
}

func newLocalGameState(theme chess.ColorTheme) localGameState {
	var board chess.Board
	board.Reset()

	return localGameState{
		board:          board,
		theme:          theme,
		perspective:    chess.White,
		positionCounts: map[uint64]uint{board.PositionKey(): 1},
		positions:      []chess.Board{board},
		startedAt:      time.Now(),
	}
}

func (g *localGameState) handleKey(event *tcell.EventKey, width, height int) bool {
	if _, ok := localBoardLayout(width, height); !ok {
		return event.Key() == tcell.KeyEsc
	}
	if g.handleSoundVolumeKey(event) {
		return false
	}
	if g.hasTerminalResult() {
		if g.review.active {
			if event.Key() == tcell.KeyEsc {
				g.leaveReview()
			}
			return false
		}
		if event.Key() == tcell.KeyRune && (event.Rune() == 'g' || event.Rune() == 'G') {
			g.startReview()
			return false
		}
		return event.Key() == tcell.KeyEsc
	}
	if g.isComputerTurn() {
		return false
	}

	if len(g.promotionChoices) != 0 {
		switch event.Key() {
		case tcell.KeyEsc:
			g.promotionChoices = nil
			g.message = "Promotion cancelled"
		case tcell.KeyRune:
			if move, ok := g.promotionMove(event.Rune()); ok {
				g.makeMove(move)
			}
		}
		return false
	}

	switch event.Key() {
	case tcell.KeyEsc:
		if g.resignPending {
			g.resignPending = false
			return false
		}
		if g.uciInput != "" {
			g.uciInput = ""
			return false
		}
		if g.selectedSource != nil {
			g.selectedSource = nil
			return false
		}
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(g.uciInput) != 0 {
			g.uciInput = g.uciInput[:len(g.uciInput)-1]
		}
	case tcell.KeyEnter:
		g.submitUCI()
	case tcell.KeyRune:
		character := unicode.ToLower(event.Rune())
		if (character >= 'a' && character <= 'h') ||
			(character >= '1' && character <= '8') ||
			strings.ContainsRune("qrbn", character) {
			g.uciInput += string(character)
		}
	}

	return false
}

func (g *localGameState) handleSoundVolumeKey(event *tcell.EventKey) bool {
	if event.Key() != tcell.KeyRune || g.adjustSoundVolume == nil {
		return false
	}
	var delta int
	switch event.Rune() {
	case ',':
		delta = -10
	case '.':
		delta = 10
	default:
		return false
	}
	g.message = fmt.Sprintf("Sound volume: %d%%", g.adjustSoundVolume(delta))
	return true
}

func (g *localGameState) handleMouse(event *tcell.EventMouse, width, height int) {
	x, y := event.Position()
	layout, ok := localBoardLayout(width, height)
	if !ok {
		return
	}
	if g.hasTerminalResult() {
		g.handleReviewMouse(event, layout)
		return
	}
	if g.isComputerTurn() {
		return
	}
	if event.Buttons() == tcell.ButtonNone {
		g.hoveredResign = g.resignButtonAt(x, y, layout)
		g.hoveredFlip = flipButtonAt(x, y, layout)
		if g.hoveredResign || g.hoveredFlip {
			return
		}
	}

	if len(g.promotionChoices) != 0 {
		character, ok := g.promotionOptionAt(x, y, layout)
		if event.Buttons() == tcell.ButtonNone {
			g.hoveredPromotion = 0
			if ok {
				g.hoveredPromotion = character
			}
			return
		}
		if event.Buttons() == tcell.Button1 && ok {
			if move, ok := g.promotionMove(character); ok {
				g.makeMove(move)
			}
			return
		}
		if event.Buttons() == tcell.Button1 {
			g.promotionChoices = nil
			g.hoveredPromotion = 0
			g.message = "Promotion cancelled"
		}
		return
	}
	if event.Buttons() == tcell.Button1 {
		if g.resignButtonAt(x, y, layout) {
			g.hoveredResign = true
			if g.resignPending {
				g.resign()
			} else {
				g.resignPending = true
				g.selectedSource = nil
				g.hoveredSquare = nil
			}
			return
		}
		if g.resignPending {
			g.resignPending = false
			g.hoveredResign = false
			return
		}
		if flipButtonAt(x, y, layout) {
			g.flipBoard()
			return
		}
		g.hoveredResign = false
		g.hoveredFlip = false
	}

	square, ok := squareAtScreenPosition(x, y, layout, g.perspective)
	if !ok {
		if event.Buttons() == tcell.Button1 {
			g.selectedSource = nil
			g.hoveredSquare = nil
		}
		return
	}

	if event.Buttons() == tcell.ButtonNone {
		g.updateHover(square)
		return
	}
	if event.Buttons() != tcell.Button1 {
		return
	}

	g.hoveredSquare = nil
	legalMoves := chess.GenerateLegalMoves(g.board)
	if g.selectedSource == nil {
		if g.isSelectableSource(square, legalMoves) {
			g.selectSource(square)
		}
		return
	}

	candidates := g.movesForTarget(square, legalMoves)
	switch len(candidates) {
	case 0:
		if g.isSelectableSource(square, legalMoves) {
			g.selectSource(square)
		} else {
			g.selectedSource = nil
			g.message = ""
		}
	case 1:
		g.makeMove(candidates[0])
	default:
		g.promotionChoices = candidates
		g.message = ""
	}
}

func (g *localGameState) submitUCI() {
	if g.hasTerminalResult() {
		return
	}
	uci := g.uciInput
	if len(uci) != 4 && len(uci) != 5 {
		g.message = "Enter UCI like e2e4 or e7e8q"
		return
	}

	from, err := chess.ParseCoordinate(uci[:2])
	if err != nil {
		g.message = err.Error()
		return
	}
	to, err := chess.ParseCoordinate(uci[2:4])
	if err != nil {
		g.message = err.Error()
		return
	}

	var candidates []chess.Move
	for _, move := range chess.GenerateLegalMoves(g.board) {
		if move.From() == from && move.To() == to {
			candidates = append(candidates, move)
		}
	}
	if len(candidates) == 0 {
		g.message = "That is not a legal move"
		return
	}
	if len(candidates) == 1 {
		g.makeMove(candidates[0])
		return
	}
	if len(uci) == 4 {
		g.promotionChoices = candidates
		g.message = ""
		return
	}

	if move, ok := promotionMoveForRune(candidates, rune(uci[4])); ok {
		g.makeMove(move)
		return
	}

	g.message = "Promotion must be q, r, b, or n"
}

func (g *localGameState) selectSource(square chess.Square) {
	g.selectedSource = new(chess.Square)
	*g.selectedSource = square
	g.message = ""
}

func (g *localGameState) updateHover(square chess.Square) {
	if g.hasTerminalResult() {
		g.hoveredSquare = nil
		return
	}
	legalMoves := chess.GenerateLegalMoves(g.board)
	if g.isSelectableSource(square, legalMoves) {
		g.hoveredSquare = new(chess.Square)
		*g.hoveredSquare = square
		return
	}

	g.hoveredSquare = nil
}

func (g *localGameState) isSelectableSource(square chess.Square, legalMoves []chess.Move) bool {
	piece := g.board.PieceAt(square)
	if piece.Type == chess.PieceNone || piece.Color != g.board.ColorToMove {
		return false
	}
	for _, move := range legalMoves {
		if move.From() == square {
			return true
		}
	}

	return false
}

func (g *localGameState) movesForTarget(target chess.Square, legalMoves []chess.Move) []chess.Move {
	var candidates []chess.Move
	for _, move := range legalMoves {
		if move.From() == *g.selectedSource && g.targetForMove(move) == target {
			candidates = append(candidates, move)
		}
	}

	return candidates
}

func (g *localGameState) targetForMove(move chess.Move) chess.Square {
	if move.Flag() != chess.KingSideCastle && move.Flag() != chess.QueenSideCastle {
		return move.To()
	}

	right := g.board.CastlingRights[g.board.ColorToMove][chess.CastlingSide(move.Flag())]
	return right.RookFrom
}

func (g *localGameState) promotionMove(character rune) (chess.Move, bool) {
	return promotionMoveForRune(g.promotionChoices, character)
}

func promotionMoveForRune(candidates []chess.Move, character rune) (chess.Move, bool) {
	for _, move := range candidates {
		switch unicode.ToLower(character) {
		case 'n':
			if move.Flag() == chess.PromoteKnight || move.Flag() == chess.PromoteCaptureKnight {
				return move, true
			}
		case 'b':
			if move.Flag() == chess.PromoteBishop || move.Flag() == chess.PromoteCaptureBishop {
				return move, true
			}
		case 'r':
			if move.Flag() == chess.PromoteRook || move.Flag() == chess.PromoteCaptureRook {
				return move, true
			}
		case 'q':
			if move.Flag() == chess.PromoteQueen || move.Flag() == chess.PromoteCaptureQueen {
				return move, true
			}
		}
	}

	return chess.Move(0), false
}

func (g *localGameState) makeMove(move chess.Move) {
	if g.hasTerminalResult() {
		return
	}
	g.advanceClock(time.Now())
	if g.hasTerminalResult() {
		return
	}
	movingColor := g.board.ColorToMove
	san, err := g.board.SAN(move)
	if err != nil {
		g.message = err.Error()
		return
	}
	if err := g.board.MakeMove(move); err != nil {
		g.message = err.Error()
		return
	}

	g.history = append(g.history, san)
	g.moves = append(g.moves, move)
	g.positions = append(g.positions, g.board)
	g.viewPly = len(g.moves)
	g.lastMove = move
	g.hasLastMove = true
	g.selectedSource = nil
	g.hoveredSquare = nil
	g.promotionChoices = nil
	g.hoveredPromotion = 0
	g.uciInput = ""
	g.message = ""
	g.recordPosition()
	g.updateTerminalResult()
	g.playMove(move)
	if g.timeControl.isTimed() && !g.hasTerminalResult() {
		g.clockRemaining[movingColor] += g.timeControl.increment
		g.turnStarted = time.Now()
	}
	if g.timeControl.isTimed() {
		g.clockHistory = append(g.clockHistory, g.clockRemaining[movingColor])
	}
	if g.hasTerminalResult() {
		g.syncHistoryToView(1)
	}
	if g.autoFlip {
		g.perspective = g.board.ColorToMove
	}
	g.startEngineTurn()
}

func (g *localGameState) startSounds() {
	if g.suppressSoundEvents {
		return
	}
	if g.soundStarted {
		return
	}
	g.soundStarted = true
	g.play(sound.Game)
}

func (g *localGameState) endSounds() {
	if g.suppressSoundEvents {
		return
	}
	if g.soundEnded {
		return
	}
	g.soundEnded = true
	g.play(sound.Game)
}

func (g *localGameState) playMove(move chess.Move) {
	g.play(sound.Move)
	if isCapture(move) {
		g.play(sound.Capture)
	}
	if g.hasTerminalResult() {
		if len(chess.GenerateLegalMoves(g.board)) == 0 && g.board.IsColorInCheck(g.board.ColorToMove) {
			g.play(sound.Checkmate)
		} else if len(chess.GenerateLegalMoves(g.board)) == 0 {
			g.play(sound.Stalemate)
		} else {
			g.play(sound.Draw)
		}
		g.endSounds()
		return
	}
	if g.board.IsColorInCheck(g.board.ColorToMove) {
		g.play(sound.Check)
	}
}

func (g *localGameState) playReplayMove(move chess.Move) {
	g.play(sound.Move)
	if isCapture(move) {
		g.play(sound.Capture)
	}
}

func (g *localGameState) play(event sound.Event) {
	if !g.suppressSoundEvents && g.sounds != nil {
		g.sounds.Play(event)
	}
}

func isCapture(move chess.Move) bool {
	switch move.Flag() {
	case chess.Capture, chess.EnPassant, chess.PromoteCaptureKnight, chess.PromoteCaptureBishop, chess.PromoteCaptureRook, chess.PromoteCaptureQueen:
		return true
	}
	return false
}

func (g *localGameState) flipBoard() {
	g.perspective = g.perspective.Opponent()
	g.selectedSource = nil
	g.hoveredSquare = nil
	g.hoveredFlip = true
}

func (g localGameState) boardForDisplay() chess.Board {
	if g.review.variation != nil {
		return g.review.variation.positions[g.review.variation.viewPly]
	}
	if g.hasTerminalResult() && g.viewPly >= 0 && g.viewPly < len(g.positions) {
		return g.positions[g.viewPly]
	}
	return g.board
}

func (g localGameState) viewedMove() (chess.Move, bool) {
	if g.hasTerminalResult() {
		if g.viewPly > 0 && g.viewPly <= len(g.moves) {
			return g.moves[g.viewPly-1], true
		}
		return chess.Move(0), false
	}
	return g.lastMove, g.hasLastMove
}

func (g *localGameState) recordPosition() {
	if g.positionCounts == nil {
		g.positionCounts = make(map[uint64]uint)
	}
	g.positionCounts[g.board.PositionKey()]++
}

func (g *localGameState) updateTerminalResult() {
	legalMoves := chess.GenerateLegalMoves(g.board)
	if len(legalMoves) == 0 {
		if g.board.IsColorInCheck(g.board.ColorToMove) {
			winner := g.board.ColorToMove.Opponent()
			g.terminalTitle = fmt.Sprintf("Checkmate · %s wins %s", winner, winner.EndResult())
		} else {
			g.terminalTitle = "Stalemate · ½-½"
		}
		return
	}
	if g.board.HasInsufficientMaterial() {
		g.drawReason = "Draw by insufficient material"
		g.terminalTitle = g.drawReason + " · ½-½"
		return
	}
	if g.positionCounts[g.board.PositionKey()] >= 3 {
		g.drawReason = "Draw by threefold repetition"
		g.terminalTitle = g.drawReason + " · ½-½"
		return
	}
	if g.board.HalfmoveClock >= 100 {
		g.drawReason = "Draw by fifty-move rule"
		g.terminalTitle = g.drawReason + " · ½-½"
	}
}

func (g *localGameState) resign() {
	resigningColor := g.board.ColorToMove
	g.resignedBy = &resigningColor
	g.resignPending = false
	g.hoveredResign = false
	g.selectedSource = nil
	g.hoveredSquare = nil
	g.promotionChoices = nil
	g.hoveredPromotion = 0
	g.uciInput = ""
	g.message = ""
	g.endSounds()
}

func (g localGameState) hasResigned() bool {
	return g.resignedBy != nil
}

func (g localGameState) hasTerminalResult() bool {
	return g.hasResigned() || g.terminalTitle != ""
}

func (g localGameState) pgnGame() pgn.Game {
	white := "White"
	black := "Black"
	if g.computerColor != nil {
		name := fmt.Sprintf("Stockfish level %d", g.engineDifficulty)
		if *g.computerColor == chess.White {
			white = name
		} else {
			black = name
		}
	}

	return pgn.Game{
		Event:       g.timeControl.event,
		Site:        "https://github.com/ahmadnaufalhakim/gochs",
		Date:        g.startedAt,
		Round:       "-",
		White:       white,
		Black:       black,
		Result:      g.pgnResult(),
		Variant:     "Standard",
		TimeControl: g.timeControl.pgnValue(),
		Termination: g.pgnTermination(),
		Moves:       g.history,
		Clocks:      g.clockHistory,
	}
}

func (g localGameState) pgnResult() string {
	if g.resignedBy != nil {
		return g.resignedBy.Opponent().EndResult()
	}
	if g.timedOutBy != nil {
		return g.timedOutBy.Opponent().EndResult()
	}
	if g.terminalTitle == "" {
		return "*"
	}
	if len(chess.GenerateLegalMoves(g.board)) == 0 && g.board.IsColorInCheck(g.board.ColorToMove) {
		return g.board.ColorToMove.Opponent().EndResult()
	}

	return "1/2-1/2"
}

func (g localGameState) pgnTermination() string {
	if g.timedOutBy != nil {
		return "Time forfeit"
	}

	return "Normal"
}

func (c timeControl) pgnValue() string {
	if !c.isTimed() {
		return "-"
	}

	return fmt.Sprintf("%d+%d", int(c.initial/time.Second), int(c.increment/time.Second))
}

func (g *localGameState) savePGN(includeClocks bool) error {
	path, err := savePGNBytes(g.startedAt, includeClocks, []byte(pgn.Format(g.pgnGame(), includeClocks)))
	if err != nil {
		return err
	}
	g.exportMessage = "Saved " + filepath.Base(path)
	return nil
}

func savePGNBytes(startedAt time.Time, includeClocks bool, contents []byte) (string, error) {
	directory, err := pgnDirectory()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}

	format := "plain"
	if includeClocks {
		format = "clocks"
	}
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	path := filepath.Join(directory, startedAt.Format("20060102-150405.000000000")+"-"+format+".pgn")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func pgnDirectory() (string, error) {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "gochs", "games"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "gochs", "games"), nil
}

func (g *localGameState) handleReviewMouse(event *tcell.EventMouse, layout boardLayout) {
	x, y := event.Position()
	buttons := event.Buttons()
	if g.review.active {
		g.handleGameReviewMouse(event, layout, layout.y+12)
		return
	}
	if buttons == tcell.ButtonNone {
		g.hoveredFlip = flipButtonAt(x, y, layout)
		g.hoveredSavePGN = savePGNButtonAt(x, y, layout, false)
		g.hoveredSaveClocks = savePGNButtonAt(x, y, layout, true)
		g.review.hoveredStart = gameReviewButtonAt(x, y, layout)
		return
	}
	if buttons == tcell.Button1 {
		if gameReviewButtonAt(x, y, layout) {
			g.startReview()
			return
		}
		if flipButtonAt(x, y, layout) {
			g.flipBoard()
			return
		}
		if savePGNButtonAt(x, y, layout, false) {
			if err := g.savePGN(false); err != nil {
				g.exportMessage = "Could not save PGN: " + err.Error()
			}
			return
		}
		if savePGNButtonAt(x, y, layout, true) {
			if err := g.savePGN(true); err != nil {
				g.exportMessage = "Could not save PGN: " + err.Error()
			}
			return
		}
		if ply, ok := g.historyPlyAt(x, y, layout); ok {
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

func (g *localGameState) syncHistoryAfterClick(row int) {
	switch row {
	case 0:
		g.historyScroll = max(0, g.historyScroll-1)
	case 3:
		g.historyScroll = min(g.historyMaxScroll(), g.historyScroll+1)
	}
}

func (g *localGameState) moveReplay(direction int) {
	if g.review.variation != nil {
		variation := g.review.variation
		if direction < 0 && variation.viewPly <= 1 {
			g.review.variation = nil
			if variation.sourcePly > 0 && variation.sourcePly <= len(g.moves) {
				g.playReplayMove(g.moves[variation.sourcePly-1])
			}
			return
		}
		previousPly := variation.viewPly
		viewPly := min(max(1, variation.viewPly+direction), len(variation.positions)-1)
		if viewPly == previousPly {
			return
		}
		variation.viewPly = viewPly
		if review, ok := g.selectedMoveReview(); ok {
			moveIndex := viewPly - 1
			if direction < 0 {
				moveIndex = previousPly - 1
			}
			if moveIndex >= 0 && moveIndex < len(review.bestLine) {
				g.playReplayMove(review.bestLine[moveIndex])
			}
		}
		return
	}
	previousPly := g.viewPly
	viewPly := min(max(0, g.viewPly+direction), len(g.moves))
	if viewPly == previousPly {
		return
	}
	g.viewPly = viewPly
	moveIndex := viewPly - 1
	if direction < 0 {
		moveIndex = previousPly - 1
	}
	if moveIndex >= 0 && moveIndex < len(g.moves) {
		g.playReplayMove(g.moves[moveIndex])
	}
	if replayFullmoveRow(previousPly) != replayFullmoveRow(viewPly) {
		g.syncHistoryToView(direction)
	}
}

func replayFullmoveRow(ply int) int {
	if ply == 0 {
		return -1
	}
	return (ply - 1) / 2
}

func (g *localGameState) syncHistoryToView(direction int) {
	if g.viewPly == 0 {
		g.historyScroll = 0
		return
	}
	targetRow := (g.viewPly - 1) / 2
	anchorRow := 1
	if direction > 0 {
		anchorRow = 2
	}
	g.historyScroll = min(max(0, targetRow-anchorRow), g.historyMaxScroll())
}

func (g localGameState) historyMaxScroll() int {
	fullmoves := (len(g.history) + 1) / 2
	return max(0, fullmoves-4)
}

func (g localGameState) historyPlyAt(x, y int, layout boardLayout) (int, bool) {
	if !historyAreaAt(x, y, layout) {
		return 0, false
	}
	row := y - (layout.y + 1)
	index := (g.historyScroll + row) * 2
	startX := moveHistoryBoxX(layout) + 1
	if x < startX+moveNumberWidth+2 {
		return 0, false
	}
	if x < startX+moveNumberWidth+2+sanWidth {
		if index < len(g.history) {
			return index + 1, true
		}
		return 0, false
	}
	if x < startX+moveNumberWidth+2+sanWidth+2 {
		return 0, false
	}
	if index+1 < len(g.history) {
		return index + 2, true
	}
	return 0, false
}

func historyAreaAt(x, y int, layout boardLayout) bool {
	startX := moveHistoryBoxX(layout) + 1
	return x >= startX && x < startX+moveHistoryRowWidth && y >= layout.y+1 && y < layout.y+5
}

func moveHistoryBoxX(layout boardLayout) int {
	return layout.x + 24
}

func drawLocalGame(screen tcell.Screen, game localGameState) {
	width, height := screen.Size()
	layout, ok := localBoardLayout(width, height)
	if !ok {
		drawCentered(screen, height/2-1, "Terminal too small", titleStyle)
		drawCentered(screen, height/2+1, "Resize or Esc to return", mutedStyle)
		return
	}

	drawBoard(screen, game, layout)
	drawMoveHistory(screen, game, layout, width)
	drawClocks(screen, game, layout)
	drawGameInput(screen, game, layout)
	if len(game.promotionChoices) != 0 {
		drawPromotionPicker(screen, game, layout)
	}
}

func gameTitle(game localGameState) string {
	if game.review.variation != nil {
		return "Viewing engine variation"
	}
	if game.resignedBy != nil {
		if game.viewPly != len(game.moves) {
			return game.viewingTitle()
		}
		return fmt.Sprintf("%s resigned · %s wins %s", *game.resignedBy, game.resignedBy.Opponent(), game.resignedBy.EndResult())
	}
	if game.terminalTitle != "" {
		if game.viewPly != len(game.moves) {
			return game.viewingTitle()
		}
		return game.terminalTitle
	}
	legalMoves := chess.GenerateLegalMoves(game.board)
	if len(legalMoves) == 0 {
		if game.board.IsColorInCheck(game.board.ColorToMove) {
			winner := game.board.ColorToMove.Opponent()
			return fmt.Sprintf("Checkmate · %s wins %s", winner, winner.EndResult())
		}
		return "Stalemate · ½-½"
	}
	if game.board.IsColorInCheck(game.board.ColorToMove) {
		return fmt.Sprintf("%s to move - check", game.board.ColorToMove)
	}

	return fmt.Sprintf("%s to move", game.board.ColorToMove)
}

func (g localGameState) viewingTitle() string {
	if g.viewPly == 0 {
		return "Viewing starting position"
	}
	san := g.history[g.viewPly-1]
	fullmove := (g.viewPly + 1) / 2
	if g.viewPly%2 == 0 {
		return fmt.Sprintf("Viewing %d... %s", fullmove, san)
	}
	return fmt.Sprintf("Viewing %d. %s", fullmove, san)
}

func drawBoard(screen tcell.Screen, game localGameState, layout boardLayout) {
	renderer := chess.Renderer{Perspective: game.perspective}
	board := game.boardForDisplay()
	var legalMoves []chess.Move
	if !game.hasTerminalResult() {
		legalMoves = chess.GenerateLegalMoves(board)
	}
	for row := range 8 {
		square := renderer.SquareAt(row, 0)
		drawString(screen, layout.x-2, layout.y+row, fmt.Sprintf("%d", square.Rank()+1), mutedStyle)
		for column := range 8 {
			square := renderer.SquareAt(row, column)
			style, color := squareStyle(game, board, square, legalMoves)
			piece := board.PieceAt(square)
			drawPiece(screen, layout.x+column*squareWidth, layout.y+row, piece, style, color)
		}
	}

	for column := range 8 {
		square := renderer.SquareAt(0, column)
		drawString(screen, layout.x+column*squareWidth, layout.y+8, fmt.Sprintf("%c ", 'a'+square.File()), mutedStyle)
	}
}

func squareStyle(game localGameState, board chess.Board, square chess.Square, legalMoves []chess.Move) (tcell.Style, chess.RGB) {
	light, dark := game.theme.SquareColors()
	color := light
	if square.IsDark() {
		color = dark
	}

	kingInCheck := false
	if king, ok := board.Pieces[board.ColorToMove][chess.King].SingleSquare(); ok {
		kingInCheck = board.IsColorInCheck(board.ColorToMove) && king == square
	}

	if kingInCheck {
		color = chess.RGB{R: 205, G: 50, B: 50}
	}
	if game.selectedSource != nil && *game.selectedSource == square {
		color = adjustRGB(color, 28, 28, 70)
	} else if game.selectedSource != nil && containsTarget(game, square, legalMoves) {
		color.G = 255
	} else if game.hoveredSquare != nil && *game.hoveredSquare == square {
		color = darkenRGB(color, 71)
	} else if lastMove, ok := game.viewedMove(); ok && (lastMove.From() == square || lastMove.To() == square) {
		color = brightenRGB(color, 31)
	}

	return backgroundStyle.Background(tcell.NewRGBColor(int32(color.R), int32(color.G), int32(color.B))), color
}

func containsTarget(game localGameState, square chess.Square, legalMoves []chess.Move) bool {
	for _, move := range legalMoves {
		if move.From() == *game.selectedSource && game.targetForMove(move) == square {
			return true
		}
	}

	return false
}

func adjustRGB(color chess.RGB, red, green, blue uint8) chess.RGB {
	return chess.RGB{
		R: saturatingAdd(color.R, red),
		G: saturatingAdd(color.G, green),
		B: saturatingAdd(color.B, blue),
	}
}

func brightenRGB(color chess.RGB, amount uint8) chess.RGB {
	return chess.RGB{
		R: saturatingAdd(color.R, amount),
		G: saturatingAdd(color.G, amount),
		B: saturatingAdd(color.B, amount),
	}
}

func darkenRGB(color chess.RGB, amount uint8) chess.RGB {
	return chess.RGB{
		R: saturatingSubtract(color.R, amount),
		G: saturatingSubtract(color.G, amount),
		B: saturatingSubtract(color.B, amount),
	}
}

func saturatingAdd(value, amount uint8) uint8 {
	if 255-value < amount {
		return 255
	}

	return value + amount
}

func saturatingSubtract(value, amount uint8) uint8 {
	if value < amount {
		return 0
	}

	return value - amount
}

func drawPiece(screen tcell.Screen, x, y int, piece chess.Piece, style tcell.Style, background chess.RGB) {
	if piece.Type == chess.PieceNone {
		screen.SetContent(x, y, ' ', nil, style)
		screen.SetContent(x+1, y, ' ', nil, style)
		return
	}

	foreground := tcell.ColorWhite
	if piece.Color == chess.Black {
		foreground = tcell.ColorBlack
	} else if luminance(background) > 150 {
		foreground = tcell.ColorBlack
	}
	label := []rune(piece.Label())
	screen.SetContent(x, y, label[0], nil, style.Foreground(foreground))
	screen.SetContent(x+1, y, ' ', nil, style.Foreground(foreground))
}

func luminance(color chess.RGB) uint8 {
	return uint8((uint16(color.R)*299 + uint16(color.G)*587 + uint16(color.B)*114) / 1000)
}

func drawMoveHistory(screen tcell.Screen, game localGameState, layout boardLayout, width int) {
	x := moveHistoryBoxX(layout)
	if x+moveHistoryBoxWidth > width {
		return
	}

	drawString(screen, x, layout.y-1, gameTitle(game), titleStyle)
	drawMoveHistoryBorder(screen, x, layout.y)
	start := game.historyScroll
	if !game.hasTerminalResult() {
		start, _ = moveHistoryWindow(game.history)
	}
	for row := range 4 {
		fullmove := start + row
		index := fullmove * 2
		if index >= len(game.history) {
			break
		}
		blackMove := ""
		if index+1 < len(game.history) {
			blackMove = game.history[index+1]
		}
		drawMoveHistoryRow(screen, x+1, layout.y+1+row, fullmove+1, game, index, blackMove)
	}
	if !game.hasTerminalResult() {
		drawResignButton(screen, game, layout)
	}
	drawFlipButton(screen, game, layout)
}

func drawMoveHistoryBorder(screen tcell.Screen, x, y int) {
	topLabel := "━ Moves "
	topBorder := "┏" + topLabel + strings.Repeat("━", moveHistoryRowWidth-runewidth.StringWidth(topLabel)) + "┓"
	drawString(screen, x, y, topBorder, mutedStyle)
	drawString(screen, x+3, y, "Moves", titleStyle)
	for row := 1; row <= 4; row++ {
		drawString(screen, x, y+row, "┃", mutedStyle)
		drawString(screen, x+moveHistoryBoxWidth-1, y+row, "┃", mutedStyle)
	}
	drawString(screen, x, y+5, "┗"+strings.Repeat("━", moveHistoryRowWidth)+"┛", mutedStyle)
}

func drawMoveHistoryRow(screen tcell.Screen, x, y, fullmove int, game localGameState, index int, blackMove string) {
	drawString(screen, x, y, fmt.Sprintf("%*d. ", moveNumberWidth, fullmove), backgroundStyle)
	whiteStyle := backgroundStyle
	blackStyle := backgroundStyle
	if game.hasTerminalResult() && game.viewPly == index+1 {
		whiteStyle = selectedStyle
	}
	if game.hasTerminalResult() && game.viewPly == index+2 {
		blackStyle = selectedStyle
	}
	whiteMove := game.annotatedSAN(index)
	drawString(screen, x+moveNumberWidth+2, y, fmt.Sprintf("%-*s", sanWidth, whiteMove), whiteStyle)
	drawString(screen, x+moveNumberWidth+2+sanWidth, y, "  ", backgroundStyle)
	blackIndex := index + 1
	if blackMove != "" {
		blackMove = game.annotatedSAN(blackIndex)
	}
	drawString(screen, x+moveNumberWidth+2+sanWidth+2, y, fmt.Sprintf("%-*s", sanWidth, blackMove), blackStyle)
}

func formatMoveHistoryRow(fullmove int, whiteMove, blackMove string) string {
	return fmt.Sprintf("%*d. %-*s  %-*s", moveNumberWidth, fullmove, sanWidth, whiteMove, sanWidth, blackMove)
}

func moveHistoryWindow(history []string) (start, end int) {
	end = (len(history) + 1) / 2
	return max(0, end-4), end
}

func drawResignButton(screen tcell.Screen, game localGameState, layout boardLayout) {
	x, y := resignButtonPosition(layout)
	label := resignButtonLabel(game.resignPending)
	style := backgroundStyle
	if game.hoveredResign && game.resignPending {
		style = backgroundStyle.Background(tcell.NewRGBColor(130, 50, 50)).Foreground(tcell.NewRGBColor(255, 230, 211)).Bold(true)
	} else if game.hoveredResign {
		style = selectedStyle
	}
	drawString(screen, x, y, label, style)
}

func resignButtonPosition(layout boardLayout) (int, int) {
	return moveHistoryBoxX(layout), layout.y + 6
}

func resignButtonLabel(pending bool) string {
	if pending {
		return "[🏳️] Are you sure?"
	}
	return "[🏳️] Resign"
}

func (g localGameState) resignButtonAt(x, y int, layout boardLayout) bool {
	buttonX, buttonY := resignButtonPosition(layout)
	return y == buttonY && x >= buttonX && x < buttonX+runewidth.StringWidth(resignButtonLabel(g.resignPending))
}

func drawFlipButton(screen tcell.Screen, game localGameState, layout boardLayout) {
	x, y := flipButtonPosition(layout)
	style := backgroundStyle
	if game.hoveredFlip {
		style = selectedStyle
	}
	drawString(screen, x, y, "[🔄] Flip board", style)
}

func flipButtonPosition(layout boardLayout) (int, int) {
	return moveHistoryBoxX(layout), layout.y + 7
}

func flipButtonAt(x, y int, layout boardLayout) bool {
	buttonX, buttonY := flipButtonPosition(layout)
	return y == buttonY && x >= buttonX && x < buttonX+runewidth.StringWidth("[🔄] Flip board")
}

func drawGameInput(screen tcell.Screen, game localGameState, layout boardLayout) {
	x := layout.x - 2
	y := layout.y + 10
	if game.hasTerminalResult() {
		if game.review.active {
			drawGameReviewPanel(screen, game, layout, layout.y+12, true)
			return
		}
		drawSavePGNButton(screen, game, layout, false)
		drawSavePGNButton(screen, game, layout, true)
		drawGameReviewButton(screen, game, layout)
		if game.exportMessage != "" {
			drawString(screen, x, y+3, game.exportMessage, mutedStyle)
		}
		drawString(screen, x, y+6, reviewSoundHint(game.message), mutedStyle)
		return
	}
	drawUCIInputBorder(screen, x, y)
	drawPaddedString(screen, x+1, y+1, uciInputBoxWidth-2, game.uciInput+"_", backgroundStyle)
	if game.message != "" {
		drawString(screen, x, y+4, game.message, mutedStyle)
	} else if game.stockfishStatus() != "" {
		drawString(screen, x, y+4, game.stockfishStatus(), mutedStyle)
	}
	drawString(screen, x, y+5, ",/.: sound volume  Esc: clear selection or return to menu", mutedStyle)
}

func reviewSoundHint(message string) string {
	hint := ",/.: Sound volume  Esc: return to menu"
	if message == "" {
		return hint
	}
	return message + "  " + hint
}

func drawSavePGNButton(screen tcell.Screen, game localGameState, layout boardLayout, includeClocks bool) {
	x, y := savePGNButtonPosition(layout, includeClocks)
	label := savePGNButtonLabel(includeClocks)
	style := backgroundStyle
	if (!includeClocks && game.hoveredSavePGN) || (includeClocks && game.hoveredSaveClocks) {
		style = selectedStyle
	}
	drawString(screen, x, y, label, style)
}

func savePGNButtonPosition(layout boardLayout, includeClocks bool) (int, int) {
	y := layout.y + 10
	if includeClocks {
		y++
	}
	return layout.x - 2, y
}

func savePGNButtonLabel(includeClocks bool) string {
	if includeClocks {
		return "[⏱️] Save PGN + clocks"
	}
	return "[💾] Save PGN"
}

func savePGNButtonAt(x, y int, layout boardLayout, includeClocks bool) bool {
	buttonX, buttonY := savePGNButtonPosition(layout, includeClocks)
	return y == buttonY && x >= buttonX && x < buttonX+runewidth.StringWidth(savePGNButtonLabel(includeClocks))
}

func drawUCIInputBorder(screen tcell.Screen, x, y int) {
	topLabel := "━ Your move (UCI) "
	topBorder := "┏" + topLabel + strings.Repeat("━", uciInputBoxWidth-runewidth.StringWidth(topLabel)-2) + "┓"
	drawString(screen, x, y, topBorder, mutedStyle)
	drawString(screen, x+3, y, "Your move (UCI)", titleStyle)
	drawString(screen, x, y+1, "┃", mutedStyle)
	drawString(screen, x+uciInputBoxWidth-1, y+1, "┃", mutedStyle)
	drawString(screen, x, y+2, "┗"+strings.Repeat("━", uciInputBoxWidth-2)+"┛", mutedStyle)
}

func drawPromotionPicker(screen tcell.Screen, game localGameState, layout boardLayout) {
	target, ok := game.promotionTarget()
	if !ok {
		return
	}

	x, targetY := promotionScreenPosition(target, layout, game.perspective)
	promotingColor := game.board.ColorToMove
	foreground := tcell.ColorWhite
	background := tcell.ColorBlack
	if promotingColor == chess.Black {
		foreground = tcell.ColorBlack
		background = tcell.ColorWhite
	}

	for index, character := range promotionOptions {
		style := backgroundStyle.Foreground(foreground).Background(background)
		if game.hoveredPromotion == character {
			style = style.Underline(true)
		}
		piece := chess.Piece{Color: promotingColor, Type: promotionPieceType(character)}
		drawString(screen, x, targetY-index-1, piece.Label()+" ", style)
	}
}

var promotionOptions = []rune{'q', 'r', 'b', 'n'}

func (g *localGameState) promotionOptionAt(x, y int, layout boardLayout) (rune, bool) {
	target, ok := g.promotionTarget()
	if !ok {
		return 0, false
	}

	popupX, targetY := promotionScreenPosition(target, layout, g.perspective)
	if x < popupX || x >= popupX+squareWidth {
		return 0, false
	}
	index := targetY - y - 1
	if index < 0 || index >= len(promotionOptions) {
		return 0, false
	}

	return promotionOptions[index], true
}

func (g *localGameState) promotionTarget() (chess.Square, bool) {
	if len(g.promotionChoices) == 0 {
		return chess.Square(0), false
	}
	return g.promotionChoices[0].To(), true
}

func promotionScreenPosition(square chess.Square, layout boardLayout, perspective chess.PieceColor) (int, int) {
	if perspective == chess.Black {
		return layout.x + (7-int(square.File()))*squareWidth, layout.y + int(square.Rank())
	}
	return layout.x + int(square.File())*squareWidth, layout.y + 7 - int(square.Rank())
}

func promotionPieceType(character rune) chess.PieceType {
	switch character {
	case 'r':
		return chess.Rook
	case 'b':
		return chess.Bishop
	case 'n':
		return chess.Knight
	default:
		return chess.Queen
	}
}

func localBoardLayout(width, height int) (boardLayout, bool) {
	if width < 52 || height < 21 {
		return boardLayout{}, false
	}

	const panelHeight = 20
	return boardLayout{x: (width - 48) / 2, y: (height-panelHeight)/2 + 4}, true
}

func squareAtScreenPosition(x, y int, layout boardLayout, perspective chess.PieceColor) (chess.Square, bool) {
	if x < layout.x || x >= layout.x+8*squareWidth || y < layout.y || y >= layout.y+8 {
		return chess.Square(0), false
	}

	row := y - layout.y
	column := (x - layout.x) / squareWidth
	renderer := chess.Renderer{Perspective: perspective}
	return renderer.SquareAt(row, column), true
}
