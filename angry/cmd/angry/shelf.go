package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/janearc/angry-dist/kingfisher"
	"github.com/janearc/angry-dist/textmap"
)

// The shelf: kingfisher's priced door, and the only way this client asks for
// cells by default. A window and a budget, never a resolution; the server folds
// to fit the budget and its own memory and says what it did. The budget is the
// one exact figure a terminal has:
//
// cols times rows times two, the pixel count of the map, since every cell
// past that averages into a bin that already holds one.
//
// The fetched window is grown past the screen so a small pan costs nothing. A
// fetch happens when the view leaves what is held, or zooms in far enough
// that a refetch would buy resolution.

// shelf is the fetch-on-move state.
type shelf struct {
	kf      *kingfisher.Client
	fetched *kingfisher.BBox // what is held, or nil
	span    float64          // the view's scale at the last fetch
	last    *kingfisher.Window
	err     error         // the last refusal, shown until the next success
	at      time.Time     // when the last fetch was sent
	waiting bool          // a fetch is due and a timer will send it
	gap     time.Duration // least time between fetches; minFetchGap
}

// minFetchGap is the least time between two shelf fetches. This client has
// knocked kingfisher over before by asking faster than it could fold, and on
// the horizon an arrow key held down to fly leaves the held window several
// times a second.
//
// Between fetches the view draws from what it holds, which is what the grown
// window is for.
const minFetchGap = 2 * time.Second

// budget is what this frame can hold.
func (a *app) budget() int {
	w, h := a.proj.Pixels()
	return max(1, w*h)
}

// window is the view's bounding box as the server takes it, clamped to
// the world. The globe's Bounds already says the world when the disc
// wraps or holds a pole.
func (a *app) window() kingfisher.BBox {
	w, s, e, n := a.proj.Bounds()
	return kingfisher.BBox{
		max(-180, w),
		max(-90, s),
		min(180, e),
		min(90, n),
	}
}

// refresh fetches when the screen has left what is held, or when it has
// shrunk far enough inside it that finer cells are worth asking for. It
// blocks the loop for the request, which is fine for a request sized to
// the screen, and says so in the status line meanwhile.
func (a *app) refresh(force bool) {
	if a.shelf == nil {
		return
	}
	want := a.window()
	zoomedIn := a.shelf.fetched != nil && a.proj.Scale() < a.shelf.span*0.5
	if !force && a.shelf.fetched != nil && a.shelf.fetched.Covers(want) &&
		!zoomedIn {
		return
	}
	if since := time.Since(a.shelf.at); !force && since < a.shelf.gap {
		a.later(a.shelf.gap - since)
		return
	}
	box := want.Grown(1.6)
	box = kingfisher.BBox{
		max(-180, box[0]),
		max(-90, box[1]),
		min(180, box[2]),
		min(90, box[3]),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	a.shelf.at = time.Now()
	budget := a.budget()
	w, err := a.shelf.kf.Window(ctx, box, budget)
	if err != nil {
		a.shelf.err = err
		var r *kingfisher.Refused
		if errors.As(err, &r) {
			a.msg = r.Error()
		} else {
			a.msg = fmt.Sprintf("fetch failed: %v", err)
		}
		return
	}
	a.shelf.err = nil
	a.shelf.last = w
	a.shelf.fetched = &box
	a.shelf.span = a.proj.Scale()
	src := textmap.NewCellSource(
		"shelf",
		w.Cells,
		map[string]string{"point_density": "LAYER_KIND_EXTENSIVE"},
	)
	a.src = src
	a.layers = src.Layers()
	if len(a.layers) == 0 {
		a.layers = []string{"value"}
	}
	if a.layer >= len(a.layers) {
		a.layer = 0
	}
	a.pinScale()
	a.dirty = true
	served := ""
	if w.ClippedBy != "" {
		served = fmt.Sprintf(
			", clipped by %s to %d",
			w.ClippedBy,
			w.BudgetServed,
		)
	}
	a.msg = fmt.Sprintf(
		"fetched %d cells at res %d from %d datasets for budget %d%s",
		len(w.Cells),
		w.Res,
		w.Datasets,
		budget,
		served,
	)
}

// later sends one refresh after a wait, so a view that stops moving
// inside the gap still catches up without another key press. Only one is
// ever pending; the refresh it sends decides afresh whether to fetch.
func (a *app) later(wait time.Duration) {
	if a.shelf.waiting || a.t == nil {
		return
	}
	a.shelf.waiting = true
	a.msg = "fetch waiting: kingfisher is asked at most every two seconds"
	time.AfterFunc(wait, func() {
		a.t.SendEvent(callEvent{f: func(a *app) {
			// the shelf can be gone by the time this fires: a
			// change of map puts the view on a file and drops it
			if a.shelf == nil {
				return
			}
			a.shelf.waiting = false
			a.refresh(false)
		}, done: make(chan struct{})})
	})
}

// pinScale fixes the colour ramp to everything fetched, not to what is on
// screen, so panning within a fetch is a move and not a repaint.
func (a *app) pinScale() {
	a.pin = nil
	if a.src == nil || len(a.layers) == 0 {
		return
	}
	vals := a.src.Values(a.layers[a.layer])
	if lo, hi, ok := a.style.scale.Bounds(vals); ok {
		a.pin = &[2]float64{lo, hi}
	}
}

// shelfNote is the header's account of what the shelf served.
func (a *app) shelfNote() string {
	if a.shelf == nil || a.shelf.last == nil {
		return ""
	}
	w := a.shelf.last
	s := fmt.Sprintf(
		"  shelf res %d, %d cells of budget %d",
		w.Res,
		len(w.Cells),
		a.budget(),
	)
	if w.ClippedBy != "" {
		s += " clipped by " + w.ClippedBy
	}
	return s
}
