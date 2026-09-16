package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"

	"github.com/janearc/angry-dist/kingfisher"
	"github.com/janearc/angry-dist/textmap"
)

// aim is where the middle of the view sits, in degrees. A nil aim means fit to
// the data, which is what every caller wanted before there was a way to say
// otherwise.
//
// The operator's word for it is "look", because centre and center are one word
// spelled two ways and nobody who has lived on both sides of the Atlantic can
// keep them straight.
type aim struct{ lon, lat float64 }

// parseLook turns a lon,lat flag into an aim, or nil when it was not
// given. Parsing at the command line rather than at the render keeps a bad
// value an error the operator sees instead of a frame that is quietly
// somewhere else.
func parseLook(s string) (*aim, error) {
	if s == "" {
		return nil, nil
	}
	var c aim
	if _, err := fmt.Sscanf(s, "%f,%f", &c.lon, &c.lat); err != nil {
		return nil, fmt.Errorf("-look wants lon,lat: %v", err)
	}
	// Both projections clamp latitude themselves, so this is not about
	// safety: it is so that a typo is reported rather than silently drawn
	// from somewhere near the pole.
	if c.lon < -180 || c.lon > 180 || c.lat < -90 || c.lat > 90 {
		return nil, fmt.Errorf(
			"-look out of range: lon %.3f, lat %.3f",
			c.lon,
			c.lat,
		)
	}
	return &c, nil
}

// frameSize derives each dimension only when it was not given, so -cols on
// its own is honoured rather than discarded with the missing -rows.
func frameSize(cols, rows int) (int, int) {
	tc, tr := 120, 40
	if c, r, err := term.GetSize(os.Stdout.Fd()); err == nil && c > 0 &&
		r > 3 {
		tc, tr = c, r-3
	}
	if cols <= 0 {
		cols = tc
	}
	if rows <= 0 {
		rows = tr
	}
	return cols, rows
}

// frameOnce draws the fitted view a single time as 24-bit half blocks, so
// the picture can be looked at anywhere a terminal shows colour, tmux
// included. No mouse, no alternate screen: the demo without the
// interaction, and the way to check the projection without a window.
func frameOnce(
	w io.Writer,
	src *textmap.CellSource,
	m *kingfisher.Manifest,
	layer string,
	globe bool,
	cols, rows int,
	span float64,
	c *aim,
	st style,
) error {
	cols, rows = frameSize(cols, rows)
	a := newFrameApp(src, m, layer, globe, cols, rows)
	a.style = st
	if span > 0 {
		a.setScale(span)
	}
	// reset() fitted the view to the data, which is why every -bbox gave
	// the same picture in -dataset mode. An explicit centre overrides that
	// fit and is what makes the ball turnable frame by frame.
	if c != nil {
		a.setCentre(c.lon, c.lat)
	}
	return printFrame(w, a, layer, cols, rows)
}

// newFrameApp builds the app the frame path draws through, fitted to the
// data. It is separate from drawing so the repl can hold one app across
// many frames and mutate it between them, rather than rebuilding a view
// the operator just aimed.
func newFrameApp(
	src *textmap.CellSource,
	m *kingfisher.Manifest,
	layer string,
	globe bool,
	cols, rows int,
) *app {
	a := &app{
		src:   src,
		cols:  cols,
		rows:  rows + chromeRows,
		globe: globe,
		style: defaultStyle(),
	}
	a.layers = src.Layers()
	for i, l := range a.layers {
		if l == layer {
			a.layer = i
		}
	}
	a.reset(m)
	return a
}

// printFrame renders one frame of an app that is already aimed. Every
// caller draws through here, so the format cannot drift between the
// one-shot flag and the repl: a decoder written against one reads both.
func printFrame(w io.Writer, a *app, layer string, cols, rows int) error {
	src := a.src
	f := textmap.RenderWith(
		src,
		a.proj,
		layer,
		a.res(),
		textmap.Options{Scale: a.style.scale},
	)
	lon, lat := a.proj.Centre()
	view := a.viewName()
	fmt.Fprintf(
		w,
		"%s  %s  res %d of %d  %s  centre %.3f, %.3f  span %.3f deg  "+
			"%d of %d pixels\n",
		src.Name,
		layer,
		f.Res,
		src.NativeRes(),
		view,
		lon,
		lat,
		a.proj.Scale(),
		f.Filled,
		f.W*f.H,
	)
	var b strings.Builder
	for row := 0; row < rows; row++ {
		b.Reset()
		lastFg, lastBg := "", ""
		for x := 0; x < cols; x++ {
			top, okT := frameShade(f.At(x, row*2), a.style.palette)
			bot, okB := frameShade(
				f.At(x, row*2+1),
				a.style.palette,
			)
			if !okT && !okB {
				if lastFg != "" || lastBg != "" {
					b.WriteString("\x1b[0m")
					lastFg, lastBg = "", ""
				}
				b.WriteByte(' ')
				continue
			}
			var glyph string
			var fg, bg string
			switch {
			case okT && okB:
				glyph, fg, bg = "▀", sgr(38, top), sgr(48, bot)
			case okT:
				glyph, fg, bg = "▀", sgr(38, top), "\x1b[49m"
			default:
				glyph, fg, bg = "▄", sgr(38, bot), "\x1b[49m"
			}
			if fg != lastFg {
				b.WriteString(fg)
				lastFg = fg
			}
			if bg != lastBg {
				b.WriteString(bg)
				lastBg = bg
			}
			b.WriteString(glyph)
		}
		b.WriteString("\x1b[0m\n")
		if _, err := io.WriteString(w, b.String()); err != nil {
			return err
		}
	}
	fmt.Fprintf(
		w,
		"values %.3f to %.3f%s; blank is unsurveyed\n",
		f.Lo,
		f.Hi,
		a.style.note(),
	)
	return nil
}

// frameShade is a pixel's colour: the palette for data, the tint for
// unsurveyed planet, nothing for space.
func frameShade(p textmap.Pixel, pal textmap.Palette) (textmap.RGB, bool) {
	switch {
	case p.Set:
		return pal.At(p.T), true
	case p.Planet && p.Grid:
		return gridTint, true
	case p.Planet:
		return planetTint, true
	case p.Glow > 0:
		return textmap.Atmosphere(p.Glow), true
	}
	return textmap.RGB{}, false
}

// sgr is one true-colour escape, foreground or background by kind, written
// by hand rather than through a library: a frame is thousands of these and
// the allocation shows.
func sgr(kind int, c textmap.RGB) string {
	return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", kind, c.R, c.G, c.B)
}
