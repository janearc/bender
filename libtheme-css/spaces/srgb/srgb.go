// Package srgb is the swatch as a screen is driven: three lamp levels, red,
// green and blue, on the curve every screen since 1996 assumes. It is the door
// to hex, and to the two cylinders, hsl and hsv, which are the same lamps
// described by angle.
//
// A swatch may fall outside what the lamps can make; every exit here says
// whether it did.
//
// The standard, IEC 61966-2-1, states four things and this package types in
// only those: where each lamp sits on the 1931 chromaticity diagram (x and y
// for red, green and blue), and the curve. The white is the swatch's own D65.
// The 3x3 matrix everyone else copies is derived from the four, below.
package srgb

import (
	"fmt"
	"math"
	"strings"

	"github.com/janearc/libtheme-css/internal/mat"
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
)

// RGB is the three lamp levels as fractions, 0 off to 1 full.
type RGB struct {
	R, G, B float64
}

// HSL is the lamps described by angle: hue around the wheel, saturation,
// and lightness as the midpoint of the brightest and dimmest lamp.
type HSL struct {
	H, S, L float64
}

// HSV is the other cylinder: hue, saturation, and value, the brightest
// lamp on its own, so full red and white both have V 1.
type HSV struct {
	H, S, V float64
}

// The three primaries as the standard states them: where on the 1931
// diagram each lamp's light falls. x and y are XYZ with the brightness
// divided out, so a lamp is a place, not an amount.
var (
	redXY   = [2]float64{0.6400, 0.3300}
	greenXY = [2]float64{0.3000, 0.6000}
	blueXY  = [2]float64{0.1500, 0.0600}
)

// Gamut is the triangle the primaries make: what srgb can show. Derived
// from the same three chromaticities as the matrix, so the two cannot
// disagree.
var Gamut = swatch.Gamut{
	Red:   swatch.XY{X: redXY[0], Y: redXY[1]},
	Green: swatch.XY{X: greenXY[0], Y: greenXY[1]},
	Blue:  swatch.XY{X: blueXY[0], Y: blueXY[1]},
}

// rgbToXYZ is derived: each primary's column is its chromaticity turned
// back into XYZ at unit brightness, and the three columns are then
// scaled so that all three lamps at full add up to exactly the white.
// That one constraint fixes the nine numbers.
var rgbToXYZ = func() mat.M {
	col := func(c [2]float64) [3]float64 {
		x, y := c[0], c[1]
		return [3]float64{x / y, 1, (1 - x - y) / y}
	}
	r, g, b := col(redXY), col(greenXY), col(blueXY)
	p := mat.M{{r[0], g[0], b[0]}, {r[1], g[1], b[1]}, {r[2], g[2], b[2]}}
	sr, sg, sb := p.Inverse().Apply(swatch.White.XYZ())
	return p.Scale(sr, sg, sb)
}()

var xyzToRGB = rgbToXYZ.Inverse()

// The curve. A lamp level is not light: the standard spends more of its
// numbers on the dark end, where eyes can tell shades apart, by a
// straight piece near zero and a power of 2.4 above it. These four
// constants are the standard's.
func toLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// fromLinear is the srgb curve from linear light to a lamp's value.
func fromLinear(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

// gamutSlack is how far outside 0..1, in linear light, a channel may fall and
// still count as in gamut: a display's darkest step is about 3e-4 linear, so a
// hair less than that is clipped without comment.
//
// At 1e-6 a dark blue whose red channel hovers just under zero across a band of
// chroma is called out, and a fit walks it back by a sixth.
const gamutSlack = 1e-4

// FromSwatch is the swatch as lamp levels, and whether the lamps can make it.
// Out of gamut, the levels are clipped to 0..1 and inGamut is false: the
// nearest thing the lamps can do, and an honest word that it is not the same
// colour.
//
// The tolerance is a millionth, so a colour on the edge of the triangle, like a
// primary at full, counts as in.
func FromSwatch(s swatch.Swatch) (c RGB, inGamut bool) {
	r, g, b := xyzToRGB.Apply(s.XYZ())
	inGamut = true
	clip := func(v float64) float64 {
		if v < -gamutSlack || v > 1+gamutSlack {
			inGamut = false
		}
		return math.Max(0, math.Min(1, v))
	}
	return RGB{fromLinear(clip(r)), fromLinear(clip(g)),
		fromLinear(clip(b))}, inGamut
}

// Swatch stores the lamp levels back as a swatch.
func (c RGB) Swatch() swatch.Swatch {
	return swatch.FromXYZ(rgbToXYZ.Apply(toLinear(c.R), toLinear(c.G),
		toLinear(c.B)))
}

// Bytes is the lamps as the three bytes a terminal wants.
func (c RGB) Bytes() (r, g, b uint8) {
	round := func(v float64) uint8 {
		return uint8(math.Round(math.Max(0,
			math.Min(1, v)) * 255))
	}
	return round(c.R), round(c.G), round(c.B)
}

// Hex is the lamps as "#rrggbb".
func (c RGB) Hex() string {
	r, g, b := c.Bytes()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// MustHex is FromHex for a literal you know is well formed.
func MustHex(h string) RGB {
	c, err := FromHex(h)
	if err != nil {
		panic(err)
	}
	return c
}

// FromHex reads "#rrggbb" or "rrggbb", which is what every hex code you
// have ever typed silently was: three lamp levels in this space. The
// short form "#rgb" is each digit doubled, as css has always read it.
func FromHex(h string) (RGB, error) {
	h = strings.TrimPrefix(strings.TrimSpace(h), "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return RGB{}, fmt.Errorf("hex colour wants three or six "+
			"digits, not %q", h)
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b); err != nil {
		return RGB{}, fmt.Errorf("hex colour %q: %v", h, err)
	}
	return RGB{float64(r) / 255, float64(g) / 255, float64(b) / 255}, nil
}

// The lamps at full, and all three at once, which the derived matrix
// makes come out as exactly the white.
var (
	Red   = RGB{1, 0, 0}
	Green = RGB{0, 1, 0}
	Blue  = RGB{0, 0, 1}
	White = RGB{1, 1, 1}
	Black = RGB{0, 0, 0}
)

// Mix is the swatch t of the way from a to b along a straight line
// through the lamps: what every tool that blends hex codes does, and the
// line that goes through mud. Here so a ramp can choose it on purpose,
// and so the difference can be drawn next to oklab's.
var Mix = functions.Mixer{Name: "srgb", Mix: func(a, b swatch.Swatch,
	t float64) swatch.Swatch {
	p, _ := FromSwatch(a)
	q, _ := FromSwatch(b)
	return RGB{p.R + (q.R-p.R)*t, p.G + (q.G-p.G)*t, p.B +
		(q.B-p.B)*t}.Swatch()
}}

// hueAndRange is the arithmetic the two cylinders share: which lamp is
// brightest decides the sixth of the wheel, and the spread between the
// brightest and dimmest lamp is the colourfulness.
func (c RGB) hueAndRange() (h, max, min float64) {
	max = math.Max(c.R, math.Max(c.G, c.B))
	min = math.Min(c.R, math.Min(c.G, c.B))
	d := max - min
	if d < 1e-12 {
		return 0, max, min
	}
	switch max {
	case c.R:
		h = math.Mod((c.G-c.B)/d, 6)
	case c.G:
		h = (c.B-c.R)/d + 2
	default:
		h = (c.R-c.G)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, max, min
}

// HSL is the lamps as hue, saturation, lightness.
func (c RGB) HSL() HSL {
	h, max, min := c.hueAndRange()
	l := (max + min) / 2
	var s float64
	if d := max - min; d > 1e-12 {
		s = d / (1 - math.Abs(2*l-1))
	}
	return HSL{h, s, l}
}

// HSV is the lamps as hue, saturation, value.
func (c RGB) HSV() HSV {
	h, max, min := c.hueAndRange()
	var s float64
	if max > 1e-12 {
		s = (max - min) / max
	}
	return HSV{h, s, max}
}

// fromHue is the shared reverse: a hue, a colourfulness and a floor
// become three lamps.
func fromHue(h, chroma, m float64) RGB {
	h = math.Mod(math.Mod(h, 360)+360, 360) / 60
	x := chroma * (1 - math.Abs(math.Mod(h, 2)-1))
	var r, g, b float64
	switch {
	case h < 1:
		r, g = chroma, x
	case h < 2:
		r, g = x, chroma
	case h < 3:
		g, b = chroma, x
	case h < 4:
		g, b = x, chroma
	case h < 5:
		r, b = x, chroma
	default:
		r, b = chroma, x
	}
	return RGB{r + m, g + m, b + m}
}

// RGB is the cylinder back as lamps.
func (c HSL) RGB() RGB {
	chroma := (1 - math.Abs(2*c.L-1)) * c.S
	return fromHue(c.H, chroma, c.L-chroma/2)
}

// RGB is the cylinder back as lamps.
func (c HSV) RGB() RGB {
	chroma := c.V * c.S
	return fromHue(c.H, chroma, c.V-chroma)
}

// RGB8 is a colour from the 0-255 bytes a person or a file usually has.
func RGB8(r, g, b uint8) RGB {
	return RGB{float64(r) / 255, float64(g) / 255, float64(b) / 255}
}

// RGBA makes an RGB an image/color.Color, so it can be handed to
// anything that draws without a conversion at every call.
func (c RGB) RGBA() (r, g, b, a uint32) {
	q := func(v float64) uint32 {
		return uint32(math.Round(math.Max(0, math.Min(1, v)) * 0xffff))
	}
	return q(c.R), q(c.G), q(c.B), 0xffff
}

// Equal is whether two colours are the same to within a byte per
// channel, which is the resolution anything downstream can show.
func (c RGB) Equal(o RGB) bool {
	r1, g1, b1 := c.Bytes()
	r2, g2, b2 := o.Bytes()
	return r1 == r2 && g1 == g2 && b1 == b2
}

// In is the display's gamut as a rule a fit or a picker can ask: whether
// a swatch is one this space can show.
func In(s swatch.Swatch) bool {
	_, in := FromSwatch(s)
	return in
}
