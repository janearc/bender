package render

import (
	"github.com/janearc/libdaffy/canvas"
	"github.com/janearc/libdaffy/part"
	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// The half-block rule, and nothing else.
//
// A cell carries one glyph, one foreground and one background. The half-block
// trick spends the foreground and background on colour to buy two pixels. The
// rule that matters is what happens to the half of a cell that has not been
// painted: it MUST NOT be painted.
//
// The underlay image sits at z=-1, above each cell's background colour and
// below its text, so an unpainted half is given no background at all and the
// image shows through. The painted half always goes in the foreground, because
// the foreground is the opaque one.
//
// Measured in reference/spike/draw.go, 2026-09-07. The first version filled
// the unpainted half with an opaque dark and punched holes through the image.

const (
	// UpperHalf is the glyph whose foreground is the top pixel.
	UpperHalf = "▀"
	// LowerHalf is the glyph whose foreground is the bottom pixel.
	LowerHalf = "▄"
)

// Paint is what one terminal cell should show. Set false means write nothing:
// the cell belongs to whatever is beneath. HasBg false means leave the
// background unset so the underlay shows through that half.
type Paint struct {
	Glyph string
	Fg    srgb.RGB
	Bg    srgb.RGB
	HasBg bool
	Set   bool
}

// HalfBlock applies the rule to one cell. A nil pointer is an unpainted half.
func HalfBlock(top, bot *srgb.RGB) Paint {
	switch {
	case top == nil && bot == nil:
		return Paint{}
	case top != nil && bot != nil:
		return Paint{
			Glyph: UpperHalf,
			Fg:    *top,
			Bg:    *bot,
			HasBg: true,
			Set:   true,
		}
	case top != nil:
		return Paint{Glyph: UpperHalf, Fg: *top, Set: true}
	default:
		return Paint{Glyph: LowerHalf, Fg: *bot, Set: true}
	}
}

// Resolver gives the colour a part id draws in, and false for a pixel that is
// not painted, which is what None is.
type Resolver func(part.PartID) (srgb.RGB, bool)

// CellPaint renders one cell of a canvas: column x, cell row cy.
func CellPaint(c *canvas.Canvas, x, cy int, res Resolver) Paint {
	var top, bot *srgb.RGB
	if col, ok := res(c.At(x, cy*2)); ok {
		top = &col
	}
	if col, ok := res(c.At(x, cy*2+1)); ok {
		bot = &col
	}
	return HalfBlock(top, bot)
}

// Resolve builds a Resolver from a part registry and a colourway's roles. None
// is not painted; a part the roles do not bind is painted in the placeholder,
// because an unbound part is a thing to see, not a thing to hide.
func Resolve(parts *part.Parts, roles *css.Sheet) Resolver {
	return ResolveWith(
		func() (*part.Parts, *css.Sheet) { return parts, roles },
	)
}

// ResolveWith builds a Resolver that asks for the registry and roles on
// every call. A resolver is the one object every drawing path holds, so
// it must follow a colourway swap rather than keep the one it was made with.
func ResolveWith(current func() (*part.Parts, *css.Sheet)) Resolver {
	return func(id part.PartID) (srgb.RGB, bool) {
		if id == part.None {
			return srgb.RGB{}, false
		}
		parts, roles := current()
		p := parts.Get(id)
		if p == nil {
			return Placeholder, true
		}
		col, _ := Bind(p.Ink, roles)
		return col, true
	}
}

// Own decides who draws a cell. A cell is owned by exactly one layer: a glyph
// there hides the pixels beneath; no glyph, the pixels draw; nothing in either,
// nothing is written and the underlay shows through. A glyph with no background
// part leaves the background unset, so the image shows behind the letterforms.
func Own(glyph canvas.Glyph, hasGlyph bool, pixels Paint, res Resolver) Paint {
	if !hasGlyph {
		return pixels
	}
	p := Paint{Glyph: glyph.Ch, Set: true}
	p.Fg, _ = res(glyph.Fg)
	if glyph.Bg != part.None {
		p.Bg, p.HasBg = res(glyph.Bg)
	}
	return p
}

// Placeholder is the loud colour an unbound part draws in. Visible rather
// than plausible, on purpose: a silently defaulted part would look like a
// choice.
var Placeholder = srgb.RGB8(0xff, 0x00, 0xff)

// Bind is the colour an ink draws in against a colourway's roles: a
// pinned ink is its own colour whatever the roles say; a role the sheet
// has is that colour; anything else is the placeholder, and false.
func Bind(ink part.Ink, roles *css.Sheet) (srgb.RGB, bool) {
	if ink.Pinned {
		c, _ := srgb.FromSwatch(ink.Literal)
		return c, true
	}
	if roles == nil {
		return Placeholder, false
	}
	s, ok := roles.Get(ink.Role)
	if !ok {
		return Placeholder, false
	}
	c, _ := srgb.FromSwatch(s)
	return c, true
}

// UnboundParts lists the parts whose role the sheet does not bind, in
// registry order, so an editor can say so before a save.
func UnboundParts(parts *part.Parts, roles *css.Sheet) []part.Part {
	var out []part.Part
	for _, p := range parts.All() {
		if _, ok := Bind(p.Ink, roles); !ok {
			out = append(out, p)
		}
	}
	return out
}
