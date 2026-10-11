package chess

func (b *Board) IsColorInCheck(c PieceColor) bool {
	king := b.Pieces[c][King]

	for square := a1; square <= h8; square++ {
		if king&square.Mask() != 0 {
			return b.IsSquareAttackedBy(square, c.Opponent())
		}
	}

	return false
}

func (b *Board) IsSquareAttackedBy(s Square, by PieceColor) bool {
	mask := s.Mask()

	return GeneratePawnAttacks(*b, by)&mask != 0 ||
		GenerateKnightAttacks(*b, by)&mask != 0 ||
		GenerateBishopAttacks(*b, by)&mask != 0 ||
		GenerateRookAttacks(*b, by)&mask != 0 ||
		GenerateQueenAttacks(*b, by)&mask != 0 ||
		GenerateKingAttacks(*b, by)&mask != 0
}

func GeneratePawnAttacks(b Board, c PieceColor) Bitboard {
	var attacks Bitboard
	pawns := b.Pieces[c][Pawn]

	for _, delta := range pawnAttackMoveDeltas[c] {
		sources := pawns & allowedSources(delta)
		shift := delta.File + delta.Rank*8

		if shift > 0 {
			attacks |= sources << shift
		} else {
			attacks |= sources >> -shift
		}
	}

	return attacks
}

func GenerateKnightAttacks(b Board, c PieceColor) Bitboard {
	var attacks Bitboard
	knights := b.Pieces[c][Knight]

	for _, delta := range knightMoveDeltas {
		sources := knights & allowedSources(delta)
		shift := delta.File + delta.Rank*8

		if shift > 0 {
			attacks |= sources << shift
		} else {
			attacks |= sources >> -shift
		}
	}

	return attacks
}

func GenerateBishopAttacks(b Board, c PieceColor) Bitboard {
	var attacks Bitboard
	bishops := b.Pieces[c][Bishop]

	for from := a1; from <= h8; from++ {
		if bishops&from.Mask() != 0 {
			attacks |= slidingAttacksFrom(b, from, bishopMoveDeltas)
		}
	}

	return attacks
}

func GenerateRookAttacks(b Board, c PieceColor) Bitboard {
	var attacks Bitboard
	rooks := b.Pieces[c][Rook]

	for from := a1; from <= h8; from++ {
		if rooks&from.Mask() != 0 {
			attacks |= slidingAttacksFrom(b, from, rookMoveDeltas)
		}
	}

	return attacks
}

func GenerateQueenAttacks(b Board, c PieceColor) Bitboard {
	var attacks Bitboard
	queens := b.Pieces[c][Queen]

	for from := a1; from <= h8; from++ {
		if queens&from.Mask() != 0 {
			attacks |= slidingAttacksFrom(b, from, queenMoveDeltas)
		}
	}

	return attacks
}

func GenerateKingAttacks(b Board, c PieceColor) Bitboard {
	var attacks Bitboard
	king := b.Pieces[c][King]

	for _, delta := range queenMoveDeltas {
		sources := king & allowedSources(delta)
		shift := delta.File + delta.Rank*8

		if shift > 0 {
			attacks |= sources << shift
		} else {
			attacks |= sources >> -shift
		}
	}

	return attacks
}
