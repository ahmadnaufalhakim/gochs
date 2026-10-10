package chess

import (
	"strings"
	"testing"
)

func TestRendererSquareBackground(t *testing.T) {
	tests := []struct {
		name     string
		renderer Renderer
		square   Square
		want     string
	}{
		{"default dark", Renderer{Theme: DEFAULT}, a1, defaultDarkSquare},
		{"default light", Renderer{Theme: DEFAULT}, b1, defaultLightSquare},
		{"wood dark", Renderer{Theme: WOOD}, a1, woodDarkSquare},
		{"wood light", Renderer{Theme: WOOD}, b1, woodLightSquare},
		{"pastel dark", Renderer{Theme: PASTEL}, a1, pastelDarkSquare},
		{"pastel light", Renderer{Theme: PASTEL}, b1, pastelLightSquare},
		{"cyberpunk dark", Renderer{Theme: CYBERPUNK}, a1, cyberpunkDarkSquare},
		{"cyberpunk light", Renderer{Theme: CYBERPUNK}, b1, cyberpunkLightSquare},
		{"forest dark", Renderer{Theme: FOREST}, a1, forestDarkSquare},
		{"forest light", Renderer{Theme: FOREST}, b1, forestLightSquare},
		{"ocean dark", Renderer{Theme: OCEAN}, a1, oceanDarkSquare},
		{"ocean light", Renderer{Theme: OCEAN}, b1, oceanLightSquare},
		{"sakura dark", Renderer{Theme: SAKURA}, a1, sakuraDarkSquare},
		{"sakura light", Renderer{Theme: SAKURA}, b1, sakuraLightSquare},
		{"sunset dark", Renderer{Theme: SUNSET}, a1, sunsetDarkSquare},
		{"sunset light", Renderer{Theme: SUNSET}, b1, sunsetLightSquare},
		{"lavender dark", Renderer{Theme: LAVENDER}, a1, lavenderDarkSquare},
		{"lavender light", Renderer{Theme: LAVENDER}, b1, lavenderLightSquare},
		{"nord dark", Renderer{Theme: NORD}, a1, nordDarkSquare},
		{"nord light", Renderer{Theme: NORD}, b1, nordLightSquare},
		{"monochrome dark", Renderer{Theme: MONOCHROME}, a1, monochromeDarkSquare},
		{"monochrome light", Renderer{Theme: MONOCHROME}, b1, monochromeLightSquare},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.renderer.squareBackground(test.square); got != test.want {
				t.Errorf("squareBackground(%s) = %q, want %q", test.square, got, test.want)
			}
		})
	}
}

func TestRendererSquareAtExported(t *testing.T) {
	tests := []struct {
		name     string
		renderer Renderer
		row      int
		column   int
		want     Square
	}{
		{"white top left", Renderer{Perspective: White}, 0, 0, a8},
		{"white bottom right", Renderer{Perspective: White}, 7, 7, h1},
		{"black top left", Renderer{Perspective: Black}, 0, 0, h1},
		{"black bottom right", Renderer{Perspective: Black}, 7, 7, a8},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.renderer.squareAt(test.row, test.column); got != test.want {
				t.Errorf("squareAt(%d, %d) = %s, want %s", test.row, test.column, got, test.want)
			}
		})
	}
}

func TestPieceForeground(t *testing.T) {
	if got := pieceForeground(Piece{Type: PieceNone}); got != "" {
		t.Errorf("empty piece foreground = %q, want empty string", got)
	}
	if got := pieceForeground(Piece{Color: White, Type: King}); got != whitePieceColor {
		t.Errorf("white piece foreground = %q, want %q", got, whitePieceColor)
	}
	if got := pieceForeground(Piece{Color: Black, Type: King}); got != blackPieceColor {
		t.Errorf("black piece foreground = %q, want %q", got, blackPieceColor)
	}
}

func TestRendererPrint(t *testing.T) {
	var board Board
	board.SetPieceAt(e4, Piece{Color: White, Type: Queen})

	renderer := Renderer{Theme: DEFAULT, Perspective: White}
	output := captureStdout(t, func() {
		renderer.Print(board)
	})

	for _, want := range []string{"8 ", "1 ", "   a  b  c  d  e  f  g  h ", "♕", ansiReset} {
		if !strings.Contains(output, want) {
			t.Errorf("Print() output does not contain %q", want)
		}
	}
}

func TestColorThemeMetadata(t *testing.T) {
	themes := ColorThemes()
	if len(themes) == 0 || themes[0] != DEFAULT {
		t.Fatalf("ColorThemes() = %v, want Default first", themes)
	}

	light, dark := WOOD.SquareColors()
	if light != (RGB{R: 214, G: 158, B: 93}) || dark != (RGB{R: 122, G: 73, B: 35}) {
		t.Errorf("WOOD.SquareColors() = %#v, %#v", light, dark)
	}
	if WOOD.String() != "Wood" {
		t.Errorf("WOOD.String() = %q, want Wood", WOOD.String())
	}
}

func TestRendererSquareAt(t *testing.T) {
	white := Renderer{Perspective: White}
	black := Renderer{Perspective: Black}
	if got := white.SquareAt(0, 0); got != a8 {
		t.Errorf("white SquareAt(0, 0) = %s, want a8", got)
	}
	if got := black.SquareAt(0, 0); got != h1 {
		t.Errorf("black SquareAt(0, 0) = %s, want h1", got)
	}
}
