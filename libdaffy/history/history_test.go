package history_test

import (
	"errors"
	"testing"

	"github.com/janearc/libdaffy/history"
)

// grid is a member small enough to read: a row of numbers.
type grid struct{ cells []int }

// Clone copies the row.
func (g *grid) Clone() *grid {
	return &grid{cells: append([]int(nil), g.cells...)}
}

// Equal is whether two rows hold the same numbers.
func (g *grid) Equal(o *grid) bool {
	if len(g.cells) != len(o.cells) {
		return false
	}
	for i := range g.cells {
		if g.cells[i] != o.cells[i] {
			return false
		}
	}
	return true
}

// a step is walked back and forward, a new step after an undo forgets the
// redo, and a change that fails leaves the member as it was and no step.
func TestDoUndoRedo(t *testing.T) {
	h := history.New(0)
	g := &grid{cells: []int{0, 0}}
	history.Do(h, "one", &g, func() error { g.cells[0] = 1; return nil })
	history.Do(h, "two", &g, func() error { g.cells[1] = 2; return nil })
	if what, ok := h.Undo(); !ok || what != "two" || g.cells[1] != 0 ||
		g.cells[0] != 1 {
		t.Fatalf("undo: %q %v %v", what, ok, g.cells)
	}
	if what, ok := h.Redo(); !ok || what != "two" || g.cells[1] != 2 {
		t.Fatalf("redo: %q %v %v", what, ok, g.cells)
	}
	h.Undo()
	history.Do(h, "three", &g, func() error { g.cells[1] = 3; return nil })
	if h.CanRedo() {
		t.Fatal("a new step should forget the redo stack")
	}
	err := history.Do(
		h,
		"bad",
		&g,
		func() error { g.cells[0] = 9; return errors.New("no") },
	)
	if err == nil || g.cells[0] != 1 || h.Len() != 2 {
		t.Fatalf("a failed change: %v %v %d", err, g.cells, h.Len())
	}
	h.Undo()
	h.Undo()
	if _, ok := h.Undo(); ok || g.cells[0] != 0 {
		t.Fatalf("past the bottom: %v", g.cells)
	}
}

// undo puts back a copy, so editing after an undo does not rewrite the
// history's record of what was.
func TestRecordsAreNotAliased(t *testing.T) {
	h := history.New(0)
	g := &grid{cells: []int{0}}
	history.Do(h, "one", &g, func() error { g.cells[0] = 1; return nil })
	h.Undo()
	g.cells[0] = 7
	h.Redo()
	if g.cells[0] != 1 {
		t.Fatalf("redo gave %v", g.cells)
	}
	h.Undo()
	if g.cells[0] != 0 {
		t.Fatalf("undo after a stray edit gave %v", g.cells)
	}
}

// a plain value is recorded by assignment.
func TestValue(t *testing.T) {
	h := history.New(0)
	name := "a"
	history.Value(h, "rename", &name, func() { name = "b" })
	if what, ok := h.Undo(); !ok || what != "rename" || name != "a" {
		t.Fatalf("undo: %q %v %q", what, ok, name)
	}
	if _, ok := h.Redo(); !ok || name != "b" {
		t.Fatal("redo")
	}
}

// a gesture of many writes is one step, and one that changed nothing is
// not recorded.
func TestGesture(t *testing.T) {
	h := history.New(0)
	g := &grid{cells: make([]int, 5)}
	ge := history.Begin(h, "stroke", &g)
	for i := range g.cells {
		g.cells[i] = i
	}
	if !ge.End() || h.Len() != 1 {
		t.Fatalf("a gesture that changed cells: %d steps", h.Len())
	}
	if history.Begin(h, "nothing", &g).End() || h.Len() != 1 {
		t.Fatal("an empty gesture was recorded")
	}
	h.Undo()
	if g.cells[4] != 0 {
		t.Fatalf("undo of the gesture: %v", g.cells)
	}
}

// the oldest step falls off when the history is full.
func TestLimit(t *testing.T) {
	h := history.New(2)
	n := 0
	for i := 1; i <= 3; i++ {
		history.Value(h, "n", &n, func() { n = i })
	}
	if h.Len() != 2 {
		t.Fatalf("len %d", h.Len())
	}
	h.Undo()
	h.Undo()
	if _, ok := h.Undo(); ok || n != 1 {
		t.Fatalf("the oldest step should be gone: %d", n)
	}
}
