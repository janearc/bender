package render

import "strconv"

// fgSGR is the escape that sets a cell's 24-bit foreground.
func fgSGR(p Paint) string {
	r, g, b := p.Fg.Bytes()
	return "\x1b[38;2;" + strconv.Itoa(int(r)) + ";" + strconv.Itoa(
		int(g),
	) + ";" + strconv.Itoa(int(b)) + "m"
}

// bgSGR is the escape that sets a cell's background: 24-bit, or the
// terminal's own when the cell has none.
func bgSGR(p Paint) string {
	if !p.HasBg {
		return "\x1b[49m"
	}
	r, g, b := p.Bg.Bytes()
	return "\x1b[48;2;" + strconv.Itoa(int(r)) + ";" + strconv.Itoa(
		int(g),
	) + ";" + strconv.Itoa(int(b)) + "m"
}

// colours is the terminal's colour state as the writer last set it, so a
// cell sends only the half that changed.
type colours struct{ fg, bg string }

// set appends what a cell needs beyond the state, and updates it.
func (c *colours) set(b []byte, p Paint) []byte {
	if fg := fgSGR(p); fg != c.fg {
		b = append(b, fg...)
		c.fg = fg
	}
	if bg := bgSGR(p); bg != c.bg {
		b = append(b, bg...)
		c.bg = bg
	}
	return b
}

// reset appends a reset if any colour is set, and forgets the state.
func (c *colours) reset(b []byte) []byte {
	if c.fg != "" || c.bg != "" {
		b = append(b, "\x1b[0m"...)
		c.fg, c.bg = "", ""
	}
	return b
}

// Text is a frame as plain characters: the glyph where a cell is painted,
// a space where it is not, trailing spaces dropped, one line per row.
func Text(f Frame) string {
	out := ""
	for y := 0; y < f.h; y++ {
		line := ""
		for x := 0; x < f.w; x++ {
			p := f.At(x, y)
			if !p.Set {
				line += " "
				continue
			}
			line += p.Glyph
		}
		out += trimRight(line) + "\n"
	}
	return out
}

// trimRight drops the trailing spaces a text export need not keep.
func trimRight(s string) string {
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}

// ANSI is a frame as 24-bit sgr, colours set only where they change and a
// reset at the end of every line. It is a file to cat, not a file to edit.
func ANSI(f Frame) string {
	var b []byte
	for y := 0; y < f.h; y++ {
		var c colours
		for x := 0; x < f.w; x++ {
			p := f.At(x, y)
			if !p.Set {
				b = c.reset(b)
				b = append(b, ' ')
				continue
			}
			b = c.set(b, p)
			b = append(b, p.Glyph...)
		}
		b = append(b, "\x1b[0m\n"...)
	}
	return string(b)
}
