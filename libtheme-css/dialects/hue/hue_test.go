package hue

import (
	"math"
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// near is whether two values agree within a tolerance.
func near(a, b swatch.XY, tol float64) bool {
	return math.Abs(a.X-b.X) <= tol && math.Abs(a.Y-b.Y) <= tol
}

// what a lamp says comes back as what it said: white's place through a
// swatch and back is white's place.
func TestPlaceRoundTrip(t *testing.T) {
	w := Of(swatch.White)
	if !near(Of(Swatch(w)), w, 1e-12) {
		t.Errorf("white did not round-trip: %v", Of(Swatch(w)))
	}
}

// a ramp made from a lamp's five places sampled back to five is those
// places, to the precision of going through oklab and back.
func TestRampToPointsIsIdentityAtTheStops(t *testing.T) {
	in := []swatch.XY{{X: 0.64, Y: 0.33}, {X: 0.5, Y: 0.4}, {X: 0.3127,
		Y: 0.329}, {X: 0.2, Y: 0.5}, {X: 0.15, Y: 0.06}}
	out := Points(Ramp(in...), 5)
	for i := range in {
		if !near(out[i], in[i], 1e-6) {
			t.Errorf("point %d: in %v out %v", i, in[i], out[i])
		}
	}
}

// 370 mirek is 2703 K, a warm bulb; it must sit on the locus between
// illuminant A (2856 K, x 0.4476) and warmer, so x above 0.44.
func TestMirekIsWarm(t *testing.T) {
	p := Of(Mirek(370))
	if p.X < 0.44 || p.X > 0.48 {
		t.Errorf("370 mirek gave x %.4f", p.X)
	}
}

// gamuts a and c hold white; gamut b, the old bulbs, misses d65 by six
// ten-thousandths past its green-blue edge, which is a known fact about
// those lamps, so Fit moves white less than a thousandth. spectral red
// fits to gamut c's red corner, which lies short of it.
func TestPublishedGamuts(t *testing.T) {
	w := swatch.White.XY()
	for name, g := range map[string]swatch.Gamut{"A": GamutA, "C": GamutC} {
		if !g.Contains(w) {
			t.Errorf("white is outside gamut %s", name)
		}
	}
	if f := GamutB.Fit(w); math.Hypot(f.X-w.X, f.Y-w.Y) > 0.001 {
		t.Errorf("gamut b should miss white by a hair, not %v", f)
	}
	red := swatch.XY{X: 0.7347, Y: 0.2653}
	if f := GamutC.Fit(red); !near(f, GamutC.Red, 1e-9) {
		t.Errorf("spectral red should fit to gamut c's red corner, "+
			"got %v", f)
	}
}
