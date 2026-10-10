package gui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

const (
	squareWidth         = 2
	moveNumberWidth     = 4
	sanWidth            = 8
	moveHistoryRowWidth = moveNumberWidth + 2 + sanWidth*2 + 2
)

type localGameState struct {
	board            chess.Board
	theme            chess.ColorTheme
	perspective      chess.PieceColor
	autoFlip         bool
	selectedSource   *chess.Square
	hoveredSquare    *chess.Square
	promotionChoices []chess.Move
	hoveredPromotion rune
	uciInput         string
	message          string
	history          []string
	moves            []chess.Move
	positions        []chess.Board
	viewPly          int
	historyScroll    int
	lastMove         chess.Move
	hasLastMove      bool
	positionCounts   map[uint64]uint
	drawReason       string
	terminalTitle    string
	resignPending    bool
	hoveredResign    bool
	hoveredFlip      bool
	resignedBy       *chess.PieceColor
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
	}
}

func (g *localGameState) handleKey(event *tcell.EventKey, width, height int) bool {
	if _, ok := localBoardLayout(width, height); !ok {
		return event.Key() == tcell.KeyEsc
	}
	if g.hasTerminalResult() {
		return event.Key() == tcell.KeyEsc
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
	if g.hasTerminalResult() {
		g.syncHistoryToView(1)
	}
	if g.autoFlip {
		g.perspective = g.board.ColorToMove
	}
}

func (g *localGameState) flipBoard() {
	g.perspective = g.perspective.Opponent()
	g.selectedSource = nil
	g.hoveredSquare = nil
	g.hoveredFlip = true
}

func (g localGameState) boardForDisplay() chess.Board {
	if g.hasTerminalResult() && g.viewPly >= 0 && g.viewPly < len(g.positions) {
		return g.positions[g.viewPly]
	}
	return g.board
}

func (g localGameState) viewedMove() (chess.Move, bool) {
	if g.hasTerminalResult() && g.viewPly > 0 && g.viewPly <= len(g.moves) {
		return g.moves[g.viewPly-1], true
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
}

func (g localGameState) hasResigned() bool {
	return g.resignedBy != nil
}

func (g localGameState) hasTerminalResult() bool {
	return g.hasResigned() || g.terminalTitle != ""
}

func (g *localGameState) handleReviewMouse(event *tcell.EventMouse, layout boardLayout) {
	x, y := event.Position()
	buttons := event.Buttons()
	if buttons == tcell.ButtonNone {
		g.hoveredFlip = flipButtonAt(x, y, layout)
		return
	}
	if buttons == tcell.Button1 {
		if flipButtonAt(x, y, layout) {
			g.flipBoard()
			return
		}
		if ply, ok := g.historyPlyAt(x, y, layout); ok {
			g.viewPly = ply
			g.syncHistoryToView(0)
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

func (g *localGameState) moveReplay(direction int) {
	viewPly := min(max(0, g.viewPly+direction), len(g.moves))
	if viewPly == g.viewPly {
		return
	}
	g.viewPly = viewPly
	g.syncHistoryToView(direction)
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
	row := y - (layout.y + 2)
	index := (g.historyScroll + row) * 2
	if x < layout.x+20+moveNumberWidth+2 {
		return 0, false
	}
	if x < layout.x+20+moveNumberWidth+2+sanWidth {
		if index < len(g.history) {
			return index + 1, true
		}
		return 0, false
	}
	if x < layout.x+20+moveNumberWidth+2+sanWidth+2 {
		return 0, false
	}
	if index+1 < len(g.history) {
		return index + 2, true
	}
	return 0, false
}

func historyAreaAt(x, y int, layout boardLayout) bool {
	startX := layout.x + 20
	return x >= startX && x < startX+moveHistoryRowWidth && y >= layout.y+2 && y < layout.y+6
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
	drawGameInput(screen, game, layout)
	if len(game.promotionChoices) != 0 {
		drawPromotionPicker(screen, game, layout)
	}
}

func gameTitle(game localGameState) string {
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
	x := layout.x + 20
	if x+moveHistoryRowWidth > width {
		return
	}

	drawString(screen, x, layout.y-2, gameTitle(game), titleStyle)
	drawString(screen, x, layout.y, "Moves", titleStyle)
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
		drawMoveHistoryRow(screen, x, layout.y+2+row, fullmove+1, game, index, blackMove)
	}
	if !game.hasTerminalResult() {
		drawResignButton(screen, game, layout)
	}
	drawFlipButton(screen, game, layout)
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
	drawString(screen, x+moveNumberWidth+2, y, fmt.Sprintf("%-*s", sanWidth, game.history[index]), whiteStyle)
	drawString(screen, x+moveNumberWidth+2+sanWidth, y, "  ", backgroundStyle)
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
	return layout.x + 20, layout.y + 7
}

func resignButtonLabel(pending bool) string {
	if pending {
		return "[🏳️ ] Are you sure?"
	}
	return "[🏳️ ] Resign"
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
	drawString(screen, x, y, "[↻ ] Flip board", style)
}

func flipButtonPosition(layout boardLayout) (int, int) {
	return layout.x + 20, layout.y + 8
}

func flipButtonAt(x, y int, layout boardLayout) bool {
	buttonX, buttonY := flipButtonPosition(layout)
	return y == buttonY && x >= buttonX && x < buttonX+runewidth.StringWidth("[↻ ] Flip board")
}

func drawGameInput(screen tcell.Screen, game localGameState, layout boardLayout) {
	y := layout.y + 10
	if game.hasTerminalResult() {
		drawString(screen, layout.x-2, y+3, "Esc: return to menu", mutedStyle)
		return
	}
	drawString(screen, layout.x-2, y, "Your move (UCI): "+game.uciInput+"_", backgroundStyle)
	if game.message != "" {
		drawString(screen, layout.x-2, y+1, game.message, mutedStyle)
	}
	drawString(screen, layout.x-2, y+3, "Esc: clear selection or return to menu", mutedStyle)
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
	if width < 48 || height < 20 {
		return boardLayout{}, false
	}

	const panelHeight = 18
	return boardLayout{x: (width - 42) / 2, y: (height-panelHeight)/2 + 4}, true
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
