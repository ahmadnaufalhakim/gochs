package chess

import "testing"

func TestParseFENRoundTrip(t *testing.T) {
	const fen = "r3k2r/ppp2ppp/2n5/4p3/4P3/2N5/PPP2PPP/R3K2R w KQkq - 12 19"
	board, err := ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	got, err := board.FEN()
	if err != nil {
		t.Fatal(err)
	}
	if got != fen {
		t.Errorf("FEN() = %q, want %q", got, fen)
	}
}

func TestParseFENRejectsInvalidPlacement(t *testing.T) {
	if _, err := ParseFEN("8/8/8/8/8/8/8/8 w - - 0 1"); err == nil {
		t.Fatal("ParseFEN accepted a position without kings")
	}
}

func TestParseUCIMoves(t *testing.T) {
	var board Board
	board.Reset()
	moves, err := ParseUCIMoves(board, "e2e4 e7e5 g1f3")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(moves); got != 3 {
		t.Fatalf("move count = %d, want 3", got)
	}
	for index, want := range []string{"e2e4", "e7e5", "g1f3"} {
		if got := moves[index].UCI(); got != want {
			t.Errorf("move %d = %q, want %q", index, got, want)
		}
	}
}

func TestParseUCIPromotion(t *testing.T) {
	board, err := ParseFEN("7k/P7/8/8/8/8/8/K7 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	move, err := board.ParseUCI("a7a8q")
	if err != nil {
		t.Fatal(err)
	}
	if move.Flag() != PromoteQueen {
		t.Errorf("promotion flag = %d, want queen promotion", move.Flag())
	}
}
