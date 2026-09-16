package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// caps is what the terminal answered. The one capability this program is
// built on is the pixel mouse; inside tmux it is absent rather than
// reduced, so detect says plainly what is missing and the program refuses.
type caps struct {
	tmux         bool
	pixelMouse   bool
	cellW, cellH int
	cols, rows   int
}

// missing is what this terminal cannot do, in words a person can act on. it
// is said once at startup rather than discovered when the mouse does
// nothing.
func (c caps) missing() []string {
	var out []string
	if c.tmux {
		out = append(
			out,
			"running inside tmux, which does not forward pixel "+
				"mouse reports",
		)
	}
	if !c.pixelMouse {
		out = append(
			out,
			"no SGR-pixel mouse reporting (DEC mode 1016): the "+
				"terminal reports cells, not pixels",
		)
	}
	if c.cellW <= 0 || c.cellH <= 0 {
		out = append(
			out,
			"no pixel geometry: the terminal did not say"+
				" how big a character is",
		)
	}
	return out
}

// String is the terminal's measurements on one line, for the status bar and
// for -probe.
func (c caps) String() string {
	yn := map[bool]string{true: "yes", false: "no"}
	return fmt.Sprintf(
		"window %d by %d cells, character %d by %d pixels,"+
			" pixel mouse %s, tmux %s",
		c.cols,
		c.rows,
		c.cellW,
		c.cellH,
		yn[c.pixelMouse],
		yn[c.tmux],
	)
}

// detect asks a started terminal for DEC 1016 and reads the answer off the
// event stream with a deadline. It consumes the terminal's first window
// size event, so the caller sizes the screen from the measurement here.
func detect(
	t *uv.Terminal,
	env func(string) string,
	timeout time.Duration,
) caps {
	var c caps
	c.tmux = env("TMUX") != "" || strings.HasPrefix(env("TERM"), "tmux") ||
		strings.HasPrefix(env("TERM"), "screen")
	if ws, err := t.GetWinsize(); err == nil && ws != nil && ws.Col > 0 &&
		ws.Row > 0 {
		c.cols, c.rows = int(ws.Col), int(ws.Row)
		c.cellW, c.cellH = int(ws.Xpixel)/c.cols, int(ws.Ypixel)/c.rows
	}
	_, _ = t.Write([]byte(ansi.RequestMode(ansi.DECMode(1016))))
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return c
		case ev, ok := <-t.Events():
			if !ok {
				return c
			}
			if e, ok := ev.(uv.ModeReportEvent); ok &&
				e.Mode == ansi.DECMode(1016) {
				c.pixelMouse = !e.Value.IsNotRecognized()
				return c
			}
		}
	}
}

// probe prints what the terminal answered.
func probe(w io.Writer) error {
	t := uv.DefaultTerminal()
	if err := t.Start(); err != nil {
		return err
	}
	c := detect(t, os.Getenv, 500*time.Millisecond)
	time.Sleep(30 * time.Millisecond) // see stop in app.go
	_ = t.Stop()
	fmt.Fprintln(w, c)
	for _, m := range c.missing() {
		fmt.Fprintln(w, "missing:", m)
	}
	return nil
}
