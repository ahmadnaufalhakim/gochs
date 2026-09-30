package chess

import "errors"

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

type Board struct {
	WhitePawn   Bitboard
	WhiteKnight Bitboard
	WhiteBishop Bitboard
	WhiteRook   Bitboard
	WhiteQueen  Bitboard
	WhiteKing   Bitboard

	BlackPawn   Bitboard
	BlackKnight Bitboard
	BlackBishop Bitboard
	BlackRook   Bitboard
	BlackQueen  Bitboard
	BlackKing   Bitboard

	ColorToMove PieceColor
}

func (b *Board) Reset() *Board {
	b.WhitePawn = a2.Mask() | b2.Mask() | c2.Mask() | d2.Mask() | e2.Mask() | f2.Mask() | g2.Mask() | h2.Mask()
	b.WhiteKnight = b1.Mask() | g1.Mask()
	b.WhiteBishop = c1.Mask() | f1.Mask()
	b.WhiteRook = a1.Mask() | h1.Mask()
	b.WhiteQueen = d1.Mask()
	b.WhiteKing = e1.Mask()

	b.BlackPawn = a7.Mask() | b7.Mask() | c7.Mask() | d7.Mask() | e7.Mask() | f7.Mask() | g7.Mask() | h7.Mask()
	b.BlackKnight = b8.Mask() | g8.Mask()
	b.BlackBishop = c8.Mask() | f8.Mask()
	b.BlackRook = a8.Mask() | h8.Mask()
	b.BlackQueen = d8.Mask()
	b.BlackKing = e8.Mask()

	b.ColorToMove = White

	return b
}

func (b *Board) PieceAt(s Square) Piece {
	switch {
	case s.Mask()&b.WhitePawn != 0:
		return Piece{Color: White, Type: Pawn}
	case s.Mask()&b.WhiteKnight != 0:
		return Piece{Color: White, Type: Knight}
	case s.Mask()&b.WhiteBishop != 0:
		return Piece{Color: White, Type: Bishop}
	case s.Mask()&b.WhiteRook != 0:
		return Piece{Color: White, Type: Rook}
	case s.Mask()&b.WhiteQueen != 0:
		return Piece{Color: White, Type: Queen}
	case s.Mask()&b.WhiteKing != 0:
		return Piece{Color: White, Type: King}

	case s.Mask()&b.BlackPawn != 0:
		return Piece{Color: Black, Type: Pawn}
	case s.Mask()&b.BlackKnight != 0:
		return Piece{Color: Black, Type: Knight}
	case s.Mask()&b.BlackBishop != 0:
		return Piece{Color: Black, Type: Bishop}
	case s.Mask()&b.BlackRook != 0:
		return Piece{Color: Black, Type: Rook}
	case s.Mask()&b.BlackQueen != 0:
		return Piece{Color: Black, Type: Queen}
	case s.Mask()&b.BlackKing != 0:
		return Piece{Color: Black, Type: King}
	}

	return Piece{Color: 0, Type: 0}
}
