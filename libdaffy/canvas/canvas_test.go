package canvas

import (
	"github.com/janearc/libdaffy/part"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"testing"
)

// a new canvas is empty and sized as asked, and pixels set, read and count.
func TestCanvasBasics(t *testing.T) {
	c := NewCanvas(4, 3)
	if c.Height() != 4 || c.Rows() != 2 {
		t.Fatalf(
			"odd height should round up: h=%d rows=%d",
			c.Height(),
			c.Rows(),
		)
	}
	if !c.Set(1, 1, 7) || c.Set(1, 1, 7) {
		t.Error("Set should report change once")
	}
	if c.At(1, 1) != 7 || c.At(-1, 0) != part.None ||
		c.At(4, 0) != part.None {
		t.Error("At")
	}
	if c.Set(9, 9, 1) {
		t.Error("off-canvas set should be ignored")
	}
	if c.Count(7) != 1 {
		t.Error("Count")
	}
	cl := c.Clone()
	if !cl.Equal(c) {
		t.Error("clone should equal")
	}
	cl.Clear()
	if cl.Equal(c) || cl.Count(7) != 0 {
		t.Error("clear")
	}
	if n := c.Replace(7, 3); n != 1 || c.At(1, 1) != 3 {
		t.Error("Replace")
	}
	big := c.Resize(6, 6)
	if big.At(1, 1) != 3 || big.Width() != 6 {
		t.Error("Resize should keep pixels")
	}
	small := c.Resize(1, 2)
	if small.At(1, 1) != part.None {
		t.Error("Resize should crop")
	}
	if s := dump(c); s != "....\n.3..\n....\n....\n" {
		t.Errorf("String:\n%s", s)
	}
}

// parts are added, looked up, renamed and removed, and an ink names itself.
func TestParts(t *testing.T) {
	p := part.NewParts()
	a, err := p.Add("a", part.Literal(srgb.RGB{R: 1, G: 0, B: 0}.Swatch()))
	if err != nil || a.ID != 1 {
		t.Fatal(err)
	}
	if _, err := p.Add("a", part.Role("x")); err == nil {
		t.Error("duplicate name should fail")
	}
	if _, err := p.Add("", part.Role("x")); err == nil {
		t.Error("empty name should fail")
	}
	b, _ := p.Add("b", part.Role("beak"))
	if p.Lookup("b") != b || p.Get(b.ID) != b || p.Lookup("zz") != nil ||
		p.Get(9) != nil {
		t.Error("lookups")
	}
	if err := p.Rename(a.ID, "b"); err == nil {
		t.Error("rename onto a taken name should fail")
	}
	if err := p.Rename(a.ID, "aa"); err != nil || p.Lookup("aa") == nil ||
		p.Lookup("a") != nil {
		t.Error("rename")
	}
	if err := p.Rename(a.ID, ""); err == nil {
		t.Error("rename to empty should fail")
	}
	if err := p.Rename(99, "q"); err == nil {
		t.Error("rename of a missing part should fail")
	}
	if err := p.SetInk(a.ID, part.Role("cap")); err != nil ||
		p.Get(a.ID).Ink.Role != "cap" {
		t.Error("SetInk")
	}
	if err := p.SetInk(99, part.Role("cap")); err == nil {
		t.Error("SetInk of a missing part should fail")
	}
	all := p.All()
	if len(all) != 2 || all[0].ID != 1 || all[1].ID != 2 {
		t.Errorf("All %+v", all)
	}
	cl := p.Clone()
	p.Remove(a.ID)
	if p.Len() != 1 || cl.Len() != 2 || p.Lookup("aa") != nil {
		t.Error("Remove or Clone")
	}
	p.Remove(99)
	if err := p.Put(part.Part{ID: 0, Name: "z"}); err == nil {
		t.Error("Put id 0 should fail")
	}
	if err := p.Put(part.Part{ID: 2, Name: "z"}); err == nil {
		t.Error("Put a taken id should fail")
	}
	if err := p.Put(part.Part{ID: 5, Name: "b"}); err == nil {
		t.Error("Put a taken name should fail")
	}
	if err := p.Put(part.Part{ID: 5, Name: ""}); err == nil {
		t.Error("Put an empty name should fail")
	}
	if err := p.Put(part.Part{ID: 5, Name: "five"}); err != nil {
		t.Error(err)
	}
	next, _ := p.Add("six", part.Role("r"))
	if next.ID != 6 {
		t.Errorf("next id after Put(5) should be 6, got %d", next.ID)
	}
	if part.Literal(srgb.RGB{R: 1, G: 1, B: 1}.Swatch()).
		String() !=
		"#ffffff" ||
		part.Role("beak").String() != "role beak" {
		t.Error("Ink.String")
	}
}

// dump is the pixel grid as text for a failure a person can read: '.' for
// nothing, otherwise the part id as one character, 0-9 then a-z, '#'
// past that.
func dump(c *Canvas) string {
	const ids = "0123456789abcdefghijklmnopqrstuvwxyz"
	out := make([]byte, 0, (c.Width()+1)*c.Height())
	for y := 0; y < c.Height(); y++ {
		for x := 0; x < c.Width(); x++ {
			p := c.At(x, y)
			switch {
			case p == part.None:
				out = append(out, '.')
			case int(p) < len(ids):
				out = append(out, ids[p])
			default:
				out = append(out, '#')
			}
		}
		out = append(out, '\n')
	}
	return string(out)
}
