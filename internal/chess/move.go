package chess

import "fmt"

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

func slidingMoveDestinationsFrom(b Board, from Square, deltas []MoveDelta) Bitboard {
	attacks := slidingAttacksFrom(b, from, deltas)
	return attacks &^ b.OccupiedByColor(b.ColorToMove)
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

type Move uint16
type MoveFlag uint8

const (
	KingSideCastle MoveFlag = iota
	QueenSideCastle
	RESERVED_2
	RESERVED_3
	QuietMove
	Capture
	DoublePawnPush
	EnPassant
	PromoteKnight
	PromoteBishop
	PromoteRook
	PromoteQueen
	PromoteCaptureKnight
	PromoteCaptureBishop
	PromoteCaptureRook
	PromoteCaptureQueen
)

func NewMove(from, to Square, flag MoveFlag) Move {
	return Move(from) | Move(to)<<6 | Move(flag)<<12
}

func (m Move) From() Square {
	return Square(m & 0x3F)
}

func (m Move) To() Square {
	return Square((m >> 6) & 0x3F)
}

func (m Move) Flag() MoveFlag {
	return MoveFlag((m >> 12) & 0xF)
}

func (m Move) String() string {
	from := m.From().String()
	to := m.To().String()

	switch m.Flag() {
	case KingSideCastle:
		return from + "-" + to + " (O-O)"
	case QueenSideCastle:
		return from + "-" + to + " (O-O-O)"
	case QuietMove:
		return from + "-" + to
	case Capture:
		return from + "x" + to
	case DoublePawnPush:
		return from + "-" + to + " (double push)"
	case EnPassant:
		return from + "x" + to + " e.p."
	case PromoteKnight:
		return from + "-" + to + "=N"
	case PromoteBishop:
		return from + "-" + to + "=B"
	case PromoteRook:
		return from + "-" + to + "=R"
	case PromoteQueen:
		return from + "-" + to + "=Q"
	case PromoteCaptureKnight:
		return from + "x" + to + "=N"
	case PromoteCaptureBishop:
		return from + "x" + to + "=B"
	case PromoteCaptureRook:
		return from + "x" + to + "=R"
	case PromoteCaptureQueen:
		return from + "x" + to + "=Q"
	default:
		return from + "-" + to + " (invalid flag)"
	}
}

func (m Move) Print() {
	fmt.Println(m)
}

func GeneratePawnMoveDestinations(b Board) Bitboard {
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

func GeneratePawnCaptureMoveDestinations(b Board) Bitboard {
	var moves Bitboard
	pawns := b.Pieces[b.ColorToMove][Pawn]
	occupiedByOpponent := b.OccupiedByColor(b.ColorToMove.Opponent())

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

func GenerateKnightMoveDestinations(b Board) Bitboard {
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

func GenerateBishopMoveDestinations(b Board) Bitboard {
	var moves Bitboard
	bishops := b.Pieces[b.ColorToMove][Bishop]

	for from := a1; from <= h8; from++ {
		if bishops&from.Mask() != 0 {
			moves |= slidingMoveDestinationsFrom(b, from, bishopMoveDeltas)
		}
	}

	return moves
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

func GenerateRookMoveDestinations(b Board) Bitboard {
	var moves Bitboard
	rooks := b.Pieces[b.ColorToMove][Rook]

	for from := a1; from <= h8; from++ {
		if rooks&from.Mask() != 0 {
			moves |= slidingMoveDestinationsFrom(b, from, rookMoveDeltas)
		}
	}

	return moves
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

func GenerateQueenMoveDestinations(b Board) Bitboard {
	var moves Bitboard
	queens := b.Pieces[b.ColorToMove][Queen]

	for from := a1; from <= h8; from++ {
		if queens&from.Mask() != 0 {
			moves |= slidingMoveDestinationsFrom(b, from, queenMoveDeltas)
		}
	}

	return moves
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

func GenerateKingMoveDestinations(b Board) Bitboard {
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

func GenerateKingAttacks(b Board, c PieceColor) Bitboard {
	var moves Bitboard
	king := b.Pieces[c][King]

	for _, delta := range queenMoveDeltas {
		sources := king & allowedSources(delta)
		shift := delta.File + delta.Rank*8

		if shift > 0 {
			moves |= sources << shift
		} else {
			moves |= sources >> -shift
		}
	}

	return moves
}

func GenerateJumpPseudoLegalMoves(
	b Board,
	color PieceColor,
	pieceType PieceType,
	deltas []MoveDelta,
) []Move {
	var moves []Move
	pieces := b.Pieces[color][pieceType]

	for from := a1; from <= h8; from++ {
		if pieces&from.Mask() == 0 {
			continue
		}

		for _, delta := range deltas {
			source := from.Mask() & allowedSources(delta)
			if source == 0 {
				continue
			}

			file := int8(from.File()) + delta.File
			rank := int8(from.Rank()) + delta.Rank
			to := Square(file + rank*8)
			toPiece := b.PieceAt(to)
			if toPiece.Type != PieceNone {
				if toPiece.Color != color && toPiece.Type != King {
					moves = append(moves, NewMove(from, to, Capture))
				}
				break
			}
			moves = append(moves, NewMove(from, to, QuietMove))
		}
	}

	return moves
}

func GenerateSlidePseudoLegalMoves(
	b Board,
	color PieceColor,
	pieceType PieceType,
	deltas []MoveDelta,
) []Move {
	var moves []Move
	pieces := b.Pieces[color][pieceType]

	for from := a1; from <= h8; from++ {
		if pieces&from.Mask() == 0 {
			continue
		}

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
				toPiece := b.PieceAt(to)
				if toPiece.Type != PieceNone {
					if toPiece.Color != color && toPiece.Type != King {
						moves = append(moves, NewMove(from, to, Capture))
					}
					break
				}
				moves = append(moves, NewMove(from, to, QuietMove))
			}
		}
	}

	return moves
}
