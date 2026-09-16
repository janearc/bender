package functions

import (
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// A mixer that averages XYZ: wrong for an eye, right for a test, because
// it makes every expected value easy to compute by hand.
var flat = Mixer{"test", func(a, b swatch.Swatch, t float64) swatch.Swatch {
	ax, ay, az := a.XYZ()
	bx, by, bz := b.XYZ()
	return swatch.FromXYZ(ax+(bx-ax)*t, ay+(by-ay)*t, az+(bz-az)*t)
}}

// y is a swatch's luminance.
func y(s swatch.Swatch) float64 { _, v, _ := s.XYZ(); return v }

// Even spacing puts n stops at 0, 1/(n-1), ... 1; one stop sits at 0.
func TestEven(t *testing.T) {
	r := Even(flat, swatch.Black, swatch.White, swatch.Black)
	if len(r.Stops) != 3 || r.Stops[1].At != 0.5 || r.Stops[2].At != 1 {
		t.Errorf("stops = %+v", r.Stops)
	}
	if one := Even(flat, swatch.White); one.Stops[0].At != 0 ||
		one.At(0.7) != swatch.White {
		t.Errorf("a one-stop ramp is not that stop everywhere")
	}
}

// The ends are the end stops, the middle is the mixer's middle, and
// outside 0..1 the ramp holds its ends.
func TestAt(t *testing.T) {
	r := Even(flat, swatch.Black, swatch.White)
	if r.At(0) != swatch.Black || r.At(1) != swatch.White {
		t.Errorf("ends moved")
	}
	if got := y(r.At(0.5)); got != 0.5 {
		t.Errorf("middle Y = %v", got)
	}
	if r.At(-1) != swatch.Black || r.At(2) != swatch.White {
		t.Errorf("outside 0..1 the ramp should hold its ends")
	}
}

// Explicit stops are sorted, and the line between neighbours is local:
// a stop at 0.25 means the first quarter goes there, not the first half.
func TestStops(t *testing.T) {
	r := New(flat, Stop{1, swatch.White}, Stop{0.25, swatch.Black}, Stop{0,
		swatch.White})
	if r.Stops[0].At != 0 || r.Stops[1].At != 0.25 || r.Stops[2].At != 1 {
		t.Errorf("not sorted: %+v", r.Stops)
	}
	if got := y(r.At(0.125)); got != 0.5 {
		t.Errorf("Y at 0.125 = %v, want 0.5", got)
	}
	if got := y(r.At(0.625)); got != 0.5 {
		t.Errorf("Y at 0.625 = %v, want 0.5", got)
	}
}

// Samples runs 0 to 1 inclusive; one sample is the start; none is none.
func TestSamples(t *testing.T) {
	r := Even(flat, swatch.Black, swatch.White)
	s := r.Samples(5)
	if len(s) != 5 || s[0] != swatch.Black || s[4] != swatch.White ||
		y(s[2]) != 0.5 {
		t.Errorf("samples = %v", s)
	}
	if one := r.Samples(1); len(one) != 1 || one[0] != swatch.Black {
		t.Errorf("one sample = %v", one)
	}
	if r.Samples(0) != nil {
		t.Errorf("zero samples should be nil")
	}
}

// String is what CSS accepts, in the mixer's space.
func TestString(t *testing.T) {
	r := Even(flat, swatch.Black, swatch.White)
	got := r.String(func(s swatch.Swatch) string {
		if s == swatch.Black {
			return "#000000"
		}
		return "#ffffff"
	})
	if got != "linear-gradient(in test, #000000 0%, #ffffff 100%)" {
		t.Errorf("String = %q", got)
	}
}
