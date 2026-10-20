package gui3d

import (
	"testing"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
)

func TestLegalDestinationForSelectedPiece(t *testing.T) {
	var board chess.Board
	board.Reset()
	e2, err := chess.ParseCoordinate("e2")
	if err != nil {
		t.Fatal(err)
	}
	e4, err := chess.ParseCoordinate("e4")
	if err != nil {
		t.Fatal(err)
	}
	e5, err := chess.ParseCoordinate("e5")
	if err != nil {
		t.Fatal(err)
	}
	game := state{board: board, selected: &e2}
	if !game.isLegalDestination(e4) {
		t.Error("e4 should be a legal target for the selected pawn")
	}
	if game.isLegalDestination(e5) {
		t.Error("e5 should not be a legal target for the selected pawn")
	}
}
