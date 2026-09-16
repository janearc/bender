package document

import (
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"

	"github.com/janearc/libdaffy/part"
)

// PartFor gives the same part for the same colour, a new one for another,
// and a name that does not collide with a part of the same hex.
func TestPartFor(t *testing.T) {
	d := NewDocument("t", 2, 2)
	red := srgb.RGB8(255, 0, 0).Swatch()
	a := d.PartFor(red)
	if a == nil || d.PartFor(red) != a || a.Name != "#ff0000" {
		t.Fatalf("same colour: %+v", a)
	}
	if b := d.PartFor(srgb.RGB8(0, 255, 0).Swatch()); b == nil ||
		b.ID == a.ID {
		t.Fatal("another colour should be another part")
	}
	d.Parts.Rename(a.ID, "crimson")
	d.Parts.Add("#ff0000", srgbRole())
	if c := d.PartFor(red); c == nil || c.ID != a.ID {
		t.Fatal("the renamed part is still the part for its colour")
	}
	d.Parts.Rename(a.ID, "x")
	d.Parts.SetInk(a.ID, srgbRole())
	if c := d.PartFor(red); c == nil || c.Name != "#ff0000-2" {
		t.Fatalf("a taken hex name gets a number: %+v", c)
	}
}

// srgbRole is an ink that is not pinned, for the test above.
func srgbRole() part.Ink { return part.Role("beak") }

// a role bound with WithRole is bound in the copy and not in the given
// colourway, and removed the same way.
func TestWithRole(t *testing.T) {
	cw := EmptyColourway()
	with := WithRole(cw, "beak", srgb.RGB8(1, 2, 3).Swatch(), false)
	if _, ok := with.Roles.Get("beak"); !ok {
		t.Fatal("not bound")
	}
	if _, ok := cw.Roles.Get("beak"); ok {
		t.Fatal("the given colourway was changed")
	}
	without := WithRole(with, "beak", srgb.RGB8(0, 0, 0).Swatch(), true)
	if _, ok := without.Roles.Get("beak"); ok {
		t.Fatal("not removed")
	}
	if _, ok := with.Roles.Get("beak"); !ok {
		t.Fatal("the copy lost its role")
	}
}
