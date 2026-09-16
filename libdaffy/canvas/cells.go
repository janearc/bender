package canvas

// The cell layer: glyphs, text boxes and borders.
//
// A terminal cell can show one glyph with one foreground and one background.
//
// The half-block trick spends both slots on colour to buy two pixels, so a cell
// showing text cannot also show two pixels: the glyph is the character, not the
// block. A cell is therefore owned by exactly one layer.
//
// If the cell layer has a glyph there, it draws and the pixels beneath are
// hidden; if it does not, the pixels draw; if neither, the underlay shows
// through.
//
// A glyph with no background part leaves the background unset, which is the
// half-block rule seen from below: the image shows behind the letterforms.

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"github.com/janearc/libdaffy/part"
	"image"

	"strings"
)

// Glyph is one cell of text: a character with a foreground part and an
// optional background part. Ch is one grapheme; an empty Ch is no glyph.
type Glyph struct {
	Ch string
	Fg part.PartID
	Bg part.PartID // None leaves the background unset
}

// Empty is whether nothing is here, the word Canvas and Mask use too.
func (g Glyph) Empty() bool { return g.Ch == "" }

// Cells is the loose-glyph grid, in cells.
type Cells struct {
	w, h int
	g    []Glyph
}

// NewCells makes an empty grid of a size in cells.
func NewCells(w, h int) *Cells {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return &Cells{w: w, h: h, g: make([]Glyph, w*h)}
}

// Width in cells.
func (c *Cells) Width() int { return c.w }

// Height in cells.
func (c *Cells) Height() int { return c.h }

// In reports whether a cell is on the grid.
func (c *Cells) In(x, y int) bool {
	return x >= 0 && y >= 0 && x < c.w && y < c.h
}

// At is the glyph at a cell, or an empty glyph.
func (c *Cells) At(x, y int) Glyph {
	if !c.In(x, y) {
		return Glyph{}
	}
	return c.g[y*c.w+x]
}

// Set writes a glyph, ignoring anything off the grid. It reports whether
// the cell changed.
func (c *Cells) Set(x, y int, g Glyph) bool {
	if !c.In(x, y) {
		return false
	}
	i := y*c.w + x
	if c.g[i] == g {
		return false
	}
	c.g[i] = g
	return true
}

// Clear removes every glyph.
func (c *Cells) Clear() {
	for i := range c.g {
		c.g[i] = Glyph{}
	}
}

// Count is how many cells hold a glyph.
func (c *Cells) Count() int {
	n := 0
	for _, g := range c.g {
		if !g.Empty() {
			n++
		}
	}
	return n
}

// Clone copies the grid.
func (c *Cells) Clone() *Cells {
	n := &Cells{w: c.w, h: c.h, g: make([]Glyph, len(c.g))}
	copy(n.g, c.g)
	return n
}

// Equal compares two grids.
func (c *Cells) Equal(o *Cells) bool {
	if c.w != o.w || c.h != o.h {
		return false
	}
	for i := range c.g {
		if c.g[i] != o.g[i] {
			return false
		}
	}
	return true
}

// Glyphs exposes the grid row-major, live.
func (c *Cells) Glyphs() []Glyph { return c.g }

// Resize makes a grid of another size holding these glyphs at the same
// coordinates.
func (c *Cells) Resize(w, h int) *Cells {
	n := NewCells(w, h)
	for y := 0; y < c.h && y < n.h; y++ {
		for x := 0; x < c.w && x < n.w; x++ {
			n.g[y*n.w+x] = c.g[y*c.w+x]
		}
	}
	return n
}

// PutString writes a run of graphemes left to right from a cell, one per
// cell, and returns how many cells it took. It stops at the edge.
func (c *Cells) PutString(x, y int, s string, fg, bg part.PartID) int {
	n := 0
	for _, r := range s {
		if !c.In(x+n, y) {
			break
		}
		c.Set(x+n, y, Glyph{Ch: string(r), Fg: fg, Bg: bg})
		n++
	}
	return n
}

// PutBlock writes lines of text as a rectangular paste of characters, one
// glyph per rune, with the top-left at a cell. Spaces are written as glyphs
// too: a paste is opaque within its rectangle, which is what makes it a
// paste rather than a stencil.
func (c *Cells) PutBlock(x, y int, text string, fg, bg part.PartID) {
	for i, line := range strings.Split(text, "\n") {
		c.PutString(x, y+i, line, fg, bg)
	}
}

// --- text boxes --------------------------------------------------------------

// Align is a text box's horizontal alignment.
type Align int

// The alignments.
const (
	Left Align = iota
	Centre
	Right
)

// String names the alignment.
func (a Align) String() string {
	switch a {
	case Centre:
		return "centre"
	case Right:
		return "right"
	}
	return "left"
}

// TextBox is a rectangle a person types into. It is stored as its string
// and its box, not as loose glyphs, so changing the text reflows it and
// changing the width reflows it again.
type TextBox struct {
	ID    int
	Box   image.Rectangle // in cells
	Text  string
	Align Align
	Fg    part.PartID
	Bg    part.PartID // None leaves the background unset
}

// Lines is the text wrapped to the box's width, at most its height, each
// line padded or positioned by the alignment. Newlines in the text break
// lines; words longer than the width are broken.
func (t TextBox) Lines() []string {
	w := t.Box.Dx()
	if w < 1 {
		return nil
	}
	var out []string
	for _, para := range strings.Split(t.Text, "\n") {
		out = append(out, wrap(para, w)...)
		if len(out) >= t.Box.Dy() {
			break
		}
	}
	if len(out) > t.Box.Dy() {
		out = out[:t.Box.Dy()]
	}
	return out
}

// wrap breaks a paragraph into lines no wider than w cells, on spaces
// where it can and inside a word where it must, measuring width the
// way a terminal does. The wrapping is charm's; only the box is ours.
func wrap(s string, w int) []string {
	if s == "" || w < 1 {
		return []string{""}
	}
	return strings.Split(ansi.Wrap(s, w, ""), "\n")
}

// At is the glyph at a cell of the box once its text is laid out, and
// whether there is one: a box owns every cell in its rectangle but only
// paints where a character landed.
func (t TextBox) At(x, y int) (Glyph, bool) {
	if !image.Pt(x, y).In(t.Box) {
		return Glyph{}, false
	}
	lines := t.Lines()
	row := y - t.Box.Min.Y
	ch := " "
	if row < len(lines) {
		line := []rune(lines[row])
		w := t.Box.Dx()
		pad := 0
		switch t.Align {
		case Centre:
			pad = (w - len(line)) / 2
		case Right:
			pad = w - len(line)
		}
		col := x - t.Box.Min.X - pad
		if col >= 0 && col < len(line) {
			ch = string(line[col])
		}
	}
	return Glyph{Ch: ch, Fg: t.Fg, Bg: t.Bg}, true
}

// --- the layer ---------------------------------------------------------------

// Layer is the whole cell layer: text boxes over loose glyphs. Later boxes
// are on top of earlier ones.
type Layer struct {
	Cells  *Cells
	Boxes  []*TextBox
	nextID int
}

// NewLayer makes an empty layer the size of a canvas in cells.
func NewLayer(w, h int) *Layer {
	return &Layer{Cells: NewCells(w, h), nextID: 1}
}

// AddBox registers a text box under a fresh id and returns it.
func (l *Layer) AddBox(b TextBox) *TextBox {
	if b.ID == 0 {
		b.ID = l.nextID
	}
	if b.ID >= l.nextID {
		l.nextID = b.ID + 1
	}
	cp := b
	l.Boxes = append(l.Boxes, &cp)
	return &cp
}

// Box finds a text box by id, or nil.
func (l *Layer) Box(id int) *TextBox {
	for _, b := range l.Boxes {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// BoxAt finds the topmost text box covering a cell, or nil.
func (l *Layer) BoxAt(x, y int) *TextBox {
	for i := len(l.Boxes) - 1; i >= 0; i-- {
		if image.Pt(x, y).In(l.Boxes[i].Box) {
			return l.Boxes[i]
		}
	}
	return nil
}

// SetBox replaces a text box's contents, box and style, keeping its id.
func (l *Layer) SetBox(id int, after TextBox) error {
	b := l.Box(id)
	if b == nil {
		return fmt.Errorf("layer: no text box %d", id)
	}
	after.ID = id
	*b = after
	return nil
}

// RemoveBox deletes a text box by id and reports whether it was there.
func (l *Layer) RemoveBox(id int) bool {
	for i, b := range l.Boxes {
		if b.ID == id {
			l.Boxes = append(l.Boxes[:i], l.Boxes[i+1:]...)
			return true
		}
	}
	return false
}

// At is the glyph the layer shows at a cell, and whether it shows one. Text
// boxes win over loose glyphs.
func (l *Layer) At(x, y int) (Glyph, bool) {
	if b := l.BoxAt(x, y); b != nil {
		return b.At(x, y)
	}
	g := l.Cells.At(x, y)
	return g, !g.Empty()
}

// Equal is whether two layers hold the same glyphs and the same boxes, so
// a gesture on a layer that changed nothing can be dropped.
func (l *Layer) Equal(o *Layer) bool {
	if !l.Cells.Equal(o.Cells) || len(l.Boxes) != len(o.Boxes) {
		return false
	}
	for i := range l.Boxes {
		if *l.Boxes[i] != *o.Boxes[i] {
			return false
		}
	}
	return true
}

// Clone copies the layer.
func (l *Layer) Clone() *Layer {
	n := &Layer{Cells: l.Cells.Clone(), nextID: l.nextID}
	for _, b := range l.Boxes {
		cp := *b
		n.Boxes = append(n.Boxes, &cp)
	}
	return n
}

// --- borders -----------------------------------------------------------------

// BorderStyle is the set of rules a border is drawn with.
type BorderStyle int

// The styles.
const (
	Light BorderStyle = iota
	Heavy
	Double
	Rounded
	ASCII
)

// String names the style.
func (s BorderStyle) String() string {
	switch s {
	case Heavy:
		return "heavy"
	case Double:
		return "double"
	case Rounded:
		return "rounded"
	case ASCII:
		return "ascii"
	}
	return "light"
}

// borderSet is the eight characters of a style: the four corners clockwise
// from top-left, then horizontal, then vertical.
var borderSets = map[BorderStyle][6]string{
	Light:   {"┌", "┐", "┘", "└", "─", "│"},
	Heavy:   {"┏", "┓", "┛", "┗", "━", "┃"},
	Double:  {"╔", "╗", "╝", "╚", "═", "║"},
	Rounded: {"╭", "╮", "╯", "╰", "─", "│"},
	ASCII:   {"+", "+", "+", "+", "-", "|"},
}

// Border writes a border around the inside edge of a cell rectangle into
// the loose glyphs, with an optional title in the top rule. A rectangle
// narrower or shorter than two cells gets a rule rather than a box.
func (c *Cells) Border(
	r image.Rectangle,
	style BorderStyle,
	title string,
	fg, bg part.PartID,
) {
	if r.Empty() {
		return
	}
	set := borderSets[style]
	put := func(x, y int, ch string) {
		c.Set(x, y, Glyph{Ch: ch, Fg: fg, Bg: bg})
	}
	x0, y0, x1, y1 := r.Min.X, r.Min.Y, r.Max.X-1, r.Max.Y-1
	if r.Dx() == 1 || r.Dy() == 1 {
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if r.Dx() == 1 {
					put(x, y, set[5])
				} else {
					put(x, y, set[4])
				}
			}
		}
		return
	}
	for x := x0 + 1; x < x1; x++ {
		put(x, y0, set[4])
		put(x, y1, set[4])
	}
	for y := y0 + 1; y < y1; y++ {
		put(x0, y, set[5])
		put(x1, y, set[5])
	}
	put(x0, y0, set[0])
	put(x1, y0, set[1])
	put(x1, y1, set[2])
	put(x0, y1, set[3])
	if title != "" && r.Dx() > 4 {
		room := r.Dx() - 4
		t := []rune(title)
		if len(t) > room {
			t = t[:room]
		}
		c.PutString(x0+1, y0, " "+string(t)+" ", fg, bg)
	}
}

// --- the glyph palette -------------------------------------------------------

// GlyphSet is a named group of characters interfaces are made of.
type GlyphSet struct {
	Name   string
	Glyphs []string
}

// Palette is the glyph sets: box drawing, blocks and arrows, because
// hunting for them by codepoint is miserable.
var Palette = []GlyphSet{
	{"box light", split("─ │ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼ ╴ ╵ ╶ ╷")},
	{"box heavy", split("━ ┃ ┏ ┓ ┗ ┛ ┣ ┫ ┳ ┻ ╋ ╸ ╹ ╺ ╻")},
	{"box double", split("═ ║ ╔ ╗ ╚ ╝ ╠ ╣ ╦ ╩ ╬ ╒ ╕ ╘ ╛")},
	{"box rounded and dashed", split("╭ ╮ ╰ ╯ ╌ ╍ ┄ ┅ ┆ ┇ ┈ ┉ ┊ ┋ ╱ ╲ ╳")},
	{"blocks", split("█ ▀ ▄ ▌ ▐ ░ ▒ ▓ ▖ ▗ ▘ ▝ ▚ ▞ ▛ ▜ ▙ ▟ ■ □ ▪ ▫")},
	{"arrows", split("← ↑ → ↓ ↔ ↕ ⇐ ⇑ ⇒ ⇓ ⇔ ⇕ ▲ ▼ ◀ ▶ △ ▽ ◁ ▷ ▴ ▾ ◂ ▸")},
	{"marks", split("• ◦ ○ ● ◌ ◍ ◎ ✓ ✗ ✕ ✖ ★ ☆ … · ‥ ⋯ ⋮ ‹ › « »")},
}

// split is the characters of a palette line, written with spaces between
// them so a person can read the line.
func split(s string) []string { return strings.Fields(s) }
