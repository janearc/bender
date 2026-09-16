package main

import (
	"fmt"

	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// srgbPageProse is what the page says.
const srgbPageProse = `
a screen makes colour with three lamps, and srgb is where each lamp
sits on the 1931 diagram: numbers written down for television
phosphors in 1990 and adopted for every computer screen in 1996. the
library types only the standard's four facts, the three lamps and
the curve, and derives the matrix everyone else copies from them
plus the white. hex is this space. so are hsl and hsv, the same
lamps described by angle. a swatch may fall outside the lamps'
reach; every exit here says whether it did.`

// louder is a green more colourful than the green lamp can make.
var louder = ok.OKLCH{L: 0.7, C: 0.4, H: 145}.Rect().Swatch()

// srgbPage paints the page: the screen's three lamps.
func srgbPage() {
	title("srgb: the screen's three lamps")
	fmt.Printf("   %s red   %s green   %s blue   %s all three: the "+
		"white\n\n",
		paint(srgb.Red.Swatch(), 6), paint(srgb.Green.Swatch(), 6),
		paint(srgb.Blue.Swatch(), 6), paint(srgb.White.Swatch(), 6))
	_, in := srgb.FromSwatch(louder)
	fmt.Printf("   %s a green greener than the green lamp. in gamut? %v.\n",
		paint(louder, 6), in)
	fmt.Printf("   what you see is the nearest the lamps can do.\n\n")
	say(lines(srgbPageProse)...)
}

// srgbCSS is the page as a css sheet: the primaries and the white.
func srgbCSS() string {
	sheet := css.New()
	sheet.Set("red", srgb.Red.Swatch())
	sheet.Set("green", srgb.Green.Swatch())
	sheet.Set("blue", srgb.Blue.Swatch())
	sheet.Set("white", srgb.White.Swatch())
	sheet.Set("louder-green", louder)
	return sheet.String()
}
