package chess

import (
	"fmt"
	"math/bits"
)

type Bitboard uint64

// Checks if bitboard only has one 1-valued bit.
func (b Bitboard) IsSingleBit() bool {
	return b != 0 && b&(b-1) == 0
}

// Checks if a single square can be extracted
// from a bitboard.
func (b Bitboard) SingleSquare() (Square, bool) {
	if !b.IsSingleBit() {
		return Square(0), false
	}

	return Square(bits.TrailingZeros64(uint64(b))), true
}

// Canonical bitboard printing function. Canonical meaning
// that its printing doesn't depend on anything (i.e. the chess board).
func (b Bitboard) Print() {
	for file := range uint8(8) {
		for rank := range uint8(8) {
			s := Square(file + rank*8)

			if b&s.Mask() != 0 {
				fmt.Print("1 ")
			} else {
				fmt.Print(". ")
			}
		}
		fmt.Println()
	}
}
