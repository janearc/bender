package css

import (
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// Names keep their order, a name set twice keeps its place, and Get
// says when a name is missing.
func TestOrderAndReplace(t *testing.T) {
	s := New()
	s.Set("ground", swatch.Black)
	s.Set("ink", swatch.White)
	s.Set("ground", srgb.Red.Swatch())
	if got := s.Names(); len(got) != 2 || got[0] != "ground" ||
		got[1] != "ink" {
		t.Errorf("names = %v", got)
	}
	if c, ok := s.Get("ground"); !ok || c != srgb.Red.Swatch() {
		t.Errorf("ground = %v, %v", c, ok)
	}
	if _, ok := s.Get("accent"); ok {
		t.Errorf("a missing name was found")
	}
}

// The sheet writes the CSS a file would hold, exactly.
func TestString(t *testing.T) {
	s := New()
	s.Set("ground", swatch.Black)
	s.Set("red", srgb.Red.Swatch())
	want := ":root {\n" +
		"  --ground: #000000; /* oklch(0% 0.000 none) */\n" +
		"  --red: #ff0000; /* oklch(63% 0.258 29.2) */\n" +
		"}\n"
	if got := s.String(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}
