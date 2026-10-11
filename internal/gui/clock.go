package gui

import (
	"fmt"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/gdamore/tcell/v2"
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
		winner := color.Opponent()
		g.terminalTitle = fmt.Sprintf("%s ran out of time · %s wins %s", color, winner, winner.EndResult())
		g.selectedSource = nil
		g.hoveredSquare = nil
		g.promotionChoices = nil
		g.uciInput = ""
		return
	}
	g.clockRemaining[color] -= elapsed
	g.turnStarted = now
}

func drawClocks(screen tcell.Screen, game localGameState, layout boardLayout) {
	if !game.timeControl.isTimed() {
		return
	}
	for color := range chess.PieceColorCount {
		style := backgroundStyle
		if !game.hasTerminalResult() && chess.PieceColor(color) == game.board.ColorToMove {
			style = selectedStyle
		}
		label := fmt.Sprintf("%-5s %s", chess.PieceColor(color), formatClock(game.clockRemaining[color]))
		drawString(screen, moveHistoryBoxX(layout), layout.y+9+int(color), label, style)
	}
}

func formatClock(remaining time.Duration) string {
	seconds := int((remaining + time.Second - 1) / time.Second)
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}
