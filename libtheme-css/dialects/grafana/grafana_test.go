package grafana

import (
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// pink is the vaporwave accent, and a hue with somewhere to go in both
// directions.
var pink = srgb.MustHex("#ff6ec7").Swatch()

// the five shades run from dark to super-light, and each is lighter than
// the one before it: what grafana's names say is what they do.
func TestShadesAscend(t *testing.T) {
	last := -1.0
	for _, s := range Shades {
		l := ok.FromSwatch(Shade(pink, s)).Polar().L
		if l <= last {
			t.Errorf("%q is not lighter than the one before: %.3f",
				s, l)
		}
		last = l
	}
}

// every shade is a colour sRGB can name, because a hex is all grafana can
// be told.
func TestShadesAreInGamut(t *testing.T) {
	for _, s := range Shades {
		if !InGamut(Shade(pink, s)) {
			t.Errorf("shade %q left the gamut", s)
		}
	}
}

// the bare hue is the colour it was given, unmoved.
func TestBareHueIsUnchanged(t *testing.T) {
	if got := Shade(pink, ""); !ok.Same(ok.FromSwatch(got),
		ok.FromSwatch(pink), 1e-12) {
		t.Error("the bare hue moved")
	}
}

// a name goes to a shade and a hue and back to the same name, bare hues
// included.
func TestNamesRoundTrip(t *testing.T) {
	for _, hue := range []string{"blue", "green", "super"} {
		for _, shade := range Shades {
			name := Name(shade, hue)
			gotShade, gotHue := Split(name)
			if gotShade != shade || gotHue != hue {
				t.Errorf("%q read back as %q %q", name,
					gotShade, gotHue)
			}
		}
	}
}

// a palette holds five names per hue, all of them hex, and a name it was
// not given is refused rather than guessed.
func TestPalette(t *testing.T) {
	p := Palette(map[string]swatch.Swatch{"pink": pink})
	if len(p) != len(Shades) {
		t.Fatalf("palette has %d names, want %d", len(p), len(Shades))
	}
	for name, hex := range p {
		if len(hex) != 7 || hex[0] != '#' {
			t.Errorf("%s is %q, which is not a hex", name, hex)
		}
	}
	bound := map[string]swatch.Swatch{"pink": pink}
	if _, found := Lookup("dark-pink", bound); !found {
		t.Error("a bound hue was not found")
	}
	if _, found := Lookup("dark-teal", bound); found {
		t.Error("an unbound hue was answered")
	}
}
