package chess

import (
	"fmt"
	"strconv"
	"strings"
)

var fenSymbolToPiece = map[rune]Piece{
	'P': {Color: White, Type: Pawn},
	'N': {Color: White, Type: Knight},
	'B': {Color: White, Type: Bishop},
	'R': {Color: White, Type: Rook},
	'Q': {Color: White, Type: Queen},
	'K': {Color: White, Type: King},
	'p': {Color: Black, Type: Pawn},
	'n': {Color: Black, Type: Knight},
	'b': {Color: Black, Type: Bishop},
	'r': {Color: Black, Type: Rook},
	'q': {Color: Black, Type: Queen},
	'k': {Color: Black, Type: King},
}

// ParseFEN decodes a Forsyth-Edwards Notation position, including standard
// and Shredder-FEN castling rights.
func ParseFEN(fen string) (Board, error) {
	fields := strings.Fields(fen)
	if len(fields) != 6 {
		return Board{}, fmt.Errorf("FEN must contain six fields, got %d", len(fields))
	}

	var board Board
	if err := parseFENPieces(&board, fields[0]); err != nil {
		return Board{}, err
	}
	switch fields[1] {
	case "w":
		board.ColorToMove = White
	case "b":
		board.ColorToMove = Black
	default:
		return Board{}, fmt.Errorf("invalid FEN active color %q", fields[1])
	}
	if err := parseFENCastlingRights(&board, fields[2]); err != nil {
		return Board{}, err
	}
	if fields[3] != "-" {
		target, err := ParseCoordinate(fields[3])
		if err != nil {
			return Board{}, fmt.Errorf("invalid FEN en-passant target: %w", err)
		}
		board.EnPassantTarget = target.Mask()
	}
	halfmoveClock, err := strconv.ParseUint(fields[4], 10, 0)
	if err != nil {
		return Board{}, fmt.Errorf("invalid FEN halfmove clock %q: %w", fields[4], err)
	}
	board.HalfmoveClock = uint(halfmoveClock)
	fullmoveNumber, err := strconv.ParseUint(fields[5], 10, 0)
	if err != nil {
		return Board{}, fmt.Errorf("invalid FEN fullmove number %q: %w", fields[5], err)
	}
	board.FullmoveNumber = uint(fullmoveNumber)
	if err := board.Validate(); err != nil {
		return Board{}, fmt.Errorf("invalid FEN position: %w", err)
	}
	return board, nil
}

func parseFENPieces(board *Board, placements string) error {
	ranks := strings.Split(placements, "/")
	if len(ranks) != 8 {
		return fmt.Errorf("FEN piece placement must contain eight ranks, got %d", len(ranks))
	}
	for fenRank, placement := range ranks {
		file := 0
		for _, symbol := range placement {
			if symbol >= '1' && symbol <= '8' {
				file += int(symbol - '0')
			} else if piece, ok := fenSymbolToPiece[symbol]; ok {
				if file >= 8 {
					return fmt.Errorf("FEN rank %d exceeds eight files", 8-fenRank)
				}
				board.SetPieceAt(Square(file+(7-fenRank)*8), piece)
				file++
			} else {
				return fmt.Errorf("invalid FEN piece symbol %q", symbol)
			}
			if file > 8 {
				return fmt.Errorf("FEN rank %d exceeds eight files", 8-fenRank)
			}
		}
		if file != 8 {
			return fmt.Errorf("FEN rank %d contains %d files", 8-fenRank, file)
		}
	}
	return nil
}

func parseFENCastlingRights(board *Board, field string) error {
	if field == "-" {
		return nil
	}
	for _, symbol := range field {
		color, rookFile, ok := fenCastlingRookFile(symbol)
		if !ok {
			return fmt.Errorf("invalid FEN castling right %q", symbol)
		}
		kingFrom, ok := board.Pieces[color][King].SingleSquare()
		if !ok {
			return fmt.Errorf("%s castling right has no unique king", color)
		}
		if kingFrom.Rank() != map[PieceColor]uint8{White: 0, Black: 7}[color] {
			return fmt.Errorf("%s castling king is not on its home rank", color)
		}
		rookFrom := Square(rookFile + int(kingFrom.Rank())*8)
		side := QueenSide
		if rookFile > int(kingFrom.File()) {
			side = KingSide
		}
		if rookFile == int(kingFrom.File()) || board.CastlingRights[color][side].Available {
			return fmt.Errorf("duplicate or invalid %s castling right", color)
		}
		board.CastlingRights[color][side] = CastlingRight{KingFrom: kingFrom, RookFrom: rookFrom, Available: true}
	}
	return nil
}

func fenCastlingRookFile(symbol rune) (PieceColor, int, bool) {
	switch symbol {
	case 'K':
		return White, 7, true
	case 'Q':
		return White, 0, true
	case 'k':
		return Black, 7, true
	case 'q':
		return Black, 0, true
	case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H':
		return White, int(symbol - 'A'), true
	case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h':
		return Black, int(symbol - 'a'), true
	default:
		return White, 0, false
	}
}
