package document

import (
	"bytes"
	"encoding/json"
	"github.com/janearc/libdaffy/canvas"
	"github.com/janearc/libdaffy/part"
	"github.com/janearc/libdaffy/sprite"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A document round trip preserves the canvas, parts, colourway, cells, boxes,
// regions, mask, underlay and sprite extras.
func TestDocumentRoundTrip(t *testing.T) {
	d := NewDocument("demo", 12, 8)
	red, _ := d.Parts.Add(
		"red",
		part.Literal(srgb.RGB8(255, 0, 0).Swatch()),
	)
	beak, _ := d.Parts.Add("beak", part.Role("beak"))
	beak.Role = 'B'
	s := d.Stroke(red.ID)
	canvas.RectFill(s, 0, 0, 5, 3)
	s.PlotPart(7, 7, beak.ID)
	d.ColourwayName = "night"
	d.Colourway.Roles.Set("beak", srgb.RGB8(1, 2, 3).Swatch())
	d.Layer.Cells.Set(2, 2, canvas.Glyph{Ch: "┼", Fg: red.ID, Bg: beak.ID})
	d.Layer.AddBox(
		canvas.TextBox{
			Box:   image.Rect(1, 1, 8, 3),
			Text:  "hi there",
			Align: canvas.Centre,
			Fg:    red.ID,
		},
	)
	m := canvas.NewMask(12, 8)
	m.Set(3, 3, true)
	d.Regions.Add(
		canvas.Region{
			Name:  "side",
			Kind:  "pane",
			Cells: image.Rect(0, 0, 4, 4),
			Mask:  m,
			Notes: map[string]string{"k": "v"},
		},
	)
	d.Regions.Add(
		canvas.Region{Name: "plain", Cells: image.Rect(4, 0, 8, 4)},
	)
	d.Mask = canvas.MaskFromRect(12, 8, image.Rect(0, 0, 6, 6))
	d.Underlay = "ref.png"
	d.Sprite = &sprite.Sprite{
		Name: "bird",
		Raw: []string{
			"sprite bird",
			"",
			"art",
			sprite.ArtMarker,
			"",
			"pose blink",
		},
	}

	var buf bytes.Buffer
	if err := d.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(buf.Bytes()) {
		t.Fatal("not json")
	}
	back, err := Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if back.Name != "demo" || !back.Canvas.Equal(d.Canvas) {
		t.Fatal("canvas")
	}
	if bp := back.Parts.Get(beak.ID); bp == nil || bp.Name != "beak" ||
		bp.Ink.Pinned ||
		bp.Ink.Role != "beak" ||
		bp.Role != 'B' {
		t.Fatalf("beak part %+v", bp)
	}
	if rp := back.Parts.Get(red.ID); rp == nil || !rp.Ink.Pinned ||
		!sameSwatch(rp.Ink.Literal, srgb.RGB8(255, 0, 0).Swatch()) {
		t.Fatalf("red part %+v", rp)
	}
	bk, _ := back.Colourway.Roles.Get("beak")
	if back.ColourwayName != "night" ||
		!sameSwatch(bk, srgb.RGB8(1, 2, 3).Swatch()) {
		t.Fatal("colourway")
	}
	if g := back.Layer.Cells.At(2, 2); g.Ch != "┼" || g.Fg != red.ID ||
		g.Bg != beak.ID {
		t.Fatalf("glyph %+v", g)
	}
	if b := back.Layer.Box(1); b == nil || b.Text != "hi there" ||
		b.Align != canvas.Centre ||
		b.Box != (image.Rect(1, 1, 8, 3)) {
		t.Fatalf("box %+v", b)
	}
	if r := back.Regions.Lookup("side"); r == nil || r.Kind != "pane" ||
		r.Mask == nil ||
		!r.Mask.At(3, 3) ||
		r.Notes["k"] != "v" {
		t.Fatalf("region %+v", r)
	}
	if r := back.Regions.Lookup("plain"); r == nil || r.Mask != nil {
		t.Fatal("plain region")
	}
	if back.Mask == nil || !back.Mask.Equal(d.Mask) ||
		back.Underlay != "ref.png" {
		t.Fatal("mask or underlay")
	}
	if back.Sprite == nil || back.Sprite.Name != "bird" ||
		len(back.Sprite.Raw) != 6 {
		t.Fatal("sprite extras")
	}
	// Encoding again gives the same bytes: the format is deterministic.
	var again bytes.Buffer
	back.Encode(&again)
	d2, _ := Decode(bytes.NewReader(again.Bytes()))
	var third bytes.Buffer
	d2.Encode(&third)
	if again.String() != third.String() {
		t.Error("encoding is not stable")
	}
	// Save and Load through a file.
	path := filepath.Join(t.TempDir(), "demo.daffy")
	if err := d.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil || !loaded.Canvas.Equal(d.Canvas) {
		t.Fatal(err)
	}
	if _, err := Load(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("missing file")
	}
	unwritable := filepath.Join(t.TempDir(), "no", "dir", "x")
	if err := d.Save(unwritable); err == nil {
		t.Error("unwritable path")
	}
}

// not json, a wrong version, an empty canvas or an unknown part id is refused.
func TestDecodeRefusesBadFiles(t *testing.T) {
	one := `{"daffy": 1, "canvas": {"w": 1, "h": 2, "pixels": [[0, 2]]}`
	themed := one + `, "theme": {"roles": {}}`
	bad := []string{
		"not json",
		`{"daffy": 2, "canvas": {"w": 1, "h": 2, "pixels": [[0, 2]]}}`,
		`{"daffy": 1, "canvas": {"w": 0, "h": 2}}`,
		`{"daffy": 1, "canvas": {"w": 1, "h": 2, "pixels": [[0, 3]]}}`,
		`{"daffy": 1, "canvas": {"w": 1, "h": 2, "pixels": [[-1, 2]]}}`,
		one + `, "parts": [{"id": 1, "name": "p", "color": "zz"}]}`,
		one + `, "parts": [{"id": 0, "name": "p"}]}`,
		one + `, "theme": {"roles": {"a": "zz"}}}`,
		themed + `, "regions": [{"name": "r", "mask": [[1, 1]]}]}`,
		themed + `, "regions": [{"name": ""}]}`,
		themed + `, "mask": [[1, 5]]}`,
	}
	for _, src := range bad {
		if _, err := Decode(strings.NewReader(src)); err == nil {
			t.Errorf("accepted: %s", src)
		}
	}
	good := `{"daffy": 1, "name": "x",
		"canvas": {"w": 2, "h": 2, "pixels": [[0, 4]]},
		"theme": {"roles": {}},
		"boxes": [{"id": 3, "box": [0,0,1,1],
			"text": "a", "align": "right"}]}`
	d, err := Decode(strings.NewReader(good))
	if err != nil || d.Layer.Box(3) == nil ||
		d.Layer.Box(3).Align != canvas.Right {
		t.Errorf("minimal file: %v", err)
	}
}

// the ansi, text and layout exports carry the picture, and the layout its
// roles.
func TestExports(t *testing.T) {
	d := NewDocument("x", 6, 4)
	red, _ := d.Parts.Add(
		"red",
		part.Literal(srgb.RGB8(255, 0, 0).Swatch()),
	)
	blue, _ := d.Parts.Add("blue", part.Role("blue"))
	d.Colourway.Roles.Set("blue", srgb.RGB8(0, 0, 255).Swatch())
	d.Canvas.Set(0, 0, red.ID)
	d.Canvas.Set(1, 0, red.ID)
	d.Canvas.Set(1, 1, blue.ID)
	d.Canvas.Set(2, 3, blue.ID)
	d.Layer.Cells.Set(4, 0, canvas.Glyph{Ch: "a", Fg: red.ID})
	d.Layer.Cells.Set(5, 1, canvas.Glyph{Ch: "b", Fg: red.ID, Bg: blue.ID})
	d.Regions.Add(
		canvas.Region{
			Name:  "r",
			Kind:  "pane",
			Cells: image.Rect(0, 0, 2, 2),
			Mask:  canvas.NewMask(6, 4),
		},
	)

	text := d.Text()
	if text != "▀▀  a\n  ▄  b\n" {
		t.Errorf("text %q", text)
	}
	ansi := d.ANSI()
	lines := strings.Split(strings.TrimSuffix(ansi, "\n"), "\n")
	if len(lines) != 2 || !strings.HasSuffix(lines[0], "\x1b[0m") {
		t.Fatalf("ansi lines %q", lines)
	}
	if !strings.Contains(lines[0], "\x1b[38;2;255;0;0m\x1b[49m▀") {
		t.Errorf(
			"top-only cell should have a reset background: %q",
			lines[0],
		)
	}
	if !strings.Contains(lines[0], "\x1b[49m▀\x1b[48;2;0;0;255m▀") {
		t.Errorf("both halves, foreground not repeated: %q", lines[0])
	}
	if !strings.Contains(lines[1], "\x1b[48;2;0;0;255mb") {
		t.Errorf("glyph with background: %q", lines[1])
	}
	if strings.Count(lines[0], "38;2;255;0;0") != 2 {
		t.Errorf("colour should not be repeated for runs: %q", lines[0])
	}
	out, err := d.Layout()
	if err != nil {
		t.Fatal(err)
	}
	var l map[string]any
	if err := json.Unmarshal(out, &l); err != nil {
		t.Fatal(err)
	}
	if l["cols"].(float64) != 6 || l["rows"].(float64) != 2 {
		t.Error("layout size")
	}
	regions := l["regions"].([]any)
	if len(regions) != 1 || regions[0].(map[string]any)["shape"] != true {
		t.Errorf("layout regions %v", regions)
	}
	parts := l["parts"].([]any)
	if len(parts) != 2 || parts[1].(map[string]any)["bound"] != true {
		t.Errorf("layout parts %v", parts)
	}
	if !regexp.MustCompile(`"#[0-9a-f]{6}"`).Match(out) {
		t.Error("layout colours as hex")
	}
	if _, err := d.Layout(); err != nil {
		t.Fatalf("layout again: %v", err)
	}
	if os.Getenv("DAFFY_DUMP") != "" {
		os.WriteFile("dump.ansi", []byte(ansi), 0o644)
	}
}
