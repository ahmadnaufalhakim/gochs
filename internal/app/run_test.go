package app

import (
	"bufio"
	"strings"
	"testing"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
)

func TestSelectMoveReturnsMatchingMove(t *testing.T) {
	from, err := chess.ParseCoordinate("e2")
	if err != nil {
		t.Fatal(err)
	}
	to, err := chess.ParseCoordinate("e4")
	if err != nil {
		t.Fatal(err)
	}

	want := chess.NewMove(from, to, chess.DoublePawnPush)
	got, err := selectMove(bufio.NewReader(strings.NewReader("")), []chess.Move{want}, from, to)
	if err != nil {
		t.Fatalf("selectMove() returned an error: %v", err)
	}
	if got != want {
		t.Errorf("selectMove() = %v, want %v", got, want)
	}
}

func TestSelectMovePromptsForPromotion(t *testing.T) {
	from, err := chess.ParseCoordinate("e7")
	if err != nil {
		t.Fatal(err)
	}
	to, err := chess.ParseCoordinate("e8")
	if err != nil {
		t.Fatal(err)
	}

	legalMoves := []chess.Move{
		chess.NewMove(from, to, chess.PromoteKnight),
		chess.NewMove(from, to, chess.PromoteBishop),
		chess.NewMove(from, to, chess.PromoteRook),
		chess.NewMove(from, to, chess.PromoteQueen),
	}

	want := chess.NewMove(from, to, chess.PromoteQueen)
	got, err := selectMove(bufio.NewReader(strings.NewReader("q\n")), legalMoves, from, to)
	if err != nil {
		t.Fatalf("selectMove() returned an error: %v", err)
	}
	if got != want {
		t.Errorf("selectMove() = %v, want %v", got, want)
	}
}
