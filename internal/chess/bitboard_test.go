package chess

import "testing"

func TestBitboardCoversAllSquares(t *testing.T) {
	if got, want := Bitboard(1)<<a1, Bitboard(0x0000000000000001); got != want {
		t.Errorf("a1 bit = %#016x, want %#016x", got, want)
	}
	if got, want := Bitboard(1)<<h8, Bitboard(0x8000000000000000); got != want {
		t.Errorf("h8 bit = %#016x, want %#016x", got, want)
	}
}
