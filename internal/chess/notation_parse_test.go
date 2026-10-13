package chess

import "testing"

func TestParseSAN(t *testing.T) {
	var board Board
	board.Reset()
	move, err := board.ParseSAN("e4")
	if err != nil {
		t.Fatal(err)
	}
	if move.UCI() != "e2e4" {
		t.Errorf("ParseSAN(e4) = %s, want e2e4", move.UCI())
	}
}
