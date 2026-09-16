package document

import (
	"image"
	"strconv"

	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"

	"github.com/janearc/libdaffy/canvas"
	"github.com/janearc/libdaffy/part"
	"github.com/janearc/libdaffy/render"
	"github.com/janearc/libdaffy/sprite"
)

// Document is one drawing as a file holds it: a container of members and
// nothing else. It has no memory of how it got this way; an editing
// session keeps that in a history of its own, and a viewer keeps none.
type Document struct {
	Name   string
	Canvas *canvas.Canvas
	Parts  *part.Parts
	// Colourway is the roles and ramps the parts bind against, libtheme's;
	// ColourwayName is what the file calls it. A colourway is never
	// changed in place: WithRole makes a new one, so the value copies by
	// assignment.
	Colourway     css.Colourway
	ColourwayName string
	// Mask is the active stencil, or nil for none. Every stroke is clipped
	// by it.
	Mask *canvas.Mask
	// Layer is the cell layer: glyphs and text boxes above the pixels.
	Layer *canvas.Layer
	// Regions are the named areas.
	Regions *canvas.Regions
	// Underlay is the path of the reference image, or empty. The image is
	// not part of the document; the reference to it is.
	Underlay string
	// Sprite carries what an imported .sprite had beyond its art, poses
	// included, so a round trip preserves them. Nil for a drawing that did
	// not come from one.
	Sprite *sprite.Sprite
}

// NewDocument makes an empty drawing of a size in pixels, with an empty
// colourway and no parts.
func NewDocument(name string, w, h int) *Document {
	return &Document{
		Name:          name,
		Canvas:        canvas.NewCanvas(w, h),
		Parts:         part.NewParts(),
		Colourway:     EmptyColourway(),
		ColourwayName: "untitled",
		Layer:         canvas.NewLayer(w, h/2),
		Regions:       canvas.NewRegions(),
	}
}

// EmptyColourway is a colourway with no roles and no ramps, the way
// libtheme's reader would build one from an empty sheet.
func EmptyColourway() css.Colourway {
	return css.Colourway{
		Roles: css.New(),
		Ramps: map[string]functions.Ramp{},
	}
}

// WithRole is a colourway with one role bound to a colour, or, with the
// zero swatch and remove set, with that role left out. The one given is
// not touched: a sheet only grows, so a copy is made and the value can
// be recorded by assignment.
func WithRole(
	cw css.Colourway,
	role string,
	c swatch.Swatch,
	remove bool,
) css.Colourway {
	out := css.Colourway{Roles: css.New(), Ramps: cw.Ramps, Order: cw.Order}
	if cw.Roles != nil {
		for _, name := range cw.Roles.Names() {
			if name == role {
				continue
			}
			if s, ok := cw.Roles.Get(name); ok {
				out.Roles.Set(name, s)
			}
		}
	}
	if !remove {
		out.Roles.Set(role, c)
	}
	return out
}

// Resolver is the colour lookup for rendering this document. It reads the
// document's parts and colourway at each call, so a resolver taken before
// a colourway swap follows the swap: the lifetime rule is that there is none.
func (d *Document) Resolver() render.Resolver {
	return render.ResolveWith(func() (*part.Parts, *css.Sheet) {
		return d.Parts, d.Colourway.Roles
	})
}

// Frame is the drawing as painted cells, every layer considered: what a
// writer sends to a terminal, and what an export reads.
func (d *Document) Frame() render.Frame {
	f := render.NewFrame(d.Canvas.Width(), d.Canvas.Rows())
	for cy := 0; cy < d.Canvas.Rows(); cy++ {
		for x := 0; x < d.Canvas.Width(); x++ {
			f.Set(x, cy, d.PaintCell(x, cy))
		}
	}
	return f
}

// PaintCell is what one canvas cell shows with every layer considered.
func (d *Document) PaintCell(x, cy int) render.Paint {
	res := d.Resolver()
	g, ok := d.Layer.At(x, cy)
	return render.Own(g, ok, render.CellPaint(d.Canvas, x, cy, res), res)
}

// Stroke is one gesture's worth of pixel writes, clipped by the active mask
// and recorded for undo. Plot into it, then Commit.
type Stroke struct {
	doc    *Document
	id     part.PartID
	bounds image.Rectangle
	// Clip further restricts writes, on top of the mask. Empty means the
	// whole canvas.
	Clip image.Rectangle
	// MaskResolved says the caller has already worked the mask into the
	// pixels it is plotting, so the stroke must not apply it a second time.
	//
	// It is not an exemption from the mask, and the difference matters. The
	// one operation that needs this is erasing everything outside the mask:
	//
	// it walks the canvas, skips what the mask covers, and plots the rest.
	// Applying the mask again on top of that rejects every pixel it just
	// chose, and the operation silently does nothing. The mask still
	// governs; the caller has simply read it from the other side.
	//
	// So a caller sets this only when it has resolved the mask itself. A
	// caller that wants a plot to land somewhere the mask does not cover
	// wants something else, and there is no field for it on purpose.
	MaskResolved bool
}

// Stroke starts a stroke that paints one part through the mask. Whether
// the stroke is remembered is the session's business, not the
// document's: an editor opens a history gesture on the canvas around it.
func (d *Document) Stroke(id part.PartID) *Stroke {
	return &Stroke{doc: d, id: id, Clip: d.Canvas.Bounds()}
}

// Plot paints one pixel, if the mask and clip allow it.
func (s *Stroke) Plot(x, y int) { s.PlotPart(x, y, s.id) }

// PlotPart paints one pixel with a specific part, for tools that paint more
// than one, if the mask and clip allow it.
func (s *Stroke) PlotPart(x, y int, id part.PartID) {
	if !image.Pt(x, y).In(s.Clip) {
		return
	}
	if !s.MaskResolved && s.doc.Mask != nil && !s.doc.Mask.At(x, y) {
		return
	}
	if s.doc.Canvas.Set(x, y, id) {
		px := image.Rect(x, y, x+1, y+1)
		if s.bounds.Empty() {
			s.bounds = px
		} else {
			s.bounds = s.bounds.Union(px)
		}
	}
}

// Bounds is the pixel rectangle the stroke has changed so far.
func (s *Stroke) Bounds() image.Rectangle { return s.bounds }

// PartFor is the part with a pinned ink of this colour, made if none
// exists, named by its hex and, if that name is taken by a different
// colour, by a number after it. Choosing from the picker makes such a
// part; recording the making is the session's business.
func (d *Document) PartFor(c swatch.Swatch) *part.Part {
	for _, p := range d.Parts.All() {
		if p.Ink.Pinned && sameSwatch(p.Ink.Literal, c) {
			return d.Parts.Get(p.ID)
		}
	}
	name := hexOf(c)
	for i := 2; d.Parts.Lookup(name) != nil; i++ {
		name = hexOf(c) + "-" + strconv.Itoa(i)
	}
	p, err := d.Parts.Add(name, part.Literal(c))
	if err != nil {
		return nil
	}
	return p
}

// sameSwatch is whether two swatches are the same colour to a display,
// within a byte per channel, which is what a file can tell apart.
func sameSwatch(a, b swatch.Swatch) bool {
	x, _ := srgb.FromSwatch(a)
	y, _ := srgb.FromSwatch(b)
	return x.Equal(y)
}
