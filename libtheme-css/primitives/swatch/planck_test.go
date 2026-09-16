package swatch

import (
	"math"
	"testing"
)

// CIE illuminant A is defined as a black body at 2856 K, and its
// chromaticity is published to four places: x 0.4476, y 0.4074. The
// derived locus must land on it; this is the check that Planck's law,
// the observer tables and Illuminant agree.
func TestPlanckianIsIlluminantA(t *testing.T) {
	c := Planckian(2856)
	x, y := c.XY().X, c.XY().Y
	if math.Abs(x-0.4476) > 0.001 || math.Abs(y-0.4074) > 0.001 {
		t.Errorf("2856 K gave xy %.4f %.4f, illuminant A is 0.4476 "+
			"0.4074", x, y)
	}
}

// Daylight is not a black body, so 6504 K does not land exactly on D65;
// it lands close, on the warm side of it, and the gap is a known fact
// about daylight rather than an error here.
func TestPlanckianNearD65(t *testing.T) {
	p, w := Planckian(6504).XY(), White.XY()
	x, y, wx, wy := p.X, p.Y, w.X, w.Y
	if math.Hypot(x-wx, y-wy) > 0.01 {
		t.Errorf("6504 K gave xy %.4f %.4f, d65 is %.4f %.4f", x, y, wx,
			wy)
	}
}

// A chromaticity put back at unit luminance is the swatch it came from.
func TestXYRoundTrip(t *testing.T) {
	s := FromXY(White.XY(), 1)
	sx, sy, sz := s.XYZ()
	wx, wy, wz := White.XYZ()
	if math.Abs(sx-wx) > 1e-12 || math.Abs(sy-wy) > 1e-12 ||
		math.Abs(sz-wz) > 1e-12 {
		t.Errorf("white through xy came back %v %v %v", sx, sy, sz)
	}
	if FromXY(XY{0.3, 0}, 1) != Black {
		t.Error("y of zero is not black")
	}
}

// the gamut is a triangle: srgb's primaries typed here as a test may;
// its white is inside, spectral red is outside and fits to the red
// corner, srgb blue is inside by definition, and whatever is outside,
// Fit's answer is inside.
func TestGamut(t *testing.T) {
	g := Gamut{Red: XY{0.64, 0.33}, Green: XY{0.30, 0.60}, Blue: XY{0.15,
		0.06}}
	if !g.Contains(White.XY()) {
		t.Error("white is outside srgb")
	}
	if !g.Contains(XY{0.15, 0.06}) {
		t.Error("a corner is not inside")
	}
	red := XY{0.7347, 0.2653}
	if g.Contains(red) {
		t.Error("spectral red is inside srgb")
	}
	if f := g.Fit(red); math.Abs(f.X-0.64) > 1e-9 ||
		math.Abs(f.Y-0.33) > 1e-9 {
		t.Errorf("spectral red should fit to the red corner, got %v", f)
	}
	for _, p := range []XY{{0, 0}, {0.1, 0.8}, {0.9, 0.1}, {0.3, 0.9}, {0.5,
		0.02}, {0.7347, 0.2653}} {
		if f := g.Fit(p); !g.Contains(f) {
			t.Errorf("fit of %v gave %v, which Contains refuses", p,
				f)
		}
	}
	if w := White.XY(); g.Fit(w) != w {
		t.Error("fit moved a point that was inside")
	}
}
