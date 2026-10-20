package gui3d

import (
	"testing"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
)

func TestSquareWorldRoundTrip(t *testing.T) {
	for _, coordinate := range []string{"a1", "e4", "h8"} {
		square, err := chess.ParseCoordinate(coordinate)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := SquareAtWorld(SquareCenter(square))
		if !ok || got != square {
			t.Errorf("SquareAtWorld(%s) = %s, %t", coordinate, got, ok)
		}
	}
}

func TestWhitePerspectiveUsesStandardFileDirection(t *testing.T) {
	a1, err := chess.ParseCoordinate("a1")
	if err != nil {
		t.Fatal(err)
	}
	h1, err := chess.ParseCoordinate("h1")
	if err != nil {
		t.Fatal(err)
	}
	if SquareCenter(a1).X <= SquareCenter(h1).X {
		t.Error("a1 should be left of h1 from the White camera perspective")
	}
	if !a1.IsDark() {
		t.Error("a1 must be a dark square")
	}
}
