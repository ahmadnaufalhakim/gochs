package gui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

const (
	puzzleInitialDelay   = time.Second
	puzzleHelpDelay      = 5 * time.Second
	puzzleWrongDelay     = 500 * time.Millisecond
	puzzleReplyDelay     = time.Second
	puzzleThemesBoxWidth = 24
)

type puzzlePhase uint8

const (
	puzzlePreview puzzlePhase = iota
	puzzleSolverTurn
	puzzleWrongMove
	puzzleOpponentReply
	puzzleSolved
	puzzleSolutionViewed
)

type puzzleAction uint8

const (
	puzzleStay puzzleAction = iota
	puzzleReturnToMenu
	puzzleContinueTraining
)

type puzzleState struct {
	puzzle           entity.Puzzle
	board            chess.Board
	theme            chess.ColorTheme
	solverColor      chess.PieceColor
	perspective      chess.PieceColor
	phase            puzzlePhase
	nextSolutionMove int
	phaseDeadline    time.Time
	helpAvailableAt  time.Time
	selectedSource   *chess.Square
	hoveredSquare    *chess.Square
	promotionChoices []chess.Move
	hoveredPromotion rune
	uciInput         string
	message          string
	hintCount        int
	hintMoveIndex    int
	history          []string
	moves            []chess.Move
	positions        []chess.Board
	viewPly          int
	historyScroll    int
	lastMove         chess.Move
	hasLastMove      bool
	wrongBoard       chess.Board
	wrongLastMove    chess.Move
	wrongHasLastMove bool
	hoveredHint      bool
	hoveredSolution  bool
	hoveredFlip      bool
	hoveredContinue  bool
}

func newPuzzleState(puzzle entity.Puzzle, theme chess.ColorTheme, now time.Time) (puzzleState, error) {
	if len(puzzle.Moves) < 2 {
		return puzzleState{}, fmt.Errorf("puzzle %s has fewer than two moves", puzzle.ID)
	}
	board, err := chess.ParseFEN(puzzle.FEN)
	if err != nil {
		return puzzleState{}, err
	}
	return puzzleState{
		puzzle:          puzzle,
		board:           board,
		theme:           theme,
		solverColor:     board.ColorToMove.Opponent(),
		perspective:     board.ColorToMove.Opponent(),
		phase:           puzzlePreview,
		phaseDeadline:   now.Add(puzzleInitialDelay),
		helpAvailableAt: now.Add(puzzleHelpDelay),
		positions:       []chess.Board{board},
		hintMoveIndex:   -1,
	}, nil
}

func (p *puzzleState) advance(now time.Time) {
	switch p.phase {
	case puzzlePreview:
		if !now.Before(p.phaseDeadline) {
			p.applyRecordedMove(p.puzzle.Moves[0])
			p.nextSolutionMove = 1
			p.phase = puzzleSolverTurn
		}
	case puzzleWrongMove:
		if !now.Before(p.phaseDeadline) {
			p.board = p.wrongBoard
			p.lastMove = p.wrongLastMove
			p.hasLastMove = p.wrongHasLastMove
			p.phase = puzzleSolverTurn
		}
	case puzzleOpponentReply:
		if !now.Before(p.phaseDeadline) {
			p.applyRecordedMove(p.puzzle.Moves[p.nextSolutionMove])
			p.nextSolutionMove++
			if p.nextSolutionMove == len(p.puzzle.Moves) {
				p.finishSolved()
			} else {
				p.phase = puzzleSolverTurn
			}
		}
	}
}

func (p *puzzleState) handleKey(event *tcell.EventKey, width, height int) puzzleAction {
	if _, ok := localBoardLayout(width, height); !ok {
		if event.Key() == tcell.KeyEsc {
			return puzzleReturnToMenu
		}
		return puzzleStay
	}
	if p.isComplete() {
		if event.Key() == tcell.KeyEsc {
			return puzzleReturnToMenu
		}
		return puzzleStay
	}
	if p.phase != puzzleSolverTurn {
		if event.Key() == tcell.KeyEsc {
			return puzzleReturnToMenu
		}
		return puzzleStay
	}
	if len(p.promotionChoices) != 0 {
		switch event.Key() {
		case tcell.KeyEsc:
			p.promotionChoices = nil
			p.message = "Promotion cancelled"
		case tcell.KeyRune:
			if move, ok := promotionMoveForRune(p.promotionChoices, event.Rune()); ok {
				p.submitMove(move, time.Now())
			}
		}
		return puzzleStay
	}
	switch event.Key() {
	case tcell.KeyEsc:
		if p.uciInput != "" {
			p.uciInput = ""
		} else if p.selectedSource != nil {
			p.selectedSource = nil
		} else {
			return puzzleReturnToMenu
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(p.uciInput) != 0 {
			p.uciInput = p.uciInput[:len(p.uciInput)-1]
		}
	case tcell.KeyEnter:
		p.submitUCI(time.Now())
	case tcell.KeyRune:
		character := unicode.ToLower(event.Rune())
		if (character >= 'a' && character <= 'h') ||
			(character >= '1' && character <= '8') ||
			strings.ContainsRune("qrbn", character) {
			p.uciInput += string(character)
		}
	}
	return puzzleStay
}

func (p *puzzleState) handleMouse(event *tcell.EventMouse, width, height int) puzzleAction {
	x, y := event.Position()
	layout, ok := localBoardLayout(width, height)
	if !ok {
		return puzzleStay
	}
	if p.isComplete() {
		return p.handleCompleteMouse(event, layout)
	}
	if event.Buttons() == tcell.ButtonNone {
		p.hoveredFlip = puzzleActiveFlipButtonAt(x, y, layout)
		p.hoveredHint = p.helpAvailable() && puzzleHintButtonAt(x, y, layout)
		p.hoveredSolution = p.helpAvailable() && puzzleSolutionButtonAt(x, y, layout)
		if p.hoveredFlip || p.hoveredHint || p.hoveredSolution {
			return puzzleStay
		}
	}
	if event.Buttons() == tcell.Button1 {
		if puzzleActiveFlipButtonAt(x, y, layout) {
			p.flipBoard()
			return puzzleStay
		}
		if p.helpAvailable() && puzzleHintButtonAt(x, y, layout) {
			p.selectedSource = nil
			p.hoveredSquare = nil
			if p.hintMoveIndex != p.nextSolutionMove {
				p.hintMoveIndex = p.nextSolutionMove
				p.hintCount = 0
			}
			p.hintCount++
			return puzzleStay
		}
		if p.helpAvailable() && puzzleSolutionButtonAt(x, y, layout) {
			p.viewSolution()
			return puzzleStay
		}
	}
	if p.phase != puzzleSolverTurn {
		return puzzleStay
	}
	if len(p.promotionChoices) != 0 {
		character, ok := p.promotionOptionAt(x, y, layout)
		if event.Buttons() == tcell.ButtonNone {
			p.hoveredPromotion = 0
			if ok {
				p.hoveredPromotion = character
			}
			return puzzleStay
		}
		if event.Buttons() == tcell.Button1 && ok {
			if move, ok := promotionMoveForRune(p.promotionChoices, character); ok {
				p.submitMove(move, time.Now())
			}
		}
		return puzzleStay
	}
	square, ok := squareAtScreenPosition(x, y, layout, p.perspective)
	if !ok {
		if event.Buttons() == tcell.Button1 {
			p.selectedSource = nil
			p.hoveredSquare = nil
		}
		return puzzleStay
	}
	if event.Buttons() == tcell.ButtonNone {
		p.updateHover(square)
		return puzzleStay
	}
	if event.Buttons() != tcell.Button1 {
		return puzzleStay
	}
	legalMoves := chess.GenerateLegalMoves(p.board)
	p.hoveredSquare = nil
	if p.selectedSource == nil {
		if p.isSelectableSource(square, legalMoves) {
			p.selectSource(square)
		}
		return puzzleStay
	}
	candidates := p.movesForTarget(square, legalMoves)
	switch len(candidates) {
	case 0:
		if p.isSelectableSource(square, legalMoves) {
			p.selectSource(square)
		} else {
			p.selectedSource = nil
		}
	case 1:
		p.submitMove(candidates[0], time.Now())
	default:
		p.promotionChoices = candidates
	}
	return puzzleStay
}

func (p *puzzleState) handleCompleteMouse(event *tcell.EventMouse, layout boardLayout) puzzleAction {
	x, y := event.Position()
	buttons := event.Buttons()
	if buttons == tcell.ButtonNone {
		p.hoveredFlip = puzzleCompleteFlipButtonAt(x, y, layout)
		p.hoveredContinue = puzzleContinueButtonAt(x, y, layout)
		return puzzleStay
	}
	if buttons == tcell.Button1 {
		if puzzleContinueButtonAt(x, y, layout) {
			return puzzleContinueTraining
		}
		if puzzleCompleteFlipButtonAt(x, y, layout) {
			p.flipBoard()
			return puzzleStay
		}
		if ply, ok := p.historyPlyAt(x, y, layout); ok {
			p.viewPly = ply
			p.syncHistoryAfterClick(y - (layout.y + 1))
		}
		return puzzleStay
	}
	if buttons&tcell.WheelUp != 0 || buttons&tcell.WheelDown != 0 {
		if _, ok := squareAtScreenPosition(x, y, layout, p.perspective); ok {
			direction := 1
			if buttons&tcell.WheelUp != 0 {
				direction = -1
			}
			p.moveReplay(direction)
		} else if historyAreaAt(x, y, layout) {
			if buttons&tcell.WheelUp != 0 {
				p.historyScroll = max(0, p.historyScroll-1)
			} else {
				p.historyScroll = min(p.historyMaxScroll(), p.historyScroll+1)
			}
		}
	}
	return puzzleStay
}

func (p *puzzleState) submitUCI(now time.Time) {
	if len(p.uciInput) != 4 && len(p.uciInput) != 5 {
		p.message = "Enter UCI like e2e4 or e7e8q"
		return
	}
	move, err := p.board.ParseUCI(p.uciInput)
	if err != nil {
		p.message = "That is not a legal move"
		return
	}
	p.submitMove(move, now)
}

func (p *puzzleState) submitMove(move chess.Move, now time.Time) {
	if move == p.puzzle.Moves[p.nextSolutionMove] || p.isAlternativeFinalMate(move) {
		p.applyRecordedMove(move)
		p.nextSolutionMove++
		p.message = "Best move! Keep going.."
		if p.nextSolutionMove == len(p.puzzle.Moves) {
			p.finishSolved()
			return
		}
		p.phase = puzzleOpponentReply
		p.phaseDeadline = now.Add(puzzleReplyDelay)
		return
	}
	p.wrongBoard = p.board
	p.wrongLastMove = p.lastMove
	p.wrongHasLastMove = p.hasLastMove
	if err := p.board.MakeMove(move); err != nil {
		p.message = err.Error()
		return
	}
	p.lastMove = move
	p.hasLastMove = true
	p.clearMoveInput()
	p.message = "That's not the move! Try again.."
	p.phase = puzzleWrongMove
	p.phaseDeadline = now.Add(puzzleWrongDelay)
}

func (p puzzleState) isAlternativeFinalMate(move chess.Move) bool {
	if p.nextSolutionMove != len(p.puzzle.Moves)-1 || !p.hasMateTheme() {
		return false
	}
	next := p.board
	if err := next.MakeMove(move); err != nil {
		return false
	}
	return len(chess.GenerateLegalMoves(next)) == 0 && next.IsColorInCheck(next.ColorToMove)
}

func (p puzzleState) hasMateTheme() bool {
	for _, theme := range p.puzzle.Themes {
		if theme.ID == "mate" || strings.HasPrefix(theme.ID, "mateIn") || strings.HasSuffix(theme.ID, "Mate") {
			return true
		}
	}
	return false
}

func (p *puzzleState) applyRecordedMove(move chess.Move) {
	san, err := p.board.SAN(move)
	if err != nil {
		return
	}
	if err := p.board.MakeMove(move); err != nil {
		return
	}
	p.history = append(p.history, san)
	p.moves = append(p.moves, move)
	p.positions = append(p.positions, p.board)
	p.viewPly = len(p.moves)
	p.lastMove = move
	p.hasLastMove = true
	p.clearMoveInput()
}

func (p *puzzleState) viewSolution() {
	p.applyRecordedMove(p.puzzle.Moves[p.nextSolutionMove])
	p.nextSolutionMove++
	shownPly := len(p.moves)
	for p.nextSolutionMove < len(p.puzzle.Moves) {
		p.applyRecordedMove(p.puzzle.Moves[p.nextSolutionMove])
		p.nextSolutionMove++
	}
	p.viewPly = shownPly
	p.phase = puzzleSolutionViewed
	p.message = ""
}

func (p *puzzleState) finishSolved() {
	p.phase = puzzleSolved
	p.viewPly = len(p.moves)
	p.clearMoveInput()
}

func (p *puzzleState) clearMoveInput() {
	p.selectedSource = nil
	p.hoveredSquare = nil
	p.promotionChoices = nil
	p.hoveredPromotion = 0
	p.uciInput = ""
}

func (p *puzzleState) flipBoard() {
	p.perspective = p.perspective.Opponent()
	p.clearMoveInput()
	p.hoveredFlip = true
}

func (p puzzleState) isComplete() bool {
	return p.phase == puzzleSolved || p.phase == puzzleSolutionViewed
}

func (p puzzleState) helpAvailable() bool {
	return p.phase == puzzleSolverTurn && !time.Now().Before(p.helpAvailableAt)
}

func (p *puzzleState) selectSource(square chess.Square) {
	p.selectedSource = new(chess.Square)
	*p.selectedSource = square
	p.message = ""
}

func (p *puzzleState) updateHover(square chess.Square) {
	legalMoves := chess.GenerateLegalMoves(p.board)
	if p.isSelectableSource(square, legalMoves) {
		p.hoveredSquare = new(chess.Square)
		*p.hoveredSquare = square
		return
	}
	p.hoveredSquare = nil
}

func (p puzzleState) isSelectableSource(square chess.Square, legalMoves []chess.Move) bool {
	piece := p.board.PieceAt(square)
	if piece.Type == chess.PieceNone || piece.Color != p.board.ColorToMove {
		return false
	}
	for _, move := range legalMoves {
		if move.From() == square {
			return true
		}
	}
	return false
}

func (p puzzleState) movesForTarget(target chess.Square, legalMoves []chess.Move) []chess.Move {
	var candidates []chess.Move
	for _, move := range legalMoves {
		if move.From() == *p.selectedSource && p.targetForMove(move) == target {
			candidates = append(candidates, move)
		}
	}
	return candidates
}

func (p puzzleState) targetForMove(move chess.Move) chess.Square {
	if move.Flag() != chess.KingSideCastle && move.Flag() != chess.QueenSideCastle {
		return move.To()
	}
	return p.board.CastlingRights[p.board.ColorToMove][chess.CastlingSide(move.Flag())].RookFrom
}

func (p puzzleState) promotionOptionAt(x, y int, layout boardLayout) (rune, bool) {
	if len(p.promotionChoices) == 0 {
		return 0, false
	}
	target := p.promotionChoices[0].To()
	popupX, targetY := promotionScreenPosition(target, layout, p.perspective)
	if x < popupX || x >= popupX+squareWidth {
		return 0, false
	}
	index := targetY - y - 1
	if index < 0 || index >= len(promotionOptions) {
		return 0, false
	}
	return promotionOptions[index], true
}

func (p puzzleState) displayBoard() chess.Board {
	if p.isComplete() && p.viewPly >= 0 && p.viewPly < len(p.positions) {
		return p.positions[p.viewPly]
	}
	return p.board
}

func (p puzzleState) viewedMove() (chess.Move, bool) {
	if p.isComplete() {
		if p.viewPly > 0 && p.viewPly <= len(p.moves) {
			return p.moves[p.viewPly-1], true
		}
		return chess.Move(0), false
	}
	return p.lastMove, p.hasLastMove
}

func (p *puzzleState) moveReplay(direction int) {
	previousPly := p.viewPly
	p.viewPly = min(max(0, p.viewPly+direction), len(p.moves))
	if replayFullmoveRow(previousPly) != replayFullmoveRow(p.viewPly) {
		p.syncHistoryToView(direction)
	}
}

func (p *puzzleState) syncHistoryAfterClick(row int) {
	if row == 0 {
		p.historyScroll = max(0, p.historyScroll-1)
	} else if row == 3 {
		p.historyScroll = min(p.historyMaxScroll(), p.historyScroll+1)
	}
}

func (p *puzzleState) syncHistoryToView(direction int) {
	if p.viewPly == 0 {
		p.historyScroll = 0
		return
	}
	targetRow := (p.viewPly - 1) / 2
	anchorRow := 1
	if direction > 0 {
		anchorRow = 2
	}
	p.historyScroll = min(max(0, targetRow-anchorRow), p.historyMaxScroll())
}

func (p puzzleState) historyMaxScroll() int {
	return max(0, (len(p.history)+1)/2-4)
}

func (p puzzleState) historyPlyAt(x, y int, layout boardLayout) (int, bool) {
	if !historyAreaAt(x, y, layout) {
		return 0, false
	}
	row := y - (layout.y + 1)
	index := (p.historyScroll + row) * 2
	startX := moveHistoryBoxX(layout) + 1
	if x < startX+moveNumberWidth+2 {
		return 0, false
	}
	if x < startX+moveNumberWidth+2+sanWidth {
		if index < len(p.history) {
			return index + 1, true
		}
		return 0, false
	}
	if x < startX+moveNumberWidth+2+sanWidth+2 {
		return 0, false
	}
	if index+1 < len(p.history) {
		return index + 2, true
	}
	return 0, false
}

func drawPuzzle(screen tcell.Screen, puzzle puzzleState) {
	width, height := screen.Size()
	layout, ok := localBoardLayout(width, height)
	if !ok {
		drawCentered(screen, height/2-1, "Terminal too small", titleStyle)
		drawCentered(screen, height/2+1, "Resize or Esc to return", mutedStyle)
		return
	}
	drawPuzzleBoard(screen, puzzle, layout)
	drawPuzzleHistory(screen, puzzle, layout, width)
	drawPuzzleControls(screen, puzzle, layout)
	if puzzle.isComplete() {
		drawPuzzleThemes(screen, puzzle, layout)
	} else {
		drawPuzzleInput(screen, puzzle, layout)
	}
	if len(puzzle.promotionChoices) != 0 {
		drawPuzzlePromotionPicker(screen, puzzle, layout)
	}
}

func drawPuzzleBoard(screen tcell.Screen, puzzle puzzleState, layout boardLayout) {
	renderer := chess.Renderer{Perspective: puzzle.perspective}
	board := puzzle.displayBoard()
	var legalMoves []chess.Move
	if puzzle.phase == puzzleSolverTurn {
		legalMoves = chess.GenerateLegalMoves(board)
	}
	for row := range 8 {
		square := renderer.SquareAt(row, 0)
		drawString(screen, layout.x-2, layout.y+row, fmt.Sprintf("%d", square.Rank()+1), mutedStyle)
		for column := range 8 {
			square := renderer.SquareAt(row, column)
			style, color := puzzleSquareStyle(puzzle, board, square, legalMoves)
			drawPiece(screen, layout.x+column*squareWidth, layout.y+row, board.PieceAt(square), style, color)
		}
	}
	for column := range 8 {
		square := renderer.SquareAt(0, column)
		drawString(screen, layout.x+column*squareWidth, layout.y+8, fmt.Sprintf("%c ", 'a'+square.File()), mutedStyle)
	}
}

func puzzleSquareStyle(puzzle puzzleState, board chess.Board, square chess.Square, legalMoves []chess.Move) (tcell.Style, chess.RGB) {
	light, dark := puzzle.theme.SquareColors()
	color := light
	if square.IsDark() {
		color = dark
	}
	if king, ok := board.Pieces[board.ColorToMove][chess.King].SingleSquare(); ok && board.IsColorInCheck(board.ColorToMove) && king == square {
		color = chess.RGB{R: 205, G: 50, B: 50}
	}
	if puzzle.phase == puzzleSolverTurn && puzzle.hintCount > 0 && puzzle.hintMoveIndex == puzzle.nextSolutionMove {
		hint := puzzle.puzzle.Moves[puzzle.hintMoveIndex]
		if square == hint.From() || (puzzle.hintCount > 1 && square == hint.To()) {
			color.B = 255
		}
	}
	if puzzle.selectedSource != nil && *puzzle.selectedSource == square {
		color = adjustRGB(color, 28, 28, 70)
	} else if puzzle.selectedSource != nil && puzzleContainsTarget(puzzle, square, legalMoves) {
		color.G = 255
	} else if puzzle.hoveredSquare != nil && *puzzle.hoveredSquare == square {
		color = darkenRGB(color, 71)
	} else if lastMove, ok := puzzle.viewedMove(); ok && (lastMove.From() == square || lastMove.To() == square) {
		color = brightenRGB(color, 31)
	}
	return backgroundStyle.Background(tcell.NewRGBColor(int32(color.R), int32(color.G), int32(color.B))), color
}

func puzzleContainsTarget(puzzle puzzleState, square chess.Square, legalMoves []chess.Move) bool {
	for _, move := range legalMoves {
		if move.From() == *puzzle.selectedSource && puzzle.targetForMove(move) == square {
			return true
		}
	}
	return false
}

func drawPuzzleHistory(screen tcell.Screen, puzzle puzzleState, layout boardLayout, width int) {
	x := moveHistoryBoxX(layout)
	if x+moveHistoryBoxWidth > width {
		return
	}
	drawString(screen, x, layout.y-2, puzzle.title(), titleStyle)
	drawMoveHistoryBorder(screen, x, layout.y)
	start := puzzle.historyScroll
	if !puzzle.isComplete() {
		start, _ = moveHistoryWindow(puzzle.history)
	}
	view := localGameState{history: puzzle.history, viewPly: puzzle.viewPly}
	if puzzle.isComplete() {
		view.terminalTitle = "review"
	}
	for row := range 4 {
		fullmove := start + row
		index := fullmove * 2
		if index >= len(puzzle.history) {
			break
		}
		blackMove := ""
		if index+1 < len(puzzle.history) {
			blackMove = puzzle.history[index+1]
		}
		drawMoveHistoryRow(screen, x+1, layout.y+1+row, fullmove+1, view, index, blackMove)
	}
}

func (p puzzleState) title() string {
	switch p.phase {
	case puzzleSolved:
		return "Success!"
	case puzzleSolutionViewed:
		return "Puzzle complete!"
	default:
		return fmt.Sprintf("%s to move", p.solverColor)
	}
}

func drawPuzzleControls(screen tcell.Screen, puzzle puzzleState, layout boardLayout) {
	x := moveHistoryBoxX(layout)
	if puzzle.isComplete() {
		drawPuzzleButton(screen, x, layout.y+6, "[💪] Continue training", puzzle.hoveredContinue)
		drawPuzzleButton(screen, x, layout.y+7, "[🔄] Flip board", puzzle.hoveredFlip)
	} else {
		if puzzle.helpAvailable() {
			drawPuzzleButton(screen, x, layout.y+6, "[💡] Get a hint", puzzle.hoveredHint)
			drawPuzzleButton(screen, x, layout.y+7, "[🤷] View the solution", puzzle.hoveredSolution)
		}
		drawPuzzleButton(screen, x, layout.y+8, "[🔄] Flip board", puzzle.hoveredFlip)
	}
	drawString(screen, x, layout.y+9, fmt.Sprintf("Rating: %d", puzzle.puzzle.Rating), mutedStyle)
}

func drawPuzzleButton(screen tcell.Screen, x, y int, label string, hovered bool) {
	style := backgroundStyle
	if hovered {
		style = selectedStyle
	}
	drawString(screen, x, y, label, style)
}

func puzzleHintButtonAt(x, y int, layout boardLayout) bool {
	return puzzleButtonAt(x, y, layout, layout.y+6, "[💡] Get a hint")
}

func puzzleSolutionButtonAt(x, y int, layout boardLayout) bool {
	return puzzleButtonAt(x, y, layout, layout.y+7, "[🤷] View the solution")
}

func puzzleActiveFlipButtonAt(x, y int, layout boardLayout) bool {
	return puzzleButtonAt(x, y, layout, layout.y+8, "[🔄] Flip board")
}

func puzzleCompleteFlipButtonAt(x, y int, layout boardLayout) bool {
	return puzzleButtonAt(x, y, layout, layout.y+7, "[🔄] Flip board")
}

func puzzleContinueButtonAt(x, y int, layout boardLayout) bool {
	return puzzleButtonAt(x, y, layout, layout.y+6, "[💪] Continue training")
}

func puzzleButtonAt(x, y int, layout boardLayout, buttonY int, label string) bool {
	buttonX := moveHistoryBoxX(layout)
	return y == buttonY && x >= buttonX && x < buttonX+runewidth.StringWidth(label)
}

func drawPuzzleInput(screen tcell.Screen, puzzle puzzleState, layout boardLayout) {
	x := layout.x - 2
	y := layout.y + 10
	if puzzle.phase != puzzleSolverTurn {
		if puzzle.message != "" {
			drawString(screen, x, y, puzzle.message, mutedStyle)
		}
		return
	}
	drawUCIInputBorder(screen, x, y)
	drawPaddedString(screen, x+1, y+1, uciInputBoxWidth-2, puzzle.uciInput+"_", backgroundStyle)
	if puzzle.message != "" {
		drawString(screen, x, y+4, puzzle.message, mutedStyle)
	}
	drawString(screen, x, y+5, "Esc: clear selection or return to menu", mutedStyle)
}

func drawPuzzleThemes(screen tcell.Screen, puzzle puzzleState, layout boardLayout) {
	_, height := screen.Size()
	x := layout.x - 2
	y := layout.y + 10
	labels := make([]string, 0, len(puzzle.puzzle.Themes))
	for _, theme := range puzzle.puzzle.Themes {
		labels = append(labels, theme.DisplayName)
	}
	rows := min(len(labels), max(1, height-y-4))
	if len(labels) > rows {
		labels = append(labels[:rows-1], fmt.Sprintf("+%d more", len(labels)-rows+1))
	}
	topLabel := "━ Puzzle themes "
	topBorder := "┏" + topLabel + strings.Repeat("━", puzzleThemesBoxWidth-runewidth.StringWidth(topLabel)-2) + "┓"
	drawString(screen, x, y, topBorder, mutedStyle)
	drawString(screen, x+3, y, "Puzzle themes", titleStyle)
	for row, label := range labels {
		drawString(screen, x, y+row+1, "┃", mutedStyle)
		drawPaddedString(screen, x+1, y+row+1, puzzleThemesBoxWidth-2, label, backgroundStyle)
		drawString(screen, x+puzzleThemesBoxWidth-1, y+row+1, "┃", mutedStyle)
	}
	drawString(screen, x, y+len(labels)+1, "┗"+strings.Repeat("━", puzzleThemesBoxWidth-2)+"┛", mutedStyle)
	if y+len(labels)+3 < height {
		drawString(screen, x, y+len(labels)+3, "Esc: return to menu", mutedStyle)
	}
}

func drawPuzzlePromotionPicker(screen tcell.Screen, puzzle puzzleState, layout boardLayout) {
	target := puzzle.promotionChoices[0].To()
	x, targetY := promotionScreenPosition(target, layout, puzzle.perspective)
	promotingColor := puzzle.board.ColorToMove
	foreground := tcell.ColorWhite
	background := tcell.ColorBlack
	if promotingColor == chess.Black {
		foreground = tcell.ColorBlack
		background = tcell.ColorWhite
	}
	for index, character := range promotionOptions {
		style := backgroundStyle.Foreground(foreground).Background(background)
		if puzzle.hoveredPromotion == character {
			style = style.Underline(true)
		}
		piece := chess.Piece{Color: promotingColor, Type: promotionPieceType(character)}
		drawString(screen, x, targetY-index-1, piece.Label()+" ", style)
	}
}
