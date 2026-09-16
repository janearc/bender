// visualdocs is the documentation, shown rather than written: one page per
// idea, a few painted rows and a few lines of text, because this is a library
// about colour and the honest way to explain a colour is to put it on the
// screen.
//
// each page can also say the same thing in css, and show the go that painted
// it, which is its own source file, embedded, so the three can never drift
// apart.
//
//	visualdocs                  every page, painted, enter for the next
//	visualdocs PAGE             one page, painted
//	visualdocs [PAGE] --css     the same, as a stylesheet
//	visualdocs [PAGE] --go      the go that produced it: the page's own file
//
// pages: swatch, observer, ok, eye, srgb, ramp, css
package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

//go:embed page_*.go
var sources embed.FS

// page is one idea: a name, how to paint it, how to say it in css.
type page struct {
	name string
	show func()
	css  func() string
}

var pages = []page{
	{"swatch", swatchPage, swatchCSS},
	{"observer", observerPage, observerCSS},
	{"ok", okPage, okCSS},
	{"eye", eyePage, eyeCSS},
	{"srgb", srgbPage, srgbCSS},
	{"ramp", rampPage, rampCSS},
	{"css", cssPage, cssCSS},
}

// main is the pager: a page name and a mode, or every page in turn.
func main() {
	// arguments in any order: an optional page name and an optional mode.
	// no page means every page; no mode means paint.
	var name, mode string
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "--") {
			mode = a
		} else {
			name = a
		}
	}
	if mode != "" && mode != "--css" && mode != "--go" {
		fmt.Fprintf(os.Stderr,
			"visualdocs: modes are --css and --go, not %q\n", mode)
		os.Exit(2)
	}
	chosen := pages
	if name != "" {
		chosen = nil
		for _, p := range pages {
			if p.name == name {
				chosen = []page{p}
			}
		}
		if chosen == nil {
			fmt.Fprintf(os.Stderr,
				"no page called %q; there is "+
					"%s\n", name, names())
			os.Exit(2)
		}
	}
	in := bufio.NewReader(os.Stdin)
	for i, p := range chosen {
		switch mode {
		case "--css":
			fmt.Printf("/* %s */\n", p.name)
			fmt.Print(p.css())
		case "--go":
			src, err := sources.ReadFile("page_" + p.name + ".go")
			if err != nil {
				fmt.Fprintln(os.Stderr, "visualdocs:", err)
				os.Exit(1)
			}
			fmt.Printf("// page_%s.go\n", p.name)
			os.Stdout.Write(src)
		default:
			p.show()
		}
		if i == len(chosen)-1 {
			return
		}
		if mode != "" {
			fmt.Println()
			continue
		}
		fmt.Printf("\n   [%d/%d] enter for %s, q to stop: ", i+1,
			len(chosen), chosen[i+1].name)
		line, _ := in.ReadString('\n')
		if strings.TrimSpace(line) == "q" {
			return
		}
		fmt.Println()
	}
}

// names is every page, for the usage line.
func names() string {
	n := make([]string, len(pages))
	for i, p := range pages {
		n[i] = p.name
	}
	return strings.Join(n, ", ")
}

// paint is a run of cells in the colour, as the terminal's lamps show
// it: the nearest they can do when the colour is outside their reach.
//
// What the docs assume about the terminal, and it is all they assume: it paints
// a 24-bit background, and its default text colour reads on its default
// background.
//
// nothing sets a foreground, nothing dims or bolds, so the words read on a
// light terminal as well as a dark one and only the swatches carry colour.
//
// with NO_COLOR set, the convention every terminal tool honours, no escape is
// written at all: a swatch is a run of hashes, its shape without its colour,
// and the hex beside it in the text is what it was.
func paint(s swatch.Swatch, width int) string {
	if noColour() {
		return strings.Repeat("#", width)
	}
	c, _ := srgb.FromSwatch(s)
	r, g, b := c.Bytes()
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm%s\x1b[0m", r, g, b,
		strings.Repeat(" ", width))
}

// noColour is the NO_COLOR convention: set to anything, colour is off.
func noColour() bool { return os.Getenv("NO_COLOR") != "" }

// bar is a ramp sampled across a width, painted.
func bar(r functions.Ramp, width int) string {
	var b strings.Builder
	for _, s := range r.Samples(width) {
		b.WriteString(paint(s, 1))
	}
	return b.String()
}

// hexOf is a swatch written the way a stop in css is.
func hexOf(s swatch.Swatch) string {
	c, _ := srgb.FromSwatch(s)
	return c.Hex()
}

// title prints a page's heading.
func title(t string) { fmt.Printf("== %s\n\n", t) }

// say prints lines of text under a heading.
func say(lines ...string) {
	for _, l := range lines {
		fmt.Println("   " + l)
	}
}

// lines is a page's prose as the lines it was written in, the newline
// after the opening backtick dropped.
func lines(prose string) []string {
	return strings.Split(strings.TrimPrefix(prose, "\n"), "\n")
}
