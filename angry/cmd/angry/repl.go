package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/janearc/angry-dist/kingfisher"
	"github.com/janearc/angry-dist/textmap"
)

// repl drives the frame path from stdin: one command line in, one frame
// out, in exactly the bytes -frame writes. It exists because a flag is a
// decision made before the process starts, and a console pane driving a
// recording has to aim the view while it is running.
type repl struct {
	kf    *kingfisher.Client
	out   io.Writer
	errw  io.Writer
	a     *app
	layer string
	cols  int
	rows  int
	fps   int // frames a second during a move
	// redraw in place, for a pane watched directly
	live  bool
	sleep func(time.Duration) // time.Sleep, replaced in tests
}

// defaultFPS is the frame rate of a move when -fps is not given. Twelve is
// smooth enough for a globe turning in half blocks and leaves each frame
// eighty milliseconds, where a horizon frame takes about twelve to draw.
const defaultFPS = 12

// runRepl reads commands until stdin ends or the operator stops it. A bad
// command is reported and the loop continues: a typo during a take must
// not kill the process, and must not put anything on stdout, which is a
// frame stream somebody is decoding.
func runRepl(
	ctx context.Context,
	kf *kingfisher.Client,
	in io.Reader,
	out, errw io.Writer,
	src *textmap.CellSource,
	m *kingfisher.Manifest,
	layer string,
	opts options,
	view *aim,
	st style,
) error {
	cols, rows := frameSize(opts.cols, opts.rows)
	r := &repl{
		kf:    kf,
		out:   out,
		errw:  errw,
		layer: layer,
		cols:  cols,
		rows:  rows,
		fps:   opts.fps,
		live:  opts.live,
		sleep: time.Sleep,
	}
	if r.fps <= 0 {
		r.fps = defaultFPS
	}
	if r.live {
		// Clear once and hide the cursor, so the frames that follow
		// land on the same rows and a pane plays them like video; show
		// the cursor again however the session ends.
		io.WriteString(out, "\x1b[2J\x1b[?25l")
		defer io.WriteString(out, "\x1b[?25h")
	}
	r.a = newFrameApp(src, m, layer, opts.globe, cols, rows)
	if st.palette.Stops != nil {
		r.a.style = st
	}
	if opts.span > 0 {
		r.a.setScale(opts.span)
	}
	if view != nil {
		r.a.setCentre(view.lon, view.lat)
	}
	// One frame before any command, so a pane that attaches and waits has
	// a picture rather than an empty region.
	if err := r.draw(); err != nil {
		return err
	}
	sc := bufio.NewScanner(in)
	// A dataset name is short, but the reader is a pipe from another
	// process and a long line should be a command error rather than a
	// truncated command silently doing something else.
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		if stop := r.do(ctx, strings.Fields(sc.Text())); stop {
			return nil
		}
	}
	return sc.Err()
}

// do runs one command line and redraws. It reports true when the operator
// asked to stop. Commands that fail report why and leave the view alone,
// so the last good frame stays on screen.
func (r *repl) do(ctx context.Context, f []string) bool {
	if len(f) == 0 {
		// A blank line redraws: a console can hold a beat without
		// moving.
		r.draw()
		return false
	}
	switch f[0] {
	case "quit", "exit":
		return true

	case "globe":
		r.a.setGlobe(true)

	case "flat":
		r.a.setGlobe(false)

	case "horizon":
		r.a.setHorizon()

	case "look", "centre", "center":
		// Both "look 10 20" and "look 10,20" are accepted: one is what
		// a person types, the other is what the flag takes.
		//
		// The two spellings of centre still work, unadvertised, because
		// the person most likely to type one is the person who asked
		// for "look" so she would not have to remember which.
		c, err := parseLook(strings.Join(f[1:], ","))
		if err != nil {
			return r.fail("%v", err)
		}
		if c == nil {
			return r.fail("look wants lon lat")
		}
		r.a.setCentre(c.lon, c.lat)

	case "palette":
		if len(f) != 2 {
			return r.fail("palette wants one name")
		}
		st, err := r.a.style.withPalette(f[1])
		if err != nil {
			return r.fail("%v", err)
		}
		r.a.style = st

	case "floor":
		if len(f) != 2 {
			return r.fail("floor wants a number or off")
		}
		st, err := r.a.style.withFloor(f[1])
		if err != nil {
			return r.fail("%v", err)
		}
		r.a.style = st

	case "fly":
		return r.move("fly", f[1:], 0, 1)

	case "spin":
		return r.move("spin", f[1:], 1, 0)

	case "span":
		if len(f) != 2 {
			return r.fail("span wants one number in degrees")
		}
		deg, err := strconv.ParseFloat(f[1], 64)
		if err != nil || deg <= 0 || deg > 360 {
			return r.fail(
				"span wants degrees above 0 and up to"+
					" 360, not %q",
				f[1],
			)
		}
		r.a.setScale(deg)

	case "layer":
		if len(f) != 2 {
			return r.fail(
				"layer wants one name of %s",
				strings.Join(r.a.layers, ", "),
			)
		}
		if !r.setLayer(f[1]) {
			return r.fail(
				"no layer %q; it has %s",
				f[1],
				strings.Join(r.a.layers, ", "),
			)
		}

	case "dataset":
		if len(f) != 2 {
			return r.fail("dataset wants one name")
		}
		if err := r.setDataset(ctx, f[1]); err != nil {
			return r.fail("%v", err)
		}

	default:
		return r.fail(
			"unknown command %q; try globe, flat, horizon,"+
				" look, span, fly, spin, palette, floor,"+
				" layer, dataset, quit",
			f[0],
		)
	}
	r.draw()
	return false
}

// setLayer picks a layer by name and repins the colour scale to it,
// reporting whether the dataset has one by that name.
func (r *repl) setLayer(name string) bool {
	for i, l := range r.a.layers {
		if l == name {
			r.a.layer = i
			r.layer = l
			r.a.pinScale()
			r.a.dirty = true
			return true
		}
	}
	return false
}

// setDataset fetches another dataset whole and fits the view to it. The
// view is refitted rather than kept, because the new data may be nowhere
// near the old centre and a frame of empty space is worse than a move the
// console can see in the header and correct with one more command.
func (r *repl) setDataset(ctx context.Context, name string) error {
	m, err := r.kf.Manifest(ctx, name)
	if err != nil {
		return err
	}
	cells, err := r.kf.Cells(ctx, name)
	if err != nil {
		return err
	}
	src := textmap.NewCellSource(name, cells.Cells, m.LayerKinds())
	if src.Len() == 0 {
		return fmt.Errorf("%s: no valid cells", name)
	}
	// Keep the layer across the switch when the new dataset has one by
	// that name, since a console stepping through datasets is usually
	// comparing the same measurement.
	layer := src.Layers()[0]
	for _, l := range src.Layers() {
		if l == r.layer {
			layer = l
		}
	}
	// The horizon is not a flag newFrameApp knows, so it is put back after
	// the refit, looking at the new data from the same height of camera.
	horizon := r.a.viewName() == "horizon"
	st := r.a.style
	r.a = newFrameApp(src, m, layer, r.a.globe, r.cols, r.rows)
	r.a.style = st
	if horizon {
		r.a.setHorizon()
	}
	r.layer = layer
	return nil
}

// turnNote is the half of the rate message that depends on the axis: an
// axis that turns accepts a negative rate to turn the other way, one that
// does not wants a positive number.
var turnNote = map[bool]string{
	true:  ", negative to turn the other way",
	false: " and above 0",
}

// move runs the view along one axis for some seconds, a frame each tick. fly is
// the vertical axis and spin the horizontal, and each projection says what its
// axes mean:
//
// on the horizon fly goes forward along the heading and spin turns the heading;
// on the globe fly tilts toward a pole, stopping at 89 degrees, and spin turns
// the planet under the camera; on the flat map fly goes north, stopping at 85,
// and spin goes east.
//
// The rate is a fraction of the view a second -- of the distance to the
// horizon, or of the field of view, on the horizon -- so a take reads the same
// at any height. A spin may be negative, to turn the other way.
//
// Frames are paced to the clock, because a take is watched or recorded as it
// runs; a frame slower than its tick is not waited for, so a slow frame costs
// time rather than piling up.
func (r *repl) move(verb string, args []string, ax, ay float64) bool {
	if len(args) < 1 || len(args) > 2 {
		return r.fail("%s wants seconds, and optionally a rate", verb)
	}
	secs, err := strconv.ParseFloat(args[0], 64)
	if err != nil || secs <= 0 || secs > 600 {
		return r.fail(
			"%s wants seconds above 0 and up to 600, not %q",
			verb,
			args[0],
		)
	}
	rate := 0.1
	if len(args) == 2 {
		rate, err = strconv.ParseFloat(args[1], 64)
		turning := ax != 0
		if err != nil || rate == 0 || math.Abs(rate) > 5 ||
			(!turning && rate < 0) {
			return r.fail(
				"%s wants a rate up to 5%s, not %q",
				verb,
				turnNote[turning],
				args[1],
			)
		}
	}
	frames := max(1, int(math.Round(secs*float64(r.fps))))
	tick := time.Second / time.Duration(r.fps)
	step := rate / float64(r.fps)
	for i := 0; i < frames; i++ {
		start := time.Now()
		r.a.proj.Pan(ax*step, ay*step)
		r.a.dirty = true
		r.draw()
		if spent := time.Since(start); spent < tick {
			r.sleep(tick - spent)
		}
	}
	return false
}

// draw writes one frame, reporting a write failure on stderr rather than
// ending the session: the pane may have gone away and come back. Live, the
// cursor goes home first so the frame overwrites the last one; otherwise
// the stream is exactly what -frame writes, for whatever decodes it.
func (r *repl) draw() error {
	if r.live {
		io.WriteString(r.out, "\x1b[H")
	}
	if err := printFrame(r.out, r.a, r.layer, r.cols, r.rows); err != nil {
		fmt.Fprintln(r.errw, "angry:", err)
		return nil
	}
	return nil
}

// fail reports a bad command without drawing, so the frame on screen
// stays the last one that was asked for. It always reports false, to be
// returned straight from a command case.
func (r *repl) fail(format string, args ...any) bool {
	fmt.Fprintf(r.errw, "angry: "+format+"\n", args...)
	return false
}
