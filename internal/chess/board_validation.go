package chess

import (
	"errors"
	"fmt"
	"math/bits"
)

const (
	fileA Bitboard = 0x0101010101010101
	fileB Bitboard = 0x0202020202020202
	fileC Bitboard = 0x0404040404040404
	fileD Bitboard = 0x0808080808080808
	fileE Bitboard = 0x1010101010101010
	fileF Bitboard = 0x2020202020202020
	fileG Bitboard = 0x4040404040404040
	fileH Bitboard = 0x8080808080808080

	rank1 Bitboard = 0x00000000000000FF
	rank2 Bitboard = 0x000000000000FF00
	rank3 Bitboard = 0x0000000000FF0000
	rank4 Bitboard = 0x00000000FF000000
	rank5 Bitboard = 0x000000FF00000000
	rank6 Bitboard = 0x0000FF0000000000
	rank7 Bitboard = 0x00FF000000000000
	rank8 Bitboard = 0xFF00000000000000
)

func (b *Board) validatePawnPositions() error {
	for color := range PieceColorCount {
		if b.Pieces[color][Pawn]&(rank1|rank8) != 0 {
			return errors.New("pawns cannot be on the first or eighth rank")
		}
	}

	return nil
}

func (b *Board) validateKingCount() error {
	if !b.Pieces[b.ColorToMove][King].IsSingleBit() ||
		!b.Pieces[b.ColorToMove.Opponent()][King].IsSingleBit() {
		return errors.New("each side must have exactly one king")
	}

	return nil
}

func (b *Board) validateKingCheck() error {
	if b.IsColorInCheck(b.ColorToMove.Opponent()) {
		return fmt.Errorf(
			"%s to move, but %s is in check",
			b.ColorToMove.String(),
			b.ColorToMove.Opponent().String(),
		)
	}

	return nil
}

func (b *Board) validateEnPassantTarget() error {
	target := b.EnPassantTarget
	if target == 0 {
		return nil
	}

	if !target.IsSingleBit() {
		return errors.New("en-passant target must contain one square")
	}

	lastMover := b.ColorToMove.Opponent()
	if target&pawnEnPassantTargetRank[lastMover] == 0 {
		return errors.New("en-passant target is on an invalid rank")
	}

	targetSquare := Square(bits.TrailingZeros64(uint64(target)))
	if b.IsSquareOccupied(targetSquare) {
		return errors.New("en-passant target must be empty")
	}

	pawnFile := int8(targetSquare.File())
	pawnRank := int8(targetSquare.Rank()) + pawnMoveDelta[lastMover].Rank
	pawnSquare := Square(pawnFile + pawnRank*8)

	expectedPawn := Piece{
		Color: lastMover,
		Type:  Pawn,
	}

	if !b.IsSquareOccupiedByPiece(pawnSquare, expectedPawn) {
		return errors.New("en-passant target has no (possibly) double-pushed pawn")
	}

	pawnSourceRank := int8(targetSquare.Rank()) - pawnMoveDelta[lastMover].Rank
	pawnSourceSquare := Square(pawnFile + pawnSourceRank*8)
	if b.IsSquareOccupied(pawnSourceSquare) {
		return errors.New("en-passant target's pawn source square must be empty")
	}

	return nil
}

func (right CastlingRight) isWellFormed(color PieceColor, side CastlingSide) bool {
	if right.KingFrom.Mask()&castlingHomeRank[color] == 0 ||
		right.RookFrom.Mask()&castlingHomeRank[color] == 0 ||
		right.KingFrom == right.RookFrom {
		return false
	}

	if side == KingSide {
		return right.RookFrom.File() > right.KingFrom.File()
	}

	return right.RookFrom.File() < right.KingFrom.File()
}

func (b *Board) validateCastlingRights() error {
	for color := range PieceColorCount {
		for side := range CastlingSideCount {
			right := b.CastlingRights[color][side]
			if !right.Available {
				continue
			}
			if !right.isWellFormed(color, CastlingSide(side)) {
				return fmt.Errorf("%s castling right %d has invalid origins", color, side)
			}
			if b.PieceAt(right.KingFrom) != (Piece{Color: color, Type: King}) {
				return fmt.Errorf("%s castling right %d has no king on its origin", color, side)
			}
			if b.PieceAt(right.RookFrom) != (Piece{Color: color, Type: Rook}) {
				return fmt.Errorf("%s castling right %d has no rook on its origin", color, side)
			}
		}
	}

	return nil
}

func (b *Board) validateHalfmoveClock() error {
	if b.HalfmoveClock < 0 {
		return fmt.Errorf("invalid halfmove clock: %v", b.HalfmoveClock)
	}

	return nil
}

func (b *Board) validateFullmoveNumber() error {
	if b.FullmoveNumber < 1 {
		return fmt.Errorf("invalid fullmove number: %v", b.FullmoveNumber)
	}

	return nil
}
