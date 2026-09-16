package canvas_test

import (
	"github.com/janearc/libdaffy/canvas"
	"image"
	"testing"
)

// regions are added by name, looked up, removed and turned into masks.
func TestRegions(t *testing.T) {
	rs := canvas.NewRegions()
	if _, err := rs.Add(canvas.Region{}); err == nil {
		t.Error("a region needs a name")
	}
	a, err := rs.Add(
		canvas.Region{
			Name:  "sidebar",
			Kind:  "pane",
			Cells: image.Rect(0, 0, 10, 5),
			Notes: map[string]string{"z": "1", "a": "2"},
		},
	)
	if err != nil || a.ID != 1 {
		t.Fatal(err)
	}
	if _, err := rs.Add(canvas.Region{Name: "sidebar"}); err == nil {
		t.Error("duplicate name")
	}
	b, _ := rs.Add(
		canvas.Region{
			ID:    5,
			Name:  "status",
			Cells: image.Rect(0, 4, 20, 6),
		},
	)
	if b.ID != 5 || canvas.NextID(rs) != 6 {
		t.Error("explicit id")
	}
	if rs.Get(5) != b || rs.Lookup("sidebar") != a ||
		rs.Lookup("x") != nil ||
		rs.Get(9) != nil ||
		rs.Len() != 2 {
		t.Error("lookups")
	}
	// The last-added region wins where they overlap.
	if rs.At(3, 4) != b || rs.At(3, 3) != a || rs.At(50, 50) != nil {
		t.Error("At")
	}
	if keys := a.NoteKeys(); len(keys) != 2 || keys[0] != "a" {
		t.Error("NoteKeys sorted")
	}
	cl := rs.Clone()
	cl.Get(1).Notes["z"] = "changed"
	if a.Notes["z"] != "1" {
		t.Error("clone should not share notes")
	}
	if !rs.Remove(5) || rs.Remove(5) || rs.Len() != 1 ||
		len(rs.All()) != 1 {
		t.Error("Remove")
	}
	// A masked region contains only cells its mask touches.
	m := canvas.NewMask(10, 10)
	m.Set(2, 3, true)
	if _, err := canvas.MaskToRegion(
		"blob", "shape", canvas.NewMask(4, 4),
	); err == nil {
		t.Error("empty mask")
	}
	if _, err := canvas.MaskToRegion("blob", "shape", nil); err == nil {
		t.Error("nil mask")
	}
	r, err := canvas.MaskToRegion("blob", "shape", m)
	if err != nil || r.Cells != (image.Rect(2, 1, 3, 2)) || r.Mask == nil {
		t.Fatalf("MaskToRegion %v %v", r.Cells, err)
	}
	if !r.Contains(2, 1) || r.Contains(2, 0) {
		t.Error("masked Contains")
	}
	if !r.AsMask(10, 10).Equal(m) {
		t.Error("AsMask keeps the mask")
	}
	plain := canvas.Region{Name: "p", Cells: image.Rect(1, 1, 3, 2)}
	if pm := plain.AsMask(10, 10); pm.Count() != 4 || !pm.At(1, 2) ||
		pm.At(1, 1) {
		t.Errorf("AsMask fills the box: %d", pm.Count())
	}
	cp := r.Clone()
	cp.Mask.Set(0, 0, true)
	if r.Mask.At(0, 0) {
		t.Error("clone shares the mask")
	}
	if (canvas.Region{}).Clone().Notes != nil {
		t.Error("nil notes stay nil")
	}
}

// a name is unique on add and on set, a set needs a name and a region
// that exists, and keeps the id.
func TestRegionRules(t *testing.T) {
	rs := canvas.NewRegions()
	a, err := rs.Add(
		canvas.Region{Name: "a", Cells: image.Rect(0, 0, 2, 2)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rs.Add(canvas.Region{Name: "a"}); err == nil {
		t.Fatal("duplicate")
	}
	rs.Add(canvas.Region{Name: "b", Cells: image.Rect(2, 2, 4, 4)})
	if err := rs.Set(a.ID, canvas.Region{Name: "b"}); err == nil {
		t.Fatal("rename onto a taken name")
	}
	if err := rs.Set(a.ID, canvas.Region{Name: ""}); err == nil {
		t.Fatal("rename to empty")
	}
	if err := rs.Set(99, canvas.Region{Name: "q"}); err == nil {
		t.Fatal("missing region")
	}
	renamed := canvas.Region{
		Name:  "aa",
		Kind:  "pane",
		Cells: image.Rect(0, 0, 3, 3),
		Notes: map[string]string{"k": "v"},
	}
	if err := rs.Set(a.ID, renamed); err != nil {
		t.Fatal(err)
	}
	if got := rs.Get(a.ID); got.Name != "aa" || got.Notes["k"] != "v" ||
		got.ID != a.ID {
		t.Fatalf("set: %+v", got)
	}
}
