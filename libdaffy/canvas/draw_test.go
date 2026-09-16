package canvas

import (
	"image"
	"strings"
	"testing"
)

// grid collects plots into a picture, for comparing against expectations.
type grid struct {
	w, h int
	on   []bool
}

// newGrid is a plotter that remembers what was plotted, for the shape tests.
func newGrid(w, h int) *grid { return &grid{w: w, h: h, on: make([]bool, w*h)} }

// Plot marks a point, dropping any outside the grid.
func (g *grid) Plot(x, y int) {
	if x >= 0 && y >= 0 && x < g.w && y < g.h {
		g.on[y*g.w+x] = true
	}
}

// String draws the grid as text, so a failure can be looked at.
func (g *grid) String() string {
	var b strings.Builder
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			if g.on[y*g.w+x] {
				b.WriteByte('#')
			} else {
				b.WriteByte('.')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// count is how many points were plotted.
func (g *grid) count() int {
	n := 0
	for _, v := range g.on {
		if v {
			n++
		}
	}
	return n
}

// a line reaches both ends in every direction, with no gaps.
func TestLine(t *testing.T) {
	g := newGrid(5, 5)
	if n := Line(g, 0, 0, 4, 4); n != 4 {
		t.Errorf("gap %d", n)
	}
	want := "#....\n.#...\n..#..\n...#.\n....#\n"
	if g.String() != want {
		t.Errorf("diagonal:\n%s", g)
	}
	g = newGrid(5, 3)
	Line(g, 4, 1, 0, 1)
	if g.String() != ".....\n#####\n.....\n" {
		t.Errorf("horizontal backwards:\n%s", g)
	}
	g = newGrid(3, 3)
	if n := Line(g, 1, 1, 1, 1); n != 0 || g.count() != 1 {
		t.Errorf("point: gap %d count %d", n, g.count())
	}
	// Steep line: every row gets exactly one pixel.
	g = newGrid(3, 7)
	Line(g, 0, 0, 2, 6)
	for y := 0; y < 7; y++ {
		row := 0
		for x := 0; x < 3; x++ {
			if g.on[y*3+x] {
				row++
			}
		}
		if row != 1 {
			t.Errorf("steep line row %d has %d pixels", y, row)
		}
	}
}

// a filled rect covers its area; an outline covers only its edge.
func TestRects(t *testing.T) {
	g := newGrid(5, 4)
	RectOutline(g, 4, 3, 0, 0)
	if g.String() != "#####\n#...#\n#...#\n#####\n" {
		t.Errorf("outline:\n%s", g)
	}
	g = newGrid(5, 4)
	RectFill(g, 1, 1, 3, 2)
	if g.String() != ".....\n.###.\n.###.\n.....\n" {
		t.Errorf("fill:\n%s", g)
	}
	g = newGrid(3, 3)
	RectOutline(g, 1, 1, 1, 1)
	if g.count() != 1 {
		t.Errorf("single-pixel outline count %d", g.count())
	}
	g = newGrid(3, 3)
	RectOutline(g, 0, 0, 2, 0)
	if g.String() != "###\n...\n...\n" {
		t.Errorf("one-row outline:\n%s", g)
	}
}

// an ellipse is symmetric about both axes and touches all four edges.
func TestEllipse(t *testing.T) {
	g := newGrid(7, 5)
	EllipseOutline(g, 0, 0, 6, 4)
	s := g.String()
	// Symmetric about both axes, touches all four edges, hollow in the
	// middle.
	if !g.on[0*7+3] || !g.on[4*7+3] || !g.on[2*7+0] || !g.on[2*7+6] {
		t.Errorf("outline does not touch the box:\n%s", s)
	}
	if g.on[2*7+3] {
		t.Errorf("outline is filled:\n%s", s)
	}
	for y := 0; y < 5; y++ {
		for x := 0; x < 7; x++ {
			if g.on[y*7+x] != g.on[y*7+(6-x)] ||
				g.on[y*7+x] != g.on[(4-y)*7+x] {
				t.Fatalf("asymmetric at %d,%d:\n%s", x, y, s)
			}
		}
	}
	f := newGrid(7, 5)
	EllipseFill(f, 6, 4, 0, 0)
	if !f.on[2*7+3] || f.count() <= g.count() {
		t.Errorf("fill:\n%s", f)
	}
	// Every outline pixel is in the fill.
	for i := range g.on {
		if g.on[i] && !f.on[i] {
			t.Fatalf(
				"fill misses outline pixel %d:"+
					"\nout\n%s\nfill\n%s",
				i,
				s,
				f,
			)
		}
	}
	// Even diameters have no centre pixel and must still close.
	e := newGrid(8, 6)
	EllipseOutline(e, 0, 0, 7, 5)
	if e.count() == 0 || !e.on[0*8+3] || !e.on[0*8+4] {
		t.Errorf("even ellipse:\n%s", e)
	}
	// Degenerate boxes are lines.
	l := newGrid(5, 1)
	EllipseOutline(l, 0, 0, 4, 0)
	if l.count() != 5 {
		t.Errorf("flat ellipse should be a line:\n%s", l)
	}
	l = newGrid(1, 5)
	EllipseFill(l, 0, 0, 0, 4)
	if l.count() != 5 {
		t.Errorf("tall ellipse fill should be a line:\n%s", l)
	}
}

// a flood fill stops at a ring: inside stays inside, outside outside.
func TestFloodFill(t *testing.T) {
	// A ring with a hole: fill inside stays inside, fill outside stays
	// outside.
	pic := []string{
		"........",
		".######.",
		".#....#.",
		".#.##.#.",
		".#....#.",
		".######.",
		"........",
	}
	w, h := 8, 7
	wall := func(x, y int) bool { return pic[y][x] == '#' }
	same := func(x, y int) bool { return !wall(x, y) }
	b := image.Rect(0, 0, w, h)

	in := newGrid(w, h)
	FloodFill(in, b, 2, 2, same)
	if in.count() != 10 {
		t.Errorf("inside fill got %d:\n%s", in.count(), in)
	}
	out := newGrid(w, h)
	FloodFill(out, b, 0, 0, same)
	if out.count() != 8+8+5+5 {
		t.Errorf("outside fill got %d:\n%s", out.count(), out)
	}
	none := newGrid(w, h)
	FloodFill(none, b, 1, 1, same)
	if none.count() != 0 {
		t.Error("seed on a wall should fill nothing")
	}
	FloodFill(none, b, -1, -1, same)
	if none.count() != 0 {
		t.Error("seed off the bounds should fill nothing")
	}
	// A large open area fills completely without recursion.
	big := newGrid(300, 300)
	FloodFill(
		big,
		image.Rect(0, 0, 300, 300),
		150,
		150,
		func(int, int) bool { return true },
	)
	if big.count() != 90000 {
		t.Errorf("open fill got %d", big.count())
	}
	// A spiral, which defeats naive scanline seeding.
	sp := []string{
		"#########",
		"#.......#",
		"#.#####.#",
		"#.#...#.#",
		"#.#.#.#.#",
		"#.#.###.#",
		"#.#.....#",
		"#.#######",
		"#........",
	}
	sg := newGrid(9, 9)
	FloodFill(
		sg,
		image.Rect(0, 0, 9, 9),
		8,
		8,
		func(x, y int) bool { return sp[y][x] == '.' },
	)
	dots := 0
	for _, r := range sp {
		dots += strings.Count(r, ".")
	}
	if sg.count() != dots {
		t.Errorf("spiral got %d want %d:\n%s", sg.count(), dots, sg)
	}
	var pf PlotFunc = func(x, y int) {}
	pf.Plot(0, 0)
}
