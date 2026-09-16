package render_test

import (
	"github.com/janearc/libdaffy/document"
	"github.com/janearc/libdaffy/part"
	"github.com/janearc/libdaffy/render"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"testing"
)

// The half-block rule as a table of its four cases.
func TestHalfBlockRule(t *testing.T) {
	a, b := srgb.RGB8(1, 2, 3), srgb.RGB8(4, 5, 6)
	cases := []struct {
		name     string
		top, bot *srgb.RGB
		want     render.Paint
	}{
		{"neither", nil, nil, render.Paint{}},
		{
			"both",
			&a,
			&b,
			render.Paint{
				Glyph: render.UpperHalf,
				Fg:    a,
				Bg:    b,
				HasBg: true,
				Set:   true,
			},
		},
		{
			"top only",
			&a,
			nil,
			render.Paint{Glyph: render.UpperHalf, Fg: a, Set: true},
		},
		{
			"bottom only",
			nil,
			&b,
			render.Paint{Glyph: render.LowerHalf, Fg: b, Set: true},
		},
	}
	for _, c := range cases {
		got := render.HalfBlock(c.top, c.bot)
		if got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
		// The invariant behind the table: a half that is not painted
		// never gets a background.
		if (c.top == nil || c.bot == nil) && got.HasBg {
			t.Errorf("%s: painted the unpainted half", c.name)
		}
	}
}

// CellPaint reads the two pixels of a cell through the resolver, and an
// unbound part is loud rather than absent.
func TestCellPaint(t *testing.T) {
	d := document.NewDocument("t", 4, 4)
	red, _ := d.Parts.Add(
		"red",
		part.Literal(srgb.RGB{R: 1, G: 0, B: 0}.Swatch()),
	)
	role, _ := d.Parts.Add("beak", part.Role("beak"))
	d.Canvas.Set(0, 0, red.ID)
	d.Canvas.Set(1, 1, role.ID)
	res := d.Resolver()
	top := render.CellPaint(d.Canvas, 0, 0, res)
	if top.Glyph != render.UpperHalf || top.HasBg ||
		!top.Fg.Equal(srgb.RGB{R: 1, G: 0, B: 0}) {
		t.Errorf("top pixel: %+v", top)
	}
	low := render.CellPaint(d.Canvas, 1, 0, res)
	if low.Glyph != render.LowerHalf || !low.Fg.Equal(render.Placeholder) {
		t.Errorf("unbound part should draw the placeholder: %+v", low)
	}
	if p := render.CellPaint(d.Canvas, 2, 0, res); p.Set {
		t.Errorf("empty cell should not be set: %+v", p)
	}
	d.Colourway.Roles.Set("beak", srgb.RGB{R: 0, G: 1, B: 0}.Swatch())
	if p := render.CellPaint(d.Canvas, 1, 0, res); !p.Fg.Equal(
		srgb.RGB{R: 0, G: 1, B: 0},
	) {
		t.Errorf("bound role should resolve: %+v", p)
	}
	// A part id the registry does not know is also loud.
	d.Canvas.Set(3, 3, 999)
	if p := render.CellPaint(d.Canvas, 3, 1, res); !p.Set ||
		!p.Fg.Equal(render.Placeholder) {
		t.Errorf("unknown part should draw the placeholder: %+v", p)
	}
}
