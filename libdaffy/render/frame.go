package render

// Frame is a picture as a grid of painted cells, whole: what a document,
// a scene or a stream says its cells are, and what a writer turns into
// bytes. It is the seam between painting and writing; neither side knows
// the other, and time is the painter's business, never the frame's.
type Frame struct {
	w, h  int
	cells []Paint
}

// NewFrame makes an unpainted frame of a size in cells.
func NewFrame(w, h int) Frame {
	return Frame{w: w, h: h, cells: make([]Paint, w*h)}
}

// Width is the frame's columns.
func (f Frame) Width() int { return f.w }

// Height is the frame's rows.
func (f Frame) Height() int { return f.h }

// At is the paint of a cell; off the frame is unpainted.
func (f Frame) At(x, y int) Paint {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return Paint{}
	}
	return f.cells[y*f.w+x]
}

// Set paints a cell, ignoring one off the frame.
func (f Frame) Set(x, y int, p Paint) {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return
	}
	f.cells[y*f.w+x] = p
}

// Clone copies the frame.
func (f Frame) Clone() Frame {
	out := Frame{w: f.w, h: f.h, cells: make([]Paint, len(f.cells))}
	copy(out.cells, f.cells)
	return out
}

// Equal is whether two frames paint every cell the same.
func (f Frame) Equal(o Frame) bool {
	if f.w != o.w || f.h != o.h {
		return false
	}
	for i := range f.cells {
		if f.cells[i] != o.cells[i] {
			return false
		}
	}
	return true
}
