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

func (t PieceType) String() string {
	switch t {
	case Pawn:
		return "pawn"
	case Knight:
		return "knight"
	case Bishop:
		return "bishop"
	case Rook:
		return "rook"
	case Queen:
		return "queen"
	case King:
		return "king"
	default:
		return "unknown"
	}
}

type PieceColor uint8

const (
	White PieceColor = iota
	Black
	PieceColorCount
)

func (c PieceColor) Opponent() PieceColor {
	return c ^ 1
}

func (c PieceColor) String() string {
	switch c {
	case White:
		return "White"
	case Black:
		return "Black"
	default:
		return (c % 2).String()
	}
}

func (c PieceColor) EndResult() string {
	switch c {
	case White:
		return "1-0"
	case Black:
		return "0-1"
	default:
		return ""
	}
}

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
