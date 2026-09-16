package document_test

import (
	"image"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"

	"github.com/janearc/libdaffy/canvas"
	"github.com/janearc/libdaffy/document"
	"github.com/janearc/libdaffy/history"
	"github.com/janearc/libdaffy/part"
)

// every member of a document walks back and forward through history's one
// door, and in order: a stroke on the canvas, a gesture on the glyphs, a
// box, a region, a part, a role, the colourway, the underlay, the mask.
func TestUndoCoversEveryMember(t *testing.T) {
	d := document.NewDocument("t", 10, 6)
	h := history.New(0)
	var red *part.Part
	if err := history.Do(h, "part", &d.Parts, func() error {
		ink := part.Literal(srgb.RGB8(255, 0, 0).Swatch())
		p, err := d.Parts.Add("red", ink)
		red = p
		return err
	}); err != nil {
		t.Fatal(err)
	}
	g := history.Begin(h, "stroke", &d.Canvas)
	s := d.Stroke(red.ID)
	canvas.RectFill(s, 0, 0, 3, 3)
	if !g.End() || d.Canvas.Count(red.ID) != 16 {
		t.Fatalf("stroke: %d pixels", d.Canvas.Count(red.ID))
	}
	if history.Begin(h, "nothing", &d.Canvas).End() {
		t.Fatal("an empty stroke was recorded")
	}
	gl := history.Begin(h, "glyphs", &d.Layer)
	d.Layer.Cells.Set(1, 1, canvas.Glyph{Ch: "c", Fg: red.ID})
	d.Layer.Cells.Border(
		image.Rect(0, 0, 10, 3),
		canvas.Light,
		"t",
		red.ID,
		part.None,
	)
	gl.End()
	var box *canvas.TextBox
	history.Do(h, "box", &d.Layer, func() error {
		box = d.Layer.AddBox(
			canvas.TextBox{
				Box:  image.Rect(2, 3, 8, 5),
				Text: "hello",
				Fg:   red.ID,
			},
		)
		return nil
	})
	history.Do(h, "edit box", &d.Layer, func() error {
		return d.Layer.SetBox(
			box.ID,
			canvas.TextBox{
				Box:  image.Rect(2, 3, 8, 5),
				Text: "bye",
				Fg:   red.ID,
			},
		)
	})
	history.Do(h, "region", &d.Regions, func() error {
		_, err := d.Regions.Add(
			canvas.Region{Name: "a", Cells: image.Rect(0, 0, 2, 2)},
		)
		return err
	})
	history.Value(h, "role", &d.Colourway, func() {
		d.Colourway = document.WithRole(
			d.Colourway,
			"beak",
			srgb.RGB8(1, 2, 3).Swatch(),
			false,
		)
	})
	history.Value(
		h,
		"underlay",
		&d.Underlay,
		func() { d.Underlay = "ref.png" },
	)
	history.Value(
		h,
		"select",
		&d.Mask,
		func() {
			r := image.Rect(0, 0, 5, 6)
			d.Mask = canvas.MaskFromRect(10, 6, r)
		},
	)
	if h.Len() != 9 {
		t.Fatalf("%d steps, want 9", h.Len())
	}
	order := []string{
		"select", "underlay", "role", "region", "edit box",
		"box", "glyphs", "stroke", "part",
	}
	for _, want := range order {
		if what, ok := h.Undo(); !ok || what != want {
			t.Fatalf("undo %q, want %q", what, want)
		}
	}
	if d.Mask != nil || d.Underlay != "" || d.Regions.Len() != 0 ||
		d.Layer.Cells.Count() != 0 ||
		d.Canvas.Count(red.ID) != 0 ||
		d.Parts.Len() != 0 {
		t.Fatal("something survived the walk back")
	}
	if _, ok := d.Colourway.Roles.Get("beak"); ok {
		t.Fatal("the role survived")
	}
	for h.CanRedo() {
		h.Redo()
	}
	if d.Layer.Box(box.ID) == nil || d.Layer.Box(box.ID).Text != "bye" ||
		d.Canvas.Count(red.ID) != 16 ||
		d.Layer.Cells.At(0, 0).Ch != "┌" ||
		d.Underlay != "ref.png" ||
		d.Mask == nil {
		t.Fatal("something did not come back")
	}
	if p := d.Parts.Lookup("red"); p == nil || p.ID != red.ID {
		t.Fatal("the part did not come back under its id")
	}
}

// a change that fails inside the door leaves the member as it was and
// records nothing.
func TestFailedChangeIsNotRecorded(t *testing.T) {
	d := document.NewDocument("t", 4, 4)
	h := history.New(0)
	d.Parts.Add("a", part.Role("a"))
	err := history.Do(h, "dup", &d.Parts, func() error {
		_, err := d.Parts.Add("a", part.Role("a"))
		return err
	})
	if err == nil || h.Len() != 0 || d.Parts.Len() != 1 {
		t.Fatalf("%v %d %d", err, h.Len(), d.Parts.Len())
	}
}

// a gesture on the layer that only touched boxes is recorded, and one
// that put a box back as it was is not.
func TestLayerGestureSeesBoxes(t *testing.T) {
	d := document.NewDocument("t", 4, 4)
	h := history.New(0)
	g := history.Begin(h, "box", &d.Layer)
	b := d.Layer.AddBox(
		canvas.TextBox{Box: image.Rect(0, 0, 2, 1), Text: "a"},
	)
	if !g.End() {
		t.Fatal("adding a box was not a change")
	}
	g = history.Begin(h, "same", &d.Layer)
	d.Layer.SetBox(
		b.ID,
		canvas.TextBox{Box: image.Rect(0, 0, 2, 1), Text: "b"},
	)
	d.Layer.SetBox(
		b.ID,
		canvas.TextBox{Box: image.Rect(0, 0, 2, 1), Text: "a"},
	)
	if g.End() || h.Len() != 1 {
		t.Fatal("a box put back as it was counted as a change")
	}
}
