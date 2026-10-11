package chess

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
