// Package css is the container everything normalises to: a sheet of named
// colours, each one a swatch. Rudimentary on purpose.
//
// It holds names in the order they were given, writes itself out as the custom
// properties a browser, daffy or a colourway file would read, and that is all
// it does until it has to do more.
package css

import (
	"fmt"
	"strings"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// Sheet is named swatches in the order they were set.
type Sheet struct {
	names    []string
	swatches map[string]swatch.Swatch
}

// New is an empty sheet.
func New() *Sheet { return &Sheet{swatches: map[string]swatch.Swatch{}} }

// Set names a swatch. Setting a name again replaces the swatch and keeps
// the name's place in the order.
func (s *Sheet) Set(name string, c swatch.Swatch) {
	if _, ok := s.swatches[name]; !ok {
		s.names = append(s.names, name)
	}
	s.swatches[name] = c
}

// Get is the swatch under a name, and whether there is one.
func (s *Sheet) Get(name string) (swatch.Swatch, bool) {
	c, ok := s.swatches[name]
	return c, ok
}

// Names are the names, in order.
func (s *Sheet) Names() []string { return append([]string(nil), s.names...) }

// Rule is one custom property as the sheet would write it: the name, the
// value a screen can make, and the comment beside it. Exposed so that a
// printer can paint the parts without parsing the text back.
type Rule struct {
	Name, Value, Comment string
	Swatch               swatch.Swatch
}

// Rules are the sheet's rules, in order.
func (s *Sheet) Rules() []Rule {
	out := make([]Rule, 0, len(s.names))
	for _, name := range s.names {
		c := s.swatches[name]
		rgb, in := srgb.FromSwatch(c)
		note := ok.FromSwatch(c).Polar().String()
		if !in {
			note += ", clipped"
		}
		out = append(out, Rule{name, rgb.Hex(), note, c})
	}
	return out
}

// String is the sheet as CSS: one custom property per name on :root, the value
// as the hex a screen can make, and beside it in a comment the same colour as
// oklch, which is the form that says what it is. Out of gamut, the hex is the
// nearest the lamps can do and the comment says so.
func (s *Sheet) String() string {
	var b strings.Builder
	b.WriteString(":root {\n")
	for _, r := range s.Rules() {
		fmt.Fprintf(&b, "  --%s: %s; /* %s */\n", r.Name, r.Value,
			r.Comment)
	}
	b.WriteString("}\n")
	return b.String()
}
