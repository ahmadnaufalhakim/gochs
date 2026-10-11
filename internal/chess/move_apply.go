package chess

import (
	"fmt"
	"slices"
)

func (b *Board) applyMove(m Move) {
	movingPiece := b.PieceAt(m.From())
	capturedPiece := b.PieceAt(m.To())
	isPawnMove := movingPiece.Type == Pawn
	isCaptureMove := capturedPiece.Type != PieceNone || m.Flag() == EnPassant
	pawnDelta := pawnMoveDelta[movingPiece.Color]

	isCastling := slices.Contains([]MoveFlag{KingSideCastle, QueenSideCastle}, m.Flag())
	var castlingRook Piece
	var castlingRookDestination Square

	if isCastling {
		castlingSide := CastlingSide(m.Flag())
		castlingRight := b.CastlingRights[movingPiece.Color][castlingSide]

		castlingRookSquare := castlingRight.RookFrom
		castlingRook = b.PieceAt(castlingRookSquare)
		b.ClearSquare(castlingRookSquare)

		_, castlingRookDestination = castlingDestinations(movingPiece.Color, castlingSide)

		for side := range CastlingSideCount {
			b.CastlingRights[movingPiece.Color][side].Available = false
		}
	}

	if m.Flag() == EnPassant {
		capturedFile := int8(m.To().File())
		capturedRank := int8(m.To().Rank()) - pawnDelta.Rank
		capturedSquare := Square(capturedFile + capturedRank*8)

		b.Pieces[movingPiece.Color.Opponent()][Pawn] &^= capturedSquare.Mask()
	}

	b.ClearSquare(m.From())

	if movingPiece.Type == Pawn && pawnPromotionRank[movingPiece.Color]&m.To().Mask() != 0 {
		switch m.Flag() {
		case PromoteKnight, PromoteCaptureKnight:
			movingPiece.Type = Knight
		case PromoteBishop, PromoteCaptureBishop:
			movingPiece.Type = Bishop
		case PromoteRook, PromoteCaptureRook:
			movingPiece.Type = Rook
		case PromoteQueen, PromoteCaptureQueen:
			movingPiece.Type = Queen
		}
	}

	if movingPiece.Type == King {
		for side := range CastlingSideCount {
			b.CastlingRights[movingPiece.Color][side].Available = false
		}
	}
	if movingPiece.Type == Rook {
		b.disableRookCastlingRight(movingPiece.Color, m.From())
	}
	if capturedPiece.Type == Rook {
		b.disableRookCastlingRight(capturedPiece.Color, m.To())
	}

	b.SetPieceAt(m.To(), movingPiece)
	if isCastling {
		b.SetPieceAt(castlingRookDestination, castlingRook)
	}

	b.EnPassantTarget = Bitboard(0)
	if m.Flag() == DoublePawnPush {
		enPassantTargetFile := int8(m.From().File())
		enPassantTargetRank := int8(m.From().Rank()) + pawnDelta.Rank
		enPassantTargetSquare := Square(enPassantTargetFile + enPassantTargetRank*8)

		b.EnPassantTarget = enPassantTargetSquare.Mask()
	}

	b.ColorToMove = b.ColorToMove.Opponent()
	if m.Flag() == DoublePawnPush && !b.hasLegalEnPassantCapture() {
		b.EnPassantTarget = 0
	}

	if isPawnMove || isCaptureMove {
		b.HalfmoveClock = 0
	} else {
		b.HalfmoveClock++
	}

	if movingPiece.Color == Black {
		b.FullmoveNumber++
	}
}

func (b Board) hasLegalEnPassantCapture() bool {
	target, ok := b.EnPassantTarget.SingleSquare()
	if !ok {
		return false
	}

	color := b.ColorToMove
	for _, delta := range pawnAttackMoveDeltas[color] {
		fromFile := int8(target.File()) - delta.File
		fromRank := int8(target.Rank()) - delta.Rank
		if fromFile < 0 || fromFile > 7 || fromRank < 0 || fromRank > 7 {
			continue
		}
		from := Square(fromFile + fromRank*8)
		if b.Pieces[color][Pawn]&from.Mask() == 0 {
			continue
		}

		next := b
		next.applyMove(NewMove(from, target, EnPassant))
		if !next.IsColorInCheck(color) {
			return true
		}
	}

	return false
}

func (b *Board) MakeMove(m Move) error {
	legalMoves := GenerateLegalMoves(*b)
	if slices.Contains(legalMoves, m) {
		b.applyMove(m)
		return nil
	}

	return fmt.Errorf("%s is an illegal move", m.UCI())
}
