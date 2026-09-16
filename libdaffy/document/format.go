package document

// Daffy's own document format. It is JSON, it is new, and it owes nothing
// to ANSI, XBin or PCBoard. Pixels and masks are run-length encoded as
// pairs, which keeps a diff of a drawing readable: shape changes and
// palette changes show up separately, which is the point of parts.

import (
	"github.com/janearc/libdaffy/canvas"
	"github.com/janearc/libdaffy/part"
	"github.com/janearc/libdaffy/render"
	"github.com/janearc/libdaffy/sprite"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"image"

	"encoding/json"
	"fmt"
	"io"
	"os"
)

// FormatVersion is written into every file and checked on the way in.
const FormatVersion = 1

// LayoutVersion is written into every `export json` payload. It starts at 1
// alongside FormatVersion and moves independently of it: the document format
// and the interchange format are versioned separately because they change for
// different reasons and are read by different people.
const LayoutVersion = 1

// fileDoc is the on-disk shape.
type fileDoc struct {
	Daffy     int           `json:"daffy"`
	Name      string        `json:"name"`
	Canvas    fileCanvas    `json:"canvas"`
	Parts     []filePart    `json:"parts"`
	Colourway fileColourway `json:"theme"`
	Cells     []fileGlyph   `json:"cells,omitempty"`
	Boxes     []fileBox     `json:"boxes,omitempty"`
	Regions   []fileRegion  `json:"regions,omitempty"`
	Mask      [][2]int      `json:"mask,omitempty"`
	Underlay  string        `json:"underlay,omitempty"`
	Sprite    *fileSprite   `json:"sprite,omitempty"`
}

type fileCanvas struct {
	W      int      `json:"w"`
	H      int      `json:"h"`
	Pixels [][2]int `json:"pixels"` // run-length: [part, count]
}

type filePart struct {
	ID    part.PartID `json:"id"`
	Name  string      `json:"name"`
	Role  string      `json:"role,omitempty"`
	Color string      `json:"color,omitempty"` // hex, when pinned
	Char  string      `json:"char,omitempty"`  // sprite role character
}

type fileColourway struct {
	Name      string              `json:"name"`
	Roles     map[string]string   `json:"roles"`
	Gradients map[string][]string `json:"gradients,omitempty"`
}

type fileGlyph struct {
	X  int         `json:"x"`
	Y  int         `json:"y"`
	Ch string      `json:"ch"`
	Fg part.PartID `json:"fg"`
	Bg part.PartID `json:"bg,omitempty"`
}

type fileBox struct {
	ID    int         `json:"id"`
	Box   [4]int      `json:"box"`
	Text  string      `json:"text"`
	Align string      `json:"align"`
	Fg    part.PartID `json:"fg"`
	Bg    part.PartID `json:"bg,omitempty"`
}

type fileRegion struct {
	ID    int               `json:"id"`
	Name  string            `json:"name"`
	Kind  string            `json:"kind"`
	Cells [4]int            `json:"cells"`
	Mask  [][2]int          `json:"mask,omitempty"`
	Notes map[string]string `json:"notes,omitempty"`
}

type fileSprite struct {
	Name string   `json:"name"`
	Raw  []string `json:"raw"`
}

// rle encodes a run of ids.
func rle(px []part.PartID) [][2]int {
	var out [][2]int
	for i := 0; i < len(px); {
		j := i
		for j < len(px) && px[j] == px[i] {
			j++
		}
		out = append(out, [2]int{int(px[i]), j - i})
		i = j
	}
	return out
}

// unrle decodes into a slice of a known length, refusing a mismatch.
func unrle(runs [][2]int, n int) ([]part.PartID, error) {
	out := make([]part.PartID, 0, n)
	for _, r := range runs {
		if r[1] < 0 || r[0] < 0 || r[0] > 0xffff {
			return nil, fmt.Errorf("bad run %v", r)
		}
		for i := 0; i < r[1]; i++ {
			out = append(out, part.PartID(r[0]))
		}
	}
	if len(out) != n {
		return nil, fmt.Errorf("run length %d, want %d", len(out), n)
	}
	return out, nil
}

// rleBits encodes a mask.
func rleBits(bits []bool) [][2]int {
	px := make([]part.PartID, len(bits))
	for i, b := range bits {
		if b {
			px[i] = 1
		}
	}
	return rle(px)
}

// unrleBits decodes a mask.
func unrleBits(runs [][2]int, n int) ([]bool, error) {
	px, err := unrle(runs, n)
	if err != nil {
		return nil, err
	}
	out := make([]bool, n)
	for i, p := range px {
		out[i] = p != 0
	}
	return out, nil
}

// rectOf is a rect as the four numbers the file keeps.
func rectOf(r image.Rectangle) [4]int {
	return [4]int{r.Min.X, r.Min.Y, r.Max.X, r.Max.Y}
}

// ofRect is rectOf undone.
func ofRect(a [4]int) image.Rectangle {
	return image.Rect(a[0], a[1], a[2], a[3])
}

// Encode writes the document as JSON.
func (d *Document) Encode(w io.Writer) error {
	f := fileDoc{
		Daffy: FormatVersion,
		Name:  d.Name,
		Canvas: fileCanvas{
			W:      d.Canvas.Width(),
			H:      d.Canvas.Height(),
			Pixels: rle(d.Canvas.Pixels()),
		},
		Colourway: fileColourway{
			Name:  d.ColourwayName,
			Roles: map[string]string{},
		},
		Underlay: d.Underlay,
	}
	for _, p := range d.Parts.All() {
		fp := filePart{ID: p.ID, Name: p.Name}
		if p.Ink.Pinned {
			fp.Color = hexOf(p.Ink.Literal)
		} else {
			fp.Role = p.Ink.Role
		}
		if p.Role != 0 {
			fp.Char = string(p.Role)
		}
		f.Parts = append(f.Parts, fp)
	}
	for _, role := range d.Colourway.Roles.Names() {
		if c, ok := d.Colourway.Roles.Get(role); ok {
			f.Colourway.Roles[role] = hexOf(c)
		}
	}
	cells := d.Layer.Cells
	for y := 0; y < cells.Height(); y++ {
		for x := 0; x < cells.Width(); x++ {
			if g := cells.At(x, y); !g.Empty() {
				f.Cells = append(
					f.Cells,
					fileGlyph{
						X:  x,
						Y:  y,
						Ch: g.Ch,
						Fg: g.Fg,
						Bg: g.Bg,
					},
				)
			}
		}
	}
	for _, b := range d.Layer.Boxes {
		f.Boxes = append(
			f.Boxes,
			fileBox{
				ID:    b.ID,
				Box:   rectOf(b.Box),
				Text:  b.Text,
				Align: b.Align.String(),
				Fg:    b.Fg,
				Bg:    b.Bg,
			},
		)
	}
	for _, r := range d.Regions.All() {
		fr := fileRegion{
			ID:    r.ID,
			Name:  r.Name,
			Kind:  r.Kind,
			Cells: rectOf(r.Cells),
			Notes: r.Notes,
		}
		if r.Mask != nil {
			fr.Mask = rleBits(r.Mask.Bits())
		}
		f.Regions = append(f.Regions, fr)
	}
	if d.Mask != nil {
		f.Mask = rleBits(d.Mask.Bits())
	}
	if d.Sprite != nil {
		f.Sprite = &fileSprite{Name: d.Sprite.Name, Raw: d.Sprite.Raw}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return enc.Encode(f)
}

// Decode reads a document. Anything malformed is an error naming what.
func Decode(r io.Reader) (*Document, error) {
	var f fileDoc
	if err := json.NewDecoder(r).Decode(&f); err != nil {
		return nil, fmt.Errorf("daffy file: %w", err)
	}
	if f.Daffy != FormatVersion {
		return nil, fmt.Errorf(
			"daffy file: version %d, this daffy reads %d",
			f.Daffy,
			FormatVersion,
		)
	}
	if f.Canvas.W < 1 || f.Canvas.H < 1 {
		return nil, fmt.Errorf(
			"daffy file: canvas %d by %d",
			f.Canvas.W,
			f.Canvas.H,
		)
	}
	d := NewDocument(f.Name, f.Canvas.W, f.Canvas.H)
	px, err := unrle(f.Canvas.Pixels, d.Canvas.Width()*d.Canvas.Height())
	if err != nil {
		return nil, fmt.Errorf("daffy file: pixels: %w", err)
	}
	copy(d.Canvas.Pixels(), px)
	for _, fp := range f.Parts {
		p := part.Part{ID: fp.ID, Name: fp.Name}
		if fp.Color != "" {
			c, err := srgb.FromHex(fp.Color)
			if err != nil {
				return nil, fmt.Errorf(
					"daffy file: part %q colour %q",
					fp.Name,
					fp.Color,
				)
			}
			p.Ink = part.Literal(c.Swatch())
		} else {
			p.Ink = part.Role(fp.Role)
		}
		if fp.Char != "" {
			p.Role = fp.Char[0]
		}
		if err := d.Parts.Put(p); err != nil {
			return nil, fmt.Errorf("daffy file: %w", err)
		}
	}
	d.ColourwayName = f.Colourway.Name
	d.Colourway = EmptyColourway()
	for role, hex := range f.Colourway.Roles {
		c, err := srgb.FromHex(hex)
		if err != nil {
			return nil, fmt.Errorf(
				"daffy file: colourway role %q colour %q",
				role,
				hex,
			)
		}
		d.Colourway.Roles.Set(role, c.Swatch())
	}
	for _, g := range f.Cells {
		d.Layer.Cells.Set(
			g.X,
			g.Y,
			canvas.Glyph{Ch: g.Ch, Fg: g.Fg, Bg: g.Bg},
		)
	}
	for _, b := range f.Boxes {
		align := canvas.Left
		switch b.Align {
		case "centre":
			align = canvas.Centre
		case "right":
			align = canvas.Right
		}
		d.Layer.AddBox(
			canvas.TextBox{
				ID:    b.ID,
				Box:   ofRect(b.Box),
				Text:  b.Text,
				Align: align,
				Fg:    b.Fg,
				Bg:    b.Bg,
			},
		)
	}
	for _, fr := range f.Regions {
		r := canvas.Region{
			ID:    fr.ID,
			Name:  fr.Name,
			Kind:  fr.Kind,
			Cells: ofRect(fr.Cells),
			Notes: fr.Notes,
		}
		if fr.Mask != nil {
			bits, err := unrleBits(
				fr.Mask,
				d.Canvas.Width()*d.Canvas.Height(),
			)
			if err != nil {
				return nil, fmt.Errorf(
					"daffy file: region %q mask: %w",
					fr.Name,
					err,
				)
			}
			r.Mask = canvas.NewMask(
				d.Canvas.Width(),
				d.Canvas.Height(),
			)
			r.Mask.Load(bits)
		}
		if _, err := d.Regions.Add(r); err != nil {
			return nil, fmt.Errorf("daffy file: %w", err)
		}
	}
	if f.Mask != nil {
		bits, err := unrleBits(
			f.Mask,
			d.Canvas.Width()*d.Canvas.Height(),
		)
		if err != nil {
			return nil, fmt.Errorf("daffy file: mask: %w", err)
		}
		d.Mask = canvas.NewMask(d.Canvas.Width(), d.Canvas.Height())
		d.Mask.Load(bits)
	}
	d.Underlay = f.Underlay
	if f.Sprite != nil {
		d.Sprite = &sprite.Sprite{
			Name: f.Sprite.Name,
			Raw:  f.Sprite.Raw,
		}
	}
	return d, nil
}

// Save writes the document to a path.
func (d *Document) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := d.Encode(f); err != nil {
		f.Close() //nolint:errcheck // reporting the encode error
		return err
	}
	return f.Close()
}

// Load reads a document from a path.
func Load(path string) (*Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only
	return Decode(f)
}

// --- exports -----------------------------------------------------------------

// Text is the drawing as plain characters, one line per row.
func (d *Document) Text() string { return render.Text(d.Frame()) }

// ANSI is the drawing as 24-bit sgr, a file to cat.
func (d *Document) ANSI() string { return render.ANSI(d.Frame()) }

// Layout is the regions and the colourway as JSON, for whatever consumes them
// next. It is data, not code: nothing here says how a pane is built.
func (d *Document) Layout() ([]byte, error) {
	type region struct {
		Name  string            `json:"name"`
		Kind  string            `json:"kind"`
		Cells [4]int            `json:"cells"`
		Shape bool              `json:"shape,omitempty"` // has a mask
		Notes map[string]string `json:"notes,omitempty"`
	}
	type part struct {
		Name  string `json:"name"`
		Role  string `json:"role,omitempty"`
		Color string `json:"color"`
		Bound bool   `json:"bound"`
	}
	// LayoutVersion rides on the export for the same reason FormatVersion
	// rides on the document, and it matters more: the document has one
	// writer and one reader, both here, while the export is an interface
	// between trees with separate owners.
	//
	// A consumer that finds a version it does not know can say so; one that
	// finds no version at all can only guess, and guesses wrong quietly.
	type layout struct {
		Daffy     int                 `json:"daffy"`
		Name      string              `json:"name"`
		Cols      int                 `json:"cols"`
		Rows      int                 `json:"rows"`
		Colourway string              `json:"theme"`
		Roles     map[string]string   `json:"roles"`
		Parts     []part              `json:"parts"`
		Regions   []region            `json:"regions"`
		Grads     map[string][]string `json:"gradients,omitempty"`
	}
	l := layout{
		Daffy:     LayoutVersion,
		Name:      d.Name,
		Cols:      d.Canvas.Width(),
		Rows:      d.Canvas.Rows(),
		Colourway: d.ColourwayName,
		Roles:     map[string]string{},
		Regions:   []region{},
	}
	for _, role := range d.Colourway.Roles.Names() {
		if c, ok := d.Colourway.Roles.Get(role); ok {
			l.Roles[role] = hexOf(c)
		}
	}
	for _, p := range d.Parts.All() {
		c, bound := render.Bind(p.Ink, d.Colourway.Roles)
		l.Parts = append(
			l.Parts,
			part{
				Name:  p.Name,
				Role:  p.Ink.Role,
				Color: c.Hex(),
				Bound: bound,
			},
		)
	}
	for _, r := range d.Regions.All() {
		l.Regions = append(
			l.Regions,
			region{
				Name:  r.Name,
				Kind:  r.Kind,
				Cells: rectOf(r.Cells),
				Shape: r.Mask != nil,
				Notes: r.Notes,
			},
		)
	}
	return json.MarshalIndent(l, "", " ")
}

// hexOf is a swatch as the hex the file keeps: the display's colour.
func hexOf(s swatch.Swatch) string {
	c, _ := srgb.FromSwatch(s)
	return c.Hex()
}
