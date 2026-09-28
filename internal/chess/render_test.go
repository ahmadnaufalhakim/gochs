package chess

import "testing"

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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.renderer.squareBackground(test.square); got != test.want {
				t.Errorf("squareBackground(%s) = %q, want %q", test.square, got, test.want)
			}
		})
	}
}

func TestRendererSquareAt(t *testing.T) {
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
