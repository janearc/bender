package main

import (
	"context"
	"fmt"
	"image/color"
	"math"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/screen"
	"github.com/charmbracelet/x/ansi"
	"github.com/uber/h3-go/v4"

	"github.com/janearc/angry-dist/kingfisher"
	"github.com/janearc/angry-dist/textmap"
)

// The window, top to bottom: a header row, the map, a status row.
const chromeRows = 2

// probeTimeout is how long detect waits for the terminal to answer.
var probeTimeout = 500 * time.Millisecond

// app is the program's state. One event loop touches it.
type app struct {
	t    *uv.Terminal
	scr  *uv.TerminalScreen
	ctx  *screen.Context
	caps caps

	src    *textmap.CellSource
	title  string
	layers []string
	layer  int

	globe bool
	proj  textmap.Projection
	cols  int
	rows  int
	frame *textmap.Frame
	pin   *[2]float64 // the colour scale, fixed to everything fetched
	style style       // palette and scale
	// fetch-on-move through the priced door, or nil for a file
	shelf *shelf
	// kf is the door itself, kept whether or not the shelf is in use, so
	// the view can change maps without being restarted. names is what it
	// said it had, asked for once.
	kf    *kingfisher.Client
	names []string
	which int
	dirty bool // the view changed; render a new frame
	// the frame was drawn quickly mid-drag; redraw fully when idle
	quick bool
	// background draws unsurveyed planet in a dark tint with a graticule,
	// so a globe reads as a globe against space.
	background bool

	pointer pointer
	drag    *dragState
	hover   h3.Cell
	hovered bool
	clicked string
	msg     string
	quit    bool
}

// pointer is where the mouse is, in map pixels.
type pointer struct {
	seen   bool
	sx, sy int // screen cell
	x, y   int // map pixel, which may be outside the map area
	on     bool
}

// dragState is a grab in progress: where the pointer was last, and whether
// it has moved, which tells a click from a drag.
type dragState struct {
	lastX, lastY int
	moved        bool
}

// start is what the program opens with: a source already fetched from a
// file with its manifest, or a shelf client and a window to ask it for.
type start struct {
	src    *textmap.CellSource
	m      *kingfisher.Manifest
	layer  string
	globe  bool
	kf     *kingfisher.Client // the priced door, when set
	window kingfisher.BBox    // the first view, for the shelf
	style  style              // palette and scale; the default when zero
}

// styled is the start's style, or the default when none was given, so a
// caller that predates styles still draws in the ramp it always did.
func (st start) styled() style {
	if st.style.palette.Stops == nil {
		return defaultStyle()
	}
	return st.style
}

// runApp opens the terminal, probes it, refuses or proceeds, and loops.
func runApp(st start) error {
	return runOn(uv.DefaultTerminal(), os.Getenv, st, nil)
}

// runOn runs on a terminal not yet started; the tests hand in a fake
// console and a hook that receives the app once the screen is up.
func runOn(
	t *uv.Terminal,
	env func(string) string,
	st start,
	ready func(*app),
) error {
	// Pin the width method so ultraviolet never enables Unicode core mode
	// (DEC 2027): in Ghostty 1.3.1 the frame never appears with it on,
	// found in daffy on 2026-09-07 and confirmed by flipping it. Every
	// glyph here is single width, so nothing is lost.
	t.Screen().SetWidthMethod(ansi.WcWidth)
	if err := t.Start(); err != nil {
		return err
	}
	stopped := false
	stop := func() {
		if !stopped {
			stopped = true
			// ultraviolet's Stop can nil its reader under the input
			// goroutine; a short settle closes the window in
			// practice.
			time.Sleep(30 * time.Millisecond)
			_ = t.Stop()
		}
	}
	defer stop()

	c := detect(t, env, probeTimeout)
	if missing := c.missing(); len(missing) > 0 {
		stop()
		fmt.Fprintln(
			os.Stderr,
			"this needs a terminal it cannot find here:",
		)
		for _, mm := range missing {
			fmt.Fprintln(os.Stderr, "  -", mm)
		}
		fmt.Fprintln(
			os.Stderr,
			"Ghostty, kitty and foot qualify, outside tmux.",
		)
		return fmt.Errorf("cannot run here")
	}

	a := newApp(t, c, st)
	if err := a.paint(); err != nil {
		return err
	}
	if ready != nil {
		ready(a)
	}
	for ev := range t.Events() {
		a.handle(ev)
		// A drag sends samples faster than frames can be drawn, so take
		// everything already queued before painting once. Otherwise the
		// queue grows and the map answers a pointer that has moved on.
	drain:
		for !a.quit {
			select {
			case more, ok := <-t.Events():
				if !ok {
					break drain
				}
				a.handle(more)
			default:
				break drain
			}
		}
		if a.quit {
			return nil
		}
		if err := a.paint(); err != nil {
			return err
		}
	}
	return nil
}

// newApp takes over the terminal and returns the view: alternate screen,
// true colour, cursor hidden, sized to the window it was handed.
//
// the shelf branch is taken only when there is no source of its own. a view
// opened from a file still carries the client, so it can ask for another
// dataset later, and testing the client alone sent a -dataset run down the
// shelf's path and crashed on a nil shelf.
func newApp(t *uv.Terminal, c caps, st start) *app {
	src, m, layer, globe := st.src, st.m, st.layer, st.globe
	scr := t.Screen()
	scr.Resize(c.cols, c.rows)
	scr.SetColorProfile(colorprofile.TrueColor)
	scr.EnterAltScreen()
	scr.HideCursor()
	scr.SetMouseMode(uv.MouseModeMotion)
	scr.SetMouseEncoding(uv.MouseEncodingSGRPixel)
	title := "shelf"
	if src != nil {
		title = src.Name
	}
	scr.SetWindowTitle("angry " + title)

	a := &app{
		t:          t,
		scr:        scr,
		ctx:        screen.NewContext(scr),
		caps:       c,
		src:        src,
		cols:       c.cols,
		rows:       c.rows,
		globe:      globe,
		background: true,
		style:      st.styled(),
		kf:         st.kf,
	}
	// the shelf is the mode with no source of its own, not merely one
	// with a door: a view opened on a dataset now carries the door too,
	// so that it can ask what else kingfisher has
	if st.kf != nil && st.src == nil {
		// The shelf: the first view is the window asked for, the source
		// arrives from the first refresh.
		a.shelf = &shelf{kf: st.kf, gap: minFetchGap}
		a.title = "shelf"
		a.src = textmap.NewCellSource("shelf", nil, nil)
		a.layers = []string{"value"}
		m = &kingfisher.Manifest{}
		m.Pipeline.BBox = st.window[:]
		a.reset(m)
		a.refresh(true)
		for i, l := range a.layers {
			if l == layer {
				a.layer = i
			}
		}
		a.pinScale()
		a.dirty = true
		return a
	}
	a.title = m.Title
	if a.title == "" {
		a.title = src.Name
	}
	a.layers = src.Layers()
	for i, l := range a.layers {
		if l == layer {
			a.layer = i
		}
	}
	a.reset(m)
	a.msg = "sub-cell mouse: DEC 1016 reports the pixel within the " +
		"character. drag pans, scroll zooms at the pointer, click " +
		"names a cell; g globe, l layer, ? help, q quit"
	return a
}

// mapRows is how many character rows the map has.
func (a *app) mapRows() int { return max(1, a.rows-chromeRows) }

// reset puts the view over the data: the flat window fitted to the
// manifest's bounding box, or the world; the globe facing the data's centre.
func (a *app) reset(m *kingfisher.Manifest) {
	lon, lat := a.src.Centre()
	span := 360.0
	if m == nil && a.shelf != nil && a.shelf.fetched != nil {
		b := a.shelf.fetched.Grown(1 / 1.6)
		m = &kingfisher.Manifest{}
		m.Pipeline.BBox = b[:]
	}
	if m != nil && len(m.Pipeline.BBox) == 4 {
		bb := m.Pipeline.BBox
		w, s, e, n := bb[0], bb[1], bb[2], bb[3]
		lon, lat = (w+e)/2, (s+n)/2
		span = (e - w) * 1.06
	}
	if a.globe {
		g := textmap.NewGlobe(lon, lat, a.cols, a.mapRows())
		if span < 180 {
			g.SetScale(span)
		}
		a.proj = g
	} else {
		a.proj = textmap.NewViewport(
			lon, lat, span, a.cols, a.mapRows())
	}
	a.dirty = true
}

// setCentre moves the view to a coordinate.
func (a *app) setCentre(lon, lat float64) {
	a.proj.SetCentre(lon, lat)
	a.dirty = true
}

// setGlobe switches projection, facing the same place at the same scale.
// The repl and the interactive "g" key both come through here, so the two
// ways of switching cannot drift apart on what switching means.
func (a *app) setGlobe(on bool) {
	lon, lat := a.proj.Centre()
	span := a.proj.Scale()
	a.globe = on
	if on {
		g := textmap.NewGlobe(lon, lat, a.cols, a.mapRows())
		if span < 180 {
			g.SetScale(span)
		}
		a.proj = g
	} else {
		a.proj = textmap.NewViewport(
			lon, lat, math.Min(360, span), a.cols, a.mapRows())
	}
	a.dirty = true
}

// setHorizon switches to the low-orbit view, looking at the same place at
// the same width. It clears globe, so "g" from here goes to the globe
// rather than to a flat map the operator never asked for.
func (a *app) setHorizon() {
	lon, lat := a.proj.Centre()
	span := a.proj.Scale()
	a.globe = false
	h := textmap.NewHorizon(lon, lat, a.cols, a.mapRows())
	if span < 90 {
		h.SetScale(span)
	}
	h.SetCentre(lon, lat)
	a.proj = h
	a.dirty = true
}

// viewName is what the header calls the projection. It reads the
// projection itself rather than the globe flag, since a flag has two
// values and there are now three views.
func (a *app) viewName() string {
	switch a.proj.(type) {
	case *textmap.Globe:
		return "globe"
	case *textmap.Horizon:
		return "horizon"
	}
	return "flat"
}

// setScale sets how wide the window is. Its pair is setCentre: scale is
// the size of the window and centre is where it sits, and neither derives
// from the other, so they may be set in either order.
func (a *app) setScale(span float64) {
	switch p := a.proj.(type) {
	case *textmap.Globe:
		p.SetScale(span)
	case *textmap.Horizon:
		p.SetScale(span)
	case *textmap.Viewport:
		p.SpanX = math.Max(1e-7, math.Min(360, span))
	}
	a.dirty = true
}

// res is the H3 resolution the current view draws at.
func (a *app) res() int {
	return textmap.ResForScale(a.proj.Scale(), a.cols, a.src.NativeRes())
}

// callEvent runs a function on the event loop, for tests.
type callEvent struct {
	f    func(*app)
	done chan struct{}
}

// handle routes one event.
func (a *app) handle(ev uv.Event) {
	switch e := ev.(type) {
	case callEvent:
		e.f(a)
		close(e.done)
	case uv.WindowSizeEvent:
		a.cols, a.rows = e.Width, e.Height
		a.scr.Resize(e.Width, e.Height)
		ws, err := a.t.GetWinsize()
		if err == nil && ws != nil && ws.Col > 0 && ws.Row > 0 {
			a.caps.cellW = int(ws.Xpixel) / int(ws.Col)
			a.caps.cellH = int(ws.Ypixel) / int(ws.Row)
		}
		a.proj.Resize(a.cols, a.mapRows())
		a.dirty = true
		a.refresh(false)
	case uv.KeyPressEvent:
		a.key(e)
	case uv.MouseClickEvent:
		a.track(e.X, e.Y)
		if e.Button == uv.MouseLeft && a.pointer.on {
			a.drag = &dragState{
				lastX: a.pointer.x,
				lastY: a.pointer.y,
			}
		}
	case uv.MouseReleaseEvent:
		a.track(e.X, e.Y)
		if a.drag != nil && !a.drag.moved {
			a.click()
		}
		if a.drag != nil && a.drag.moved {
			a.dirty = true // the full frame, now the drag is over
			a.refresh(false)
		}
		a.drag = nil
	case uv.MouseMotionEvent:
		a.track(e.X, e.Y)
		if a.drag != nil && e.Button == uv.MouseLeft {
			moved := a.pointer.x != a.drag.lastX ||
				a.pointer.y != a.drag.lastY
			if moved {
				fx, fy := float64(a.drag.lastX)+0.5,
					float64(a.drag.lastY)+0.5
				tx, ty := float64(a.pointer.x)+0.5,
					float64(a.pointer.y)+0.5
				textmap.Drag(a.proj, fx, fy, tx, ty)
				a.drag.lastX = a.pointer.x
				a.drag.lastY = a.pointer.y
				a.drag.moved = true
				a.dirty = true
			}
		}
		a.updateHover()
	case uv.MouseWheelEvent:
		a.track(e.X, e.Y)
		factor := 1.0
		switch e.Button {
		case uv.MouseWheelUp:
			factor = 1 / 1.2
		case uv.MouseWheelDown:
			factor = 1.2
		}
		if factor != 1 && a.pointer.on {
			px, py := float64(a.pointer.x)+0.5,
				float64(a.pointer.y)+0.5
			textmap.ZoomAt(a.proj, factor, px, py)
			a.dirty = true
			a.refresh(false)
			a.updateHover()
		}
	}
}

// track converts a pixel mouse report into a screen cell, the half of it
// the pointer is in, and the map pixel. With DEC 1016 the report carries
// pixels, so the half is known, which is the whole point.
func (a *app) track(x, y int) {
	cw, ch := a.caps.cellW, a.caps.cellH
	if cw <= 0 || ch <= 0 {
		cw, ch = 1, 2
	}
	a.pointer.seen = true
	a.pointer.sx, a.pointer.sy = x/cw, y/ch
	half := 0
	if y%ch >= ch/2 {
		half = 1
	}
	a.pointer.x = a.pointer.sx
	a.pointer.y = (a.pointer.sy-1)*2 + half
	w, h := a.proj.Pixels()
	a.pointer.on = a.pointer.sy >= 1 && a.pointer.sy < 1+a.mapRows() &&
		a.pointer.x >= 0 &&
		a.pointer.x < w &&
		a.pointer.y >= 0 &&
		a.pointer.y < h
}

// updateHover finds the cell under the pointer.
func (a *app) updateHover() {
	if !a.pointer.on {
		a.hovered = false
		return
	}
	c, _, ok := textmap.CellAt(
		a.src,
		a.proj,
		a.layers[a.layer],
		a.res(),
		float64(a.pointer.x)+0.5,
		float64(a.pointer.y)+0.5,
	)
	if ok != a.hovered || c != a.hover {
		// Moving to another cell lets the hover line back in.
		a.clicked = ""
	}
	a.hover, a.hovered = c, ok
}

// click names the cell under the pointer in the status line.
func (a *app) click() {
	if !a.pointer.on {
		return
	}
	c, v, ok := textmap.CellAt(
		a.src,
		a.proj,
		a.layers[a.layer],
		a.res(),
		float64(a.pointer.x)+0.5,
		float64(a.pointer.y)+0.5,
	)
	if !ok {
		lon, lat, on := a.proj.Unproject(
			float64(a.pointer.x)+0.5,
			float64(a.pointer.y)+0.5,
		)
		if !on {
			a.clicked = "clicked space"
		} else {
			a.clicked = fmt.Sprintf("clicked %.3f, %.3f: nothing "+
				"surveyed there", lon, lat)
		}
		return
	}
	ll, _ := h3.CellToLatLng(c)
	a.clicked = fmt.Sprintf(
		"clicked %s  res %d  %s %.3f  centre %.3f, %.3f",
		c,
		c.Resolution(),
		a.layers[a.layer],
		v,
		ll.Lng,
		ll.Lat,
	)
}

// key handles one key press.
func (a *app) key(k uv.KeyPressEvent) {
	key := uv.Key(k)
	switch {
	case key.MatchString("q", "ctrl+c"):
		a.quit = true
	case key.MatchString("g"):
		a.setGlobe(!a.globe)
		which := map[bool]string{true: "globe", false: "flat map"}
		a.msg = which[a.globe]
		a.refresh(false)
	case key.MatchString("h"):
		a.setHorizon()
		a.msg = "horizon: up flies forward, left and right turn," +
			" + and - change height, [ ] the lens, { } the pitch"
		a.refresh(false)
	case key.MatchString("l"):
		a.layer = (a.layer + 1) % len(a.layers)
		a.pinScale()
		a.dirty = true
		a.msg = "layer " + a.layers[a.layer]
	case key.MatchString("r"):
		a.reset(nil)
		a.refresh(false)
		a.msg = "view reset"
	case k.Text == "+" || key.MatchString("="):
		a.proj.Zoom(1 / 1.2)
		a.dirty = true
		a.refresh(false)
	case key.MatchString("-"):
		a.proj.Zoom(1.2)
		a.dirty = true
		a.refresh(false)
	case key.MatchString("left"):
		a.proj.Pan(-0.25, 0)
		a.dirty = true
		a.refresh(false)
	case key.MatchString("right"):
		a.proj.Pan(0.25, 0)
		a.dirty = true
		a.refresh(false)
	case key.MatchString("up"):
		a.proj.Pan(0, 0.25)
		a.dirty = true
		a.refresh(false)
	case key.MatchString("down"):
		a.proj.Pan(0, -0.25)
		a.dirty = true
		a.refresh(false)
	case key.MatchString("["), key.MatchString("]"):
		// focal length, on the horizon camera: a narrower field of
		// view is a longer lens, and the limb stays where Sky put it
		// because the pitch is derived from it rather than stored
		if h, ok := a.proj.(*textmap.Horizon); ok {
			by := 1.15
			if key.MatchString("[") {
				by = 1 / 1.15
			}
			h.FOV = math.Max(8, math.Min(140, h.FOV*by))
			a.msg = fmt.Sprintf("field of view %.0f degrees", h.FOV)
			a.dirty = true
			a.refresh(false)
		} else {
			a.msg = "the lens is the horizon camera's; press h"
		}
	case key.MatchString("{"),
		key.MatchString("}"),
		key.MatchString("shift+["),
		key.MatchString("shift+]"):
		// pitch, said as where the limb sits: Sky is the share of the
		// frame above it, so raising Sky tilts the camera up
		if h, ok := a.proj.(*textmap.Horizon); ok {
			by := 0.04
			if key.MatchString("{") || key.MatchString("shift+[") {
				by = -0.04
			}
			h.Sky = math.Max(0.05, math.Min(0.95, h.Sky+by))
			a.msg = fmt.Sprintf(
				"horizon at %.0f%% of the frame",
				(1-h.Sky)*100,
			)
			a.dirty = true
			a.refresh(false)
		} else {
			a.msg = "the pitch is the horizon camera's; press h"
		}
	case key.MatchString("d"), key.MatchString("shift+d"):
		// change maps without restarting. the palette follows the data
		// rather than the session: a night-lights layer is unreadable
		// in heat and a radius layer is unreadable in night.
		back := key.MatchString("shift+d")
		if err := a.cycleDataset(back); err != nil {
			a.msg = err.Error()
		}
	case key.MatchString("b"):
		a.background = !a.background
		on := "background on: unsurveyed planet is tinted with a " +
			"graticule; space is blank"
		a.msg = map[bool]string{
			true:  on,
			false: "background off",
		}[a.background]
	case key.MatchString("?"):
		a.msg = "drag pans, wheel zooms at the pointer, click names a" +
			" cell; g globe, h horizon, b background, l layer, r" +
			" reset, arrows pan, + - zoom, [ ] lens, { } pitch, q" +
			" quit"
	}
}

// Colours for the chrome and the highlight.
var (
	chromeBg = color.RGBA{0x10, 0x10, 0x16, 0xff}
	chromeFg = color.RGBA{0xd8, 0xd8, 0xdc, 0xff}
	chromeHi = color.RGBA{0xff, 0xd2, 0x2a, 0xff}
)

// paint renders a frame if the view changed, then draws it with the
// hovered cell lit, and flushes.
func (a *app) paint() error {
	if a.dirty || a.frame == nil {
		quick := a.drag != nil && a.drag.moved
		a.frame = textmap.RenderWith(
			a.src,
			a.proj,
			a.layers[a.layer],
			a.res(),
			textmap.Options{
				Quick: quick,
				Pin:   a.pin,
				Scale: a.style.scale,
			},
		)
		a.quick = quick
		a.dirty = false
	}
	screen.Clear(a.scr)
	a.ctx.Reset()
	a.paintMap()
	a.paintHeader()
	a.paintStatus()
	a.scr.Render()
	return a.scr.Flush()
}

// The tints for unsurveyed planet and its graticule: dark, unmistakably
// not a value on the ramp, and enough to show the disc against space.
var (
	planetTint = textmap.RGB{R: 0x14, G: 0x1c, B: 0x2a}
	gridTint   = textmap.RGB{R: 0x2c, G: 0x3a, B: 0x50}
)

// shade is the colour of a pixel, or false for nothing: a data pixel in
// the ramp, lit when it belongs to the hovered cell; unsurveyed planet in
// the tint when the background is on; space never.
func (a *app) shade(p textmap.Pixel) (color.RGBA, bool) {
	switch {
	case p.Set:
		c := a.style.palette.At(p.T)
		if a.hovered && p.Cell == a.hover {
			return color.RGBA{
				uint8(int(c.R)/2 + 128),
				uint8(int(c.G)/2 + 128),
				uint8(int(c.B)/2 + 128),
				0xff,
			}, true
		}
		return color.RGBA{c.R, c.G, c.B, 0xff}, true
	case p.Planet && a.background:
		t := planetTint
		if p.Grid {
			t = gridTint
		}
		return color.RGBA{t.R, t.G, t.B, 0xff}, true
	case p.Glow > 0 && a.background:
		c := textmap.Atmosphere(p.Glow)
		return color.RGBA{c.R, c.G, c.B, 0xff}, true
	}
	return color.RGBA{}, false
}

// paintMap draws the frame as half blocks. The unpainted half of a cell is
// never painted: space stays blank.
func (a *app) paintMap() {
	f := a.frame
	for row := 0; row < a.mapRows(); row++ {
		for x := 0; x < a.cols && x < f.W; x++ {
			top, okT := a.shade(f.At(x, row*2))
			bot, okB := a.shade(f.At(x, row*2+1))
			switch {
			case !okT && !okB:
				continue
			case okT && okB:
				a.ctx.SetForeground(top)
				a.ctx.SetBackground(bot)
				a.ctx.DrawString("▀", x, row+1)
			case okT:
				a.ctx.SetForeground(top)
				a.ctx.SetBackground(nil)
				a.ctx.DrawString("▀", x, row+1)
			default:
				a.ctx.SetForeground(bot)
				a.ctx.SetBackground(nil)
				a.ctx.DrawString("▄", x, row+1)
			}
		}
	}
}

// paintHeader names the dataset, the view and the thing being demonstrated.
func (a *app) paintHeader() {
	a.ctx.SetForeground(chromeFg)
	a.ctx.SetBackground(chromeBg)
	a.ctx.DrawString(strings.Repeat(" ", a.cols), 0, 0)
	view := a.viewName()
	lon, lat := a.proj.Centre()
	left := fmt.Sprintf(
		" %s  %s  res %d of %d  %s  centre %.2f, %.2f  span %.1f°%s",
		a.title,
		a.layers[a.layer],
		a.frame.Res,
		a.src.NativeRes(),
		view,
		lon,
		lat,
		a.proj.Scale(),
		a.shelfNote(),
	)
	a.ctx.DrawString(clip(left, a.cols), 0, 0)
	right := "sub-cell mouse: DEC 1016 reports the pixel within the " +
		"character "
	if len(left)+len(right) < a.cols {
		a.ctx.SetForeground(chromeHi)
		a.ctx.DrawString(right, a.cols-len(right), 0)
	}
}

// paintStatus says what is under the pointer, and what was clicked.
func (a *app) paintStatus() {
	y := a.rows - 1
	a.ctx.SetForeground(chromeFg)
	a.ctx.SetBackground(chromeBg)
	a.ctx.DrawString(strings.Repeat(" ", a.cols), 0, y)
	s := a.msg
	switch {
	case a.clicked != "":
		s = a.clicked
	case a.hovered:
		_, v, _ := textmap.CellAt(
			a.src,
			a.proj,
			a.layers[a.layer],
			a.res(),
			float64(a.pointer.x)+0.5,
			float64(a.pointer.y)+0.5,
		)
		s = fmt.Sprintf(
			"hover %s  res %d  %s %.3f  pixel %d,%d in cell %d,%d",
			a.hover,
			a.hover.Resolution(),
			a.layers[a.layer],
			v,
			a.pointer.x,
			a.pointer.y,
			a.pointer.sx,
			a.pointer.sy,
		)
	case a.pointer.on:
		s = fmt.Sprintf(
			"pixel %d,%d: nothing surveyed",
			a.pointer.x,
			a.pointer.y,
		)
	}
	a.ctx.DrawString(clip(" "+s, a.cols), 0, y)
}

// clip cuts a string to n runes, not n bytes: the status line carries
// dataset names, and cutting mid-rune paints a replacement character.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:max(0, n)])
	}
	return s
}

// cycleDataset moves to the next map kingfisher has, or the one before.
//
// The catalogue is asked for once and kept: it is a list of names and it
// does not change while somebody is looking at a globe.
func (a *app) cycleDataset(back bool) error {
	if a.kf == nil {
		return fmt.Errorf(
			"this view has no kingfisher to ask; it was opened " +
				"from a file",
		)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if len(a.names) == 0 {
		names, err := a.kf.Datasets(ctx)
		if err != nil {
			return fmt.Errorf("kingfisher: %v", err)
		}
		a.names = names
		for i, n := range names {
			if a.src != nil && n == a.src.Name {
				a.which = i
			}
		}
	}
	if len(a.names) == 0 {
		return fmt.Errorf("kingfisher has no datasets")
	}
	step := 1
	if back {
		step = -1
	}
	a.which = (a.which + step + len(a.names)) % len(a.names)
	name := a.names[a.which]
	m, err := a.kf.Manifest(ctx, name)
	if err != nil {
		return fmt.Errorf("%s: %v", name, err)
	}
	cells, err := a.kf.Cells(ctx, name)
	if err != nil {
		return fmt.Errorf("%s: %v", name, err)
	}
	src := textmap.NewCellSource(name, cells.Cells, m.LayerKinds())
	if src.Len() == 0 {
		return fmt.Errorf("%s: no cells", name)
	}
	// keep the layer across the switch when the new map has one by that
	// name, since stepping through maps is usually comparing the same
	// measurement
	held := ""
	if a.layer < len(a.layers) {
		held = a.layers[a.layer]
	}
	a.src, a.layers, a.layer = src, src.Layers(), 0
	for i, l := range a.layers {
		if l == held {
			a.layer = i
		}
	}
	a.style = styleFor(name, a.layers[a.layer], a.style)
	a.shelf = nil
	a.reset(m)
	a.refresh(true)
	a.msg = fmt.Sprintf(
		"%s, %s, %s",
		name,
		a.layers[a.layer],
		a.style.palette.Name,
	)
	return nil
}
