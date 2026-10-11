package chess

import "fmt"

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

// UCI returns the move in Universal Chess Interface coordinate notation.
func (m Move) UCI() string {
	uci := m.From().String() + m.To().String()
	switch m.Flag() {
	case PromoteKnight, PromoteCaptureKnight:
		return uci + "n"
	case PromoteBishop, PromoteCaptureBishop:
		return uci + "b"
	case PromoteRook, PromoteCaptureRook:
		return uci + "r"
	case PromoteQueen, PromoteCaptureQueen:
		return uci + "q"
	}

	return uci
}

func (m Move) String() string {
	return m.UCI()
}

func (m Move) Print() {
	fmt.Println(m.UCI())
}

func GeneratePawnPseudoLegalMoves(b Board, color PieceColor) []Move {
	var moves []Move
	pawns := b.Pieces[color][Pawn]
	delta := pawnMoveDelta[color]
	occupied := b.Occupied()
	capturableOpponentPieces := b.OccupiedByColor(color.Opponent()) & ^b.Pieces[color.Opponent()][King]

	for from := a1; from <= h8; from++ {
		if pawns&from.Mask() == 0 {
			continue
		}

		source := from.Mask() & allowedSources(pawnMoveDelta[color])
		if source == 0 {
			continue
		}

		toFile := int8(from.File()) + delta.File
		toRank := int8(from.Rank()) + delta.Rank
		to := Square(toFile + toRank*8)
		if occupied&to.Mask() == 0 {
			if pawnPromotionRank[color]&to.Mask() != 0 {
				moves = append(moves, NewMove(from, to, PromoteKnight))
				moves = append(moves, NewMove(from, to, PromoteBishop))
				moves = append(moves, NewMove(from, to, PromoteRook))
				moves = append(moves, NewMove(from, to, PromoteQueen))
			} else {
				moves = append(moves, NewMove(from, to, QuietMove))
			}

			if pawnDoublePushSourceRank[color]&from.Mask() != 0 {
				doubleToFile := toFile + delta.File
				doubleToRank := toRank + delta.Rank
				doubleTo := Square(doubleToFile + doubleToRank*8)
				if occupied&doubleTo.Mask() == 0 {
					moves = append(moves, NewMove(from, doubleTo, DoublePawnPush))
				}
			}
		}

		for _, delta := range pawnAttackMoveDeltas[color] {
			attackSource := from.Mask() & allowedSources(delta)
			if attackSource == 0 {
				continue
			}

			attackFile := int8(from.File()) + delta.File
			attackRank := int8(from.Rank()) + delta.Rank
			attackTo := Square(attackFile + attackRank*8)
			if capturableOpponentPieces&attackTo.Mask() != 0 {
				if pawnPromotionRank[color]&attackTo.Mask() != 0 {
					moves = append(moves, NewMove(from, attackTo, PromoteCaptureKnight))
					moves = append(moves, NewMove(from, attackTo, PromoteCaptureBishop))
					moves = append(moves, NewMove(from, attackTo, PromoteCaptureRook))
					moves = append(moves, NewMove(from, attackTo, PromoteCaptureQueen))
				} else {
					moves = append(moves, NewMove(from, attackTo, Capture))
				}
			} else if b.EnPassantTarget&attackTo.Mask() != 0 {
				capturedFile := int8(attackTo.File())
				capturedRank := int8(attackTo.Rank()) - delta.Rank
				capturedPawnSquare := Square(capturedFile + capturedRank*8)

				capturedPawn := Piece{
					Color: color.Opponent(),
					Type:  Pawn,
				}

				if occupied&attackTo.Mask() == 0 &&
					b.IsSquareOccupiedByPiece(capturedPawnSquare, capturedPawn) {
					moves = append(moves, NewMove(from, attackTo, EnPassant))
				}
			}
		}
	}

	return moves
}

func GenerateKnightPseudoLegalMoves(b Board, color PieceColor) []Move {
	return GenerateJumpPseudoLegalMoves(b, color, Knight, knightMoveDeltas)
}

func GenerateBishopPseudoLegalMoves(b Board, color PieceColor) []Move {
	return GenerateSlidePseudoLegalMoves(b, color, Bishop, bishopMoveDeltas)
}

func GenerateRookPseudoLegalMoves(b Board, color PieceColor) []Move {
	return GenerateSlidePseudoLegalMoves(b, color, Rook, rookMoveDeltas)
}

func GenerateQueenPseudoLegalMoves(b Board, color PieceColor) []Move {
	return GenerateSlidePseudoLegalMoves(b, color, Queen, queenMoveDeltas)
}

func GenerateKingPseudoLegalMoves(b Board, color PieceColor) []Move {
	return GenerateJumpPseudoLegalMoves(b, color, King, queenMoveDeltas)
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
			if toPiece.Type == PieceNone {
				moves = append(moves, NewMove(from, to, QuietMove))
				continue
			}
			if toPiece.Color != color && toPiece.Type != King {
				moves = append(moves, NewMove(from, to, Capture))
			}
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

func GenerateCastlingPseudoLegalMoves(b Board, color PieceColor) []Move {
	var moves []Move

	for side := range CastlingSideCount {
		castlingRight := b.CastlingRights[color][side]
		if !castlingRight.Available ||
			!castlingRight.isWellFormed(color, CastlingSide(side)) {
			continue
		}

		king := b.PieceAt(castlingRight.KingFrom)
		rook := b.PieceAt(castlingRight.RookFrom)
		if king != (Piece{Color: color, Type: King}) ||
			rook != (Piece{Color: color, Type: Rook}) {
			continue
		}

		kingTo, rookTo := castlingDestinations(color, side)
		if b.isKingCastlingPathSafe(castlingRight.KingFrom, kingTo, color) &&
			b.isCastlingPathUnoccupied(castlingRight.KingFrom, kingTo, castlingRight.RookFrom) &&
			b.isCastlingPathUnoccupied(castlingRight.RookFrom, rookTo, castlingRight.KingFrom) {
			moves = append(moves, NewMove(castlingRight.KingFrom, kingTo, MoveFlag(side)))
		}
	}

	return moves
}

func GeneratePseudoLegalMoves(b Board, color PieceColor) []Move {
	var moves []Move

	moves = append(moves, GeneratePawnPseudoLegalMoves(b, color)...)
	moves = append(moves, GenerateKnightPseudoLegalMoves(b, color)...)
	moves = append(moves, GenerateBishopPseudoLegalMoves(b, color)...)
	moves = append(moves, GenerateRookPseudoLegalMoves(b, color)...)
	moves = append(moves, GenerateQueenPseudoLegalMoves(b, color)...)
	moves = append(moves, GenerateKingPseudoLegalMoves(b, color)...)
	moves = append(moves, GenerateCastlingPseudoLegalMoves(b, color)...)

	return moves
}

func GenerateLegalMoves(b Board) []Move {
	var legalMoves []Move
	movingColor := b.ColorToMove
	pseudoMoves := GeneratePseudoLegalMoves(b, movingColor)

	for _, pseudoMove := range pseudoMoves {
		nextBoard := b
		nextBoard.applyMove(pseudoMove)
		if !nextBoard.IsColorInCheck(movingColor) {
			legalMoves = append(legalMoves, pseudoMove)
		}
	}

	return legalMoves
}
