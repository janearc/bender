package canvas_test

import (
	"github.com/janearc/libdaffy/canvas"
	"github.com/janearc/libdaffy/document"
	"github.com/janearc/libdaffy/part"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"image"
	"testing"
)

// a mask sets, counts, grows, shrinks and inverts, and stays in bounds.
func TestMaskOps(t *testing.T) {
	m := canvas.NewMask(6, 6)
	if !m.Empty() || m.Width() != 6 || m.Height() != 6 {
		t.Fatal("new")
	}
	m.Set(2, 2, true)
	m.Plot(3, 2)
	m.Set(-1, 0, true)
	m.Set(9, 9, true)
	if m.Count() != 2 || !m.At(2, 2) || m.At(-1, 0) || m.At(9, 9) {
		t.Fatal("set/at")
	}
	if b, want := m.Bounds(), (image.Rect(2, 2, 4, 3)); b != want {
		t.Fatalf("bounds %v", b)
	}
	if empty := (image.Rectangle{}); canvas.NewMask(2, 2).
		Bounds() !=
		empty {
		t.Fatal("empty bounds")
	}
	g := m.Grow(1)
	if g.Count() != 2+2+2+1+1 || !g.At(2, 1) || !g.At(1, 2) ||
		!g.At(4, 2) ||
		g.At(1, 1) {
		t.Fatalf("grow:\n%v", g.Count())
	}
	s := g.Shrink(1)
	if !s.Equal(m) {
		t.Fatal("shrink of grow should give the original " +
			"for a convex blob")
	}
	if !m.Shrink(1).Empty() {
		t.Fatal("shrinking a thin mask empties it")
	}
	inv := m.Clone()
	inv.Invert()
	if inv.Count() != 34 || inv.At(2, 2) {
		t.Fatal("invert")
	}
	inv.Clear()
	if !inv.Empty() {
		t.Fatal("clear")
	}
	if m.Equal(canvas.NewMask(5, 5)) || !m.Equal(m.Clone()) {
		t.Fatal("equal")
	}
	if len(m.Bits()) != 36 {
		t.Fatal("bits")
	}
	// The canvas edge counts as out for shrink.
	full := canvas.FullMask(4, 4)
	if full.Count() != 16 || full.Shrink(1).Count() != 4 {
		t.Fatalf("edge shrink %d", full.Shrink(1).Count())
	}
	r := canvas.MaskFromRect(4, 4, image.Rect(-1, 1, 2, 9))
	if r.Count() != 6 || !r.At(0, 1) || r.At(2, 1) {
		t.Fatal("from rect")
	}
	c := canvas.NewCanvas(4, 4)
	c.Set(1, 1, 5)
	c.Set(3, 3, 5)
	c.Set(0, 0, 6)
	p := canvas.MaskFromPart(c, 5)
	if p.Count() != 2 || !p.At(3, 3) || p.At(0, 0) {
		t.Fatal("from part")
	}
	th := canvas.NewMask(2, 2)
	th.Threshold([]float64{0, 0.4, 0.5, 1}, 0.5)
	if th.Count() != 2 || th.At(1, 0) || !th.At(0, 1) {
		t.Fatal("threshold")
	}
	th.Threshold([]float64{1}, 0.5)
	if !th.At(0, 0) {
		t.Fatal("short threshold input")
	}
}

// Every drawing operation is clipped by the active mask.
func TestMaskClipsEveryOperation(t *testing.T) {
	d := document.NewDocument("t", 10, 10)
	p, _ := d.Parts.Add("p", part.Literal(srgb.RGB{}.Swatch()))
	d.Mask = canvas.MaskFromRect(10, 10, image.Rect(2, 2, 6, 6))
	ops := map[string]func(s *document.Stroke){
		"pencil": func(s *document.Stroke) {
			s.Plot(0, 0)
			s.Plot(3, 3)
		},
		"line": func(s *document.Stroke) {
			canvas.Line(s, 0, 0, 9, 9)
		},
		"rect": func(s *document.Stroke) {
			canvas.RectOutline(s, 0, 0, 5, 5)
		},
		"rectfil": func(s *document.Stroke) {
			canvas.RectFill(s, 0, 0, 9, 9)
		},
		"ellipse": func(s *document.Stroke) {
			canvas.EllipseOutline(s, 1, 1, 8, 8)
		},
		"ellfill": func(s *document.Stroke) {
			canvas.EllipseFill(s, 0, 0, 9, 9)
		},
		"flood": func(s *document.Stroke) {
			canvas.FloodFill(
				s,
				d.Canvas.Bounds(),
				0,
				0,
				func(int, int) bool { return true },
			)
		},
		"plotpart": func(s *document.Stroke) {
			s.PlotPart(0, 0, p.ID)
			s.PlotPart(4, 4, p.ID)
		},
	}
	for name, op := range ops {
		s := d.Stroke(p.ID)
		op(s)
		for y := 0; y < 10; y++ {
			for x := 0; x < 10; x++ {
				if d.Canvas.At(x, y) != part.None &&
					!d.Mask.At(x, y) {
					t.Fatalf(
						"%s painted %d,%d outside mask",
						name, x, y,
					)
				}
			}
		}
		if d.Canvas.Count(p.ID) == 0 {
			t.Fatalf("%s painted nothing inside the mask", name)
		}
		d.Canvas.Clear()
	}
	// The stroke's own clip stacks on top of the mask.
	s := d.Stroke(p.ID)
	s.Clip = image.Rect(0, 0, 4, 4)
	canvas.RectFill(s, 0, 0, 9, 9)
	if d.Canvas.Count(p.ID) != 4 {
		t.Fatalf(
			"clip and mask together should leave 4, got %d",
			d.Canvas.Count(p.ID),
		)
	}
	// Erase is a stroke of None and is clipped the same way.
	d.Mask = nil
	s = d.Stroke(p.ID)
	canvas.RectFill(s, 0, 0, 9, 9)
	d.Mask = canvas.MaskFromRect(10, 10, image.Rect(0, 0, 5, 10))
	s = d.Stroke(part.None)
	canvas.RectFill(s, 0, 0, 9, 9)
	if d.Canvas.Count(p.ID) != 50 {
		t.Fatalf(
			"erase should be clipped: %d left",
			d.Canvas.Count(p.ID),
		)
	}
}
