package canvas

import (
	"image"
	"math/bits"

	"github.com/janearc/libdaffy/part"
)

// Mask is a per-pixel stencil the size of a canvas, kept as a bit vector:
//
// sixty-four pixels a word, so the questions asked of it in a loop, is
// anything in, how many, are these the same, are answered a word at a
// time by the standard library's bit tricks rather than a pixel at a
// time.
type Mask struct {
	w, h  int
	words []uint64
}

// NewMask makes a mask of a size with nothing selected.
func NewMask(w, h int) *Mask {
	return &Mask{w: w, h: h, words: make([]uint64, (w*h+63)/64)}
}

// FullMask makes a mask with everything selected.
func FullMask(w, h int) *Mask {
	m := NewMask(w, h)
	for i := range m.words {
		m.words[i] = ^uint64(0)
	}
	m.trim()
	return m
}

// trim clears the bits past the last pixel, so a whole-word operation
// never counts pixels that do not exist.
func (m *Mask) trim() {
	if n := m.w * m.h; n%64 != 0 && len(m.words) > 0 {
		m.words[len(m.words)-1] &= (uint64(1) << uint(n%64)) - 1
	}
}

// Width is the mask width in pixels.
func (m *Mask) Width() int { return m.w }

// Height is the mask height in pixels.
func (m *Mask) Height() int { return m.h }

// At reports whether a pixel is in the mask. Off the edge is out.
func (m *Mask) At(x, y int) bool {
	if x < 0 || y < 0 || x >= m.w || y >= m.h {
		return false
	}
	i := y*m.w + x
	return m.words[i/64]&(uint64(1)<<uint(i%64)) != 0
}

// Set puts a pixel in or out of the mask.
func (m *Mask) Set(x, y int, v bool) {
	if x < 0 || y < 0 || x >= m.w || y >= m.h {
		return
	}
	i := y*m.w + x
	if v {
		m.words[i/64] |= uint64(1) << uint(i%64)
	} else {
		m.words[i/64] &^= uint64(1) << uint(i%64)
	}
}

// Plot puts a pixel in, so a mask is a Plotter for the drawing algorithms.
func (m *Mask) Plot(x, y int) { m.Set(x, y, true) }

// Invert flips every pixel.
func (m *Mask) Invert() {
	for i := range m.words {
		m.words[i] = ^m.words[i]
	}
	m.trim()
}

// Clear takes everything out.
func (m *Mask) Clear() {
	for i := range m.words {
		m.words[i] = 0
	}
}

// Count is how many pixels are in.
func (m *Mask) Count() int {
	n := 0
	for _, w := range m.words {
		n += bits.OnesCount64(w)
	}
	return n
}

// Empty reports whether nothing is in, without counting.
func (m *Mask) Empty() bool {
	for _, w := range m.words {
		if w != 0 {
			return false
		}
	}
	return true
}

// Clone copies the mask.
func (m *Mask) Clone() *Mask {
	out := &Mask{w: m.w, h: m.h, words: make([]uint64, len(m.words))}
	copy(out.words, m.words)
	return out
}

// Equal compares two masks.
func (m *Mask) Equal(o *Mask) bool {
	if m.w != o.w || m.h != o.h {
		return false
	}
	for i := range m.words {
		if m.words[i] != o.words[i] {
			return false
		}
	}
	return true
}

// Bits is a copy of the pixels row-major, for an encoder; writing to it
// changes nothing, which is the point of the copy.
func (m *Mask) Bits() []bool {
	out := make([]bool, m.w*m.h)
	for i := range out {
		out[i] = m.words[i/64]&(uint64(1)<<uint(i%64)) != 0
	}
	return out
}

// Load sets the pixels from a row-major slice, the way an encoder wrote
// them; a short slice leaves the rest out.
func (m *Mask) Load(on []bool) {
	m.Clear()
	for i, v := range on {
		if i >= m.w*m.h {
			break
		}
		if v {
			m.words[i/64] |= uint64(1) << uint(i%64)
		}
	}
}

// Grow expands the mask by n pixels in every direction, four-connected, so a
// selection gains a border. It returns a new mask.
func (m *Mask) Grow(n int) *Mask {
	out := m.Clone()
	for step := 0; step < n; step++ {
		src := out.Clone()
		for y := 0; y < m.h; y++ {
			for x := 0; x < m.w; x++ {
				if src.At(x, y) || src.At(x-1, y) ||
					src.At(x+1, y) ||
					src.At(x, y-1) ||
					src.At(x, y+1) {
					out.Set(x, y, true)
				}
			}
		}
	}
	return out
}

// Shrink contracts the mask by n pixels: a pixel stays only if all four of
// its neighbours were in. The canvas edge counts as out, so a mask that
// touches the edge pulls away from it. It returns a new mask.
func (m *Mask) Shrink(n int) *Mask {
	out := m.Clone()
	for step := 0; step < n; step++ {
		src := out.Clone()
		for y := 0; y < m.h; y++ {
			for x := 0; x < m.w; x++ {
				all := src.At(x, y) && src.At(x-1, y) &&
					src.At(x+1, y) && src.At(x, y-1) &&
					src.At(x, y+1)
				out.Set(x, y, all)
			}
		}
	}
	return out
}

// Bounds is the smallest pixel rectangle holding every pixel that is in, or
// an empty rectangle.
func (m *Mask) Bounds() image.Rectangle {
	var r image.Rectangle
	first := true
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			if !m.At(x, y) {
				continue
			}
			px := image.Rect(x, y, x+1, y+1)
			if first {
				r, first = px, false
			} else {
				r = r.Union(px)
			}
		}
	}
	return r
}

// MaskFromRect makes a mask holding one rectangle.
func MaskFromRect(w, h int, r image.Rectangle) *Mask {
	m := NewMask(w, h)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			m.Set(x, y, true)
		}
	}
	return m
}

// MaskFromPart makes a mask of every pixel carrying one part. This is what
// makes the parts model pay twice: every part is already a mask over the
// pixels that carry it, so selecting one costs nothing.
func MaskFromPart(c *Canvas, id part.PartID) *Mask {
	m := NewMask(c.Width(), c.Height())
	for i, p := range c.Pixels() {
		if p == id {
			m.words[i/64] |= uint64(1) << uint(i%64)
		}
	}
	return m
}

// Threshold turns a field's values into a mask: pixels at or above the cut
// are in. The values are row-major, 0-1, the mask's size.
func (m *Mask) Threshold(values []float64, cut float64) {
	n := m.w * m.h
	for i, v := range values {
		if i >= n {
			break
		}
		m.Set(i%m.w, i/m.w, v >= cut)
	}
}
