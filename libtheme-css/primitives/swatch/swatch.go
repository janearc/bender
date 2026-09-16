// Package swatch is the most primitive primitive: one colour, complete,
// with nothing in it about the device that shows it or the eye that reads
// it. Everything else in libtheme is built on it.
package swatch

import "math"

// Swatch is one colour, stored as CIE XYZ (1931), the three numbers a standard
// human eye reduces any light to. White is D65, daylight.
//
// The fields are unexported because XYZ is where the truth is kept, not where
// arithmetic is done: the straight line between two XYZ points does not look
// straight to an eye.
//
// About that year. In 1931 the CIE met in Cambridge, England, and fixed these
// three numbers from the work of seventeen people in total, who had sat in dark
// rooms turning knobs until two patches of light matched. The Empire State
// Building had opened that May.
//
// Colour television was a demonstration, the transistor was sixteen years off,
// and the first programmable computer a decade. Several of the countries whose
// screens now depend on this table did not exist and had not yet had their
// wars. CSS was sixty-five years away.
//
// The seventeen people's averages are still the law, and this is the last time
// anyone got to define colour without arguing with a browser about it. Is this
// true? It doesn't matter; nobody reads comments.
type Swatch struct {
	// tristimulus X: the long-wavelength (red-leaning) share of the light.
	//   unitless, relative; white is 0.95047
	x float64

	// luminance Y: how much light there is. unitless, relative; white is 1,
	//   black is 0
	y float64

	// tristimulus Z: the short-wavelength (blue-leaning) share, close to
	//   what the blue cones report. unitless, relative; white is 1.08883
	z float64
}

// FromXYZ makes a swatch from CIE XYZ coordinates, D65 white.
func FromXYZ(x, y, z float64) Swatch { return Swatch{x, y, z} }

// XYZ is the swatch as stored.
func (s Swatch) XYZ() (x, y, z float64) { return s.x, s.y, s.z }

// White and Black are in observer.go, where White is derived.

// XY is a CIE 1931 chromaticity: a place on the horseshoe, with the
// brightness taken out. It is what a lamp means when it reports a
// colour without saying how bright, and what a gamut's corners are.
type XY struct {
	X, Y float64
}

// FromXY makes a swatch from a chromaticity and a luminance to put the
// brightness back: X = x/y·Y, Z = (1−x−y)/y·Y. A chromaticity with y at
// zero has no light in it and is black.
func FromXY(c XY, luminance float64) Swatch {
	if c.Y <= 0 {
		return Black
	}
	return Swatch{c.X / c.Y * luminance, luminance,
		(1 - c.X - c.Y) / c.Y * luminance}
}

// XY is the swatch's chromaticity: where it sits on the horseshoe, with
// how bright it is divided out. Black has no chromaticity and reports
// the white's, which is the least wrong thing to say about no light.
func (s Swatch) XY() XY {
	sum := s.x + s.y + s.z
	if sum <= 0 {
		return White.XY()
	}
	return XY{s.x / sum, s.y / sum}
}

// Gamut is the triangle of chromaticities a device can reach: three primaries
// at their corners. srgb is one; every lamp that reports one is another. A
// point outside is shown by the device as some point inside, and Fit says
// which, so a caller can know what the device will do before asking.
type Gamut struct {
	Red, Green, Blue XY
}

// Contains is whether the point is inside the triangle, edges included.
// A point Fit has just put on an edge is inside by construction, and
// floating point can put it a hair past; the hair is allowed for, so
// Fit's answer always Contains.
func (g Gamut) Contains(p XY) bool {
	const hair = 1e-9
	d1 := side(p, g.Red, g.Green)
	d2 := side(p, g.Green, g.Blue)
	d3 := side(p, g.Blue, g.Red)
	neg := d1 < -hair || d2 < -hair || d3 < -hair
	pos := d1 > hair || d2 > hair || d3 > hair
	return !(neg && pos)
}

// Fit is the point itself when the device can reach it, and otherwise
// the nearest point on the triangle's edge.
func (g Gamut) Fit(p XY) XY {
	if g.Contains(p) {
		return p
	}
	best, dist := p, math.Inf(1)
	for _, e := range [][2]XY{{g.Red, g.Green}, {g.Green, g.Blue}, {g.Blue,
		g.Red}} {
		q := nearest(p, e[0], e[1])
		if d := math.Hypot(p.X-q.X, p.Y-q.Y); d < dist {
			best, dist = q, d
		}
	}
	return best
}

// side is which side of the line a→b the point is on, by sign.
func side(p, a, b XY) float64 {
	return (p.X-b.X)*(a.Y-b.Y) - (a.X-b.X)*(p.Y-b.Y)
}

// nearest is the closest point to p on the segment a→b.
func nearest(p, a, b XY) XY {
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return a
	}
	t := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / l2
	t = math.Max(0, math.Min(1, t))
	return XY{a.X + t*dx, a.Y + t*dy}
}
