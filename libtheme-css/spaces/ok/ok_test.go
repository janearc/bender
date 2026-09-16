package ok

import (
	"math"
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// near is whether two values agree within a tolerance.
func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// The reference values from Ottosson's description of the space, XYZ in
// and L, a, b out, to the three places he printed them.
func TestReferenceValues(t *testing.T) {
	cases := []struct {
		x, y, z float64
		l, a, b float64
	}{
		{0.950, 1.000, 1.089, 1.000, 0.000, 0.000},
		{1.000, 0.000, 0.000, 0.450, 1.236, -0.019},
		{0.000, 1.000, 0.000, 0.922, -0.671, 0.263},
		{0.000, 0.000, 1.000, 0.153, -1.415, -0.449},
	}
	for _, c := range cases {
		got := FromSwatch(swatch.FromXYZ(c.x, c.y, c.z))
		if !near(got.L, c.l, 1e-3) || !near(got.A, c.a, 1e-3) ||
			!near(got.B, c.b, 1e-3) {
			t.Errorf("xyz %v %v %v gave %+v, want L %v a %v b %v",
				c.x, c.y, c.z, got, c.l, c.a, c.b)
		}
	}
}

// The library's derived white is the space's defined white, to three
// places, and black is black exactly.
func TestWhiteAndBlack(t *testing.T) {
	w := FromSwatch(swatch.White)
	if !near(w.L, White.L, 1e-3) || !near(w.A, 0, 1e-3) || !near(w.B, 0,
		1e-3) {
		t.Errorf("white = %+v", w)
	}
	if FromSwatch(swatch.Black) != Black {
		t.Errorf("black = %+v", FromSwatch(swatch.Black))
	}
}

// Going to OKLab and back lands on the swatch you started from, to a
// trillionth (measured on darwin/arm64): the inverses are computed, not
// typed, and the cube and cube root undo each other.
func TestRoundTrip(t *testing.T) {
	for _, in := range []swatch.Swatch{swatch.White, swatch.FromXYZ(0.2,
		0.1, 0.05), swatch.FromXYZ(0.4, 0.6, 0.9)} {
		out := FromSwatch(in).Swatch()
		ix, iy, iz := in.XYZ()
		ox, oy, oz := out.XYZ()
		if !near(ix, ox, 1e-12) || !near(iy, oy, 1e-12) || !near(iz, oz,
			1e-12) {
			t.Errorf("%v came back as %v", in, out)
		}
	}
}

// Polar and rectangular are the same point: a grey has no chroma and hue
// 0, a lean straight along +a is hue 0, straight along +b is hue 90, and
// the trip there and back is exact.
func TestPolar(t *testing.T) {
	if p := Grey(0.5).Polar(); p.C != 0 || p.H != 0 || p.L != 0.5 {
		t.Errorf("grey polar = %+v", p)
	}
	if p := (OKLab{0.5, 0.1, 0}).Polar(); !near(p.H, 0, 1e-9) || !near(p.C,
		0.1, 1e-12) {
		t.Errorf("+a polar = %+v", p)
	}
	if p := (OKLab{0.5, 0, 0.1}).Polar(); !near(p.H, 90, 1e-9) {
		t.Errorf("+b polar = %+v", p)
	}
	in := OKLab{0.7, -0.05, 0.12}
	out := in.Polar().Rect()
	if !near(in.A, out.A, 1e-12) || !near(in.B, out.B, 1e-12) {
		t.Errorf("%+v came back as %+v", in, out)
	}
}

// Distance is zero to itself, symmetric, and the tolerances mean what
// they say: a round trip is within Exact, a step of 0.01 in lightness is
// within Eye, and a step of 0.1 is not.
func TestDistance(t *testing.T) {
	a := OKLab{0.6, 0.1, -0.05}
	if Distance(a, a) != 0 {
		t.Errorf("distance to self = %v", Distance(a, a))
	}
	b := OKLab{0.2, -0.1, 0.1}
	if Distance(a, b) != Distance(b, a) {
		t.Errorf("distance is not symmetric")
	}
	if !Same(a, FromSwatch(a.Swatch()), Exact) {
		t.Errorf("round trip is not exact: %v", Distance(a,
			FromSwatch(a.Swatch())))
	}
	if !Same(a, OKLab{0.61, 0.1, -0.05}, Eye) {
		t.Errorf("a hundredth of lightness should be invisible")
	}
	if Same(a, OKLab{0.7, 0.1, -0.05}, Eye) {
		t.Errorf("a tenth of lightness should be visible")
	}
}

// String is what CSS accepts, with none for a hue no eye could see.
func TestString(t *testing.T) {
	if got := (OKLCH{0.74, 0.2,
		345.3}).String(); got != "oklch(74% 0.200 345.3)" {
		t.Errorf("String = %q", got)
	}
	if got := Grey(1).Polar().String(); got != "oklch(100% 0.000 none)" {
		t.Errorf("grey String = %q", got)
	}
}
