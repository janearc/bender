package render

// A gradient is a tool, not a fill option. Stops are an ordered list of
// colours of any length; two is the common case and is not special. Linear
// takes its position by projection onto a dragged axis, radial by distance
// from a dragged centre. Interpolation is in Oklab, always.
//
// The half-block pixel is nearly square, so no aspect correction is applied,
// and none is to be added.

import (
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"image"
	"math"
)

// GradientKind is linear or radial.
type GradientKind int

// The kinds.
const (
	Linear GradientKind = iota
	Radial
)

// String names the kind.
func (k GradientKind) String() string {
	if k == Radial {
		return "radial"
	}
	return "linear"
}

// Gradient is a ramp laid across the pixel grid.
type Gradient struct {
	Kind  GradientKind
	Stops []swatch.Swatch
	// X0,Y0 to X1,Y1 is the axis for linear, or centre to rim for radial,
	// in pixels. Pixel centres are at integer plus one half.
	X0, Y0, X1, Y1 float64
	// ramp is libtheme's ramp over the stops, built once.
	ramp functions.Ramp
}

// ramped is the gradient with its ramp built, for a value made by hand.
func (g Gradient) ramped() Gradient {
	if len(g.ramp.Stops) == 0 && len(g.Stops) > 0 {
		g.ramp = functions.Even(ok.Mix, g.Stops...)
	}
	return g
}

// NewGradient makes a gradient from a drag between two pixels.
func NewGradient(
	kind GradientKind,
	stops []swatch.Swatch,
	x0, y0, x1, y1 int,
) Gradient {
	return Gradient{
		Kind: kind, Stops: stops,
		X0: float64(x0) + 0.5, Y0: float64(y0) + 0.5,
		X1: float64(x1) + 0.5, Y1: float64(y1) + 0.5,
	}.ramped()
}

// T is a pixel's position along the ramp, clamped to 0-1. For a degenerate
// axis every pixel is at zero.
func (g Gradient) T(x, y int) float64 {
	px, py := float64(x)+0.5, float64(y)+0.5
	dx, dy := g.X1-g.X0, g.Y1-g.Y0
	switch g.Kind {
	case Radial:
		r := math.Hypot(dx, dy)
		if r == 0 {
			return 0
		}
		return clamp(math.Hypot(px-g.X0, py-g.Y0)/r, 0, 1)
	default:
		l2 := dx*dx + dy*dy
		if l2 == 0 {
			return 0
		}
		return clamp(((px-g.X0)*dx+(py-g.Y0)*dy)/l2, 0, 1)
	}
}

// Color is the interpolated colour at a pixel.
func (g Gradient) Color(x, y int) srgb.RGB {
	switch len(g.Stops) {
	case 0:
		return srgb.RGB{}
	case 1:
		c, _ := srgb.FromSwatch(g.Stops[0])
		return c
	}
	c, _ := srgb.FromSwatch(g.ramped().ramp.At(g.T(x, y)))
	return c
}

// Values is T for every pixel of a rectangle, row-major, which is what a mask
// thresholds.
func (g Gradient) Values(r image.Rectangle) []float64 {
	out := make([]float64, 0, r.Dx()*r.Dy())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			out = append(out, g.T(x, y))
		}
	}
	return out
}

// Steps is the ramp resampled onto n colours, for painting with parts. A
// canvas holds parts, not colours, so a gradient on the canvas is a small
// number of literal parts, and n says how many.
func (g Gradient) Steps(n int) []swatch.Swatch {
	switch {
	case n < 1 || len(g.Stops) == 0:
		return nil
	case len(g.Stops) == 1:
		out := make([]swatch.Swatch, n)
		for i := range out {
			out[i] = g.Stops[0]
		}
		return out
	case n == 1:
		return []swatch.Swatch{g.Stops[len(g.Stops)/2]}
	}
	return g.ramped().ramp.Samples(n)
}

// Step is which of n steps a pixel falls in, and how far into it, 0-1, for
// dithering between neighbours. The last step has nothing above it, so its
// fraction is zero.
func (g Gradient) Step(x, y, n int) (k int, frac float64) {
	if n <= 1 {
		return 0, 0
	}
	t := g.T(x, y) * float64(n-1)
	k = int(t)
	if k >= n-1 {
		return n - 1, 0
	}
	return k, t - float64(k)
}
