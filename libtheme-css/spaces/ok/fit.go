package ok

import "github.com/janearc/libtheme-css/primitives/swatch"

// Fit pulls a colour into a gamut, holding its lightness and hue.
//
// Chroma is what gives way, never lightness and never hue. A person who picked
// a colour chose a hue and a level; the saturation is the part they will accept
// less of.
//
// Projecting to the nearest point of the gamut bends hue instead, worst around
// blue to magenta, so the colour comes back a different colour rather than a
// less intense one.
//
// The gamut is whatever the predicate says is inside it, so the same rule fits
// a display, a lamp or a printer. The second return is how much chroma was
// given up, zero when nothing moved, so a caller can report the fit rather than
// clip in silence.
//
// A colour with no achievable chroma at that lightness comes back grey.
func Fit(c OKLCH, in func(swatch.Swatch) bool) (OKLCH, float64) {
	if in(c.Rect().Swatch()) {
		return c, 0
	}
	lo, hi := 0.0, c.C
	for i := 0; i < 24; i++ {
		mid := (lo + hi) / 2
		if in((OKLCH{c.L, mid, c.H}).Rect().Swatch()) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return OKLCH{c.L, lo, c.H}, c.C - lo
}
