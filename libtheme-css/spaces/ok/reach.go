package ok

import "github.com/janearc/libtheme-css/primitives/swatch"

// Reach is a grid of the colours a gamut can reach: hue across, lightness down,
// and chroma as a separate pull rather than a third axis, because at every
// point there is a range from grey to the edge and only the edge can be drawn.
//
// A picker draws this; a person picks from it; every cell is inside the gamut
// by construction, so a colour the display cannot show is not on the grid.
type Reach struct {
	// Cols and Rows are the grid size.
	Cols, Rows int
	// HueFrom and HueTo bound the horizontal axis, in degrees.
	HueFrom, HueTo float64
	// LightFrom and LightTo bound the vertical axis, 0-1, top to bottom;
	// bright at the top is the way a person expects light to fall.
	LightFrom, LightTo float64
	// Chroma is how far toward the edge each cell sits, 0-1. One is the
	// most saturated colour the gamut has at that hue and lightness.
	Chroma float64
	// In is the gamut: what can be shown. A display, a lamp, a printer.
	In func(swatch.Swatch) bool
}

// Grid is the starting view of a gamut: the whole hue circle, most of
// the useful lightness range, full chroma.
func Grid(cols, rows int, in func(swatch.Swatch) bool) Reach {
	return Reach{
		Cols: cols, Rows: rows,
		HueFrom: 0, HueTo: 360,
		LightFrom: 0.95, LightTo: 0.15,
		Chroma: 1,
		In:     in,
	}
}

// At is the colour at one cell: the edge is found by fitting an
// over-saturated colour at that hue and lightness, and the cell sits
// at Chroma of the way to it.
func (r Reach) At(col, row int) OKLCH {
	h := r.HueFrom
	if r.Cols > 1 {
		h += (r.HueTo - r.HueFrom) * float64(col) / float64(r.Cols-1)
	}
	l := r.LightFrom
	if r.Rows > 1 {
		l += (r.LightTo - r.LightFrom) *
			float64(row) / float64(r.Rows-1)
	}
	edge, _ := Fit(OKLCH{L: l, C: 0.4, H: h}, r.In)
	chroma := r.Chroma
	if chroma < 0 {
		chroma = 0
	}
	if chroma > 1 {
		chroma = 1
	}
	return OKLCH{L: l, C: edge.C * chroma, H: h}
}

// Nearest is the cell closest to a colour, so a colour already chosen
// can be shown on the grid.
func (r Reach) Nearest(c OKLab) (col, row int) {
	best := -1.0
	for y := 0; y < r.Rows; y++ {
		for x := 0; x < r.Cols; x++ {
			d := Distance(r.At(x, y).Rect(), c)
			if best < 0 || d < best {
				best, col, row = d, x, y
			}
		}
	}
	return col, row
}
