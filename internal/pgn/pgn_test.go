package pgn

import (
	"strings"
	"testing"
	"time"
)

func TestFormatPlainGame(t *testing.T) {
	game := Game{
		Event:       "rapid game",
		Site:        "https://github.com/ahmadnaufalhakim/gochs",
		Date:        time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC),
		Round:       "-",
		White:       "White",
		Black:       "Black",
		Result:      "1-0",
		Variant:     "Standard",
		TimeControl: "600+5",
		Termination: "Normal",
		Moves:       []string{"e4", "e5", "Qh5", "Nc6", "Bc4", "Nf6", "Qxf7#"},
	}

	got := Format(game, false)
	for _, tag := range []string{
		"[Event \"rapid game\"]",
		"[Site \"https://github.com/ahmadnaufalhakim/gochs\"]",
		"[Date \"2026.10.12\"]",
		"[Round \"-\"]",
		"[White \"White\"]",
		"[Black \"Black\"]",
		"[Result \"1-0\"]",
		"[Annotator \"github.com/ahmadnaufalhakim/gochs\"]",
		"[Variant \"Standard\"]",
		"[TimeControl \"600+5\"]",
		"[Termination \"Normal\"]",
	} {
		if !strings.Contains(got, tag) {
			t.Errorf("PGN is missing %q:\n%s", tag, got)
		}
	}
	if !strings.Contains(got, "1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0\n") {
		t.Errorf("movetext = %q", got)
	}
}

func TestFormatClockAnnotatedGame(t *testing.T) {
	game := Game{
		Event:       "bullet game",
		Site:        "site",
		Date:        time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC),
		Round:       "-",
		White:       "White",
		Black:       "Black",
		Result:      "1/2-1/2",
		Variant:     "Standard",
		TimeControl: "60+0",
		Termination: "Normal",
		Moves:       []string{"e4", "e5", "Nf3"},
		Clocks:      []time.Duration{time.Minute, 59 * time.Second, 58300 * time.Millisecond},
	}

	got := Format(game, true)
	want := "1. e4 { [%clk 0:01:00] } 1... e5 { [%clk 0:00:59] } 2. Nf3 { [%clk 0:00:59] } 1/2-1/2\n"
	if !strings.Contains(got, want) {
		t.Errorf("clock movetext = %q, want %q", got, want)
	}
}

func TestFormatEscapesTagValues(t *testing.T) {
	game := Game{Event: "A \"quoted\" game", Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	if got := Format(game, false); !strings.Contains(got, "[Event \"A \\\"quoted\\\" game\"]") {
		t.Errorf("escaped Event = %q", got)
	}
}
