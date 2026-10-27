package engine

import (
	"testing"
	"time"
)

func TestDifficultySkill(t *testing.T) {
	for level, want := range []int{0, 3, 6, 9, 11, 14, 17, 20} {
		got, err := DifficultySkill(level + 1)
		if err != nil || got != want {
			t.Errorf("DifficultySkill(%d) = %d, %v; want %d, nil", level+1, got, err, want)
		}
	}
}

func TestUnlimitedMoveTime(t *testing.T) {
	first, _ := UnlimitedMoveTime(1)
	last, _ := UnlimitedMoveTime(8)
	if first != 150*time.Millisecond || last != 1200*time.Millisecond {
		t.Errorf("unlimited move times = %s, %s", first, last)
	}
}

func TestSelectMove(t *testing.T) {
	levelOne, _ := DifficultyPolicy(1)
	legal := []string{"e2e4", "d2d4", "g1f3"}
	if got := SelectMove(levelOne, "e2e4", legal, func(int) int { return 0 }); got != "d2d4" {
		t.Errorf("level one move = %q, want a legal alternative", got)
	}
	levelEight, _ := DifficultyPolicy(8)
	if got := SelectMove(levelEight, "e2e4", legal, func(int) int { return 0 }); got != "e2e4" {
		t.Errorf("level eight move = %q, want best move", got)
	}
}

func TestParseAnalysis(t *testing.T) {
	analysis, ok := parseAnalysis("info depth 18 seldepth 26 score cp -37 nodes 1024 pv e2e4 e7e5 g1f3")
	if !ok {
		t.Fatal("parseAnalysis() returned false")
	}
	if analysis.Depth != 18 || analysis.Score.IsMate || analysis.Score.Centipawns != -37 {
		t.Errorf("analysis = %#v", analysis)
	}
	if len(analysis.Lines) != 1 || analysis.Lines[0].Rank != 1 {
		t.Errorf("lines = %#v, want one rank-1 line", analysis.Lines)
	}
	if got, want := analysis.PV, []string{"e2e4", "e7e5", "g1f3"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("PV = %q, want %q", got, want)
	}
}

func TestParseAnalysisMateAndSinglePV(t *testing.T) {
	analysis, ok := parseAnalysis("info depth 9 multipv 1 score mate -3 pv h7h8q")
	if !ok || !analysis.Score.IsMate || analysis.Score.Mate != -3 {
		t.Errorf("parseAnalysis(mate) = %#v, %t", analysis, ok)
	}
	analysis, ok = parseAnalysis("info depth 9 multipv 2 score cp 10 pv h7h8q")
	if !ok || len(analysis.Lines) != 1 || analysis.Lines[0].Rank != 2 {
		t.Errorf("parseAnalysis(rank 2) = %#v, %t", analysis, ok)
	}
	if analysis.Depth != 0 || analysis.PV != nil {
		t.Errorf("non-primary analysis populated legacy fields: %#v", analysis)
	}
}

func TestParseAnalysisRejectsBoundScore(t *testing.T) {
	if _, ok := parseAnalysis("info depth 16 multipv 2 score cp 50 lowerbound pv d2d4"); ok {
		t.Fatal("parseAnalysis() accepted a bounded score")
	}
}

func TestValidConfig(t *testing.T) {
	for _, multiPV := range []int{0, 1, 2, 3} {
		if err := validConfig(Config{MultiPV: multiPV}); err != nil {
			t.Errorf("validConfig(%d) = %v", multiPV, err)
		}
	}
	for _, multiPV := range []int{-1, 4} {
		if err := validConfig(Config{MultiPV: multiPV}); err == nil {
			t.Errorf("validConfig(%d) returned nil", multiPV)
		}
	}
}

func TestMergeAnalysisKeepsFreshLinesByRank(t *testing.T) {
	first, _ := parseAnalysis("info depth 10 multipv 1 score cp 25 pv e2e4 e7e5")
	second, _ := parseAnalysis("info depth 10 multipv 2 score cp 12 pv d2d4 d7d5")
	updated, _ := parseAnalysis("info depth 11 multipv 1 score mate 3 pv g1f3 d7d5")

	analysis := mergeAnalysis(Analysis{}, first)
	analysis = mergeAnalysis(analysis, second)
	analysis = mergeAnalysis(analysis, updated)
	analysis.BestMove = "g1f3"

	if analysis.BestMove != "g1f3" || analysis.Depth != 11 || !analysis.Score.IsMate || analysis.Score.Mate != 3 {
		t.Errorf("primary analysis = %#v", analysis)
	}
	if len(analysis.Lines) != 1 {
		t.Fatalf("line count = %d, want 1", len(analysis.Lines))
	}
	if analysis.Lines[0].Rank != 1 || analysis.Lines[0].Depth != 11 {
		t.Errorf("lines = %#v", analysis.Lines)
	}
}

func TestMergeAnalysisRequiresCurrentPrimaryVariation(t *testing.T) {
	secondary, _ := parseAnalysis("info depth 16 multipv 2 score cp 20 pv d2d4 d7d5")
	primary, _ := parseAnalysis("info depth 16 multipv 1 score cp 150 pv e2e4 e7e5")

	analysis := mergeAnalysis(Analysis{}, secondary)
	if len(analysis.Lines) != 0 {
		t.Fatalf("secondary line without primary = %#v", analysis.Lines)
	}
	analysis = mergeAnalysis(analysis, primary)
	analysis = mergeAnalysis(analysis, secondary)
	if len(analysis.Lines) != 2 {
		t.Fatalf("current primary and secondary lines = %#v", analysis.Lines)
	}
}

func TestValidAnalysisRequest(t *testing.T) {
	for _, test := range []struct {
		fen   string
		depth int
		valid bool
	}{
		{"8/8/8/8/8/8/8/8 w - - 0 1", 12, true},
		{"", 12, false},
		{"8/8\nstop", 12, false},
		{"8/8/8/8/8/8/8/8 w - - 0 1", 0, false},
	} {
		err := validAnalysisRequest(test.fen, test.depth)
		if (err == nil) != test.valid {
			t.Errorf("validAnalysisRequest(%q, %d) error = %v, valid = %t", test.fen, test.depth, err, test.valid)
		}
	}
}
