// Package history is an editing session's memory of what changed, in
// memory only, forwards and backwards. It knows nothing about documents:
//
// a change is a member's value before and after, and the only thing it
// asks of a member is that it can be cloned. Recording nouns rather than
// verbs is what keeps it this small; a new kind of member teaches it
// nothing, and there is no change it cannot walk back.
//
// A slot is the address of a member in its container: &doc.Layer, not
// &doc.Layer.Cells. Undo puts a clone back into the slot, so a slot inside
// a member that is itself swapped by another step would point into the
// orphaned old one. Hold the door on the member, and give it an Equal.
package history

// Cloner is a member that can be copied whole, deeply enough that
// changing the copy does not change the original.
type Cloner[T any] interface {
	Clone() T
}

// Equaler is a member that can say whether another is the same, so a
// gesture that changed nothing can be dropped rather than recorded.
type Equaler[T any] interface {
	Cloner[T]
	Equal(T) bool
}

// entry is one step: what it was called, and how to go each way.
type entry struct {
	what       string
	undo, redo func()
}

// History is the two stacks: what can be undone and what can be redone.
type History struct {
	limit      int
	undo, redo []entry
}

// New makes an empty history that keeps at most limit steps, or every
// step when limit is zero or less.
func New(limit int) *History {
	return &History{limit: limit}
}

// push records a step, forgets the redo stack, and drops the oldest step
// when the history is full.
func (h *History) push(e entry) {
	h.undo = append(h.undo, e)
	h.redo = nil
	if h.limit > 0 && len(h.undo) > h.limit {
		h.undo = append([]entry(nil), h.undo[len(h.undo)-h.limit:]...)
	}
}

// Do records what a member holds, makes the change, and keeps both
// sides, so the step can be walked either way. A change that returns an
// error is not recorded and the member is put back as it was.
func Do[T Cloner[T]](
	h *History,
	what string,
	slot *T,
	change func() error,
) error {
	before := (*slot).Clone()
	if err := change(); err != nil {
		*slot = before
		return err
	}
	after := (*slot).Clone()
	h.push(entry{what,
		func() { *slot = before.Clone() },
		func() { *slot = after.Clone() },
	})
	return nil
}

// Value is Do for a member that copies by assignment: a string, a number,
// a struct whose parts are never changed in place.
func Value[T any](h *History, what string, slot *T, change func()) {
	before := *slot
	change()
	after := *slot
	h.push(entry{what,
		func() { *slot = before },
		func() { *slot = after },
	})
}

// Gesture is the door held open: many writes to one member, recorded as
// one step when it ends.
type Gesture[T Equaler[T]] struct {
	h      *History
	what   string
	slot   *T
	before T
}

// Begin takes a member's value and hands back the gesture to end.
func Begin[T Equaler[T]](h *History, what string, slot *T) *Gesture[T] {
	return &Gesture[T]{
		h:      h,
		what:   what,
		slot:   slot,
		before: (*slot).Clone(),
	}
}

// End records the gesture as one step, or drops it if the member is as it
// was, and says which.
func (g *Gesture[T]) End() bool {
	if (*g.slot).Equal(g.before) {
		return false
	}
	before, after, slot := g.before, (*g.slot).Clone(), g.slot
	g.h.push(entry{g.what,
		func() { *slot = before.Clone() },
		func() { *slot = after.Clone() },
	})
	return true
}

// Undo walks the latest step back and says what it was called.
func (h *History) Undo() (string, bool) {
	if len(h.undo) == 0 {
		return "", false
	}
	e := h.undo[len(h.undo)-1]
	h.undo = h.undo[:len(h.undo)-1]
	e.undo()
	h.redo = append(h.redo, e)
	return e.what, true
}

// Redo walks the latest undone step forward again.
func (h *History) Redo() (string, bool) {
	if len(h.redo) == 0 {
		return "", false
	}
	e := h.redo[len(h.redo)-1]
	h.redo = h.redo[:len(h.redo)-1]
	e.redo()
	h.undo = append(h.undo, e)
	return e.what, true
}

// CanUndo is whether there is a step to walk back.
func (h *History) CanUndo() bool { return len(h.undo) > 0 }

// CanRedo is whether there is a step to walk forward.
func (h *History) CanRedo() bool { return len(h.redo) > 0 }

// Len is how many steps can be undone.
func (h *History) Len() int { return len(h.undo) }
