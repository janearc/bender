package canvas

// Canvas is the pixel grid at half-block resolution: one terminal cell holds
// two pixels, stacked, so a canvas of W by H pixels renders on W columns and
// H/2 rows. Each pixel holds a part id, never a colour.

import (
	"github.com/janearc/libdaffy/part"
	"image"
)

// Canvas is the drawing surface.
type Canvas struct {
	w, h int
	px   []part.PartID
}

// NewCanvas makes an empty canvas. Height is in pixels and is rounded up to
// an even number, because a half cell does not exist.
func NewCanvas(w, h int) *Canvas {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if h%2 == 1 {
		h++
	}
	return &Canvas{w: w, h: h, px: make([]part.PartID, w*h)}
}

// Width is the canvas width in pixels, which is also its width in cells.
func (c *Canvas) Width() int { return c.w }

// Height is the canvas height in pixels, twice its height in cells.
func (c *Canvas) Height() int { return c.h }

// Rows is the canvas height in cells.
func (c *Canvas) Rows() int { return c.h / 2 }

// In reports whether a pixel coordinate is on the canvas.
func (c *Canvas) In(x, y int) bool {
	return x >= 0 && y >= 0 && x < c.w && y < c.h
}

// At returns the part at a pixel. Off the canvas it is None.
func (c *Canvas) At(x, y int) part.PartID {
	if !c.In(x, y) {
		return part.None
	}
	return c.px[y*c.w+x]
}

// Set writes a part at a pixel, ignoring anything off the canvas. It reports
// whether the pixel changed.
func (c *Canvas) Set(x, y int, id part.PartID) bool {
	if !c.In(x, y) {
		return false
	}
	i := y*c.w + x
	if c.px[i] == id {
		return false
	}
	c.px[i] = id
	return true
}

// Clear sets every pixel to None.
func (c *Canvas) Clear() {
	for i := range c.px {
		c.px[i] = part.None
	}
}

// Clone copies the canvas.
func (c *Canvas) Clone() *Canvas {
	n := &Canvas{w: c.w, h: c.h, px: make([]part.PartID, len(c.px))}
	copy(n.px, c.px)
	return n
}

// Pixels exposes the grid row-major, for encoders and tests. It is the live
// slice, not a copy.
func (c *Canvas) Pixels() []part.PartID { return c.px }

// Count reports how many pixels carry a part.
func (c *Canvas) Count(id part.PartID) int {
	n := 0
	for _, p := range c.px {
		if p == id {
			n++
		}
	}
	return n
}

// Replace turns every pixel of one part into another. It is what a part
// deletion does with the pixels left behind, and it reports how many moved.
func (c *Canvas) Replace(from, to part.PartID) int {
	n := 0
	for i, p := range c.px {
		if p == from {
			c.px[i] = to
			n++
		}
	}
	return n
}

// Resize makes a new canvas of another size holding this one's pixels at the
// same coordinates, cropping or padding with None.
func (c *Canvas) Resize(w, h int) *Canvas {
	n := NewCanvas(w, h)
	for y := 0; y < c.h && y < n.h; y++ {
		for x := 0; x < c.w && x < n.w; x++ {
			n.px[y*n.w+x] = c.px[y*c.w+x]
		}
	}
	return n
}

// Equal reports whether two canvases hold the same pixels.
func (c *Canvas) Equal(o *Canvas) bool {
	if c.w != o.w || c.h != o.h {
		return false
	}
	for i := range c.px {
		if c.px[i] != o.px[i] {
			return false
		}
	}
	return true
}

// Bounds is the whole canvas as a pixel rectangle.
func (c *Canvas) Bounds() image.Rectangle { return image.Rect(0, 0, c.w, c.h) }

// PixelToCell converts a pixel rectangle to the cells it touches.
func PixelToCell(r image.Rectangle) image.Rectangle {
	if r.Empty() {
		return image.Rectangle{}
	}
	return image.Rect(
		r.Min.X,
		CellRow(r.Min.Y),
		r.Max.X,
		CellRow(r.Max.Y+1),
	)
}

// CellToPixel converts a cell rectangle to the pixels it covers.
func CellToPixel(r image.Rectangle) image.Rectangle {
	if r.Empty() {
		return image.Rectangle{}
	}
	return image.Rect(r.Min.X, r.Min.Y*2, r.Max.X, r.Max.Y*2)
}
