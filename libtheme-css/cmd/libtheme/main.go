// libtheme is the console end of the library: a way to look at a swatch,
// in every form the library has, painted, so that a change lower down
// shows up as a change on screen. It is a check, not a product.
//
//	libtheme show '#ff6ec7'                one colour, every form, painted
//	libtheme show 'oklch(74% 0.20 345)'    the same, from the polar form
//	libtheme ramp '#160d2b' '#ffa2ff' 24   the line between two colours in
//	                                       oklab and, for contrast, in the
//	                                       lamps, so the mud is visible
//	libtheme known                         everything the library derives
//	                                       unprompted: the visual test
//	libtheme known --css                   the same set as a css sheet,
//	                                       plain, for piping to a file
//	libtheme known --paint                 the sheet with each colour
//	                                       painted beside its rule
//	libtheme roundtrip                     each known colour written as
//	                                       hex and oklch(), read back by
//	                                       the tool's own parser, compared
//	libtheme read FILE                     a colourway as the library sees
//	                                       it: roles painted, ramps drawn
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// main is the verb table.
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "known":
		mode := ""
		if len(os.Args) > 2 {
			mode = os.Args[2]
		}
		err = known(mode)
	case "roundtrip":
		err = roundtrip()

	case "show":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		err = show(os.Args[2])
	case "read":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		err = read(os.Args[2])
	case "ramp":
		n := 24
		if len(os.Args) > 4 {
			n, err = strconv.Atoi(os.Args[4])
			if err != nil {
				break
			}
		}
		if len(os.Args) < 4 {
			usage()
			os.Exit(2)
		}
		err = ramp(os.Args[2], os.Args[3], n)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "libtheme:", err)
		os.Exit(1)
	}
}

// usage is the one line to type when the verb was wrong.
func usage() {
	fmt.Fprintln(os.Stderr, "libtheme known [--css|--paint] | roundtrip |"+
		" show COLOUR | ramp COLOUR COLOUR [steps] | read FILE.css\n"+
		"  COLOUR is #rrggbb or oklch(L% C H)")
}

// parse reads the two spellings a person types: a hex code, or css's
// oklch(). Everything else is for later.
func parse(s string) (swatch.Swatch, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "oklch(") && strings.HasSuffix(s, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(s, "oklch("),
			")")
		fields := strings.Fields(inner)
		if len(fields) != 3 {
			return swatch.Black,
				fmt.Errorf("oklch wants three numbers, not %q",
					s)
		}
		var c ok.OKLCH
		var err error
		if c.L, err = percentOrNumber(fields[0]); err != nil {
			return swatch.Black, err
		}
		if c.C, err = strconv.ParseFloat(fields[1], 64); err != nil {
			return swatch.Black, err
		}
		// css's "none" for a hue means there isn't one to speak of: the
		// colour is grey to an eye, and any angle would do. zero does.
		if fields[2] == "none" {
			c.H = 0
		} else if c.H, err = strconv.ParseFloat(
			fields[2], 64); err != nil {
			return swatch.Black, err
		}
		return c.Rect().Swatch(), nil
	}
	c, err := srgb.FromHex(s)
	if err != nil {
		return swatch.Black, err
	}
	return c.Swatch(), nil
}

// The derived white shows a seam here: its chroma in ok is about 9e-5,
// because the fit was normalised to a white rounded to four places and
// the swatch's white is the 1 nm integration. Far below anything an eye
// could see; not zero; and OKLCH.String prints its hue as none.

// percentOrNumber reads "74%" as 0.74 and "0.74" as itself.
func percentOrNumber(s string) (float64, error) {
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		return v / 100, err
	}
	return strconv.ParseFloat(s, 64)
}

// paint is a cell of the colour, as the terminal's own lamps would show
// it: 24-bit background, the nearest the screen can do if it is out of
// gamut.
func paint(s swatch.Swatch, width int) string {
	if os.Getenv("NO_COLOR") != "" {
		return strings.Repeat("#", width)
	}
	c, _ := srgb.FromSwatch(s)
	r, g, b := c.Bytes()
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm%s\x1b[0m",
		r, g, b, strings.Repeat(" ", width))
}

// show prints one swatch in every form the library has, and paints it.
func show(arg string) error {
	s, err := parse(arg)
	if err != nil {
		return err
	}
	x, y, z := s.XYZ()
	lab := ok.FromSwatch(s)
	lch := lab.Polar()
	c, in := srgb.FromSwatch(s)
	gamut := "in gamut"
	if !in {
		gamut = "OUT OF GAMUT, clipped"
	}
	fmt.Printf("%s  %s\n", paint(s, 12), arg)
	fmt.Printf("  xyz     %.5f %.5f %.5f\n", x, y, z)
	fmt.Printf("  oklab   %.4f %.4f %.4f\n", lab.L, lab.A, lab.B)
	fmt.Printf("  oklch   %s\n", lch)
	fmt.Printf("  srgb    %.4f %.4f %.4f  %s\n", c.R, c.G, c.B, gamut)
	fmt.Printf("  hex     %s\n", c.Hex())
	h, v := c.HSL(), c.HSV()
	fmt.Printf("  hsl     %.1f %.3f %.3f\n", h.H, h.S, h.L)
	fmt.Printf("  hsv     %.1f %.3f %.3f\n", v.H, v.S, v.V)
	fmt.Printf("  to white %.3f   to black %.3f   (oklab; eye is %.2f)\n",
		ok.Distance(lab, ok.White), ok.Distance(lab, ok.Black), ok.Eye)
	return nil
}

// ramp draws the line between two colours twice: once in oklab, which is the
// line the library uses, and once as a straight line through the lamps, which
// is what every other tool does, so the difference is on screen and not in an
// argument.
//
// Both are the same Ramp with a different mixer, which is the whole point of
// the mixer being a parameter.
func ramp(a, b string, n int) error {
	sa, err := parse(a)
	if err != nil {
		return err
	}
	sb, err := parse(b)
	if err != nil {
		return err
	}
	if n < 2 {
		n = 2
	}
	for _, in := range []functions.Mixer{ok.Mix, srgb.Mix} {
		r := functions.Even(in, sa, sb)
		var line strings.Builder
		for _, s := range r.Samples(n) {
			line.WriteString(paint(s, 2))
		}
		fmt.Printf("%-6s %s\n       %s\n",
			in.Name, line.String(), r.String(hex))
	}
	return nil
}

// hex is a swatch written the way a stop in css is.
func hex(s swatch.Swatch) string {
	c, _ := srgb.FromSwatch(s)
	return c.Hex()
}

// No attributes on text at all. Dim was tried and is unreadable on a dark
// palette to the person this is for, and bold reads differently on every
// terminal; the output has to read on any palette, light or dark, so the words
// are plain and only the swatches are painted.
//
// Telling a light terminal from a dark one is a vendor question, held for the
// terminal vendor package.
const (
	dim   = ""
	bold  = ""
	plain = ""
)

// heading is one dim line saying what the block under it is, so the
// output separates itself from whatever make printed above it.
func heading(text string) { fmt.Printf("-- %s\n", text) }

// paintSheet writes the sheet with each rule's colour painted before it,
// the name in bold, the value plain, the comment dim. no parsing: the
// sheet hands over its rules as structure.
func paintSheet(sheet *css.Sheet) {
	rules := sheet.Rules()
	width := 0
	for _, r := range rules {
		if len(r.Name) > width {
			width = len(r.Name)
		}
	}
	fmt.Printf("     :root {\n")
	for _, r := range rules {
		fmt.Printf("  %s   %s--%s:%s%*s %s;  %s/* %s */%s\n",
			paint(r.Swatch,
				2), bold, r.Name, plain, width-len(r.Name), "",
			r.Value, dim, r.Comment, plain)
	}
	fmt.Printf("     }\n")
}

// roundtrip is the translation test: every known colour is written the two ways
// the tool can read, hex and oklch(), both spellings are read back through
// parse, and the two swatches are compared.
//
// within Exact they are the same colour to arithmetic; within Eye they are the
// same colour to a person, which is what the oklch spelling, printed to three
// places, can promise. a colourway file is only as good as this trip.
func roundtrip() error {
	type entry struct {
		name string
		s    swatch.Swatch
	}
	list := []entry{
		{"black", swatch.Black}, {"white", swatch.White},
		{"red", srgb.Red.Swatch()},
		{"green", srgb.Green.Swatch()},
		{"blue", srgb.Blue.Swatch()},
	}
	heading("each colour written two ways and read back through the same" +
		" parser.")
	heading("apart: the distance in oklab between the two readings.")
	heading(fmt.Sprintf("under %.0e is arithmetic noise; under %.2g a "+
		"person"+
		" cannot tell", ok.Exact, ok.Eye))
	heading("them apart; over that, they could.")
	fmt.Printf("   %-6s %-8s %-24s %-8s %s\n",
		"", "hex", "oklch", "apart", "verdict")
	failed := false
	for _, e := range list {
		h := hex(e.s)
		l := ok.FromSwatch(e.s).Polar().String()
		fromHex, err := parse(h)
		if err != nil {
			return err
		}
		fromLCH, err := parse(l)
		if err != nil {
			return err
		}
		d := ok.Distance(ok.FromSwatch(fromHex), ok.FromSwatch(fromLCH))
		verdict := "same to a person"
		switch {
		case d <= ok.Exact:
			verdict = "identical"
		case d > ok.Eye:
			verdict = "DIFFERENT"
			failed = true
		}
		fmt.Printf("%s %-6s %-8s %-24s %-8.5f %s\n",
			paint(e.s, 2), e.name, h, l, d, verdict)
	}
	if failed {
		return fmt.Errorf("a round trip lost more than an eye can miss")
	}
	return nil
}

// known is everything the library can put on screen without being told a
// colour: the points it defines and the lines between them. It is the visual
// test.
//
// Each entry names where the colour is defined, because that is the list this
// library is really keeping.
//
// Black and white are the swatch's, from the observer and the daylight; red,
// green and blue are srgb's, from the standard's chromaticities; the grey line
// is oklab's, the only line the space defines on its own.
//
// A grey by itself is not on the list, because "grey" is not a colour until you
// say how light, and the line says that better than any one point.
func known(mode string) error {
	type entry struct {
		name, from string
		s          swatch.Swatch
	}
	list := []entry{
		{"black", "swatch: no light", swatch.Black},
		{"white", "swatch: d65 via the observer", swatch.White},
		{"red", "srgb: the red lamp at full", srgb.Red.Swatch()},
		{"green", "srgb: the green lamp at full", srgb.Green.Swatch()},
		{"blue", "srgb: the blue lamp at full", srgb.Blue.Swatch()},
	}
	if mode == "--css" || mode == "--paint" {
		sheet := css.New()
		for _, e := range list {
			sheet.Set(e.name, e.s)
		}
		if mode == "--css" {
			fmt.Print(sheet.String())
			return nil
		}
		heading("the same five as a css sheet. the cell before each " +
			"rule is" +
			" its value.")
		paintSheet(sheet)
		return nil
	}
	heading("everything the library can derive without being told a" +
		" colour,")
	heading("and where each one is defined.")
	for _, e := range list {
		lch := ok.FromSwatch(e.s).Polar()
		c, _ := srgb.FromSwatch(e.s)
		fmt.Printf("%s  %-6s %s   %-24s %s\n",
			paint(e.s, 8), e.name, c.Hex(), lch, e.from)
	}
	// The one line the library defines on its own, drawn with each mixer
	// the spaces supply: the same two stops, and the disagreement between
	// the spaces about what a straight line is, on screen.
	fmt.Println()
	for _, in := range []functions.Mixer{ok.Mix, srgb.Mix} {
		r := functions.Even(in, swatch.Black, swatch.White)
		var line strings.Builder
		for _, s := range r.Samples(24) {
			line.WriteString(paint(s, 2))
		}
		fmt.Printf("%s  black to white\n   %s\n",
			line.String(), r.String(hex))
	}
	return nil
}

// read is a colourway file as the library sees it: its roles painted,
// then every ramp it holds, stops with their positions, and the ramp
// sampled across a bar. what the reader found, and nothing it inferred.
func read(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cw := css.Read(string(src))
	heading(fmt.Sprintf("%s: %d roles", path, len(cw.Roles.Names())))
	paintSheet(cw.Roles)
	for _, name := range cw.Order {
		r := cw.Ramps[name]
		heading(fmt.Sprintf("ramp %s, %d stops", name, len(r.Stops)))
		for _, st := range r.Stops {
			fmt.Printf("  %5.1f%%  %s  %s\n",
				st.At*100, paint(st.Swatch, 4), hex(st.Swatch))
		}
		var bar strings.Builder
		for _, sw := range r.Samples(48) {
			bar.WriteString(paint(sw, 1))
		}
		fmt.Printf("  %s\n", bar.String())
	}
	return nil
}
