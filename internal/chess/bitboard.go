package chess

import "fmt"

type Bitboard uint64

// Canonical bitboard printing function. Canonical meaning
// that its printing doesn't depend on anything (i.e. the chess board).
func (b Bitboard) Print() {
	for rank := range uint8(8) {
		for file := range uint8(8) {
			s := Square(rank*8 + file)

			if b&s.Mask() != 0 {
				fmt.Print("1 ")
			} else {
				fmt.Print(". ")
			}
		}
		fmt.Println()
	}
}
