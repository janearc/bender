package main

import (
	"fmt"

	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// cssPageProse is what the page says.
const cssPageProse = `
a sheet is named swatches in order, written as the custom properties
a browser, an editor theme or a lamp's config would read: the hex a
screen can make, and the same colour as oklch beside it, which is the
form that says what it is. a hex code in css is srgb by definition and
a painted cell is srgb by construction, so the bar and the rule
beside it are the same three bytes in two syntaxes; if they ever
disagree, something between here and the screen is lying.`

// vaporwave is three of the theme's tokens, enough to be a sheet.
func vaporwave() *css.Sheet {
	sheet := css.New()
	sheet.Set("ground", srgb.MustHex("#160d2b").Swatch())
	sheet.Set("ink", srgb.MustHex("#d8d0e0").Swatch())
	sheet.Set("pink", srgb.MustHex("#ff6ec7").Swatch())
	return sheet
}

// cssPage paints the page: the container everything normalises to.
func cssPage() {
	title("css: the container everything normalises to")
	for _, r := range vaporwave().Rules() {
		fmt.Printf("   %s  --%s: %s;  /* %s */\n",
			paint(r.Swatch, 2), r.Name, r.Value, r.Comment)
	}
	fmt.Println()
	say(lines(cssPageProse)...)
}

// cssCSS is the page as a css sheet: the vaporwave sheet itself.
func cssCSS() string { return vaporwave().String() }
