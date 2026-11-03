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

func TestIsBrilliantSacrificeRejectsProtectedCheckingCapture(t *testing.T) {
	board, err := ParseFEN("r2q1bnr/pp3k2/2p1p1pp/8/2B4P/5N2/PPPPQPP1/R1B1KB1R w KQ - 0 12")
	if err != nil {
		t.Fatal(err)
	}
	move, err := board.ParseUCI("e2e6")
	if err != nil {
		t.Fatal(err)
	}
	if IsBrilliantSacrifice(board, move) {
		t.Fatal("protected checking capture was recognized as a sacrifice")
	}
}

func TestIsBrilliantSacrificeRecognizesCaroKannKnightSacrifice(t *testing.T) {
	board, err := ParseFEN("r2qkbnr/pp3p1b/2p1p1pp/4N3/2B4P/5N2/PPPPQPP1/R1B1KB1R w KQkq - 0 11")
	if err != nil {
		t.Fatal(err)
	}
	move, err := board.ParseUCI("e5f7")
	if err != nil {
		t.Fatal(err)
	}
	if !IsBrilliantSacrifice(board, move) {
		t.Fatal("Caro-Kann knight sacrifice was not recognized")
	}
}

func TestIsBrilliantSacrificeRecognizesKnightOffer(t *testing.T) {
	board, err := ParseFEN("r4rk1/p1pppp1p/1p4p1/n3P1N1/2PPQ3/2PB4/q2BKP1P/6R1 w - - 0 19")
	if err != nil {
		t.Fatal(err)
	}
	move, err := board.ParseUCI("g5h7")
	if err != nil {
		t.Fatal(err)
	}
	if !IsBrilliantSacrifice(board, move) {
		t.Fatal("knight offer was not recognized")
	}
}

func TestIsBrilliantSacrificeRecognizesRookOffer(t *testing.T) {
	board, err := ParseFEN("r4r2/p1pppp1k/2p1p1p1/n3P3/2PPQ3/2PB4/q2BKP1P/6R1 w - - 0 20")
	if err != nil {
		t.Fatal(err)
	}
	move, err := board.ParseUCI("g1g6")
	if err != nil {
		t.Fatal(err)
	}
	if !IsBrilliantSacrifice(board, move) {
		t.Fatal("rook offer was not recognized")
	}
}

func TestIsBrilliantSacrificeRecognizesGoldCoinsQueenOffer(t *testing.T) {
	board, err := ParseFEN("5rk1/pp4pp/4p3/2R3Q1/3n4/2q4r/P1PB1PP1/6KR b - - 0 23")
	if err != nil {
		t.Fatal(err)
	}
	move, err := board.ParseUCI("c3g3")
	if err != nil {
		t.Fatal(err)
	}
	if !IsBrilliantSacrifice(board, move) {
		t.Fatal("Gold Coins queen offer was not recognized")
	}
}
