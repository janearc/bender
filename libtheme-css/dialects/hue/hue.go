// Package hue is the philips hue dialect: how a hue lamp names a colour, and
// how that becomes a swatch and comes back. A hue lamp never says a colour.
//
// It says a place on the horseshoe, a chromaticity, or a colour temperature in
// mirek, and separately how bright, as a percent that is the lamp's own scale
// and not a luminance. A gradient lamp says five places in order along itself.
//
// The dialect turns each of those into swatches at unit luminance and a list of
// them into a ramp; the reverse samples a ramp back to as many places as a lamp
// has. Brightness is not translated here: it is a law between a leader and a
// follower, and it lives with them.
//
// Chromaticities and gamuts are swatch's, not hue's; what is hue's is mirek,
// the places along a lamp, and the three gamuts its lamps report by letter.
package hue

import (
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
)

// Swatch is a place on the horseshoe at unit luminance.
func Swatch(p swatch.XY) swatch.Swatch { return swatch.FromXY(p, 1) }

// Of is the swatch's place on the horseshoe, as hue would want it.
func Of(s swatch.Swatch) swatch.XY { return s.XY() }

// Mirek is a colour temperature as hue sends it, reciprocal megakelvin,
// as a swatch on the planckian locus at unit luminance. Hue's lamps
// speak 153 to 500, which is 6536 K down to 2000 K.
func Mirek(m int) swatch.Swatch {
	if m <= 0 {
		return swatch.Black
	}
	return swatch.Planckian(1e6 / float64(m))
}

// Ramp is a lamp's places in order as a ramp: even stops, mixed in
// oklab, so the colour between two of a gradient's points is the one
// the eye would put there. One place is a ramp that is that colour
// everywhere.
func Ramp(places ...swatch.XY) functions.Ramp {
	swatches := make([]swatch.Swatch, len(places))
	for i, p := range places {
		swatches[i] = Swatch(p)
	}
	return functions.Even(ok.Mix, swatches...)
}

// Points is a ramp sampled back to n places, for a lamp with n of them:
// five for a gradient signe, one for a bulb. The samples are taken
// evenly from 0 to 1, so a ramp made from a lamp's own places comes back
// as those places.
func Points(r functions.Ramp, n int) []swatch.XY {
	out := make([]swatch.XY, 0, n)
	for _, s := range r.Samples(n) {
		out = append(out, Of(s))
	}
	return out
}

// The gamuts hue's lamps report, by the letter hue gives them, as philips
// publishes the corners. Vendor data, typed: these are facts about lamps, not
// derivable. A lamp reports its own triangle on the wire and that is the one to
// fit to; these are for a lamp that does not, and for tests.
var (
	GamutA = swatch.Gamut{Red: swatch.XY{X: 0.704, Y: 0.296},
		Green: swatch.XY{X: 0.2151,
			Y: 0.7106}, Blue: swatch.XY{X: 0.138, Y: 0.08}}
	GamutB = swatch.Gamut{Red: swatch.XY{X: 0.675, Y: 0.322},
		Green: swatch.XY{X: 0.409, Y: 0.518}, Blue: swatch.XY{X: 0.167,
			Y: 0.04}}
	GamutC = swatch.Gamut{Red: swatch.XY{X: 0.6915, Y: 0.3083},
		Green: swatch.XY{X: 0.17, Y: 0.7}, Blue: swatch.XY{X: 0.1532,
			Y: 0.0475}}
)
