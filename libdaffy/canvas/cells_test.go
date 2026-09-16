package canvas_test

import (
	"github.com/janearc/libdaffy/canvas"
	"github.com/janearc/libdaffy/document"
	"github.com/janearc/libdaffy/part"
	"github.com/janearc/libdaffy/render"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"image"
	"strings"
	"testing"
)

// cells hold glyphs by position, in bounds only, and count what they hold.
func TestCellsBasics(t *testing.T) {
	c := canvas.NewCells(0, 0)
	if c.Width() != 1 || c.Height() != 1 {
		t.Error("floor size")
	}
	c = canvas.NewCells(6, 3)
	g := canvas.Glyph{Ch: "x", Fg: 1}
	if !c.Set(1, 1, g) || c.Set(1, 1, g) || c.Set(9, 9, g) {
		t.Error("Set")
	}
	if c.At(1, 1) != g || !c.At(-1, 0).Empty() || c.Count() != 1 {
		t.Error("At/Count")
	}
	if n := c.PutString(4, 0, "abcd", 1, 2); n != 2 ||
		c.At(5, 0).Ch != "b" ||
		c.At(5, 0).Bg != 2 {
		t.Errorf("PutString stops at the edge: %d", n)
	}
	c.PutBlock(0, 1, "ab\n d", 1, part.None)
	if c.At(0, 1).Ch != "a" || c.At(0, 2).Ch != " " ||
		c.At(1, 2).Ch != "d" {
		t.Error("PutBlock")
	}
	cl := c.Clone()
	if !cl.Equal(c) || cl.Equal(canvas.NewCells(6, 3)) ||
		cl.Equal(canvas.NewCells(1, 1)) {
		t.Error("Clone/Equal")
	}
	r := c.Resize(3, 2)
	if r.At(0, 1).Ch != "a" || !r.At(5, 0).Empty() {
		t.Error("Resize")
	}
	if len(c.Glyphs()) != 18 {
		t.Error("Glyphs")
	}
	c.Clear()
	if c.Count() != 0 {
		t.Error("Clear")
	}
}

// wrapping breaks on spaces and keeps words whole where the width allows.
func TestWrap(t *testing.T) {
	cases := []struct {
		in   string
		w    int
		want []string
	}{
		{"", 5, []string{""}},
		{"one two three", 7, []string{"one two", "three"}},
		{"one two three", 3, []string{"one", "two", "thr", "ee"}},
		{"abcdefgh", 3, []string{"abc", "def", "gh"}},
		{"a bcdefg", 3, []string{"a", "bcd", "efg"}},
		{"  spaced   out  ", 20, []string{"  spaced   out  "}},
	}
	for _, c := range cases {
		got := canvas.Wrap(c.in, c.w)
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf(
				"canvas.Wrap(%q, %d) = %q, want %q",
				c.in,
				c.w,
				got,
				c.want,
			)
		}
	}
}

// A text box reflows when its text or width changes, and honours alignment.
func TestTextBoxReflows(t *testing.T) {
	b := canvas.TextBox{
		Box:  image.Rect(2, 1, 12, 4),
		Text: "the quick brown fox",
		Fg:   1,
	}
	if got := strings.Join(b.Lines(), "|"); got != "the quick|brown fox" {
		t.Errorf("lines %q", got)
	}
	b.Box.Max.X = 8 // width 6
	if got := strings.Join(b.Lines(), "|"); got != "the|quick|brown" {
		t.Errorf(
			"narrower box should reflow and clip to height: %q",
			got,
		)
	}
	b.Box.Max.X = 22 // width 20
	if got := strings.Join(b.Lines(), "|"); got != "the quick brown fox" {
		t.Errorf("wider box should reflow to one line: %q", got)
	}
	b.Text = "new\nlines"
	if got := strings.Join(b.Lines(), "|"); got != "new|lines" {
		t.Errorf("text change reflows: %q", got)
	}
	// Every cell inside the box is owned by it: padding is a space glyph.
	b.Align = canvas.Right
	if g, ok := b.At(21, 1); !ok || g.Ch != "w" {
		t.Errorf("right aligned last char: %+v %v", g, ok)
	}
	if g, ok := b.At(2, 1); !ok || g.Ch != " " {
		t.Errorf("right aligned padding: %+v", g)
	}
	b.Align = canvas.Centre
	if g, _ := b.At(2+8, 1); g.Ch != "n" {
		t.Errorf("centred: %q", g.Ch)
	}
	if g, _ := b.At(5, 3); g.Ch != " " {
		t.Errorf("empty row inside the box is a space: %q", g.Ch)
	}
	if _, ok := b.At(0, 0); ok {
		t.Error("outside the box")
	}
	if (canvas.TextBox{Box: image.Rect(0, 0, 0, 3)}).Lines() != nil {
		t.Error("zero width")
	}
	if canvas.Left.String() != "left" ||
		canvas.Centre.String() != "centre" ||
		canvas.Right.String() != "right" {
		t.Error("Align names")
	}
}

// a box owns its cells over loose glyphs, and is found by id or by cell.
func TestLayerOwnership(t *testing.T) {
	l := canvas.NewLayer(10, 4)
	l.Cells.Set(1, 1, canvas.Glyph{Ch: "g", Fg: 1})
	b1 := l.AddBox(
		canvas.TextBox{Box: image.Rect(0, 0, 5, 2), Text: "ab", Fg: 2},
	)
	b2 := l.AddBox(
		canvas.TextBox{Box: image.Rect(3, 0, 8, 2), Text: "cd", Fg: 3},
	)
	if b1.ID != 1 || b2.ID != 2 || l.Box(2) != b2 || l.Box(9) != nil {
		t.Error("ids")
	}
	// The later box is on top where they overlap; boxes beat loose glyphs.
	if g, ok := l.At(3, 0); !ok || g.Ch != "c" {
		t.Errorf("overlap %+v", g)
	}
	if g, ok := l.At(1, 1); !ok || g.Ch != " " || g.Fg != 2 {
		t.Errorf("box padding hides the loose glyph: %+v", g)
	}
	if g, ok := l.At(9, 3); ok || !g.Empty() {
		t.Error("empty cell")
	}
	if l.BoxAt(9, 3) != nil || l.BoxAt(4, 1) != b2 {
		t.Error("BoxAt")
	}
	cl := l.Clone()
	if !l.RemoveBox(2) || l.RemoveBox(2) || len(cl.Boxes) != 2 {
		t.Error("RemoveBox/Clone")
	}
	if g, _ := l.At(3, 0); g.Ch != " " {
		t.Error("after removal the first box shows")
	}
	l.RemoveBox(1)
	if g, ok := l.At(1, 1); !ok || g.Ch != "g" {
		t.Error("loose glyph shows once the boxes are gone")
	}
	// Adding a box with an explicit id keeps the counter ahead of it.
	l.AddBox(canvas.TextBox{ID: 7})
	if l.AddBox(canvas.TextBox{}).ID != 8 {
		t.Error("next id after an explicit one")
	}
}

// Cell ownership across all three layers, as a table.
func TestOwnTable(t *testing.T) {
	red, blue := srgb.RGB{R: 1, G: 0, B: 0}, srgb.RGB{R: 0, G: 0, B: 1}
	res := func(id part.PartID) (srgb.RGB, bool) {
		switch id {
		case 1:
			return red, true
		case 2:
			return blue, true
		}
		return srgb.RGB{}, false
	}
	pixels := render.HalfBlock(&red, nil)
	cases := []struct {
		name     string
		glyph    canvas.Glyph
		hasGlyph bool
		pixels   render.Paint
		want     render.Paint
	}{
		{
			"nothing anywhere: underlay owns it",
			canvas.Glyph{},
			false,
			render.Paint{},
			render.Paint{},
		},
		{
			"pixels only: half block",
			canvas.Glyph{},
			false,
			pixels,
			pixels,
		},
		{
			"glyph, no background: letter over the image",
			canvas.Glyph{Ch: "a", Fg: 1},
			true,
			pixels,
			render.Paint{Glyph: "a", Fg: red, Set: true},
		},
		{
			"glyph with background: opaque, pixels hidden",
			canvas.Glyph{Ch: "a", Fg: 1, Bg: 2},
			true,
			pixels,
			render.Paint{
				Glyph: "a",
				Fg:    red,
				Bg:    blue,
				HasBg: true,
				Set:   true,
			},
		},
		{
			"glyph over nothing",
			canvas.Glyph{Ch: "a", Fg: 1},
			true,
			render.Paint{},
			render.Paint{Glyph: "a", Fg: red, Set: true},
		},
	}
	for _, c := range cases {
		got := render.Own(c.glyph, c.hasGlyph, c.pixels, res)
		if got != c.want {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
	}
	d := document.NewDocument("t", 4, 4)
	p, _ := d.Parts.Add("p", part.Literal(red.Swatch()))
	d.Canvas.Set(0, 0, p.ID)
	if got := d.PaintCell(0, 0); got.Glyph != render.UpperHalf {
		t.Errorf("document paints pixels: %+v", got)
	}
	d.Layer.Cells.Set(0, 0, canvas.Glyph{Ch: "z", Fg: p.ID})
	if got := d.PaintCell(0, 0); got.Glyph != "z" || got.HasBg {
		t.Errorf("document paints the glyph: %+v", got)
	}
}

// a border is drawn in its style, inside the box, corners and sides.
func TestBorder(t *testing.T) {
	c := canvas.NewCells(10, 5)
	c.Border(image.Rect(1, 1, 8, 4), canvas.Light, "hi", 1, part.None)
	row := func(y int) string {
		s := ""
		for x := 0; x < 10; x++ {
			if g := c.At(x, y); !g.Empty() {
				s += g.Ch
			} else {
				s += "."
			}
		}
		return s
	}
	if row(1) != ".┌ hi ─┐.." || row(2) != ".│.....│.." ||
		row(3) != ".└─────┘.." {
		t.Errorf(
			"light border with title:\n%s\n%s\n%s",
			row(1),
			row(2),
			row(3),
		)
	}
	// A title wider than the room is cut; a narrow box gets no title.
	c.Clear()
	c.Border(image.Rect(0, 0, 6, 3), canvas.Double, "toolong", 1, 2)
	if row(0) != "╔ to ╗...." || c.At(0, 0).Bg != 2 {
		t.Errorf("double: %s", row(0))
	}
	c.Clear()
	c.Border(image.Rect(0, 0, 4, 3), canvas.Heavy, "x", 1, part.None)
	if row(0) != "┏━━┓......" {
		t.Errorf("heavy, no room for title: %s", row(0))
	}
	c.Clear()
	c.Border(image.Rect(0, 0, 3, 3), canvas.Rounded, "", 1, part.None)
	if row(0) != "╭─╮......." || row(2) != "╰─╯......." {
		t.Errorf("rounded: %s %s", row(0), row(2))
	}
	c.Clear()
	c.Border(image.Rect(0, 0, 3, 3), canvas.ASCII, "", 1, part.None)
	if row(0) != "+-+......." || row(1) != "|.|......." {
		t.Errorf("ascii: %s %s", row(0), row(1))
	}
	// Degenerate boxes are rules.
	c.Clear()
	c.Border(image.Rect(0, 0, 5, 1), canvas.Light, "", 1, part.None)
	if row(0) != "─────....." {
		t.Errorf("rule: %s", row(0))
	}
	c.Clear()
	c.Border(image.Rect(0, 0, 1, 3), canvas.Light, "", 1, part.None)
	if row(0) != "│........." || row(2) != "│........." {
		t.Errorf("vertical rule: %s", row(0))
	}
	c.Border(image.Rectangle{}, canvas.Light, "", 1, part.None)
	styles := map[canvas.BorderStyle]string{
		canvas.Light:   "light",
		canvas.Heavy:   "heavy",
		canvas.Double:  "double",
		canvas.Rounded: "rounded",
		canvas.ASCII:   "ascii",
	}
	for s, want := range styles {
		if s.String() != want {
			t.Error(s)
		}
	}
	if len(canvas.Palette) < 5 || canvas.Palette[0].Name != "box light" ||
		len(canvas.Palette[0].Glyphs) == 0 {
		t.Error("palette")
	}
}
