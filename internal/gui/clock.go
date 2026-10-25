package gui

import (
	"fmt"
	"math/bits"
	"strings"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/sound"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

func (g *localGameState) startClock(control timeControl) {
	g.timeControl = control
	if !control.isTimed() {
		return
	}
	g.clockRemaining[chess.White] = control.initial
	g.clockRemaining[chess.Black] = control.initial
	g.turnStarted = time.Now()
}

func (g *localGameState) advanceClock(now time.Time) {
	if !g.timeControl.isTimed() || g.hasTerminalResult() || g.turnStarted.IsZero() {
		return
	}
	color := g.board.ColorToMove
	elapsed := now.Sub(g.turnStarted)
	if elapsed <= 0 {
		return
	}
	if elapsed >= g.clockRemaining[color] {
		g.clockRemaining[color] = 0
		g.timedOutBy = new(chess.PieceColor)
		*g.timedOutBy = color
		winner := color.Opponent()
		g.terminalTitle = fmt.Sprintf("%s ran out of time · %s wins %s", color, winner, winner.EndResult())
		g.selectedSource = nil
		g.hoveredSquare = nil
		g.promotionChoices = nil
		g.uciInput = ""
		g.endSounds()
		return
	}
	g.clockRemaining[color] -= elapsed
	g.turnStarted = now
	g.playLowTime(color)
}

func (g *localGameState) playLowTime(color chess.PieceColor) {
	if !g.timeControl.isTimed() || g.lowTimePlayed[color] {
		return
	}
	threshold := (g.timeControl.initial + 40*g.timeControl.increment) / 10
	if g.clockRemaining[color] <= threshold {
		g.lowTimePlayed[color] = true
		g.play(sound.LowTime)
	}
}

func drawClocks(screen tcell.Screen, game localGameState, layout boardLayout) {
	board := game.boardForDisplay()
	for color := range chess.PieceColorCount {
		style := backgroundStyle
		if game.timeControl.isTimed() && !game.hasTerminalResult() && chess.PieceColor(color) == game.board.ColorToMove {
			style = selectedStyle
		}
		player := chess.PieceColor(color)
		label := fmt.Sprintf("%-5s", player)
		if game.timeControl.isTimed() {
			label += " " + formatClock(game.clockRemaining[color])
		}
		x := moveHistoryBoxX(layout)
		drawString(screen, x, layout.y+9+int(color), label, style)
		if material := materialIndicator(board, player); material != "" {
			drawString(screen, x+runewidth.StringWidth(label)+1, layout.y+9+int(color), material, backgroundStyle)
		}
	}
}

func materialIndicator(board chess.Board, color chess.PieceColor) string {
	deltas := materialDeltas(board)
	var indicator strings.Builder
	for _, pieceType := range []chess.PieceType{chess.Queen, chess.Rook, chess.Bishop, chess.Knight, chess.Pawn} {
		count := deltas[color][pieceType]
		if count == 0 {
			continue
		}
		indicator.WriteString(chess.PieceLabel[chess.White][pieceType])
		if count > 1 {
			fmt.Fprintf(&indicator, "×%d", count)
		}
	}

	advantage := materialValue(board, color) - materialValue(board, color.Opponent())
	if advantage > 0 {
		fmt.Fprintf(&indicator, " +%d", advantage)
	}

	return indicator.String()
}

func materialDeltas(board chess.Board) [chess.PieceColorCount][chess.PieceTypeCount]int {
	var deltas [chess.PieceColorCount][chess.PieceTypeCount]int
	for color := range chess.PieceColorCount {
		for pieceType := chess.Pawn; pieceType < chess.King; pieceType++ {
			difference := bits.OnesCount64(uint64(board.Pieces[color][pieceType])) - bits.OnesCount64(uint64(board.Pieces[chess.PieceColor(color).Opponent()][pieceType]))
			if difference > 0 {
				deltas[color][pieceType] = difference
			}
		}
	}

	return deltas
}

func materialValue(board chess.Board, color chess.PieceColor) int {
	values := map[chess.PieceType]int{
		chess.Pawn:   1,
		chess.Knight: 3,
		chess.Bishop: 3,
		chess.Rook:   5,
		chess.Queen:  9,
	}

	value := 0
	for pieceType, points := range values {
		value += bits.OnesCount64(uint64(board.Pieces[color][pieceType])) * points
	}
	return value
}

func formatClock(remaining time.Duration) string {
	seconds := int((remaining + time.Second - 1) / time.Second)
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}
