// Package textmap is the map-to-text pipeline, ported from kingfisher's
// textmap.py: cells for one dataset, a window on the world that is either
// flat or a globe, and a frame painted from the two. Nothing here knows what
// a terminal is. It knows H3 cells and floats.
package textmap

import (
	"sort"

	"github.com/uber/h3-go/v4"
)

// MaxRes is H3's ceiling. Nothing renders finer than the data was folded,
// so the real ceiling is per-source and this is only the clamp.
const MaxRes = 15

// CellSource holds one dataset's cells, re-aggregated to any H3 resolution
// on demand. Values are cell -> layer -> value.
type CellSource struct {
	Name   string
	cells  map[h3.Cell]map[string]float64
	kinds  map[string]string // layer -> kind, from the manifest
	native int
	same   bool // every cell is at the native resolution
	cache  map[cacheKey]map[h3.Cell]float64
}

type cacheKey struct {
	res   int
	layer string
}

// NewCellSource takes the cell table as kingfisher serves it: an id to
// either a number or a map of layer to number. A layer kind containing
// A kind containing `EXTENSIVE` means values add when rolled up;
// anything else takes the mean.
func NewCellSource(
	name string,
	cells map[string]any,
	kinds map[string]string,
) *CellSource {
	s := &CellSource{
		Name:  name,
		cells: map[h3.Cell]map[string]float64{},
		kinds: kinds,
		cache: map[cacheKey]map[h3.Cell]float64{},
	}
	if s.kinds == nil {
		s.kinds = map[string]string{}
	}
	for id, row := range cells {
		c := h3.Cell(h3.IndexFromString(id))
		if !c.IsValid() {
			continue
		}
		switch v := row.(type) {
		case float64:
			s.cells[c] = map[string]float64{"value": v}
		case map[string]any:
			keep := map[string]float64{}
			for k, x := range v {
				if f, ok := x.(float64); ok {
					keep[k] = f
				}
			}
			if len(keep) > 0 {
				s.cells[c] = keep
			}
		}
	}
	for c := range s.cells {
		if r := c.Resolution(); r > s.native {
			s.native = r
		}
	}
	s.same = true
	for c := range s.cells {
		if c.Resolution() != s.native {
			s.same = false
			break
		}
	}
	return s
}

// Len is how many cells the source holds.
func (s *CellSource) Len() int { return len(s.cells) }

// NativeRes is the finest resolution in the data: the ceiling for any
// rollup, because nothing can be drawn finer than it was measured.
func (s *CellSource) NativeRes() int { return s.native }

// Layers lists every measure any cell carries, sorted so cycling through
// them is stable.
func (s *CellSource) Layers() []string {
	seen := map[string]bool{}
	for _, row := range s.cells {
		for k := range row {
			seen[k] = true
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// IsExtensive reports whether a layer counts things in the cell, so that
// rolling children into a parent adds them, rather than measuring a
// property of the place, where the parent takes the mean.
func (s *CellSource) IsExtensive(layer string) bool {
	return contains(s.kinds[layer], "EXTENSIVE")
}

// contains is a substring test without importing strings, which this file
// otherwise does not need.
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Table is one layer rolled up to a resolution: parent cell to value.
// Cached, because a frame asks for it on every mouse move and folding a
// few hundred thousand cells through h3 is not free.
func (s *CellSource) Table(res int, layer string) map[h3.Cell]float64 {
	res = max(0, min(res, s.native))
	key := cacheKey{res, layer}
	if t, ok := s.cache[key]; ok {
		return t
	}
	type acc struct {
		total float64
		n     int
	}
	accs := map[h3.Cell]*acc{}
	for c, row := range s.cells {
		v, ok := row[layer]
		if !ok {
			continue
		}
		parent := c
		if c.Resolution() > res {
			p, err := c.Parent(res)
			if err != nil {
				continue
			}
			parent = p
		}
		a := accs[parent]
		if a == nil {
			a = &acc{}
			accs[parent] = a
		}
		a.total += v
		a.n++
	}
	extensive := s.IsExtensive(layer)
	t := make(map[h3.Cell]float64, len(accs))
	for parent, a := range accs {
		if extensive {
			t[parent] = a.total
		} else {
			t[parent] = a.total / float64(a.n)
		}
	}
	s.cache[key] = t
	return t
}

// Sampler answers, for a coordinate, which cell contains it and what that
// cell's value is. A source folded at one resolution is looked up in its rollup
// at the resolution asked for.
//
// A mixed source, a coarse basemap beside fine lidar, is looked up in its raw
// cells, walking coarser on a miss, because the cell containing a point is not
// always at the resolution asked for, and a coarse cell keeps its own value.
// The cell returned is the one that answered, so a caller can name it.
func (s *CellSource) Sampler(
	res int,
	layer string,
) func(lat, lon float64) (h3.Cell, float64, bool) {
	res = max(0, min(res, s.native))
	if s.uniform() {
		table := s.Table(res, layer)
		return func(lat, lon float64) (h3.Cell, float64, bool) {
			c, err := h3.LatLngToCell(
				h3.LatLng{Lat: lat, Lng: lon},
				res,
			)
			if err != nil {
				return 0, 0, false
			}
			v, ok := table[c]
			return c, v, ok
		}
	}
	raw := map[h3.Cell]float64{}
	finest := 0
	for c, row := range s.cells {
		if v, ok := row[layer]; ok {
			raw[c] = v
			finest = max(finest, c.Resolution())
		}
	}
	top := min(res, finest)
	return func(lat, lon float64) (h3.Cell, float64, bool) {
		for r := top; r >= 0; r-- {
			c, err := h3.LatLngToCell(
				h3.LatLng{Lat: lat, Lng: lon},
				r,
			)
			if err != nil {
				continue
			}
			if v, ok := raw[c]; ok {
				return c, v, true
			}
		}
		return 0, 0, false
	}
}

// uniform reports whether every cell is at the native resolution, decided
// once when the source is made rather than on every sample.
func (s *CellSource) uniform() bool { return s.same }

// Range is the least and greatest value a layer holds across every cell,
// for pinning a colour scale to everything fetched rather than to what is
// on screen.
func (s *CellSource) Range(layer string) (lo, hi float64, ok bool) {
	for _, row := range s.cells {
		v, has := row[layer]
		if !has {
			continue
		}
		if !ok {
			lo, hi, ok = v, v, true
			continue
		}
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	return lo, hi, ok
}

// Values is every value a layer holds, for a scale that needs more than
// the ends: a floor and a quantile cannot be taken from a range.
func (s *CellSource) Values(layer string) []float64 {
	out := make([]float64, 0, len(s.cells))
	for _, row := range s.cells {
		if v, has := row[layer]; has {
			out = append(out, v)
		}
	}
	return out
}

// Centre is the mean position of the data, for a first view.
func (s *CellSource) Centre() (lon, lat float64) {
	n := 0
	for c := range s.cells {
		ll, err := h3.CellToLatLng(c)
		if err != nil {
			continue
		}
		lon += ll.Lng
		lat += ll.Lat
		n++
		if n >= 5000 {
			break
		}
	}
	if n == 0 {
		return 0, 0
	}
	return lon / float64(n), lat / float64(n)
}
