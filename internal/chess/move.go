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

	pawnDoublePushDestinationRank = [PieceColorCount]Bitboard{
		White: rank4,
		Black: rank5,
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

func slidingMovesFrom(b Board, from Square, deltas []MoveDelta) Bitboard {
	own := b.OccupiedByColor(b.ColorToMove)
	occupied := b.Occupied()

	var moves Bitboard
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
			if own&to.Mask() != 0 {
				break
			}

			moves |= to.Mask()

			if occupied&to.Mask() != 0 {
				break
			}
		}
	}

	return moves
}

type Move struct {
	From      Square
	To        Square
	Promotion PieceType
}

func GeneratePawnMoves(b Board) Bitboard {
	var moves Bitboard
	pawns := b.Pieces[b.ColorToMove][Pawn]
	occupied := b.Occupied()
	shift := pawnMoveDelta[b.ColorToMove].File + pawnMoveDelta[b.ColorToMove].Rank*8

	var pawnSinglePushes, pawnDoublePushes Bitboard
	if shift > 0 {
		pawnSinglePushes = (pawns << shift) & ^occupied
		pawnDoublePushes = (pawnSinglePushes << shift) & ^occupied & pawnDoublePushDestinationRank[b.ColorToMove]
	} else {
		pawnSinglePushes = (pawns >> -shift) & ^occupied
		pawnDoublePushes = (pawnSinglePushes >> -shift) & ^occupied & pawnDoublePushDestinationRank[b.ColorToMove]
	}

	moves |= pawnSinglePushes | pawnDoublePushes

	return moves
}

func GeneratePawnCaptureMoves(b Board) Bitboard {
	var moves Bitboard
	pawns := b.Pieces[b.ColorToMove][Pawn]
	occupiedByOpponent := b.Occupied() & ^b.OccupiedByColor(b.ColorToMove)

	for _, delta := range pawnAttackMoveDeltas[b.ColorToMove] {
		sources := pawns & allowedSources(delta)
		shift := delta.File + delta.Rank*8

		if shift > 0 {
			moves |= sources << shift
		} else {
			moves |= sources >> -shift
		}
	}

	moves &= occupiedByOpponent

	return moves
}

func GenerateKnightMoves(b Board) Bitboard {
	var moves Bitboard
	knights := b.Pieces[b.ColorToMove][Knight]

	for _, delta := range knightMoveDeltas {
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

func GenerateBishopMoves(b Board) Bitboard {
	var moves Bitboard
	bishops := b.Pieces[b.ColorToMove][Bishop]

	for from := a1; from <= h8; from++ {
		if bishops&from.Mask() != 0 {
			moves |= slidingMovesFrom(b, from, bishopMoveDeltas)
		}
	}

	return moves
}

func GenerateRookMoves(b Board) Bitboard {
	var moves Bitboard
	rooks := b.Pieces[b.ColorToMove][Rook]

	for from := a1; from <= h8; from++ {
		if rooks&from.Mask() != 0 {
			moves |= slidingMovesFrom(b, from, rookMoveDeltas)
		}
	}

	return moves
}

func GenerateQueenMoves(b Board) Bitboard {
	var moves Bitboard
	queens := b.Pieces[b.ColorToMove][Queen]

	for from := a1; from <= h8; from++ {
		if queens&from.Mask() != 0 {
			moves |= slidingMovesFrom(b, from, queenMoveDeltas)
		}
	}

	return moves
}

func GenerateKingMoves(b Board) Bitboard {
	var moves Bitboard
	king := b.Pieces[b.ColorToMove][King]

	for _, delta := range queenMoveDeltas {
		sources := king & allowedSources(delta)
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
