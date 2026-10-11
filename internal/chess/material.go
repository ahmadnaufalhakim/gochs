package chess

import "math/bits"

// HasInsufficientMaterial reports positions where checkmate is impossible.
func (b Board) HasInsufficientMaterial() bool {
	for color := range PieceColorCount {
		if b.Pieces[color][Pawn]|b.Pieces[color][Rook]|b.Pieces[color][Queen] != 0 {
			return false
		}
	}

	knights := 0
	bishops := Bitboard(0)
	for color := range PieceColorCount {
		knights += bits.OnesCount64(uint64(b.Pieces[color][Knight]))
		bishops |= b.Pieces[color][Bishop]
	}
	if knights != 0 {
		return knights == 1 && bishops == 0
	}

	var bishopColor *bool
	for square := range 64 {
		if bishops&(Square(square).Mask()) == 0 {
			continue
		}
		isDark := Square(square).IsDark()
		if bishopColor == nil {
			bishopColor = &isDark
			continue
		}
		if *bishopColor != isDark {
			return false
		}
	}

	return true
}
