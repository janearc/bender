package main

import (
	"fmt"
	"strings"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// observerPageProse is what the page says.
const observerPageProse = `
the visible spectrum, one cell per five nanometres, as the 1931
observer sees it: seventeen adults in two english labs, matching
lamp light in a dark room, averaged into a table. that table is the
library's whole definition of visible; ultraviolet and infrared
contribute nothing because the table has no row for them. almost
every colour in this bar is outside what a screen can make, so what
you see is the nearest the lamps can do, which is what every screen
has always shown you of a rainbow. the white, d65, is daylight run
through the same table, derived at start-up, not typed.`

// lifted is a wavelength's colour at full brightness: the hue shows, and
// the clipping to the screen's reach is the honest part.
func lifted(s swatch.Swatch) swatch.Swatch {
	c, _ := srgb.FromSwatch(s)
	m := c.R
	if c.G > m {
		m = c.G
	}
	if c.B > m {
		m = c.B
	}
	if m > 0 {
		c = srgb.RGB{R: c.R / m, G: c.G / m, B: c.B / m}
	}
	return c.Swatch()
}

// observerPage paints the page: what visible means here.
func observerPage() {
	title("observer: what 'visible' means here")
	var b strings.Builder
	for nm := 380; nm <= 700; nm += 5 {
		b.WriteString(paint(lifted(swatch.Monochrome(nm)), 1))
	}
	fmt.Printf("   %s\n   380 nm%sup to 700 nm\n\n",
		b.String(), strings.Repeat(" ", 53))
	say(lines(observerPageProse)...)
}

// observerCSS is the page as a css sheet: the spectrum as twenty stops.
func observerCSS() string {
	// the spectrum as a gradient of hex stops, one per twenty nanometres:
	// what a browser can be handed, clipped to srgb exactly as the bar is.
	var stops []string
	for nm := 380; nm <= 700; nm += 20 {
		at := float64(nm-380) / 320 * 100
		stops = append(stops, fmt.Sprintf("    %s %.0f%%",
			hexOf(lifted(swatch.Monochrome(nm))), at))
	}
	return ":root {\n  --spectrum: linear-gradient(in srgb,\n" +
		strings.Join(stops, ",\n") + ");\n}\n"
}
