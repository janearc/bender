package render_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"

	"github.com/janearc/libdaffy/render"
)

// red is a painted cell, for the tests.
func red(glyph string) render.Paint {
	return render.Paint{Glyph: glyph, Fg: srgb.RGB8(255, 0, 0), Set: true}
}

// the first frame paints every cell inside one synchronized bracket; the
// same frame again sends nothing; one changed cell sends one cursor move
// and that cell; invalidating repaints everything.
func TestWriterSendsOnlyWhatChanged(t *testing.T) {
	var out bytes.Buffer
	w := render.NewWriter(&out)
	f := render.NewFrame(3, 2)
	f.Set(0, 0, red("a"))
	f.Set(2, 1, red("b"))
	if err := w.Write(f); err != nil {
		t.Fatal(err)
	}
	first := out.String()
	if !strings.HasPrefix(first, "\x1b[?2026h") ||
		!strings.HasSuffix(first, "\x1b[0m\x1b[?2026l") {
		t.Fatalf("no bracket: %q", first)
	}
	if strings.Count(first, "\x1b[1;1H") != 1 ||
		strings.Count(first, "\x1b[2;1H") != 1 ||
		strings.Count(first, "38;2;255;0;0m") != 2 {
		t.Fatalf("first frame: %q", first)
	}
	out.Reset()
	w.Write(f)
	if out.Len() != 0 {
		t.Fatalf("an unchanged frame sent %q", out.String())
	}
	g := f.Clone()
	g.Set(1, 1, red("c"))
	w.Write(g)
	second := out.String()
	if strings.Count(second, "H") != 1 ||
		!strings.Contains(second, "\x1b[2;2H") ||
		strings.Contains(second, "a") {
		t.Fatalf("one changed cell: %q", second)
	}
	out.Reset()
	w.Invalidate()
	w.Write(g)
	if !strings.Contains(out.String(), "\x1b[1;1H") {
		t.Fatalf("invalidate did not repaint: %q", out.String())
	}
}

// an origin moves every cursor address; an unpainted cell that changed is
// written as a reset space.
func TestWriterOriginAndBlanks(t *testing.T) {
	var out bytes.Buffer
	w := render.NewWriter(&out)
	w.Origin.X, w.Origin.Y = 10, 5
	f := render.NewFrame(2, 1)
	f.Set(0, 0, red("x"))
	w.Write(f)
	if !strings.Contains(out.String(), "\x1b[6;11H") {
		t.Fatalf("origin: %q", out.String())
	}
	out.Reset()
	g := render.NewFrame(2, 1)
	w.Write(g)
	if !strings.Contains(out.String(), "\x1b[6;11H ") {
		t.Fatalf("a cleared cell: %q", out.String())
	}
}

// text and ansi say the same frame two ways.
func TestTextAndANSI(t *testing.T) {
	f := render.NewFrame(3, 1)
	f.Set(0, 0, red("a"))
	f.Set(1, 0, red("b"))
	if got := render.Text(f); got != "ab\n" {
		t.Fatalf("text %q", got)
	}
	got := render.ANSI(f)
	if strings.Count(got, "38;2;255;0;0m") != 1 ||
		!strings.HasSuffix(got, "b\x1b[0m \x1b[0m\n") {
		t.Fatalf("ansi %q", got)
	}
}

// colour state carries across runs: a blank after a coloured run in an
// earlier row gets a reset, and a coloured cell after a blank sets its
// colour again.
func TestWriterCarriesStateAcrossRuns(t *testing.T) {
	var out bytes.Buffer
	w := render.NewWriter(&out)
	f := render.NewFrame(2, 2)
	f.Set(0, 0, red("a"))
	f.Set(1, 1, red("b"))
	w.Write(f)
	g := f.Clone()
	g.Set(0, 0, red("c"))
	g.Set(1, 1, render.Paint{})
	out.Reset()
	w.Write(g)
	s := out.String()
	if !strings.Contains(s, "\x1b[49mc") ||
		!strings.Contains(s, "\x1b[2;2H\x1b[0m ") {
		t.Fatalf("state across runs: %q", s)
	}
	if got := render.Text(func() render.Frame {
		h := render.NewFrame(4, 1)
		h.Set(0, 0, red("x"))
		return h
	}()); got != "x\n" {
		t.Fatalf("trailing spaces kept: %q", got)
	}
}
