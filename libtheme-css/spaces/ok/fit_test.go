package ok

import (
	"math"
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// a gamut that is a box on chroma: inside when chroma is at most 0.1.
func boxed(s swatch.Swatch) bool { return FromSwatch(s).Polar().C <= 0.1 }

// a colour inside the gamut is returned as it is; one outside comes
// back at the edge with the same lightness and hue, and the loss is
// the chroma it gave up.
func TestFitHoldsLightnessAndHue(t *testing.T) {
	in := OKLCH{0.7, 0.05, 40}
	if got, lost := Fit(in, boxed); got != in || lost != 0 {
		t.Fatalf("an inside colour moved: %v %v", got, lost)
	}
	out := OKLCH{0.7, 0.3, 40}
	got, lost := Fit(out, boxed)
	if got.L != out.L || got.H != out.H {
		t.Fatalf("lightness or hue moved: %v", got)
	}
	if math.Abs(got.C-0.1) > 1e-4 || math.Abs(lost-0.2) > 1e-4 {
		t.Fatalf("chroma %v lost %v, want 0.1 and 0.2", got.C, lost)
	}
}

// a gamut nothing at that lightness can enter leaves the colour grey.
func TestFitToNothingIsGrey(t *testing.T) {
	got, _ := Fit(OKLCH{0.5, 0.2, 200},
		func(swatch.Swatch) bool { return false })
	if got.C > 1e-6 {
		t.Fatalf("chroma left: %v", got.C)
	}
}
