package render

import (
	"io"
	"strconv"
)

// Writer turns frames into bytes for a terminal, sending only what changed
// since the frame before: a cursor move per run of changed cells, colours only
// where they differ from the cell just written, and the whole frame inside a
// synchronized-output bracket so it lands at once and never tears.
//
// It is the one thing in the suite that knows a terminal's escapes for
// painting; painters produce frames and stop.
type Writer struct {
	out  io.Writer
	last Frame
	// Origin is the terminal cell the frame's top left lands on, from
	// zero. A frame drawn into a region of a larger screen sets it.
	Origin struct{ X, Y int }
	dirty  bool
}

// NewWriter makes a writer whose first frame paints every cell.
func NewWriter(out io.Writer) *Writer {
	return &Writer{out: out, dirty: true}
}

// Invalidate makes the next frame paint every cell, for after a resize
// or anything else that may have disturbed the screen.
func (w *Writer) Invalidate() { w.dirty = true }

// Write sends a frame. A frame that is the same as the last sends nothing.
func (w *Writer) Write(f Frame) error {
	full := w.dirty || w.last.w != f.w || w.last.h != f.h
	if !full && w.last.Equal(f) {
		return nil
	}
	var b []byte
	b = append(b, "\x1b[?2026h\x1b[0m"...)
	var c colours
	for y := 0; y < f.h; y++ {
		run := false
		for x := 0; x < f.w; x++ {
			p := f.At(x, y)
			if !full && p == w.last.At(x, y) {
				run = false
				continue
			}
			if !run {
				b = append(b, "\x1b["...)
				b = strconv.AppendInt(
					b,
					int64(w.Origin.Y+y+1),
					10,
				)
				b = append(b, ';')
				b = strconv.AppendInt(
					b,
					int64(w.Origin.X+x+1),
					10,
				)
				b = append(b, 'H')
				run = true
			}
			if !p.Set {
				b = c.reset(b)
				b = append(b, ' ')
				continue
			}
			b = c.set(b, p)
			b = append(b, p.Glyph...)
		}
	}
	b = append(b, "\x1b[0m\x1b[?2026l"...)
	if _, err := w.out.Write(b); err != nil {
		return err
	}
	w.last, w.dirty = f.Clone(), false
	return nil
}
