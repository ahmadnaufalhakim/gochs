package pgn

import (
	"fmt"
	"strings"
	"time"
)

const Annotator = "github.com/ahmadnaufalhakim/gochs"

type Game struct {
	Event       string
	Site        string
	Date        time.Time
	Round       string
	White       string
	Black       string
	Result      string
	Variant     string
	TimeControl string
	Termination string
	Moves       []string
	Clocks      []time.Duration
}

func Format(game Game, includeClocks bool) string {
	tags := [][2]string{
		{"Event", game.Event},
		{"Site", game.Site},
		{"Date", game.Date.Format("2006.01.02")},
		{"Round", game.Round},
		{"White", game.White},
		{"Black", game.Black},
		{"Result", game.Result},
		{"Annotator", Annotator},
		{"Variant", game.Variant},
		{"TimeControl", game.TimeControl},
		{"Termination", game.Termination},
	}

	var output strings.Builder
	for _, tag := range tags {
		fmt.Fprintf(&output, "[%s \"%s\"]\n", tag[0], escapeTagValue(tag[1]))
	}
	output.WriteString("\n")
	output.WriteString(formatMoves(game.Moves, game.Clocks, includeClocks))
	if len(game.Moves) != 0 {
		output.WriteByte(' ')
	}
	output.WriteString(game.Result)
	output.WriteByte('\n')

	return output.String()
}

func escapeTagValue(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, "\"", "\\\"")
}

func formatMoves(moves []string, clocks []time.Duration, includeClocks bool) string {
	var output strings.Builder
	for index, move := range moves {
		if index > 0 {
			output.WriteByte(' ')
		}
		fullmove := index/2 + 1
		if index%2 == 0 {
			fmt.Fprintf(&output, "%d. ", fullmove)
		} else if includeClocks {
			fmt.Fprintf(&output, "%d... ", fullmove)
		}
		output.WriteString(move)
		if includeClocks && index < len(clocks) {
			fmt.Fprintf(&output, " { [%%clk %s] }", formatClock(clocks[index]))
		}
	}

	return output.String()
}

func formatClock(clock time.Duration) string {
	seconds := int((clock + time.Second - 1) / time.Second)
	return fmt.Sprintf("%d:%02d:%02d", seconds/3600, (seconds/60)%60, seconds%60)
}
