package chess

import "fmt"

const (
	ansiReset          = "\033[0m"
	defaultDarkSquare  = "\033[48;5;244m"
	defaultLightSquare = "\033[48;5;248m"
	woodDarkSquare     = "\033[48;5;94m"
	woodLightSquare    = "\033[48;5;180m"
	whitePieceColor    = "\033[97m"
	blackPieceColor    = "\033[30m"
)

type ColorTheme int

const (
	DEFAULT ColorTheme = iota
	WOOD
)

type Renderer struct {
	Theme       ColorTheme
	Perspective PieceColor
}

func (r *Renderer) squareBackground(square Square) string {
	if r.Theme == WOOD {
		if square.IsDark() {
			return woodDarkSquare
		}
		return woodLightSquare
	}

	if square.IsDark() {
		return defaultDarkSquare
	}
	return defaultLightSquare
}

func (r *Renderer) squareAt(row, column int) Square {
	rank := 7 - row
	file := column
	if r.Perspective == Black {
		rank = row
		file = 7 - column
	}

	return Square(rank*8 + file)
}

func pieceForeground(piece Piece) string {
	if piece.Type == PieceNone {
		return ""
	}
	if piece.Color == White {
		return whitePieceColor
	}
	return blackPieceColor
}

func (r *Renderer) Print(b Board) {
	for row := range 8 {
		firstSquare := r.squareAt(row, 0)
		fmt.Printf("%d ", firstSquare/8+1)
		for column := range 8 {
			square := r.squareAt(row, column)
			piece := b.PieceAt(square)
			fmt.Printf("%s%s %s %s", r.squareBackground(square), pieceForeground(piece), piece.Label(), ansiReset)
		}
		fmt.Println()
	}

	fmt.Print("  ")
	for column := range 8 {
		square := r.squareAt(0, column)
		fmt.Printf(" %c ", 'a'+square%8)
	}
	fmt.Println()
}
