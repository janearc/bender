package render_test

import (
	"testing"

	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/srgb"

	"github.com/janearc/libdaffy/part"
	"github.com/janearc/libdaffy/render"
)

// a pinned ink is its own colour whatever the roles say; a bound role
// is its colour; an unbound role, or no sheet at all, is the
// placeholder and reported so.
func TestBind(t *testing.T) {
	roles := css.New()
	roles.Set("beak", srgb.RGB{R: 1, G: 0, B: 0}.Swatch())
	if c, ok := render.Bind(part.Role("beak"), roles); !ok ||
		!c.Equal(srgb.RGB{R: 1, G: 0, B: 0}) {
		t.Fatalf("beak: %v %v", c, ok)
	}
	if c, ok := render.Bind(part.Role("nope"), roles); ok ||
		!c.Equal(render.Placeholder) {
		t.Fatalf("an unbound role: %v %v", c, ok)
	}
	green := srgb.RGB{R: 0, G: 1, B: 0}
	if c, ok := render.Bind(part.Literal(green.Swatch()), nil); !ok ||
		!c.Equal(green) {
		t.Fatalf("a pinned ink with no sheet: %v %v", c, ok)
	}
	if _, ok := render.Bind(part.Role("beak"), nil); ok {
		t.Error("no sheet bound a role")
	}
}

// the parts a sheet leaves unbound, in registry order.
func TestUnboundParts(t *testing.T) {
	roles := css.New()
	roles.Set("beak", srgb.RGB{R: 1, G: 0, B: 0}.Swatch())
	parts := part.NewParts()
	parts.Add("beak", part.Role("beak"))
	parts.Add("cap", part.Role("cap"))
	parts.Add("lit", part.Literal(srgb.RGB{}.Swatch()))
	if ub := render.UnboundParts(parts, roles); len(ub) != 1 ||
		ub[0].Name != "cap" {
		t.Errorf("unbound: %+v", ub)
	}
}
