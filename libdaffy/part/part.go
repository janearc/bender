package part

// A pixel holds a part, not a colour. The idea is a sprite's: it stores, for
// each pixel, which part of the bird it belongs to and never which colour that
// part is. A colourway supplies the ink.
//
// Daffy's canvas works the same way, so a drawing recolours for free: a
// recolour is a colourway edit, not a pixel edit.

import (
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"

	"fmt"
	"sort"
)

// PartID names a part within one document. Zero is reserved: it means no part,
// which is a transparent pixel.
type PartID uint16

// None is the part id of an unpainted pixel.
const None PartID = 0

// Ink is what a part is drawn with: a role the colourway binds, or a
// colour pinned literally. A literal is not a second mechanism; it is a part
// whose ink does not vary with the colourway.
type Ink struct {
	// Role is the colourway role this part takes its colour from, when
	// Pinned is false. "beak", "surface", "accent": whatever it names.
	Role string
	// Literal is the colour used when Pinned is true.
	Literal swatch.Swatch
	// Pinned is true when the ink is fixed and no colourway applies.
	Pinned bool
}

// Literal is a pinned ink.
func Literal(c swatch.Swatch) Ink { return Ink{Literal: c, Pinned: true} }

// Role is an ink the colourway binds.
func Role(name string) Ink { return Ink{Role: name} }

// String says what the ink is, for the inspector and for reports.
func (i Ink) String() string {
	if i.Pinned {
		c, _ := srgb.FromSwatch(i.Literal)
		return c.Hex()
	}
	return "role " + i.Role
}

// Part is a named set of pixels with one ink.
type Part struct {
	ID   PartID
	Name string
	Ink  Ink
	// Role is the sprite role character this part maps to, or zero. It is
	// used only for .sprite interchange and ignored otherwise.
	Role byte
}

// Parts is the registry of a document's parts. Ids are stable for the life of
// the document: deleting a part leaves a hole rather than renumbering, because
// the canvas holds ids and renumbering would be a pixel edit in disguise.
type Parts struct {
	byID   map[PartID]*Part
	byName map[string]PartID
	next   PartID
}

// NewParts makes an empty registry.
func NewParts() *Parts {
	return &Parts{
		byID:   map[PartID]*Part{},
		byName: map[string]PartID{},
		next:   1,
	}
}

// Add registers a part under a fresh id. A name already in use is an error,
// because two parts with one name would make a role binding ambiguous.
func (p *Parts) Add(name string, ink Ink) (*Part, error) {
	if name == "" {
		return nil, fmt.Errorf("part: a part needs a name")
	}
	if _, dup := p.byName[name]; dup {
		return nil, fmt.Errorf("part: %q already exists", name)
	}
	if p.next == 0 {
		return nil, fmt.Errorf("part: no ids left")
	}
	part := &Part{ID: p.next, Name: name, Ink: ink}
	p.byID[part.ID] = part
	p.byName[name] = part.ID
	p.next++
	return part, nil
}

// Put registers a part at a specific id, for loading a document. It refuses
// an id or a name that is already taken.
func (p *Parts) Put(part Part) error {
	if part.ID == None {
		return fmt.Errorf("part: id 0 is reserved")
	}
	if part.Name == "" {
		return fmt.Errorf("part: a part needs a name")
	}
	if _, dup := p.byID[part.ID]; dup {
		return fmt.Errorf("part: id %d already exists", part.ID)
	}
	if _, dup := p.byName[part.Name]; dup {
		return fmt.Errorf("part: %q already exists", part.Name)
	}
	cp := part
	p.byID[cp.ID] = &cp
	p.byName[cp.Name] = cp.ID
	if cp.ID >= p.next {
		p.next = cp.ID + 1
	}
	return nil
}

// Get returns the part with an id, or nil.
func (p *Parts) Get(id PartID) *Part {
	return p.byID[id]
}

// Lookup returns the part with a name, or nil.
func (p *Parts) Lookup(name string) *Part {
	id, ok := p.byName[name]
	if !ok {
		return nil
	}
	return p.byID[id]
}

// Rename changes a part's name, refusing a name in use.
func (p *Parts) Rename(id PartID, name string) error {
	part := p.byID[id]
	if part == nil {
		return fmt.Errorf("part: no part %d", id)
	}
	if name == "" {
		return fmt.Errorf("part: a part needs a name")
	}
	if other, dup := p.byName[name]; dup && other != id {
		return fmt.Errorf("part: %q already exists", name)
	}
	delete(p.byName, part.Name)
	part.Name = name
	p.byName[name] = id
	return nil
}

// SetInk changes what a part is drawn with.
func (p *Parts) SetInk(id PartID, ink Ink) error {
	part := p.byID[id]
	if part == nil {
		return fmt.Errorf("part: no part %d", id)
	}
	part.Ink = ink
	return nil
}

// Remove deletes a part. The caller is responsible for the pixels that carried
// it; this does not touch a canvas.
func (p *Parts) Remove(id PartID) {
	part := p.byID[id]
	if part == nil {
		return
	}
	delete(p.byName, part.Name)
	delete(p.byID, id)
}

// Len is how many parts are registered.
func (p *Parts) Len() int { return len(p.byID) }

// All returns every part, ordered by id, so output is stable.
func (p *Parts) All() []Part {
	out := make([]Part, 0, len(p.byID))
	for _, part := range p.byID {
		out = append(out, *part)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Clone copies the registry, for undo snapshots and documents.
func (p *Parts) Clone() *Parts {
	c := NewParts()
	for _, part := range p.All() {
		_ = c.Put(part)
	}
	c.next = p.next
	return c
}
