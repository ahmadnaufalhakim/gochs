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

func TestBitboardIsSingleBit(t *testing.T) {
	tests := []struct {
		bitboard Bitboard
		want     bool
	}{
		{0, false},
		{a1.Mask(), true},
		{h8.Mask(), true},
		{a1.Mask() | h8.Mask(), false},
	}

	for _, test := range tests {
		if got := test.bitboard.IsSingleBit(); got != test.want {
			t.Errorf("Bitboard(%#x).IsSingleBit() = %t, want %t", test.bitboard, got, test.want)
		}
	}
}

func TestBitboardPrint(t *testing.T) {
	got := captureStdout(t, func() {
		a1.Mask().Print()
	})

	want := "1 . . . . . . . \n" +
		". . . . . . . . \n" +
		". . . . . . . . \n" +
		". . . . . . . . \n" +
		". . . . . . . . \n" +
		". . . . . . . . \n" +
		". . . . . . . . \n" +
		". . . . . . . . \n"
	if got != want {
		t.Errorf("Print() = %q, want %q", got, want)
	}
}
