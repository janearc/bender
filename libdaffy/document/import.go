package document

import (
	"fmt"
	"sort"
	"strings"

	"github.com/janearc/libdaffy/sprite"
)

import (
	"github.com/janearc/libdaffy/part"
)

// ImportSprite makes a document from a sprite: one pixel per art cell, one
// part per role used, named for the role and drawn in a colourway role of
// the same name, so the drawing binds to a colourway like any other.
func ImportSprite(s *sprite.Sprite) (*Document, error) {
	d := NewDocument(s.Name, s.Width(), s.Height())
	parts := map[byte]part.PartID{}
	for y, row := range s.Art {
		for x := 0; x < len(row); x++ {
			c := row[x]
			if c == '.' {
				continue
			}
			id, ok := parts[c]
			if !ok {
				rc, _ := sprite.RoleByChar(c)
				p, err := d.Parts.Add(
					rc.Name,
					part.Role(rc.Name),
				)
				if err != nil {
					return nil, err
				}
				p.Role = c
				id = p.ID
				parts[c] = id
			}
			d.Canvas.Set(x, y, id)
		}
	}
	d.Sprite = &sprite.Sprite{Name: s.Name, Raw: s.Raw}
	return d, nil
}

// ExportSprite writes the document's canvas as a sprite. Every part in the
// drawing must map to a role in the alphabet, by its role character or by
// its name; a part that does not is named in the error, and no character is
// invented for it. Poses from an imported sprite are preserved.
func (d *Document) ExportSprite() (*sprite.Sprite, error) {
	chars := map[part.PartID]byte{}
	var offending []string
	for _, p := range d.Parts.All() {
		if d.Canvas.Count(p.ID) == 0 {
			continue
		}
		switch {
		case p.Role != 0 && p.Role != '.':
			if _, ok := sprite.RoleByChar(p.Role); !ok {
				offending = append(
					offending,
					fmt.Sprintf("%s (%q)", p.Name, p.Role),
				)
				continue
			}
			chars[p.ID] = p.Role
		default:
			rc, ok := sprite.RoleByName(p.Name)
			if !ok || rc.Char == '.' {
				offending = append(offending, p.Name)
				continue
			}
			chars[p.ID] = rc.Char
		}
	}
	// A part id the registry does not know cannot be mapped either.
	seen := map[part.PartID]bool{}
	for _, id := range d.Canvas.Pixels() {
		if id != part.None && d.Parts.Get(id) == nil && !seen[id] {
			seen[id] = true
			offending = append(
				offending,
				fmt.Sprintf("part %d (missing)", id),
			)
		}
	}
	if len(offending) > 0 {
		sort.Strings(offending)
		return nil, fmt.Errorf(
			"sprite: these parts are outside the role alphabet "+
				"and no character is invented for them: %s",
			strings.Join(offending, ", "),
		)
	}
	name := d.Name
	if d.Sprite != nil && d.Sprite.Name != "" {
		name = d.Sprite.Name
	}
	s := &sprite.Sprite{Name: name}
	if d.Sprite != nil {
		s.Raw = d.Sprite.Raw
	}
	for y := 0; y < d.Canvas.Height(); y++ {
		row := make([]byte, d.Canvas.Width())
		for x := range row {
			row[x] = '.'
			if id := d.Canvas.At(x, y); id != part.None {
				row[x] = chars[id]
			}
		}
		s.Art = append(s.Art, string(row))
	}
	return s, nil
}
