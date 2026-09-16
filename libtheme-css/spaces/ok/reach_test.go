package ok

import (
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// a gamut that is a box on chroma, as in the fit test.
func inBox(s swatch.Swatch) bool { return FromSwatch(s).Polar().C <= 0.1 }

// every cell of the grid is inside the gamut, chroma zero is grey, and
// a cell's own colour is nearest to itself.
func TestReach(t *testing.T) {
	g := Grid(12, 6, inBox)
	for y := 0; y < g.Rows; y++ {
		for x := 0; x < g.Cols; x++ {
			if !inBox(g.At(x, y).Rect().Swatch()) {
				t.Fatalf("cell %d,%d is outside the gamut", x,
					y)
			}
		}
	}
	g.Chroma = 0
	if c := g.At(3, 3); c.C != 0 {
		t.Errorf("chroma zero gave %v", c)
	}
	g.Chroma = 1
	if x, y := g.Nearest(g.At(7, 2).Rect()); x != 7 || y != 2 {
		t.Errorf("nearest to a cell's own colour was %d,%d", x, y)
	}
}
