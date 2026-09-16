package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/term"
	"github.com/uber/h3-go/v4"

	"github.com/janearc/angry-dist/kingfisher"
	"github.com/janearc/angry-dist/textmap"
)

// A console over a pipe, so the program can be driven with the bytes a
// terminal would send, without a window.
type console struct {
	in   *io.PipeReader
	feed *io.PipeWriter
	mu   sync.Mutex
	out  bytes.Buffer
	env  map[string]string
	ws   uv.Winsize
}

func newConsole(cols, rows int) *console {
	r, w := io.Pipe()
	return &console{
		in:   r,
		feed: w,
		env:  map[string]string{"TERM": "xterm-ghostty"},
		ws: uv.Winsize{
			Col:    uint16(cols),
			Row:    uint16(rows),
			Xpixel: uint16(cols * 19),
			Ypixel: uint16(rows * 42),
		},
	}
}

func (c *console) Read(p []byte) (int, error) { return c.in.Read(p) }
func (c *console) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.out.Write(p)
}
func (c *console) Close() error { return c.in.Close() }

func (c *console) Environ() []string      { return []string{"TERM=xterm-ghostty"} }
func (c *console) Getenv(k string) string { return c.env[k] }

func (c *console) LookupEnv(
	k string,
) (string, bool) {
	v, ok := c.env[k]
	return v, ok
}
func (c *console) Reader() io.Reader             { return c.in }
func (c *console) Writer() io.Writer             { return c }
func (c *console) MakeRaw() (*term.State, error) { return nil, nil }
func (c *console) Restore() error                { return nil }

func (c *console) GetSize() (int, int, error) { return int(c.ws.Col), int(c.ws.Row), nil }

func (c *console) GetWinsize() (*uv.Winsize, error) { ws := c.ws; return &ws, nil }
func (c *console) output() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.out.String()
}
func (c *console) send(s string) { _, _ = io.WriteString(c.feed, s) }

// A small synthetic dataset: every res-3 cell in a patch around San
// Francisco, valued by longitude so the picture has a gradient.
func synthetic() (*textmap.CellSource, *kingfisher.Manifest) {
	centre, _ := h3.LatLngToCell(h3.LatLng{Lat: 37.7, Lng: -122.4}, 3)
	ring, _ := centre.GridDisk(6)
	cells := map[string]any{}
	for _, c := range ring {
		ll, _ := c.LatLng()
		cells[c.String()] = map[string]any{"v": ll.Lng}
	}
	src := textmap.NewCellSource(
		"synthetic",
		cells,
		map[string]string{"v": "LAYER_KIND_INTENSIVE"},
	)
	m := &kingfisher.Manifest{ID: "synthetic", Title: "Synthetic"}
	m.Pipeline.BBox = []float64{-126, 34, -118, 41}
	return src, m
}

type session struct {
	con  *console
	t    *uv.Terminal
	app  *app
	err  error
	done chan struct{}
	mu   sync.Mutex
}

func launch(t *testing.T, cols, rows int, globe bool) *session {
	t.Helper()
	probeTimeout = 1500 * time.Millisecond
	con := newConsole(cols, rows)
	opts := uv.DefaultOptions()
	opts.EventTimeout = 30 * time.Millisecond
	tm := uv.NewTerminal(con, opts)
	s := &session{con: con, t: tm, done: make(chan struct{})}
	src, m := synthetic()
	go func() {
		err := runOn(
			tm,
			con.Getenv,
			start{src: src, m: m, layer: "v", globe: globe},
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
	// Answer the probe once it has been asked.
	for i := 0; i < 100 && !strings.Contains(con.output(), "\x1b[?1016$p"); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	con.send("\x1b[?1016;2$y")
	s.settle()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.app == nil {
		t.Fatalf("did not start: %v", s.err)
	}
	return s
}

// settle waits for everything sent to be handled, by running an empty call
// on the event loop after a pause for the bytes to cross the pipe.
func (s *session) settle() {
	time.Sleep(60 * time.Millisecond)
	select {
	case <-s.done:
		return
	default:
	}
	ev := callEvent{f: func(*app) {}, done: make(chan struct{})}
	s.t.SendEvent(ev)
	select {
	case <-ev.done:
	case <-s.done:
	case <-time.After(2 * time.Second):
	}
}

func (s *session) on(f func(*app)) {
	ev := callEvent{f: f, done: make(chan struct{})}
	s.t.SendEvent(ev)
	select {
	case <-ev.done:
	case <-s.done:
	case <-time.After(2 * time.Second):
	}
}

// Mouse in SGR-pixel form, one-based on the wire. Cell (col,row) with the
// pointer in its top or bottom half.
func px(
	col, row, half int,
) (int, int) {
	return col*19 + 9 + 1, row*42 + 10 + half*21 + 1
}

func (s *session) press(
	x, y int,
) {
	s.con.send(fmt.Sprintf("\x1b[<0;%d;%dM", x, y))
}

func (s *session) dragTo(
	x, y int,
) {
	s.con.send(fmt.Sprintf("\x1b[<32;%d;%dM", x, y))
}

func (s *session) release(
	x, y int,
) {
	s.con.send(fmt.Sprintf("\x1b[<0;%d;%dm", x, y))
}

func (s *session) move(
	x, y int,
) {
	s.con.send(fmt.Sprintf("\x1b[<35;%d;%dM", x, y))
}

func (s *session) wheelUp(
	x, y int,
) {
	s.con.send(fmt.Sprintf("\x1b[<64;%d;%dM", x, y))
}
func (s *session) key(k string) {
	s.con.send(k)
	time.Sleep(45 * time.Millisecond)
}

func TestRefusesWithoutPixelMouse(t *testing.T) {
	probeTimeout = 300 * time.Millisecond
	con := newConsole(80, 24)
	con.env["TMUX"] = "/tmp/tmux-1/default,1,0"
	src, m := synthetic()
	err := runOn(
		uv.NewTerminal(con, nil),
		con.Getenv,
		start{src: src, m: m, layer: "v"},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "cannot run here") {
		t.Fatalf("err %v", err)
	}
	if strings.Contains(con.output(), "\x1b[?1049h") {
		t.Error("entered the alternate screen before refusing")
	}
}

func TestMouseDrivesTheMap(t *testing.T) {
	s := launch(t, 80, 24, false)
	_ = s.app
	defer func() { s.key("q"); <-s.done }()

	// The first frame drew something in half blocks, with true colour, and
	// the header says what is being demonstrated.
	out := s.con.output()
	if !strings.Contains(out, "▀") && !strings.Contains(out, "▄") {
		t.Fatal("no map drawn")
	}
	if !strings.Contains(out, "38;2;") ||
		!strings.Contains(out, "DEC 1016") {
		t.Error("colour or header missing")
	}
	var before, after struct{ lon, lat float64 }
	s.on(func(a *app) { before.lon, before.lat = a.proj.Centre() })

	// Hover names a cell; the status line shows it.
	x, y := px(40, 12, 0)
	s.move(x, y)
	s.settle()
	var hovered bool
	var status string
	s.on(func(a *app) { hovered = a.hovered; status = a.msg })
	if !hovered {
		t.Fatalf("hover found nothing at the centre: %q", status)
	}
	if !strings.Contains(s.con.output(), "hover 83") {
		t.Error("status line should name the hovered res-3 cell")
	}

	// A drag moves the centre by the grabbed distance, and the coordinate
	// that was under the pointer is now under the pointer.
	var grabbed struct{ lon, lat float64 }
	s.on(
		func(a *app) { grabbed.lon, grabbed.lat, _ = a.proj.Unproject(40.5, 22.5) },
	)
	s.press(x, y)
	s.settle()
	x2, y2 := px(50, 10, 1)
	s.dragTo(x2, y2)
	s.settle()
	s.release(x2, y2)
	s.settle()
	s.on(func(a *app) { after.lon, after.lat = a.proj.Centre() })
	if after.lon >= before.lon || after.lat >= before.lat {
		t.Errorf(
			"dragging the map right and up moves the centre west and south: %+v -> %+v",
			before,
			after,
		)
	}
	var under struct{ lon, lat float64 }
	s.on(
		func(a *app) { under.lon, under.lat, _ = a.proj.Unproject(50.5, 19.5) },
	)
	if math.Abs(under.lon-grabbed.lon) > 0.05 ||
		math.Abs(under.lat-grabbed.lat) > 0.05 {
		t.Errorf(
			"grabbed %+v is now %+v under the pointer",
			grabbed,
			under,
		)
	}
	var clicked string
	s.on(func(a *app) { clicked = a.clicked })
	if clicked != "" {
		t.Error("a drag is not a click")
	}

	// A click names the cell under the pointer.
	x3, y3 := px(30, 8, 1)
	s.press(x3, y3)
	s.settle()
	s.release(x3, y3)
	s.settle()
	s.on(func(a *app) { clicked = a.clicked })
	if !strings.HasPrefix(clicked, "clicked 83") ||
		!strings.Contains(clicked, "res 3") ||
		!strings.Contains(clicked, "v -1") {
		t.Errorf("click: %q", clicked)
	}
	// Moving to another cell lets hover back in.
	s.move(px(70, 3, 0))
	s.settle()
	s.on(func(a *app) { clicked = a.clicked })
	if clicked != "" {
		t.Error("moving away should clear the click")
	}

	// Scroll zooms with the coordinate under the pointer held still.
	var anchor struct{ lon, lat float64 }
	var scale0, scale1 float64
	s.on(
		func(a *app) { anchor.lon, anchor.lat, _ = a.proj.Unproject(70.5, 4.5); scale0 = a.proj.Scale() },
	)
	s.wheelUp(px(70, 3, 0))
	s.settle()
	s.on(
		func(a *app) { under.lon, under.lat, _ = a.proj.Unproject(70.5, 4.5); scale1 = a.proj.Scale() },
	)
	if scale1 >= scale0 {
		t.Errorf("wheel up should zoom in: %v -> %v", scale0, scale1)
	}
	if math.Abs(under.lon-anchor.lon) > 0.01 ||
		math.Abs(under.lat-anchor.lat) > 0.01 {
		t.Errorf("zoom moved the anchor: %+v -> %+v", anchor, under)
	}

	// g swaps to the globe, keeping the centre, and the map still drives.
	s.key("g")
	s.settle()
	var isGlobe bool
	s.on(
		func(a *app) { _, isGlobe = a.proj.(*textmap.Globe); before.lon, before.lat = a.proj.Centre() },
	)
	if !isGlobe {
		t.Fatal("g should swap to the globe")
	}
	var globeScale float64
	s.on(func(a *app) { globeScale = a.proj.Scale() })
	if math.Abs(globeScale-scale1) > 1e-6 {
		t.Errorf(
			"the globe should keep the flat map's scale: %v vs %v",
			globeScale,
			scale1,
		)
	}
	if math.Abs(before.lon-after.lon) > 0.5 &&
		math.Abs(before.lon-anchor.lon) > 5 {
		t.Errorf("globe should face where the map was: %+v", before)
	}
	s.press(x, y)
	s.settle()
	s.dragTo(x2, y2)
	s.settle()
	s.release(x2, y2)
	s.settle()
	s.on(func(a *app) { after.lon, after.lat = a.proj.Centre() })
	if after.lon == before.lon && after.lat == before.lat {
		t.Error("dragging the globe should turn it")
	}
	s.wheelUp(x, y)
	s.settle()
	s.key("g")
	s.settle()
	s.on(func(a *app) { _, isGlobe = a.proj.(*textmap.Globe) })
	if isGlobe {
		t.Error("g again should swap back")
	}

	// Keys: layer, reset, pan and zoom; a resize follows the window.
	s.key("l")
	s.key("r")
	s.key("\x1b[C")
	s.key("+")
	s.key("-")
	s.key("?")
	s.settle()
	var msg string
	s.on(func(a *app) { msg = a.msg })
	if !strings.Contains(msg, "drag pans") {
		t.Errorf("help %q", msg)
	}
	s.con.ws.Col, s.con.ws.Row = 100, 30
	s.on(
		func(a *app) { a.handle(uv.WindowSizeEvent{Width: 100, Height: 30}) },
	)
	s.settle()
	var w, h int
	s.on(func(a *app) { w, h = a.proj.Pixels() })
	if w != 100 || h != 56 {
		t.Errorf("after resize the map is %d by %d pixels", w, h)
	}
}
