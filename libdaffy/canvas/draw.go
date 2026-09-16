package canvas

import "image"

// The drawing algorithms. Each takes a Plotter and calls it once per pixel,
// so the same code paints parts into a canvas stroke, bits into a mask, or
// nothing into a counter. None of them know about colour.

// Plotter receives pixels from an algorithm.
type Plotter interface {
	Plot(x, y int)
}

// PlotFunc adapts a function to a Plotter.
type PlotFunc func(x, y int)

// Plot calls the function.
func (f PlotFunc) Plot(x, y int) { f(x, y) }

// Line walks Bresenham's line from one point to the other, inclusive of both.
// It returns the number of pixels between the endpoints, which is what a
// stroke interpolator wants to know about the gap it filled.
func Line(p Plotter, x0, y0, x1, y1 int) int {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	err := dx + dy
	n := 0
	for {
		p.Plot(x0, y0)
		if x0 == x1 && y0 == y1 {
			return n
		}
		n++
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// RectOutline draws the four edges of a rectangle given by two corners,
// inclusive.
func RectOutline(p Plotter, x0, y0, x1, y1 int) {
	r := corners(x0, y0, x1, y1)
	for x := r.Min.X; x < r.Max.X; x++ {
		p.Plot(x, r.Min.Y)
		p.Plot(x, r.Max.Y-1)
	}
	for y := r.Min.Y + 1; y < r.Max.Y-1; y++ {
		p.Plot(r.Min.X, y)
		p.Plot(r.Max.X-1, y)
	}
}

// RectFill fills a rectangle given by two corners, inclusive.
func RectFill(p Plotter, x0, y0, x1, y1 int) {
	r := corners(x0, y0, x1, y1)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			p.Plot(x, y)
		}
	}
}

// EllipseOutline draws the ellipse inscribed in the rectangle given by two
// corners, inclusive, by the midpoint method. A degenerate box draws a line.
func EllipseOutline(p Plotter, x0, y0, x1, y1 int) {
	r := corners(x0, y0, x1, y1)
	if r.Dx() <= 1 || r.Dy() <= 1 {
		Line(p, r.Min.X, r.Min.Y, r.Max.X-1, r.Max.Y-1)
		return
	}
	ellipse(r, func(xl, xr, y int) {
		p.Plot(xl, y)
		if xr != xl {
			p.Plot(xr, y)
		}
	})
}

// EllipseFill fills the ellipse inscribed in the rectangle, inclusive.
func EllipseFill(p Plotter, x0, y0, x1, y1 int) {
	r := corners(x0, y0, x1, y1)
	if r.Dx() <= 1 || r.Dy() <= 1 {
		Line(p, r.Min.X, r.Min.Y, r.Max.X-1, r.Max.Y-1)
		return
	}
	rows := map[int][2]int{}
	ellipse(r, func(xl, xr, y int) {
		if span, ok := rows[y]; ok {
			rows[y] = [2]int{min(span[0], xl), max(span[1], xr)}
		} else {
			rows[y] = [2]int{xl, xr}
		}
	})
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if span, ok := rows[y]; ok {
			for x := span[0]; x <= span[1]; x++ {
				p.Plot(x, y)
			}
		}
	}
}

// ellipse walks the boundary of the ellipse inscribed in r and reports, for
// each boundary pixel, its mirror across the vertical axis. It is the
// integer-only midpoint ellipse for an arbitrary bounding box, which handles
// even and odd diameters without a half-pixel centre.
//
// Adapted from Alois Zingl's "A Rasterizing Algorithm for Drawing Curves",
// 2012, which is the standard reference for this.
func ellipse(r image.Rectangle, span func(xl, xr, y int)) {
	x0, y0, x1, y1 := r.Min.X, r.Min.Y, r.Max.X-1, r.Max.Y-1
	a, b := abs(x1-x0), abs(y1-y0)
	b1 := b & 1
	dx := 4 * (1 - a) * b * b
	dy := 4 * (b1 + 1) * a * a
	err := dx + dy + b1*a*a
	if x0 > x1 {
		x0, x1 = x1, x0+a
	}
	if y0 > y1 {
		y0 = y1
	}
	y0 += (b + 1) / 2
	y1 = y0 - b1
	a *= 8 * a
	b1 = 8 * b * b
	for {
		span(x0, x1, y0)
		if y0 != y1 {
			span(x0, x1, y1)
		}
		e2 := 2 * err
		if e2 <= dy {
			y0++
			y1--
			dy += a
			err += dy
		}
		if e2 >= dx || 2*err > dy {
			x0++
			x1--
			dx += b1
			err += dx
		}
		if x0 > x1 {
			break
		}
	}
	for y0-y1 < b {
		span(x0-1, x0-1, y0)
		span(x1+1, x1+1, y0)
		y0++
		span(x0-1, x0-1, y1)
		span(x1+1, x1+1, y1)
		y1--
	}
}

// FloodFill plots every pixel four-connected to the seed whose value, as
// reported by same, matches the seed's. It is a scanline fill with an explicit
// stack, so a canvas-sized region does not recurse its way off the stack.
// The bounds say where the fill may go.
func FloodFill(
	p Plotter,
	bounds image.Rectangle,
	x, y int,
	same func(x, y int) bool,
) {
	if !image.Pt(x, y).In(bounds) || !same(x, y) {
		return
	}
	w := bounds.Dx()
	done := make([]bool, w*bounds.Dy())
	idx := func(x, y int) int {
		return (y-bounds.Min.Y)*w + (x - bounds.Min.X)
	}
	inside := func(x, y int) bool {
		return image.Pt(x, y).In(bounds) && !done[idx(x, y)] &&
			same(x, y)
	}
	type seg struct{ x, y int }
	stack := []seg{{x, y}}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !inside(s.x, s.y) {
			continue
		}
		// Walk left to the start of the run.
		lx := s.x
		for inside(lx-1, s.y) {
			lx--
		}
		// Fill rightward, seeding the row above and the row below once
		// per new run of fillable pixels in each.
		above, below := false, false
		for cx := lx; inside(cx, s.y); cx++ {
			done[idx(cx, s.y)] = true
			p.Plot(cx, s.y)
			up := inside(cx, s.y-1)
			if up && !above {
				stack = append(stack, seg{cx, s.y - 1})
			}
			above = up
			down := inside(cx, s.y+1)
			if down && !below {
				stack = append(stack, seg{cx, s.y + 1})
			}
			below = down
		}
	}
}

// corners turns two inclusive corners in any order into a image.Rectangle.
func corners(x0, y0, x1, y1 int) image.Rectangle {
	r := image.Rect(x0, y0, x1, y1)
	r.Max.X++
	r.Max.Y++
	return r
}

// abs is the size of a step, whichever way it goes.
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// sign is which way to step, or nowhere for zero.
func sign(v int) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	}
	return 0
}
