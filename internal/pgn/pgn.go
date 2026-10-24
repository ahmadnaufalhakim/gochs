package pgn

import (
	"fmt"
	"strings"
	"time"
)

const Annotator = "github.com/ahmadnaufalhakim/gochs"

type Game struct {
	Event           string
	Site            string
	Date            time.Time
	Round           string
	White           string
	Black           string
	Result          string
	Variant         string
	TimeControl     string
	Termination     string
	GameID          string
	UTCDate         string
	UTCTime         string
	WhiteElo        string
	BlackElo        string
	WhiteRatingDiff string
	BlackRatingDiff string
	Moves           []string
	Clocks          []time.Duration
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
	}
	if game.GameID != "" {
		tags = append(tags, [2]string{"GameId", game.GameID})
	}
	if game.UTCDate != "" {
		tags = append(tags, [2]string{"UTCDate", game.UTCDate})
	}
	if game.UTCTime != "" {
		tags = append(tags, [2]string{"UTCTime", game.UTCTime})
	}
	if game.WhiteElo != "" {
		tags = append(tags, [2]string{"WhiteElo", game.WhiteElo})
	}
	if game.BlackElo != "" {
		tags = append(tags, [2]string{"BlackElo", game.BlackElo})
	}
	if game.WhiteRatingDiff != "" {
		tags = append(tags, [2]string{"WhiteRatingDiff", game.WhiteRatingDiff})
	}
	if game.BlackRatingDiff != "" {
		tags = append(tags, [2]string{"BlackRatingDiff", game.BlackRatingDiff})
	}
	tags = append(tags,
		[2]string{"Variant", game.Variant},
		[2]string{"TimeControl", game.TimeControl},
		[2]string{"Termination", game.Termination},
		[2]string{"Annotator", Annotator},
	)

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
