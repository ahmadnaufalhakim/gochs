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
