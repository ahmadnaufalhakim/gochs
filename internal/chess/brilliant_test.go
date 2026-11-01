package chess

import "testing"

func TestIsBrilliantSacrifice(t *testing.T) {
	board, err := ParseFEN("7k/8/6p1/8/8/8/8/3Q2K1 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	move, err := board.ParseUCI("d1h5")
	if err != nil {
		t.Fatal(err)
	}
	if !IsBrilliantSacrifice(board, move) {
		t.Fatal("queen sacrifice was not recognized")
	}
}

func TestIsBrilliantSacrificeRejectsSavingUnsafePiece(t *testing.T) {
	board, err := ParseFEN("6k1/8/6p1/7Q/8/8/8/6K1 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	move, err := board.ParseUCI("h5f3")
	if err != nil {
		t.Fatal(err)
	}
	if IsBrilliantSacrifice(board, move) {
		t.Fatal("move that saves an unsafe queen was recognized")
	}
}

func TestIsBrilliantSacrificeRejectsQueenPromotion(t *testing.T) {
	board, err := ParseFEN("7k/P7/8/8/8/8/8/7K w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	move, err := board.ParseUCI("a7a8q")
	if err != nil {
		t.Fatal(err)
	}
	if IsBrilliantSacrifice(board, move) {
		t.Fatal("queen promotion was recognized")
	}
}
