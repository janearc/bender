package main

import (
	"fmt"

	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/primitives/swatch"
)

// swatchPageProse is what the page says.
const swatchPageProse = `
a swatch is one colour with nothing in it about the screen showing it
or the eye reading it. it is three numbers, cie xyz from 1931: Y is
how much light, white is 1 and black is 0; X and Z are the rest of
what an eye reports. every other way of writing a colour in this
library is a reading of a swatch, and none of them is a new colour.
the fields are hidden: xyz is where the truth is kept, not where the
arithmetic is done, because a straight line between two xyz points
does not look straight.`

// swatchPage paints the page: one colour, complete.
func swatchPage() {
	title("swatch: one colour, complete")
	fmt.Printf("   %s black    %s white\n\n",
		paint(swatch.Black, 6), paint(swatch.White, 6))
	say(lines(swatchPageProse)...)
}

// swatchCSS is the page as a css sheet: white, black and the monochromes.
func swatchCSS() string {
	sheet := css.New()
	sheet.Set("black", swatch.Black)
	sheet.Set("white", swatch.White)
	return sheet.String()
}
