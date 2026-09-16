package document

import (
	"bytes"
	"github.com/janearc/libdaffy/part"
	"github.com/janearc/libdaffy/render"
	"github.com/janearc/libdaffy/sprite"
	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"os"
	"strings"
	"testing"
)

// The fixture is stranger.sprite, which uses all eleven roles.
// Read it, write it back, and the bytes are the same: daffy has understood
// the format rather than the puffin.
func TestSpriteRoundTrip(t *testing.T) {
	src, err := os.ReadFile("../sprite/testdata/sprites/stranger.sprite")
	if err != nil {
		t.Fatal(err)
	}
	s, err := sprite.ParseSprite(string(src))
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "stranger" || s.Width() != 14 || s.Height() != 12 {
		t.Fatalf("sprite %s %dx%d", s.Name, s.Width(), s.Height())
	}
	// The art uses nine of the eleven roles, both eye roles included, so
	// the nine-role alphabet the spec first had fails here. Stripe appears
	// only in the blink pose, which is read but not interpreted, and wing
	// does not appear in the file at all; the spec's claim that the fixture
	// uses all eleven is wider than the file.
	used := map[byte]bool{}
	for _, row := range s.Art {
		for i := 0; i < len(row); i++ {
			used[row[i]] = true
		}
	}
	for _, c := range []byte(".KWXEBYRO") {
		if !used[c] {
			t.Errorf("fixture art does not use %q", c)
		}
	}
	if used['V'] || used['D'] {
		t.Error("fixture art now uses wing or stripe; " +
			"update this test and the spec")
	}
	if !strings.Contains(string(src), "DDDD") {
		t.Error("stripe should appear in the blink pose")
	}
	if got := s.String(); got != string(src) {
		t.Fatalf("round trip differs:\n%s", got)
	}
	var buf bytes.Buffer
	s.Write(&buf)
	if buf.String() != string(src) {
		t.Fatal("Write differs")
	}
	// Through a document and back, byte for byte, poses included.
	d, err := ImportSprite(s)
	if err != nil {
		t.Fatal(err)
	}
	if d.Parts.Len() != 8 || d.Canvas.Width() != 14 ||
		d.Canvas.Height() != 12 {
		t.Fatalf(
			"import: %d parts, %dx%d",
			d.Parts.Len(),
			d.Canvas.Width(),
			d.Canvas.Height(),
		)
	}
	if p := d.Parts.Lookup("pupil"); p == nil || p.Role != 'E' ||
		p.Ink.Role != "pupil" {
		t.Fatalf("pupil part %+v", p)
	}
	out, err := d.ExportSprite()
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != string(src) {
		t.Fatalf("export differs:\n%s", out)
	}
	// The imported drawing binds to a colourway like any other drawing.
	d.ColourwayName, d.Colourway = "corvid", css.Read(
		":root { --dark: #101010; --light: #e0e0e0; "+
			"--pupil: #000000; }",
	)
	dark := d.Parts.Lookup("dark").Ink
	if c, ok := render.Bind(dark, d.Colourway.Roles); !ok ||
		!c.Equal(srgb.RGB8(0x10, 0x10, 0x10)) {
		t.Error("the colourway should bind the imported parts")
	}
	if ub := render.UnboundParts(d.Parts, d.Colourway.Roles); len(ub) != 5 {
		t.Errorf("unbound %d", len(ub))
	}
	// Loading from the path works too, and a document round trip keeps
	// the sprite extras so the export still matches.
	ls, err := sprite.LoadSprite(
		"../sprite/testdata/sprites/stranger.sprite",
	)
	if err != nil || ls.Name != "stranger" {
		t.Fatal(err)
	}
	if _, err := sprite.LoadSprite("../sprite/testdata/nope"); err == nil {
		t.Error("missing sprite")
	}
	var doc bytes.Buffer
	d.Encode(&doc)
	back, _ := Decode(&doc)
	again, err := back.ExportSprite()
	if err != nil || again.String() != string(src) {
		t.Fatalf("export after a document round trip differs: %v", err)
	}
}

// Editing the art and writing back keeps everything else in the file.
func TestSpriteEditKeepsPoses(t *testing.T) {
	src, _ := os.ReadFile("../sprite/testdata/sprites/stranger.sprite")
	s, _ := sprite.ParseSprite(string(src))
	d, _ := ImportSprite(s)
	feet := d.Parts.Lookup("feet")
	d.Canvas.Set(0, 0, feet.ID)
	out, err := d.ExportSprite()
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.HasPrefix(out.Art[0], "O") ||
		!strings.Contains(got, "pose blink") ||
		!strings.Contains(got, "# two patches") {
		t.Errorf("edited export:\n%s", got)
	}
	if strings.Count(got, "\n") != strings.Count(string(src), "\n") {
		t.Error("line count changed")
	}
}

// Export refuses a part outside the alphabet and names it, and never
// invents a role character.
func TestSpriteExportRefuses(t *testing.T) {
	d := NewDocument("mine", 4, 2)
	dark, _ := d.Parts.Add("dark", part.Role("dark"))
	tail, _ := d.Parts.Add("tail", part.Role("tail"))
	odd, _ := d.Parts.Add("odd", part.Literal(srgb.RGB{}.Swatch()))
	odd.Role = 'Z'
	unused, _ := d.Parts.Add("unused", part.Role("nope"))
	_ = unused
	d.Canvas.Set(0, 0, dark.ID)
	d.Canvas.Set(1, 0, tail.ID)
	d.Canvas.Set(2, 0, odd.ID)
	d.Canvas.Set(3, 0, 99)
	_, err := d.ExportSprite()
	if err == nil {
		t.Fatal("should refuse")
	}
	msg := err.Error()
	named := []string{"tail", "odd ('Z')", "part 99 (missing)"}
	for _, want := range named {
		if !strings.Contains(msg, want) {
			t.Errorf("error should name %q: %s", want, msg)
		}
	}
	if strings.Contains(msg, "unused") || strings.Contains(msg, "dark") {
		t.Errorf("error names parts that are fine: %s", msg)
	}
	// A drawing that fits the alphabet by part names exports canonically.
	d.Canvas.Clear()
	d.Canvas.Set(0, 0, dark.ID)
	light, _ := d.Parts.Add("light", part.Role("light"))
	d.Canvas.Set(1, 1, light.ID)
	s, err := d.ExportSprite()
	if err != nil {
		t.Fatal(err)
	}
	if s.String() != "sprite mine\n\nart\nK...\n.W..\n" {
		t.Errorf("canonical:\n%s", s)
	}
	if _, ok := sprite.RoleByName("none"); !ok {
		t.Error("none is in the alphabet")
	}
	if _, ok := sprite.RoleByName("zzz"); ok {
		t.Error("zzz is not")
	}
}
