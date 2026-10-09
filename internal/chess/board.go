package chess

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type Square uint8

func (s Square) String() string {
	fileRune := rune('a' + s.File())
	rankRune := rune('1' + s.Rank())

	return string([]rune{fileRune, rankRune})
}

func (s Square) File() uint8 {
	return uint8(s % 8)
}

func (s Square) Rank() uint8 {
	return uint8(s / 8)
}

func (s Square) IsDark() bool {
	return (s.File()+s.Rank())%2 == 0
}

func (s Square) IsLight() bool {
	return (s.File()+s.Rank())%2 == 1
}

func (s Square) Mask() Bitboard {
	return Bitboard(1) << s
}

const (
	a1 Square = iota
	b1
	c1
	d1
	e1
	f1
	g1
	h1
	a2
	b2
	c2
	d2
	e2
	f2
	g2
	h2
	a3
	b3
	c3
	d3
	e3
	f3
	g3
	h3
	a4
	b4
	c4
	d4
	e4
	f4
	g4
	h4
	a5
	b5
	c5
	d5
	e5
	f5
	g5
	h5
	a6
	b6
	c6
	d6
	e6
	f6
	g6
	h6
	a7
	b7
	c7
	d7
	e7
	f7
	g7
	h7
	a8
	b8
	c8
	d8
	e8
	f8
	g8
	h8
)

func ParseCoordinate(coordinate string) (Square, error) {
	if len(coordinate) != 2 {
		return Square(0), errors.New("coordinate must consist 2 characters")
	}
	if coordinate[0] < 'a' || coordinate[0] > 'h' || coordinate[1] < '1' || coordinate[1] > '8' {
		return Square(0), errors.New("coordinate must be between a1 and h8")
	}

	file := coordinate[0] - 'a'
	rank := coordinate[1] - '1'

	return Square(file + rank*8), nil
}

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

type Board struct {
	Pieces          [PieceColorCount][PieceTypeCount]Bitboard
	CastlingRights  [PieceColorCount][CastlingSideCount]CastlingRight
	EnPassantTarget Bitboard
	ColorToMove     PieceColor
	HalfmoveClock   uint
	FullmoveNumber  uint
}

func (b *Board) Clear() {
	for color := range PieceColorCount {
		for pieceType := Pawn; pieceType < PieceTypeCount; pieceType++ {
			b.Pieces[color][pieceType] &= Bitboard(0)
		}
	}
	b.CastlingRights = [PieceColorCount][CastlingSideCount]CastlingRight{}
	b.EnPassantTarget = Bitboard(0)
}

func (b *Board) ClearSquare(s Square) {
	mask := s.Mask()
	for color := range PieceColorCount {
		for pieceType := Pawn; pieceType < PieceTypeCount; pieceType++ {
			b.Pieces[color][pieceType] &^= mask
		}
	}
}

var defaultStartingPieces = [PieceColorCount][PieceTypeCount]Bitboard{
	White: {
		Pawn:   rank2,
		Knight: b1.Mask() | g1.Mask(),
		Bishop: c1.Mask() | f1.Mask(),
		Rook:   a1.Mask() | h1.Mask(),
		Queen:  d1.Mask(),
		King:   e1.Mask(),
	},
	Black: {
		Pawn:   rank7,
		Knight: b8.Mask() | g8.Mask(),
		Bishop: c8.Mask() | f8.Mask(),
		Rook:   a8.Mask() | h8.Mask(),
		Queen:  d8.Mask(),
		King:   e8.Mask(),
	},
}
var defaultStartingCastlingRights = [PieceColorCount][CastlingSideCount]CastlingRight{
	White: {
		QueenSide: {
			KingFrom:  e1,
			RookFrom:  a1,
			Available: true,
		},
		KingSide: {
			KingFrom:  e1,
			RookFrom:  h1,
			Available: true,
		},
	},
	Black: {
		QueenSide: {
			KingFrom:  e8,
			RookFrom:  a8,
			Available: true,
		},
		KingSide: {
			KingFrom:  e8,
			RookFrom:  h8,
			Available: true,
		},
	},
}

func (b *Board) disableRookCastlingRight(color PieceColor, rookSquare Square) {
	for side := range CastlingSideCount {
		right := &b.CastlingRights[color][side]

		if right.RookFrom == rookSquare {
			right.Available = false
		}
	}
}

func (b *Board) Reset() {
	b.Pieces = defaultStartingPieces
	b.CastlingRights = defaultStartingCastlingRights
	b.EnPassantTarget = Bitboard(0)
	b.ColorToMove = White
	b.HalfmoveClock = 0
	b.FullmoveNumber = 1
}

func (b *Board) PieceAt(s Square) Piece {
	mask := s.Mask()
	for color := range PieceColorCount {
		for pieceType := Pawn; pieceType < PieceTypeCount; pieceType++ {
			if b.Pieces[color][pieceType]&mask != 0 {
				return Piece{Color: color, Type: pieceType}
			}
		}
	}

	return Piece{Type: PieceNone}
}

func (b *Board) SetPieceAt(s Square, p Piece) {
	b.ClearSquare(s)
	if p.Type == PieceNone {
		return
	}

	mask := s.Mask()
	b.Pieces[p.Color][p.Type] |= mask
}

func (b *Board) Validate() error {
	if err := b.validateColorToMove(); err != nil {
		return err
	}
	if err := b.validatePieceOverlaps(); err != nil {
		return err
	}
	if err := b.validateKingCount(); err != nil {
		return err
	}
	if err := b.validateKingCheck(); err != nil {
		return err
	}
	if err := b.validatePawnPositions(); err != nil {
		return err
	}
	if err := b.validateEnPassantTarget(); err != nil {
		return err
	}
	if err := b.validateCastlingRights(); err != nil {
		return err
	}
	if err := b.validateFullmoveNumber(); err != nil {
		return err
	}

	return nil
}

func (b *Board) Occupied() Bitboard {
	result := Bitboard(0)
	for color := range PieceColorCount {
		for pieceType := Pawn; pieceType < PieceTypeCount; pieceType++ {
			result |= b.Pieces[color][pieceType]
		}
	}
	return result
}

func (b *Board) OccupiedByColor(c PieceColor) Bitboard {
	result := Bitboard(0)
	for pieceType := Pawn; pieceType < PieceTypeCount; pieceType++ {
		result |= b.Pieces[c][pieceType]
	}
	return result
}

func (b *Board) OccupiedByPieceType(t PieceType) Bitboard {
	result := Bitboard(0)
	for color := range PieceColorCount {
		result |= b.Pieces[color][t]
	}
	return result
}

func (b *Board) OccupiedByPiece(p Piece) Bitboard {
	return Bitboard(0) | b.Pieces[p.Color][p.Type]
}

func (b *Board) IsSquareOccupied(s Square) bool {
	return s.Mask()&b.Occupied() != 0
}

func (b *Board) IsSquareOccupiedByColor(s Square, c PieceColor) bool {
	return s.Mask()&b.OccupiedByColor(c) != 0
}

func (b *Board) IsSquareOccupiedByPieceType(s Square, t PieceType) bool {
	return s.Mask()&b.OccupiedByPieceType(t) != 0
}

func (b *Board) IsSquareOccupiedByPiece(s Square, p Piece) bool {
	return s.Mask()&b.OccupiedByPiece(p) != 0
}

func (b *Board) IsColorInCheck(c PieceColor) bool {
	king := b.Pieces[c][King]

	for square := a1; square <= h8; square++ {
		if king&square.Mask() != 0 {
			return b.IsSquareAttackedBy(square, c.Opponent())
		}
	}

	return false
}

func (b *Board) IsSquareAttackedBy(s Square, by PieceColor) bool {
	mask := s.Mask()

	opponentPawnAttacks := GeneratePawnAttacks(*b, by)
	if opponentPawnAttacks&mask != 0 {
		return true
	}
	opponentKnightAttacks := GenerateKnightAttacks(*b, by)
	if opponentKnightAttacks&mask != 0 {
		return true
	}
	opponentBishopAttacks := GenerateBishopAttacks(*b, by)
	if opponentBishopAttacks&mask != 0 {
		return true
	}
	opponentRookAttacks := GenerateRookAttacks(*b, by)
	if opponentRookAttacks&mask != 0 {
		return true
	}
	opponentQueenAttacks := GenerateQueenAttacks(*b, by)
	if opponentQueenAttacks&mask != 0 {
		return true
	}
	opponentKingAttacks := GenerateKingAttacks(*b, by)
	if opponentKingAttacks&mask != 0 {
		return true
	}

	return false
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

	if movingPiece.Type == Pawn &&
		pawnPromotionRank[movingPiece.Color]&m.To().Mask() != 0 {
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

	if isPawnMove || isCaptureMove {
		b.HalfmoveClock = 0
	} else {
		b.HalfmoveClock++
	}

	if movingPiece.Color == Black {
		b.FullmoveNumber++
	}
}

func (b *Board) MakeMove(m Move) error {
	legalMoves := GenerateLegalMoves(*b)
	if slices.Contains(legalMoves, m) {
		b.applyMove(m)
		return nil
	}

	return fmt.Errorf("%s is an illegal move", m.String())
}

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
