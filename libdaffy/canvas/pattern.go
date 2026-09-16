package canvas

import (
	"github.com/janearc/libdaffy/part"
	"math/bits"
)

// Fill patterns. A flat fill is the boring case and it is not enough. Every
// pattern operates on the pixel grid, never on characters: a texture made of
// shade glyphs breaks the pixel model and stops being editable.

// PatternKind names a two-part texture.
type PatternKind int

// The kinds.
const (
	Flat PatternKind = iota
	Dither4
	Dither8
	Stipple
	Checker
	Hatch
)

// String names the kind.
func (k PatternKind) String() string {
	switch k {
	case Dither4:
		return "dither 4x4"
	case Dither8:
		return "dither 8x8"
	case Stipple:
		return "stipple"
	case Checker:
		return "checker"
	case Hatch:
		return "hatch"
	}
	return "flat"
}

// HatchStyle is which way the rules run.
type HatchStyle int

// The hatch styles.
const (
	Diagonal HatchStyle = iota
	Horizontal
	Vertical
)

// String names the style.
func (h HatchStyle) String() string {
	switch h {
	case Horizontal:
		return "horizontal"
	case Vertical:
		return "vertical"
	}
	return "diagonal"
}

// Pattern mixes two parts, A and B, across the pixel grid.
type Pattern struct {
	Kind PatternKind
	// Mix is how much B, 0-1, for dither and stipple.
	Mix float64
	// Period is the cell of a checker or the spacing of a hatch, in pixels,
	// at least one.
	Period int
	// Weight is how many pixels thick a hatch rule is, at least one.
	Weight int
	// Style is the hatch direction.
	Style HatchStyle
	// Seed makes stipple noise stable but different between fills.
	Seed uint32
}

// bayer is the threshold at x, y in the ordered-dither matrix of side 1<<n,
// thresholds 0 to side*side-1: the bits of x^y and y interleaved, low bit
// first.
//
// That is the recursion that builds each Bayer matrix from the one below it,
// written flat, so no matrix is stored and any power-of-two side is the same
// three lines.
func bayer(x, y, n int) int {
	v := 0
	for i := 0; i < n; i++ {
		v = v<<1 | ((x^y)>>i)&1
		v = v<<1 | (y>>i)&1
	}
	return v
}

// Threshold is the ordered-dither threshold at a pixel, 0-1, for a matrix
// of the given side. The side is rounded down to a power of two, at least
// two, so 4 and 8 are themselves and 5 is 4.
func Threshold(x, y, size int) float64 {
	n := bits.Len(uint(size)) - 1
	if n < 1 {
		n = 1
	}
	side := 1 << uint(n)
	x, y = floorMod(x, side), floorMod(y, side)
	return (float64(bayer(x, y, n)) + 0.5) / float64(side*side)
}

// Noise is stable per-pixel noise, 0-1, from a small integer hash. The same
// pixel and seed always give the same value, so a stipple does not crawl when
// it is repainted.
func Noise(x, y int, seed uint32) float64 {
	h := uint32(x)*0x9E3779B1 ^ uint32(y)*0x85EBCA77 ^ seed*0xC2B2AE3D
	h ^= h >> 15
	h *= 0x2C1B3C6D
	h ^= h >> 12
	h *= 0x297A2D39
	h ^= h >> 15
	return (float64(h&0xffffff) + 0.5) / float64(1<<24)
}

// UseB reports whether a pixel takes part B rather than A.
func (p Pattern) UseB(x, y int) bool {
	period := p.Period
	if period < 1 {
		period = 1
	}
	switch p.Kind {
	case Dither4:
		return Threshold(x, y, 4) < p.Mix
	case Dither8:
		return Threshold(x, y, 8) < p.Mix
	case Stipple:
		return Noise(x, y, p.Seed) < p.Mix
	case Checker:
		return (floorDiv(x, period)+floorDiv(y, period))%2 != 0
	case Hatch:
		weight := p.Weight
		if weight < 1 {
			weight = 1
		}
		var v int
		switch p.Style {
		case Horizontal:
			v = y
		case Vertical:
			v = x
		default:
			v = x + y
		}
		return floorMod(v, period) < weight
	}
	return false
}

// PartAt is the part a pattern paints at a pixel, given its two parts.
func (p Pattern) PartAt(x, y int, a, b part.PartID) part.PartID {
	if p.UseB(x, y) {
		return b
	}
	return a
}
