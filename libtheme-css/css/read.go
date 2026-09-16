package css

import (
	"regexp"
	"sort"
	"strconv"

	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// Colourway is what a colourway file holds once read: its roles, every custom
// property that is a hex colour, in file order, and its ramps, which are
// numbered families of roles: --sunset-1 to --sunset-6 is an even ramp called
// sunset.
//
// That is the whole grammar, and it is ours: a sheet this library wrote, or a
// colourway written by hand in the same shape. Anything else in the file,
// selectors, gradients, the rest of css, is not read.
//
// This is a container, not a parser; a file that wants a ramp read promotes its
// stops to roles, as the vaporwave file does. Ramps mix in oklab, the library's
// line between two colours, which is said here so nobody is surprised.
type Colourway struct {
	Roles *Sheet
	Ramps map[string]functions.Ramp
	// Order is the ramps' names in the order they were found.
	Order []string
}

var (
	roleRe = regexp.MustCompile(
		`--([a-zA-Z0-9_-]+)\s*:\s*(#[0-9a-fA-F]{3,8})\b`)
	familyRe = regexp.MustCompile(`^(.*)-(\d+)$`)
)

// Read reads a colourway: hex custom properties as roles, numbered
// families as ramps. It is not a css parser and reads nothing else.
func Read(src string) Colourway {
	cw := Colourway{Roles: New(), Ramps: map[string]functions.Ramp{}}
	for _, m := range roleRe.FindAllStringSubmatch(src, -1) {
		if c, err := srgb.FromHex(m[2]); err == nil {
			cw.Roles.Set(m[1], c.Swatch())
		}
	}
	families := map[string][]struct {
		n    int
		name string
	}{}
	var order []string
	for _, name := range cw.Roles.Names() {
		if m := familyRe.FindStringSubmatch(name); m != nil {
			n, _ := strconv.Atoi(m[2])
			if _, seen := families[m[1]]; !seen {
				order = append(order, m[1])
			}
			families[m[1]] = append(families[m[1]], struct {
				n    int
				name string
			}{n, name})
		}
	}
	for _, f := range order {
		members := families[f]
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i,
			j int) bool {
			return members[i].n < members[j].n
		})
		var stops []swatchAt
		for i, mb := range members {
			c, _ := cw.Roles.Get(mb.name)
			stops = append(stops,
				swatchAt{float64(i) / float64(len(members)-1),
					c})
		}
		cw.add(f, stops)
	}
	return cw
}

type swatchAt struct {
	at float64
	c  swatch.Swatch
}

// add records a ramp under its name, keeping the order it was found in.
func (cw *Colourway) add(name string, stops []swatchAt) {
	cw.Ramps[name] = ramp(stops)
	cw.Order = append(cw.Order, name)
}

// ramp is stops at positions as a ramp mixed in oklab.
func ramp(stops []swatchAt) functions.Ramp {
	out := make([]functions.Stop, 0, len(stops))
	for _, s := range stops {
		out = append(out, functions.Stop{At: s.at, Swatch: s.c})
	}
	return functions.New(ok.Mix, out...)
}
