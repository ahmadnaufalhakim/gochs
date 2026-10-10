package chess

import (
	"fmt"
	"slices"
	"strings"
)

// SAN returns the move in Standard Algebraic Notation for the current board.
func (b Board) SAN(move Move) (string, error) {
	legalMoves := GenerateLegalMoves(b)
	if !slices.Contains(legalMoves, move) {
		return "", fmt.Errorf("%s is an illegal move", move.UCI())
	}

	if move.Flag() == KingSideCastle {
		return b.sanSuffix(move, "O-O"), nil
	}
	if move.Flag() == QueenSideCastle {
		return b.sanSuffix(move, "O-O-O"), nil
	}

	movingPiece := b.PieceAt(move.From())
	var notation strings.Builder
	if movingPiece.Type != Pawn {
		notation.WriteString(sanPieceName(movingPiece.Type))
		notation.WriteString(b.sanDisambiguation(move, legalMoves, movingPiece.Type))
	} else if isCapture(move.Flag()) {
		notation.WriteByte(byte('a' + move.From().File()))
	}
	if isCapture(move.Flag()) {
		notation.WriteByte('x')
	}
	notation.WriteString(move.To().String())
	if promotion := sanPromotion(move.Flag()); promotion != "" {
		notation.WriteByte('=')
		notation.WriteString(promotion)
	}

	return b.sanSuffix(move, notation.String()), nil
}

func (b Board) sanDisambiguation(move Move, legalMoves []Move, pieceType PieceType) string {
	sameFile := false
	sameRank := false
	for _, candidate := range legalMoves {
		if candidate == move || candidate.To() != move.To() || b.PieceAt(candidate.From()).Type != pieceType {
			continue
		}
		if candidate.From().File() == move.From().File() {
			sameFile = true
		}
		if candidate.From().Rank() == move.From().Rank() {
			sameRank = true
		}
	}
	if !sameFile && !sameRank {
		return ""
	}
	if !sameFile {
		return string(rune('a' + move.From().File()))
	}
	if !sameRank {
		return string(rune('1' + move.From().Rank()))
	}
	return move.From().String()
}

func (b Board) sanSuffix(move Move, notation string) string {
	next := b
	next.applyMove(move)
	if !next.IsColorInCheck(next.ColorToMove) {
		return notation
	}
	if len(GenerateLegalMoves(next)) == 0 {
		return notation + "#"
	}
	return notation + "+"
}

func isCapture(flag MoveFlag) bool {
	return flag == Capture || flag == EnPassant || flag >= PromoteCaptureKnight
}

func sanPromotion(flag MoveFlag) string {
	switch flag {
	case PromoteKnight, PromoteCaptureKnight:
		return "N"
	case PromoteBishop, PromoteCaptureBishop:
		return "B"
	case PromoteRook, PromoteCaptureRook:
		return "R"
	case PromoteQueen, PromoteCaptureQueen:
		return "Q"
	}
	return ""
}

func sanPieceName(pieceType PieceType) string {
	switch pieceType {
	case Knight:
		return "N"
	case Bishop:
		return "B"
	case Rook:
		return "R"
	case Queen:
		return "Q"
	case King:
		return "K"
	}
	return ""
}
