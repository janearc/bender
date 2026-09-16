// Package grafana is the grafana dialect: how grafana names a colour.
//
// Grafana does not take a theme. It names colours in a vocabulary of its
// own, a hue with a shade in front of it, and everywhere else it asks for
// a hex in a fixed-colour field.
//
// The five names of a hue are dark, semi-dark, the bare hue, light and
// super-light. The dialect reads them as distances along the line from
// the colour towards black or towards white, mixed in oklab, which keeps
// the hue: a lighter green stays green.
//
// Moving the lightness on its own does not, because past the gamut's edge
// the only way to keep going is to drop the chroma, and three of the five
// names come out white.
//
// It translates both ways: a swatch becomes the five shades, and a name
// becomes the swatch it stands for.
package grafana

import (
	"strings"

	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// Shades are grafana's names for one hue, darkest first. The bare hue has
// no prefix, which is why one of these is empty.
var Shades = []string{"dark", "semi-dark", "", "light", "super-light"}

// step is how far along the line each shade sits: negative towards black,
// positive towards white, as a fraction of the whole distance. A quarter
// of the way reads as one step of a palette on screen, and half is as far
// as a shade goes before it stops being the same colour.
var step = map[string]float64{
	"dark": -0.50, "semi-dark": -0.25, "": 0, "light": 0.25,
	"super-light": 0.50,
}

// Of is the five shades grafana expects of one hue, keyed by shade name.
func Of(c swatch.Swatch) map[string]swatch.Swatch {
	out := map[string]swatch.Swatch{}
	for _, s := range Shades {
		out[s] = Shade(c, s)
	}
	return out
}

// Shade is one named shade of a colour: that far along the line towards
// black or towards white, mixed in oklab, and then brought inside the
// gamut, since a hex outside it is not a colour grafana can show.
func Shade(c swatch.Swatch, shade string) swatch.Swatch {
	d, known := step[shade]
	if !known || d == 0 {
		return c
	}
	end := swatch.White
	if d < 0 {
		end, d = swatch.Black, -d
	}
	line := functions.Even(ok.Mix, c, end)
	fit, _ := ok.Fit(ok.FromSwatch(line.At(d)).Polar(), InGamut)
	return fit.Rect().Swatch()
}

// InGamut is whether a swatch is a colour sRGB can name, which is the
// only question grafana's hex field can answer.
func InGamut(s swatch.Swatch) bool {
	_, in := srgb.FromSwatch(s)
	return in
}

// Name is grafana's name for a hue at a shade: "semi-dark-blue", or
// "blue" for the bare hue.
func Name(shade, hue string) string {
	if shade == "" {
		return hue
	}
	return shade + "-" + hue
}

// Split reads one of grafana's names back into its shade and its hue. A
// name with no shade in front of it is the bare hue, which is a shade
// too.
func Split(name string) (shade, hue string) {
	for _, s := range Shades {
		if s == "" {
			continue
		}
		if strings.HasPrefix(name, s+"-") {
			return s, strings.TrimPrefix(name, s+"-")
		}
	}
	return "", name
}

// Palette is every name grafana can use for the hues it is given: five
// shades of each, as hex, ready for a dashboard's fixed-colour fields.
// The keys are grafana's names and nothing else is invented.
func Palette(hues map[string]swatch.Swatch) map[string]string {
	out := map[string]string{}
	for hue, c := range hues {
		for shade, s := range Of(c) {
			rgb, _ := srgb.FromSwatch(s)
			out[Name(shade, hue)] = rgb.Hex()
		}
	}
	return out
}

// Lookup is the swatch one of grafana's names stands for, given the hues
// it was built from. A name whose hue is not bound is not ours, and says
// so rather than guessing.
func Lookup(name string, hues map[string]swatch.Swatch) (swatch.Swatch, bool) {
	shade, hue := Split(name)
	c, bound := hues[hue]
	if !bound {
		return swatch.Swatch{}, false
	}
	return Shade(c, shade), true
}
