package chess

type PieceType uint8

const (
	PieceNone PieceType = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
	PieceTypeCount
)

type PieceColor uint8

const (
	White PieceColor = iota
	Black
	PieceColorCount
)

type Piece struct {
	Color PieceColor
	Type  PieceType
}

var PieceLabel = map[PieceColor]map[PieceType]string{
	White: {
		PieceNone: " ",
		Pawn:      "♙",
		Knight:    "♘",
		Bishop:    "♗",
		Rook:      "♖",
		Queen:     "♕",
		King:      "♔",
	},
	Black: {
		PieceNone: " ",
		Pawn:      "♙",
		Knight:    "♞",
		Bishop:    "♝",
		Rook:      "♜",
		Queen:     "♛",
		King:      "♚",
	},
}

func (p Piece) Label() string {
	return PieceLabel[p.Color][p.Type]
}
