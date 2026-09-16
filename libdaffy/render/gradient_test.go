package render

import (
	"github.com/janearc/libdaffy/canvas"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"image"
	"math"
	"testing"
)

// a linear gradient runs from its first stop to its last along the axis.
func TestLinearGradient(t *testing.T) {
	stops := []swatch.Swatch{
		srgb.RGB{R: 0, G: 0, B: 0}.Swatch(),
		srgb.RGB{R: 1, G: 1, B: 1}.Swatch(),
	}
	g := NewGradient(Linear, stops, 0, 0, 9, 0)
	if g.T(0, 5) != 0 || g.T(9, 5) != 1 {
		t.Errorf("ends: %.2f %.2f", g.T(0, 5), g.T(9, 5))
	}
	if g.T(-5, 0) != 0 || g.T(20, 0) != 1 {
		t.Error("beyond the axis should clamp")
	}
	if got := g.T(4, 0); math.Abs(got-4.0/9) > 1e-9 {
		t.Errorf("projection %.3f", got)
	}
	// Position is by projection: a pixel far off the axis at the same x has
	// the same t, because the axis is horizontal.
	if g.T(4, 100) != g.T(4, 0) {
		t.Error("projection should ignore the perpendicular")
	}
	if !g.Color(0, 0).Equal(rgb(stops[0])) ||
		!g.Color(9, 0).Equal(rgb(stops[1])) {
		t.Error("colour at the ends")
	}
	mid := g.Color(4, 0)
	if mid.R < 0.3 || mid.R > 0.8 {
		t.Errorf("mid grey %s", mid.Hex())
	}
	// A degenerate axis is all at zero.
	z := NewGradient(Linear, stops, 3, 3, 3, 3)
	if z.T(7, 7) != 0 {
		t.Error("degenerate")
	}
	if Linear.String() != "linear" || Radial.String() != "radial" {
		t.Error("names")
	}
}

// a radial gradient is its first stop at the centre and its last at the rim.
func TestRadialGradient(t *testing.T) {
	stops := []swatch.Swatch{
		srgb.RGB{R: 1, G: 0, B: 0}.Swatch(),
		srgb.RGB{R: 0, G: 0, B: 1}.Swatch(),
	}
	g := NewGradient(Radial, stops, 10, 10, 15, 10)
	if g.T(10, 10) != 0 {
		t.Error("centre")
	}
	if got := g.T(15, 10); math.Abs(got-1) > 1e-9 {
		t.Errorf("rim %.3f", got)
	}
	if g.T(10, 15) != g.T(15, 10) || g.T(5, 10) != 1 {
		t.Error("radial symmetry")
	}
	if g.T(30, 30) != 1 {
		t.Error("beyond the rim should clamp")
	}
	z := NewGradient(Radial, stops, 1, 1, 1, 1)
	if z.T(5, 5) != 0 {
		t.Error("zero radius")
	}
}

// steps keep the end stops, and values are t for every pixel of a rect.
func TestGradientStepsAndValues(t *testing.T) {
	stops := []swatch.Swatch{
		srgb.RGB{R: 0, G: 0, B: 0}.Swatch(),
		srgb.RGB{R: 1, G: 0, B: 0}.Swatch(),
		srgb.RGB{R: 1, G: 1, B: 1}.Swatch(),
	}
	g := NewGradient(Linear, stops, 0, 0, 7, 0)
	steps := g.Steps(5)
	if len(steps) != 5 || !rgb(steps[0]).Equal(rgb(stops[0])) ||
		!rgb(steps[4]).Equal(rgb(stops[2])) {
		t.Error("steps")
	}
	k, f := g.Step(0, 0, 5)
	if k != 0 || f != 0 {
		t.Errorf("first step %d %.2f", k, f)
	}
	k, f = g.Step(7, 0, 5)
	if k != 4 || f != 0 {
		t.Errorf("last step %d %.2f", k, f)
	}
	k, f = g.Step(3, 0, 5)
	if k != 1 || f < 0.5 || f > 0.8 {
		t.Errorf("mid step %d %.2f", k, f)
	}
	if k, f := g.Step(3, 0, 1); k != 0 || f != 0 {
		t.Error("one step")
	}
	v := g.Values(image.Rect(0, 0, 8, 2))
	if len(v) != 16 || v[0] != 0 || v[7] != 1 || v[8] != 0 {
		t.Errorf("values %v", v)
	}
	m := canvas.NewMask(8, 2)
	m.Threshold(v, 0.5)
	if m.Count() != 8 || m.At(3, 0) || !m.At(4, 0) {
		t.Errorf("thresholded mask %d", m.Count())
	}
}

// rgb is a swatch as the display shows it, for comparing with a painted colour.
func rgb(s swatch.Swatch) srgb.RGB {
	c, _ := srgb.FromSwatch(s)
	return c
}
