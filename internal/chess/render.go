package chess

import "fmt"

const (
	ansiReset             = "\033[0m"
	defaultDarkSquare     = "\033[48;5;244m"
	defaultLightSquare    = "\033[48;5;248m"
	woodDarkSquare        = "\033[48;5;94m"
	woodLightSquare       = "\033[48;5;180m"
	pastelDarkSquare      = "\033[48;2;52;120;240m"
	pastelLightSquare     = "\033[48;2;255;102;180m"
	cyberpunkDarkSquare   = "\033[48;2;180;0;255m"
	cyberpunkLightSquare  = "\033[48;2;200;200;35m"
	forestDarkSquare      = "\033[48;2;56;102;65m"
	forestLightSquare     = "\033[48;2;167;201;87m"
	oceanDarkSquare       = "\033[48;2;0;105;148m"
	oceanLightSquare      = "\033[48;2;0;180;216m"
	sakuraDarkSquare      = "\033[48;2;179;70;123m"
	sakuraLightSquare     = "\033[48;2;240;150;175m"
	sunsetDarkSquare      = "\033[48;2;184;80;66m"
	sunsetLightSquare     = "\033[48;2;236;179;86m"
	lavenderDarkSquare    = "\033[48;2;109;89;122m"
	lavenderLightSquare   = "\033[48;2;205;180;219m"
	nordDarkSquare        = "\033[48;2;76;86;106m"
	nordLightSquare       = "\033[48;2;216;222;233m"
	monochromeDarkSquare  = "\033[48;2;92;92;92m"
	monochromeLightSquare = "\033[48;2;207;207;207m"
	whitePieceColor       = "\033[97m"
	blackPieceColor       = "\033[30m"
)

type ColorTheme int

const (
	DEFAULT ColorTheme = iota
	WOOD
	PASTEL
	CYBERPUNK
	FOREST
	OCEAN
	SAKURA
	SUNSET
	LAVENDER
	NORD
	MONOCHROME
)

type Renderer struct {
	Theme       ColorTheme
	Perspective PieceColor
}

type squarePalette struct {
	dark  string
	light string
}

var colorThemePalettes = map[ColorTheme]squarePalette{
	DEFAULT:    {dark: defaultDarkSquare, light: defaultLightSquare},
	WOOD:       {dark: woodDarkSquare, light: woodLightSquare},
	PASTEL:     {dark: pastelDarkSquare, light: pastelLightSquare},
	CYBERPUNK:  {dark: cyberpunkDarkSquare, light: cyberpunkLightSquare},
	FOREST:     {dark: forestDarkSquare, light: forestLightSquare},
	OCEAN:      {dark: oceanDarkSquare, light: oceanLightSquare},
	SAKURA:     {dark: sakuraDarkSquare, light: sakuraLightSquare},
	SUNSET:     {dark: sunsetDarkSquare, light: sunsetLightSquare},
	LAVENDER:   {dark: lavenderDarkSquare, light: lavenderLightSquare},
	NORD:       {dark: nordDarkSquare, light: nordLightSquare},
	MONOCHROME: {dark: monochromeDarkSquare, light: monochromeLightSquare},
}

func (r *Renderer) squareBackground(square Square) string {
	palette, ok := colorThemePalettes[r.Theme]
	if !ok {
		palette = colorThemePalettes[DEFAULT]
	}

	if square.IsDark() {
		return palette.dark
	}
	return palette.light
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
