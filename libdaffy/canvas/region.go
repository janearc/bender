package canvas

// A region is a named area with layout metadata. It carries a cell-aligned
// bounding box, because a pane in a terminal interface is cell-aligned and a
// half-cell pane does not exist, and an optional pixel mask for a shape that is
// not a rectangle.
//
// The two coordinate systems are deliberate and are not collapsed.
//
// Phase one records regions, draws their bounds, uses them as masks and
// exports them as data. Nothing generates code from them: there is no real
// target yet and an exporter written against a guess gets thrown away.

import (
	"fmt"
	"image"
	"sort"
)

// Region is a named area.
type Region struct {
	ID    int
	Name  string            // "sidebar", "status", "canvas"
	Kind  string            // free text: pane, gutter, header, rule
	Cells image.Rectangle   // bounding box, in cells
	Mask  *Mask             // optional, per pixel, for a non-rectangle
	Notes map[string]string // whatever the author wants to record
}

// Clone copies the region, mask and notes included.
func (r Region) Clone() Region {
	cp := r
	if r.Mask != nil {
		cp.Mask = r.Mask.Clone()
	}
	if r.Notes != nil {
		cp.Notes = map[string]string{}
		for k, v := range r.Notes {
			cp.Notes[k] = v
		}
	}
	return cp
}

// Contains reports whether a cell is inside the region: inside its mask if
// it has one, else inside its box.
func (r Region) Contains(x, cy int) bool {
	if !image.Pt(x, cy).In(r.Cells) {
		return false
	}
	if r.Mask == nil {
		return true
	}
	return r.Mask.At(x, cy*2) || r.Mask.At(x, cy*2+1)
}

// AsMask is the region as a stencil over a canvas of a size: its mask if it
// has one, else its box filled.
func (r Region) AsMask(w, h int) *Mask {
	if r.Mask != nil {
		return r.Mask.Clone()
	}
	return MaskFromRect(w, h, CellToPixel(r.Cells))
}

// NoteKeys lists the note keys sorted, so output is stable.
func (r Region) NoteKeys() []string {
	keys := make([]string, 0, len(r.Notes))
	for k := range r.Notes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Regions is a document's regions, in creation order.
type Regions struct {
	list   []*Region
	nextID int
}

// NewRegions makes an empty list.
func NewRegions() *Regions { return &Regions{nextID: 1} }

// Add registers a region under a fresh id, or under its own id if it has
// one, and returns it. A name already in use is an error.
func (rs *Regions) Add(r Region) (*Region, error) {
	if r.Name == "" {
		return nil, fmt.Errorf("region: a region needs a name")
	}
	if rs.Lookup(r.Name) != nil {
		return nil, fmt.Errorf("region: %q already exists", r.Name)
	}
	if r.ID == 0 {
		r.ID = rs.nextID
	}
	if r.ID >= rs.nextID {
		rs.nextID = r.ID + 1
	}
	cp := r.Clone()
	rs.list = append(rs.list, &cp)
	return &cp, nil
}

// Get finds a region by id, or nil.
func (rs *Regions) Get(id int) *Region {
	for _, r := range rs.list {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// Lookup finds a region by name, or nil.
func (rs *Regions) Lookup(name string) *Region {
	for _, r := range rs.list {
		if r.Name == name {
			return r
		}
	}
	return nil
}

// At finds the last-added region containing a cell, or nil.
func (rs *Regions) At(x, cy int) *Region {
	for i := len(rs.list) - 1; i >= 0; i-- {
		if rs.list[i].Contains(x, cy) {
			return rs.list[i]
		}
	}
	return nil
}

// Remove deletes a region by id and reports whether it was there.
func (rs *Regions) Remove(id int) bool {
	for i, r := range rs.list {
		if r.ID == id {
			rs.list = append(rs.list[:i], rs.list[i+1:]...)
			return true
		}
	}
	return false
}

// All lists the regions in order.
func (rs *Regions) All() []*Region { return rs.list }

// Len is how many regions there are.
func (rs *Regions) Len() int { return len(rs.list) }

// Clone copies the list.
func (rs *Regions) Clone() *Regions {
	n := &Regions{nextID: rs.nextID}
	for _, r := range rs.list {
		cp := r.Clone()
		n.list = append(n.list, &cp)
	}
	return n
}

// MaskToRegion promotes a mask to a region: the bounding box of the mask in
// cells, and the mask itself kept for the shape. An empty mask is an error.
func MaskToRegion(name, kind string, m *Mask) (Region, error) {
	if m == nil || m.Empty() {
		return Region{}, fmt.Errorf("region: the mask is empty")
	}
	return Region{
		Name:  name,
		Kind:  kind,
		Cells: PixelToCell(m.Bounds()),
		Mask:  m.Clone(),
	}, nil
}

// Set replaces a region's fields, keeping its id. A rename onto a name in
// use, or onto no name, is an error, for the same reason Add refuses one.
func (rs *Regions) Set(id int, after Region) error {
	r := rs.Get(id)
	if r == nil {
		return fmt.Errorf("region: no region %d", id)
	}
	if after.Name == "" {
		return fmt.Errorf("region: a region needs a name")
	}
	if other := rs.Lookup(after.Name); other != nil && other.ID != id {
		return fmt.Errorf("region: %q already exists", after.Name)
	}
	after.ID = id
	*r = after.Clone()
	return nil
}
