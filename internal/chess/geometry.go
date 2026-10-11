package chess

type MoveDelta struct {
	File int8
	Rank int8
}

var (
	pawnMoveDelta = [PieceColorCount]MoveDelta{
		White: {File: 0, Rank: 1},
		Black: {File: 0, Rank: -1},
	}
	pawnAttackMoveDeltas = [PieceColorCount][]MoveDelta{
		White: {
			{File: -1, Rank: 1}, {File: 1, Rank: 1},
		},
		Black: {
			{File: -1, Rank: -1}, {File: 1, Rank: -1},
		},
	}
	knightMoveDeltas = []MoveDelta{
		{File: -1, Rank: -2}, {File: 1, Rank: -2},
		{File: -2, Rank: -1}, {File: 2, Rank: -1},
		{File: -2, Rank: 1}, {File: 2, Rank: 1},
		{File: -1, Rank: 2}, {File: 1, Rank: 2},
	}
	bishopMoveDeltas = []MoveDelta{
		{File: -1, Rank: -1}, {File: 1, Rank: -1},
		{File: -1, Rank: 1}, {File: 1, Rank: 1},
	}
	rookMoveDeltas = []MoveDelta{
		{File: 0, Rank: -1}, {File: -1, Rank: 0},
		{File: 1, Rank: 0}, {File: 0, Rank: 1},
	}
	queenMoveDeltas = append(bishopMoveDeltas, rookMoveDeltas...)

	pawnEnPassantTargetRank = [PieceColorCount]Bitboard{
		White: rank3,
		Black: rank6,
	}
	pawnDoublePushSourceRank = [PieceColorCount]Bitboard{
		White: rank2,
		Black: rank7,
	}
	pawnPromotionRank = [PieceColorCount]Bitboard{
		White: rank8,
		Black: rank1,
	}
	castlingHomeRank = [PieceColorCount]Bitboard{
		White: rank1,
		Black: rank8,
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

func slidingAttacksFrom(b Board, from Square, deltas []MoveDelta) Bitboard {
	occupied := b.Occupied()

	var attacks Bitboard
	for _, delta := range deltas {
		file := int8(from.File())
		rank := int8(from.Rank())

		for {
			file += delta.File
			rank += delta.Rank
			if file < 0 || file > 7 || rank < 0 || rank > 7 {
				break
			}

			to := Square(file + rank*8)
			attacks |= to.Mask()

			if occupied&to.Mask() != 0 {
				break
			}
		}
	}

	return attacks
}
