package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

// drive runs a script through the repl against the synthetic source and
// returns what went to stdout and to stderr.
func drive(t *testing.T, script string, opts options) (string, string) {
	t.Helper()
	src, m := synthetic()
	var out, errw bytes.Buffer
	if err := runRepl(context.Background(), nil, strings.NewReader(script), &out, &errw,
		src, m, "v", opts, nil, style{}); err != nil {
		t.Fatal(err)
	}
	return out.String(), errw.String()
}

// headers are the one line per frame that says where the view is, which is
// what a console driving the render reads back.
func headers(s string) []string {
	var h []string
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, "res ") &&
			strings.Contains(l, "centre ") {
			h = append(h, l)
		}
	}
	return h
}

// One frame goes out before any command, so a pane that attaches and waits
// is not left blank, and one more per command that succeeds.
func TestReplDrawsPerCommand(t *testing.T) {
	out, errw := drive(
		t,
		"look 10 20\nspan 90\nflat\nglobe\n",
		options{globe: true, cols: 40, rows: 10},
	)
	if got := len(headers(out)); got != 5 {
		t.Fatalf("want 5 frames for 4 commands, got %d", got)
	}
	if errw != "" {
		t.Errorf("stderr: %q", errw)
	}
}

// A bad command reports on stderr, draws nothing, and does not stop the
// session: a typo during a take must not kill the process or corrupt the
// frame stream somebody is decoding.
func TestReplBadCommandKeepsGoing(t *testing.T) {
	out, errw := drive(
		t,
		"bogus\nspan -5\nspan abc\nlook\nlayer nope\nlook 10 20\n",
		options{globe: true, cols: 40, rows: 10},
	)
	if got := len(headers(out)); got != 2 {
		t.Fatalf(
			"want 2 frames (initial plus the one good command), got %d",
			got,
		)
	}
	for _, want := range []string{"unknown command", "span wants", "look wants", "no layer"} {
		if !strings.Contains(errw, want) {
			t.Errorf("stderr missing %q: %s", want, errw)
		}
	}
}

// Switching projection keeps the view pointed where it was, so a console
// can cut between flat and globe without re-aiming.
func TestReplProjectionKeepsAim(t *testing.T) {
	out, _ := drive(
		t,
		"look 10 20\nspan 90\nflat\nglobe\n",
		options{globe: true, cols: 40, rows: 10},
	)
	h := headers(out)
	for _, l := range h[2:] {
		if !strings.Contains(l, "centre 10.000, 20.000") ||
			!strings.Contains(l, "span 90.000") {
			t.Errorf("aim moved: %s", l)
		}
	}
	if !strings.Contains(h[3], "flat") || !strings.Contains(h[4], "globe") {
		t.Errorf("projection did not switch: %q %q", h[3], h[4])
	}
}

// horizon cuts to the low-orbit view on the same ground, and the frame has
// sky in it: the top of the middle column is left empty for whatever is
// composited behind, and the bottom is planet.
func TestReplHorizon(t *testing.T) {
	out, errw := drive(
		t,
		"look 10 20\nspan 20\nhorizon\n",
		options{globe: true, cols: 40, rows: 10},
	)
	if errw != "" {
		t.Fatalf("stderr: %q", errw)
	}
	h := headers(out)
	last := h[len(h)-1]
	if !strings.Contains(last, "horizon") ||
		!strings.Contains(last, "centre 10.000, 20.000") {
		t.Fatalf("header %q, want the horizon view on 10,20", last)
	}
	frame := strings.Split(
		out[strings.LastIndex(out, last)+len(last)+1:],
		"\n",
	)
	top, bottom := plain(frame[0]), plain(frame[9])
	if strings.TrimSpace(top) != "" {
		t.Errorf("top row should be sky, got %q", top)
	}
	if strings.TrimSpace(bottom) == "" {
		t.Error("bottom row should be planet")
	}
}

// fly draws a frame per tick for the time asked, moving forward each
// time: on a horizon facing north, the ground under the middle of the
// view goes north.
func TestReplFly(t *testing.T) {
	out, errw := drive(
		t,
		"look 10 20\nhorizon\nfly 0.5\n",
		options{globe: true, cols: 40, rows: 10, fps: 4},
	)
	if errw != "" {
		t.Fatalf("stderr: %q", errw)
	}
	h := headers(out)
	if len(h) != 5 {
		t.Fatalf(
			"want 5 frames (opening, look, horizon, two flying), got %d",
			len(h),
		)
	}
	lat := func(l string) float64 {
		var lon, la float64
		i := strings.Index(l, "centre ")
		if _, err := fmt.Sscanf(l[i:], "centre %f, %f", &lon, &la); err != nil {
			t.Fatalf("header %q: %v", l, err)
		}
		return la
	}
	if !(lat(h[3]) > lat(h[2]) && lat(h[4]) > lat(h[3])) {
		t.Errorf(
			"flying north should move the view north: %.3f %.3f %.3f",
			lat(h[2]),
			lat(h[3]),
			lat(h[4]),
		)
	}
}

// spin turns the globe under the camera: longitude moves and latitude
// stays, and a negative rate turns the other way. This is the motion the
// film's globe pane needs; fly on the globe climbs toward a pole instead.
func TestReplSpin(t *testing.T) {
	coord := func(l string) (float64, float64) {
		var lon, lat float64
		i := strings.Index(l, "centre ")
		if _, err := fmt.Sscanf(l[i:], "centre %f, %f", &lon, &lat); err != nil {
			t.Fatalf("header %q: %v", l, err)
		}
		return lon, lat
	}
	for _, c := range []struct {
		script string
		east   bool
	}{{"look 10 20\nspin 0.5\n", true}, {"look 10 20\nspin 0.5 -0.1\n", false}} {
		out, errw := drive(
			t,
			c.script,
			options{globe: true, cols: 40, rows: 10, fps: 4},
		)
		if errw != "" {
			t.Fatalf("stderr: %q", errw)
		}
		h := headers(out)
		if len(h) != 4 {
			t.Fatalf(
				"want 4 frames (opening, look, two turning), got %d",
				len(h),
			)
		}
		lon0, lat0 := coord(h[1])
		lon1, lat1 := coord(h[3])
		if lat1 != lat0 {
			t.Errorf("spin moved latitude %.3f -> %.3f", lat0, lat1)
		}
		if (lon1 > lon0) != c.east {
			t.Errorf(
				"%q: longitude %.3f -> %.3f",
				c.script,
				lon0,
				lon1,
			)
		}
	}
}

// A bad fly draws nothing and says why.
func TestReplFlyBadArgs(t *testing.T) {
	out, errw := drive(
		t,
		"fly\nfly -1\nfly 1 9\nfly 1 2 3\nfly 1 -0.1\nspin 1 0\nspin 1 -9\n",
		options{globe: true, cols: 40, rows: 10, fps: 4},
	)
	if got := len(headers(out)); got != 1 {
		t.Fatalf("want only the opening frame, got %d", got)
	}
	if strings.Count(errw, "fly wants") != 5 ||
		strings.Count(errw, "spin wants") != 2 {
		t.Errorf("stderr: %q", errw)
	}
}

// Live, every frame goes home first and the cursor is hidden for the
// session and shown again at the end. Without -live the stream carries
// none of that, so a decoder reads what it always read.
func TestReplLive(t *testing.T) {
	out, _ := drive(
		t,
		"look 10 20\n",
		options{globe: true, cols: 40, rows: 10, live: true},
	)
	if !strings.HasPrefix(out, "\x1b[2J\x1b[?25l") ||
		!strings.HasSuffix(out, "\x1b[?25h") {
		t.Errorf(
			"live session should clear and hide the cursor, then show it: %q...%q",
			out[:12],
			out[len(out)-8:],
		)
	}
	if got := strings.Count(out, "\x1b[H"); got != 2 {
		t.Errorf("want one home per frame, 2, got %d", got)
	}
	plainOut, _ := drive(
		t,
		"look 10 20\n",
		options{globe: true, cols: 40, rows: 10},
	)
	if strings.Contains(plainOut, "\x1b[H") ||
		strings.Contains(plainOut, "\x1b[?25l") {
		t.Error("without -live the stream must not move the cursor")
	}
}

// palette and floor restyle the next frame, and the frame says so under
// itself; a palette that does not exist is an error and changes nothing.
func TestReplPaletteAndFloor(t *testing.T) {
	out, errw := drive(
		t,
		"palette night\nfloor 1\npalette neon\n",
		options{globe: true, cols: 40, rows: 10},
	)
	if got := len(headers(out)); got != 3 {
		t.Fatalf("want 3 frames (opening, palette, floor), got %d", got)
	}
	if !strings.Contains(errw, `no palette "neon"`) {
		t.Errorf("stderr: %q", errw)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	last := lines[len(lines)-1]
	for _, want := range []string{"night", "log", "top at p99", "floor 1.000 drawn as ground"} {
		if !strings.Contains(last, want) {
			t.Errorf("last line %q missing %q", last, want)
		}
	}
}

// plain drops the colour escapes from a frame row, leaving what would be
// on screen.
func plain(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// look takes the typed form and the flag's form, and both spellings of
// centre still work unadvertised. The header keeps saying "centre": it
// is a format a decoder reads, and changing a word in it breaks puf's.
func TestReplLookForms(t *testing.T) {
	out, errw := drive(
		t,
		"look 10 20\nlook 10,20\ncentre 10 20\ncenter 10 20\n",
		options{globe: true, cols: 40, rows: 10},
	)
	if errw != "" {
		t.Fatalf("stderr: %q", errw)
	}
	h := headers(out)
	for _, l := range h[1:] {
		if !strings.Contains(l, "centre 10.000, 20.000") {
			t.Errorf("got %s", l)
		}
	}
}

// quit ends the session and anything after it is not run.
func TestReplQuitStops(t *testing.T) {
	out, _ := drive(
		t,
		"quit\nlook 10 20\n",
		options{globe: true, cols: 40, rows: 10},
	)
	if got := len(headers(out)); got != 1 {
		t.Fatalf("want only the opening frame, got %d", got)
	}
}
