package chess

import "math/bits"

type zobristValues struct {
	piece     [PieceColorCount][PieceTypeCount][64]uint64
	blackMove uint64
	castle    [PieceColorCount][CastlingSideCount][64]uint64
	enPassant [8]uint64
}

var positionZobrist = newZobristValues()

func newZobristValues() zobristValues {
	var values zobristValues
	seed := uint64(0x8e5d6f4a3b291c07)
	for color := range PieceColorCount {
		for pieceType := Pawn; pieceType < PieceTypeCount; pieceType++ {
			for square := range 64 {
				values.piece[color][pieceType][square] = nextZobristValue(&seed)
			}
		}
		for side := range CastlingSideCount {
			for square := range 64 {
				values.castle[color][side][square] = nextZobristValue(&seed)
			}
		}
	}
	values.blackMove = nextZobristValue(&seed)
	for file := range 8 {
		values.enPassant[file] = nextZobristValue(&seed)
	}

	return values
}

func nextZobristValue(seed *uint64) uint64 {
	*seed += 0x9e3779b97f4a7c15
	value := *seed
	value = (value ^ (value >> 30)) * 0xbf58476d1ce4e5b9
	value = (value ^ (value >> 27)) * 0x94d049bb133111eb
	return value ^ (value >> 31)
}

// PositionKey returns a deterministic Zobrist hash for repetition detection.
// Move counters are intentionally excluded because they do not affect legal moves.
func (b Board) PositionKey() uint64 {
	var key uint64
	for color := range PieceColorCount {
		for pieceType := Pawn; pieceType < PieceTypeCount; pieceType++ {
			pieces := b.Pieces[color][pieceType]
			for square := range 64 {
				if pieces&(Square(square).Mask()) != 0 {
					key ^= positionZobrist.piece[color][pieceType][square]
				}
			}
		}
		for side := range CastlingSideCount {
			right := b.CastlingRights[color][side]
			if right.Available {
				key ^= positionZobrist.castle[color][side][right.RookFrom]
			}
		}
	}
	if b.ColorToMove == Black {
		key ^= positionZobrist.blackMove
	}
	if target, ok := b.EnPassantTarget.SingleSquare(); ok {
		key ^= positionZobrist.enPassant[target.File()]
	}

	return key
}

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
