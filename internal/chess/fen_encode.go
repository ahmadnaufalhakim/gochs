package chess

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// FEN returns the board in Forsyth-Edwards Notation. Positions with
// non-orthodox castling origins use Shredder-FEN rook-file castling rights.
func (b Board) FEN() (string, error) {
	if err := b.Validate(); err != nil {
		return "", fmt.Errorf("cannot encode FEN: %w", err)
	}

	var piecePlacements strings.Builder
	for rank := range 8 {
		var emptySquare int
		for file := range 8 {
			if fenSym, ok := pieceToFENSymbol[b.PieceAt(Square(file+(7-rank)*8))]; ok {
				if emptySquare > 0 {
					piecePlacements.WriteString(strconv.Itoa(emptySquare))
					emptySquare = 0
				}
				piecePlacements.WriteString(string(fenSym))
			} else {
				emptySquare++
			}
		}

		if emptySquare > 0 {
			piecePlacements.WriteString(strconv.Itoa(emptySquare))
		}

		if rank < 7 {
			piecePlacements.WriteString("/")
		}
	}

	var activeColor string
	switch b.ColorToMove {
	case White:
		activeColor = "w"
	case Black:
		activeColor = "b"
	default:
		return "", fmt.Errorf("cannot encode FEN with invalid active color %d", b.ColorToMove)
	}

	var castlingRights strings.Builder
	if !b.writeCastlingRights(&castlingRights) {
		castlingRights.WriteString("-")
	}

	enPassantField := "-"
	if b.EnPassantTarget != 0 {
		square, ok := b.EnPassantTarget.SingleSquare()
		if !ok {
			return "", errors.New("en-passant target must contain one square")
		}

		enPassantField = square.String()
	}

	return fmt.Sprintf("%s %s %s %s %s %s",
		piecePlacements.String(),
		activeColor,
		castlingRights.String(),
		enPassantField,
		strconv.Itoa(int(b.HalfmoveClock)),
		strconv.Itoa(int(b.FullmoveNumber)),
	), nil
}

func (b Board) writeCastlingRights(result *strings.Builder) bool {
	useRookFiles := false
	for color := range PieceColorCount {
		for side := range CastlingSideCount {
			right := b.CastlingRights[color][side]
			if right.Available && right != defaultStartingCastlingRights[color][side] {
				useRookFiles = true
			}
		}
	}

	if useRookFiles {
		for color := range PieceColorCount {
			for file := range uint8(8) {
				for side := range CastlingSideCount {
					right := b.CastlingRights[color][side]
					if !right.Available || right.RookFrom.File() != file {
						continue
					}

					symbol := rune('A' + file)
					if color == Black {
						symbol = rune('a' + file)
					}
					result.WriteRune(symbol)
				}
			}
		}

		return result.Len() != 0
	}

	for color := range PieceColorCount {
		for side := range CastlingSideCount {
			if b.CastlingRights[color][side].Available {
				result.WriteRune(castlingRightFENSymbols[color][side])
			}
		}
	}

	return result.Len() != 0
}
