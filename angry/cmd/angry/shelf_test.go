package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/uber/h3-go/v4"

	"github.com/janearc/angry-dist/kingfisher"
)

// A fake shelf: answers /shelf with a quote and /cells with res-3 cells
// filling the window asked for, counting the asks, and refusing when told.
type fakeShelf struct {
	asks   atomic.Int32
	refuse atomic.Bool
	last   atomic.Value // the last query string
}

func (f *fakeShelf) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch r.URL.Path {
		case "/shelf":
			w.Write(
				[]byte(
					`{"datasets": 1, "cells_native": 100, "res_native": 3, "res": 3, "cells_estimate": 50, "budget": 1}`,
				),
			)
		case "/cells":
			f.asks.Add(1)
			f.last.Store(r.URL.RawQuery)
			if f.refuse.Load() {
				w.WriteHeader(503)
				w.Write(
					[]byte(
						`{"error": "no memory left in this process for a fold", "heap_bytes": 134217728, "memory_limit_bytes": 268435456, "limits": {"rows_that_fit_now": 0}}`,
					),
				)
				return
			}
			var b [4]float64
			fmt.Sscanf(
				q.Get("bbox"),
				"%f,%f,%f,%f",
				&b[0],
				&b[1],
				&b[2],
				&b[3],
			)
			cells := map[string]any{}
			for lat := b[1]; lat <= b[3]; lat += 0.3 {
				for lon := b[0]; lon <= b[2]; lon += 0.3 {
					c, _ := h3.LatLngToCell(
						h3.LatLng{Lat: lat, Lng: lon},
						3,
					)
					cells[c.String()] = map[string]any{
						"v": lon,
					}
				}
			}
			body := map[string]any{
				"res":           3,
				"datasets":      1,
				"cells_read":    len(cells),
				"budget":        1,
				"budget_served": 1,
				"bbox":          b[:],
				"cells":         cells,
			}
			if q.Get("budget") == "1" {
				body["clipped_by"] = "memory"
			}
			json.NewEncoder(w).Encode(body)
		default:
			http.NotFound(w, r)
		}
	})
}

func TestShelfFetchOnMove(t *testing.T) {
	fs := &fakeShelf{}
	srv := httptest.NewServer(fs.handler())
	defer srv.Close()
	kf := kingfisher.New(srv.URL)
	kf.Sleep = func(time.Duration) {}

	probeTimeout = 1500 * time.Millisecond
	con := newConsole(80, 24)
	opts := uv.DefaultOptions()
	opts.EventTimeout = 30 * time.Millisecond
	tm := uv.NewTerminal(con, opts)
	s := &session{con: con, t: tm, done: make(chan struct{})}
	go func() {
		err := runOn(
			tm,
			con.Getenv,
			start{kf: kf, window: kingfisher.Regions["sfbay"]},
			func(a *app) {
				s.mu.Lock()
				s.app = a
				s.mu.Unlock()
			},
		)
		s.mu.Lock()
		s.err = err
		s.mu.Unlock()
		close(s.done)
	}()
	for i := 0; i < 100 && !strings.Contains(con.output(), "\x1b[?1016$p"); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	con.send("\x1b[?1016;2$y")
	s.settle()
	if s.app == nil {
		t.Fatalf("did not start: %v", s.err)
	}
	defer func() { s.key("q"); <-s.done }()
	// The gap between fetches is tested on its own below; here every move
	// that leaves the held window should fetch at once.
	s.on(func(a *app) { a.shelf.gap = 0 })

	// One fetch on start, sized to the screen: 80 columns by 22 rows by 2.
	if fs.asks.Load() != 1 {
		t.Fatalf("%d fetches at start", fs.asks.Load())
	}
	if q := fs.last.Load().(string); !strings.Contains(q, "budget=3520") {
		t.Errorf("budget should be the pixel count: %q", q)
	}
	var cells int
	var msg string
	var pinned bool
	s.on(
		func(a *app) { cells = a.src.Len(); msg = a.msg; pinned = a.pin != nil },
	)
	if cells == 0 || !strings.Contains(msg, "fetched") || !pinned {
		t.Fatalf(
			"after start: %d cells, %q, pinned %v",
			cells,
			msg,
			pinned,
		)
	}

	// A small pan stays inside the grown window: no fetch.
	s.key("\x1b[C")
	s.settle()
	if fs.asks.Load() != 1 {
		t.Errorf("a small pan fetched: %d", fs.asks.Load())
	}
	// The window is grown 1.6x, a margin of thirty percent of the view
	// each side; a second quarter-window pan leaves it: one fetch.
	s.key("\x1b[C")
	s.settle()
	if fs.asks.Load() != 2 {
		t.Errorf(
			"leaving the window should fetch once: %d",
			fs.asks.Load(),
		)
	}
	// Zooming in past half the fetched span fetches for finer cells: four
	// wheel steps are 1.2 to the fourth, just over two.
	x, y := px(40, 12, 0)
	before := fs.asks.Load()
	s.wheelUp(x, y)
	s.wheelUp(x, y)
	s.wheelUp(x, y)
	s.settle()
	if fs.asks.Load() != before {
		t.Errorf(
			"three wheel steps stay inside the fetch: %d",
			fs.asks.Load()-before,
		)
	}
	s.wheelUp(x, y)
	s.settle()
	if fs.asks.Load() != before+1 {
		t.Errorf(
			"zooming in past half should fetch once: %d",
			fs.asks.Load()-before,
		)
	}
	// A drag that ends outside the window fetches on release, not per sample.
	before = fs.asks.Load()
	s.press(px(10, 12, 0))
	s.settle()
	for i := 0; i < 5; i++ {
		s.dragTo(px(10+i*14, 12, 0))
		s.settle()
	}
	s.release(px(78, 12, 0))
	s.settle()
	if got := fs.asks.Load() - before; got > 1 {
		t.Errorf("a drag fetched %d times", got)
	}
	// The header says what the shelf served.
	if !strings.Contains(stripSGR(con.output()), "shelf res 3") {
		t.Error("header should carry the shelf note")
	}
	// A refusal keeps the last cells and says why, with the numbers.
	fs.refuse.Store(true)
	for i := 0; i < 6; i++ {
		s.key("\x1b[C")
	}
	s.settle()
	s.on(func(a *app) { cells = a.src.Len(); msg = a.msg })
	if cells == 0 || !strings.Contains(msg, "refused") ||
		!strings.Contains(msg, "128Mi of 256Mi") {
		t.Errorf("refusal: %d cells, %q", cells, msg)
	}

	// With a gap, a burst of moves that each leave the held window asks
	// at most once, and the fetch that was held back still arrives when
	// the gap has passed, with no further key press.
	fs.refuse.Store(false)
	s.on(
		func(a *app) { a.shelf.gap = 1500 * time.Millisecond; a.shelf.at = time.Now() },
	)
	before = fs.asks.Load()
	for i := 0; i < 8; i++ {
		s.key("\x1b[C")
	}
	s.settle()
	if got := fs.asks.Load() - before; got != 0 {
		t.Errorf(
			"a burst inside the gap fetched %d times, want none yet",
			got,
		)
	}
	s.on(func(a *app) { msg = a.msg })
	if !strings.Contains(msg, "fetch waiting") {
		t.Errorf("status should say a fetch is waiting: %q", msg)
	}
	time.Sleep(1600 * time.Millisecond)
	s.settle()
	if got := fs.asks.Load() - before; got != 1 {
		t.Errorf(
			"after the gap %d fetches arrived, want exactly one",
			got,
		)
	}
}

func TestBBoxHelpers(t *testing.T) {
	b := kingfisher.BBox{-2, -1, 2, 1}
	if b.String() != "-2.000000,-1.000000,2.000000,1.000000" {
		t.Error(b.String())
	}
	g := b.Grown(2)
	if g != (kingfisher.BBox{-4, -2, 4, 2}) || !g.Covers(b) || b.Covers(g) {
		t.Errorf("grown %v", g)
	}
	if _, ok := kingfisher.Regions["world"]; !ok {
		t.Error("regions")
	}
}
