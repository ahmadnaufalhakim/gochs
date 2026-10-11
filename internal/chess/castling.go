package chess

type CastlingSide uint8

const (
	KingSide CastlingSide = iota
	QueenSide
	CastlingSideCount
)

type CastlingRight struct {
	KingFrom  Square
	RookFrom  Square
	Available bool
}

func castlingDestinations(color PieceColor, side CastlingSide) (Square, Square) {
	switch color {
	case White:
		if side == KingSide {
			return g1, f1
		}
		return c1, d1
	case Black:
		if side == KingSide {
			return g8, f8
		}
		return c8, d8
	}

	return Square(0), Square(0)
}

func (b *Board) disableRookCastlingRight(color PieceColor, rookSquare Square) {
	for side := range CastlingSideCount {
		right := &b.CastlingRights[color][side]

		if right.RookFrom == rookSquare {
			right.Available = false
		}
	}
}

func (b *Board) isKingCastlingPathSafe(from, to Square, color PieceColor) bool {
	if from.Rank() != to.Rank() {
		return false
	}

	step := int8(1)
	if to.File() < from.File() {
		step = -1
	} else if to.File() == from.File() {
		step = 0
	}

	for file := int8(from.File()); ; file += step {
		square := Square(file + int8(from.Rank())*8)

		if b.IsSquareAttackedBy(square, color.Opponent()) {
			return false
		}
		if square == to {
			return true
		}
	}
}

func (b *Board) isCastlingPathUnoccupied(from, to, allowedOccupiedSquare Square) bool {
	if from.Rank() != to.Rank() || from == to {
		return from.Rank() == to.Rank()
	}

	step := int8(1)
	if to.File() < from.File() {
		step = -1
	} else if to.File() == from.File() {
		step = 0
	}

	for file := int8(from.File()) + step; ; file += step {
		square := Square(file + int8(from.Rank())*8)

		if b.IsSquareOccupied(square) && square != allowedOccupiedSquare {
			return false
		}
		if square == to {
			return true
		}
	}
}
