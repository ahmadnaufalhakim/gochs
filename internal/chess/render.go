package chess

import "fmt"

const (
	ansiReset             = "\033[0m"
	defaultDarkSquare     = "\033[48;5;244m"
	defaultLightSquare    = "\033[48;5;248m"
	woodDarkSquare        = "\033[48;2;122;73;35m"
	woodLightSquare       = "\033[48;2;214;158;93m"
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

type RGB struct {
	R uint8
	G uint8
	B uint8
}

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

var colorThemeColors = map[ColorTheme][2]RGB{
	DEFAULT:    {{R: 168, G: 168, B: 168}, {R: 128, G: 128, B: 128}},
	WOOD:       {{R: 214, G: 158, B: 93}, {R: 122, G: 73, B: 35}},
	PASTEL:     {{R: 255, G: 102, B: 180}, {R: 52, G: 120, B: 240}},
	CYBERPUNK:  {{R: 200, G: 200, B: 35}, {R: 180, G: 0, B: 255}},
	FOREST:     {{R: 167, G: 201, B: 87}, {R: 56, G: 102, B: 65}},
	OCEAN:      {{R: 0, G: 180, B: 216}, {R: 0, G: 105, B: 148}},
	SAKURA:     {{R: 240, G: 150, B: 175}, {R: 179, G: 70, B: 123}},
	SUNSET:     {{R: 236, G: 179, B: 86}, {R: 184, G: 80, B: 66}},
	LAVENDER:   {{R: 205, G: 180, B: 219}, {R: 109, G: 89, B: 122}},
	NORD:       {{R: 216, G: 222, B: 233}, {R: 76, G: 86, B: 106}},
	MONOCHROME: {{R: 207, G: 207, B: 207}, {R: 92, G: 92, B: 92}},
}

func (t ColorTheme) String() string {
	switch t {
	case DEFAULT:
		return "Default"
	case WOOD:
		return "Wood"
	case PASTEL:
		return "Pastel"
	case CYBERPUNK:
		return "Cyberpunk"
	case FOREST:
		return "Forest"
	case OCEAN:
		return "Ocean"
	case SAKURA:
		return "Sakura"
	case SUNSET:
		return "Sunset"
	case LAVENDER:
		return "Lavender"
	case NORD:
		return "Nord"
	case MONOCHROME:
		return "Monochrome"
	default:
		return "Default"
	}
}

func ColorThemes() []ColorTheme {
	return []ColorTheme{DEFAULT, WOOD, PASTEL, CYBERPUNK, FOREST, OCEAN, SAKURA, SUNSET, LAVENDER, NORD, MONOCHROME}
}

func (t ColorTheme) SquareColors() (light RGB, dark RGB) {
	colors, ok := colorThemeColors[t]
	if !ok {
		colors = colorThemeColors[DEFAULT]
	}

	return colors[0], colors[1]
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
	return r.SquareAt(row, column)
}

func (r *Renderer) SquareAt(row, column int) Square {
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
