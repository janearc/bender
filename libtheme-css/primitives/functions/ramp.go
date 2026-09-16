// Package functions holds the things that turn a number into a swatch. The name
// is general on purpose: a ramp is a function of one number, t from 0 to 1, and
// there may be others. They are built from swatches and a way of blending them,
// and nothing else, which is why they sit under primitives and import no space.
//
// The seam. Where t comes from is not this library's business. A ramp answers
// "what colour is t" and never asks "which cell is this, how far from the
// centre, what time is it". Those are geometry, and geometry belongs to
// whatever is drawing: a canvas, a strip of lamps, a bar of hours.
//
// It hands the ramp a t; the ramp hands back a swatch. That is the whole
// contract, and it is what lets a sun, a sky and a stripe share one ramp
// without this library knowing what a sun is. This library is containers, not
// form.
package functions

import (
	"fmt"
	"sort"
	"strings"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// Mixer is a way of blending two swatches: the colour t of the way from a to b,
// 0 giving a and 1 giving b. Each space supplies its own, and they differ: a
// straight line in oklab looks straight, a straight line through the lamps goes
// through mud. The name is what CSS writes after "in": oklab, srgb.
type Mixer struct {
	Name string
	Mix  func(a, b swatch.Swatch, t float64) swatch.Swatch
}

// Stop is a swatch at a position along the ramp, 0 to 1.
type Stop struct {
	At     float64
	Swatch swatch.Swatch
}

// Ramp is a function from t in 0..1 to a swatch: stops, and a mixer that
// draws the line between neighbouring stops. Two stops is the common
// case and is not special. Before the first stop the ramp is the first
// stop's colour; after the last it is the last's.
type Ramp struct {
	Stops []Stop
	In    Mixer
}

// Even is a ramp with its stops spaced evenly from 0 to 1, in the order
// given, which is what a list of colours with no positions means in CSS
// and in daffy. One swatch is a ramp that is that swatch everywhere.
func Even(in Mixer, swatches ...swatch.Swatch) Ramp {
	stops := make([]Stop, len(swatches))
	for i, s := range swatches {
		at := 0.0
		if len(swatches) > 1 {
			at = float64(i) / float64(len(swatches)-1)
		}
		stops[i] = Stop{at, s}
	}
	return Ramp{stops, in}
}

// New is a ramp from explicit stops, sorted by position.
func New(in Mixer, stops ...Stop) Ramp {
	s := append([]Stop(nil), stops...)
	sort.SliceStable(s, func(i, j int) bool { return s[i].At < s[j].At })
	return Ramp{s, in}
}

// At is the swatch t of the way along the ramp.
func (r Ramp) At(t float64) swatch.Swatch {
	if len(r.Stops) == 0 {
		return swatch.Black
	}
	if t <= r.Stops[0].At {
		return r.Stops[0].Swatch
	}
	last := r.Stops[len(r.Stops)-1]
	if t >= last.At {
		return last.Swatch
	}
	for i := 1; i < len(r.Stops); i++ {
		a, b := r.Stops[i-1], r.Stops[i]
		if t <= b.At {
			span := b.At - a.At
			if span <= 0 {
				return b.Swatch
			}
			return r.In.Mix(a.Swatch, b.Swatch, (t-a.At)/span)
		}
	}
	return last.Swatch
}

// Samples is the ramp taken at n points from 0 to 1 inclusive: what a
// strip with n lamps, a bar with n cells, or a palette with n entries
// gets. One sample is the ramp at 0.
func (r Ramp) Samples(n int) []swatch.Swatch {
	if n <= 0 {
		return nil
	}
	out := make([]swatch.Swatch, n)
	for i := range out {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		out[i] = r.At(t)
	}
	return out
}

// String is the ramp as CSS says it, given a way to write each stop:
// linear-gradient(in oklab, #160d2b 0%, #ffa2ff 100%). The stop writer
// is a parameter because the swatch does not know its own hex; the
// space that writes it does.
func (r Ramp) String(stop func(swatch.Swatch) string) string {
	parts := make([]string, len(r.Stops))
	for i, s := range r.Stops {
		parts[i] = fmt.Sprintf("%s %.0f%%", stop(s.Swatch), s.At*100)
	}
	return fmt.Sprintf("linear-gradient(in %s, %s)", r.In.Name,
		strings.Join(parts, ", "))
}
