package chess

type MoveDelta struct {
	File int8
	Rank int8
}

var (
	knightMoveDelta = []MoveDelta{
		{File: -1, Rank: -2}, {File: 1, Rank: -2},
		{File: -2, Rank: -1}, {File: 2, Rank: -1},
		{File: -2, Rank: 1}, {File: 2, Rank: 1},
		{File: -1, Rank: 2}, {File: 1, Rank: 2},
	}
	bishopMoveDelta = []MoveDelta{
		{File: -1, Rank: -1}, {File: 1, Rank: -1},
		{File: -1, Rank: 1}, {File: 1, Rank: 1},
	}
	rookMoveDelta = []MoveDelta{
		{File: 0, Rank: -1}, {File: -1, Rank: 0},
		{File: 1, Rank: 0}, {File: 0, Rank: 1},
	}
)

func allowedSources(delta MoveDelta) Bitboard {
	var sources Bitboard
	switch delta.File {
	case -2:
		sources = ^(fileA | fileB)
	case -1:
		sources = ^fileA
	case 0:
		sources = ^Bitboard(0)
	case 1:
		sources = ^fileH
	case 2:
		sources = ^(fileG | fileH)
	}
	switch delta.Rank {
	case -2:
		sources &= ^(rank1 | rank2)
	case -1:
		sources &= ^rank1
	case 0:
		sources &= ^Bitboard(0)
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
	for _, delta := range knightMoveDelta {
		sources := knights & allowedSources(delta)

		shift := delta.File + delta.Rank*8
		if shift > 0 {
			moves |= sources << shift
		} else {
			moves |= sources >> -shift
		}
	}
	moves &^= b.OccupiedByColor(b.ColorToMove)

	return moves
}
