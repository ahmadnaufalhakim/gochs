package chess

var knightMoveOffsets = [][2]int{
	{-1, -2}, {1, -2},
	{-2, -1}, {2, -1},
	{-2, 1}, {2, 1},
	{-1, 2}, {1, 2},
}

func knightAllowedSources(offset [2]int) Bitboard {
	fileOffset := offset[0]
	rankOffset := offset[1]

	sources := Bitboard(0)
	switch fileOffset {
	case -2:
		sources = ^(fileA | fileB)
	case -1:
		sources = ^fileA
	case 1:
		sources = ^fileH
	case 2:
		sources = ^(fileG | fileH)
	}
	switch rankOffset {
	case -2:
		sources &= ^(rank1 | rank2)
	case -1:
		sources &= ^rank1
	case 1:
		sources &= ^rank8
	case 2:
		sources &= ^(rank7 | rank8)
	}

	return sources
}

type Move struct {
	From      Square
	To        Square
	Promotion PieceType
}

func GeneratePawnMoves(b Board) Bitboard {
	unoccupied := ^b.Occupied()
	pawns := b.Pieces[b.ColorToMove][Pawn]

	var pawnSinglePushFn func(pawns Bitboard) Bitboard
	var doublePushDestinationRank Bitboard
	switch b.ColorToMove {
	case White:
		pawnSinglePushFn = func(pawns Bitboard) Bitboard {
			return (pawns << 8)
		}
		doublePushDestinationRank = rank4
	case Black:
		pawnSinglePushFn = func(pawns Bitboard) Bitboard {
			return (pawns >> 8)
		}
		doublePushDestinationRank = rank5
	}

	pawnSinglePushes := pawnSinglePushFn(pawns) & unoccupied
	pawnDoublePushes := pawnSinglePushFn(pawnSinglePushes) & unoccupied & doublePushDestinationRank

	return pawnSinglePushes | pawnDoublePushes
}

func GenerateKnightMoves(b Board) Bitboard {
	knights := b.Pieces[b.ColorToMove][Knight]

	var moves Bitboard
	for _, offset := range knightMoveOffsets {
		sources := knights & knightAllowedSources(offset)

		shift := offset[0] + offset[1]*8
		if shift > 0 {
			moves |= sources << shift
		} else {
			moves |= sources >> -shift
		}
	}
	moves &^= b.OccupiedByColor(b.ColorToMove)

	return moves
}
