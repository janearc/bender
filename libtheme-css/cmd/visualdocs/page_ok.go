package main

import (
	"fmt"
	"strings"

	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
)

// okPageProse is what the page says.
const okPageProse = `
oklab is the swatch in three numbers an eye agrees with: L, how
light; a and b, which way round the wheel and how far out. its polar
form, oklch, is the one you say aloud: lightness, chroma, hue. the
point of it is distance: equal distances look equally different
wherever they sit, so blending, ramps and 'nearest' all happen here.
it has no primaries; it has a white, a black, the grey line and the
wheel. anything with a hue is defined elsewhere and measured here.`

// okPage paints the page: the axes an eye agrees with.
func okPage() {
	title("ok: the axes an eye agrees with")
	grey := functions.Even(ok.Mix, swatch.Black, swatch.White)
	fmt.Printf("   %s  L: black to white\n", bar(grey, 40))
	var wheel strings.Builder
	for i := 0; i < 40; i++ {
		c := ok.OKLCH{L: 0.7, C: 0.12, H: float64(i) * 360 / 40}
		wheel.WriteString(paint(c.Rect().Swatch(), 1))
	}
	fmt.Printf("   %s  h: once round the wheel\n", wheel.String())
	var chroma strings.Builder
	for i := 0; i < 40; i++ {
		c := ok.OKLCH{L: 0.7, C: 0.2 * float64(i) / 39, H: 345}
		chroma.WriteString(paint(c.Rect().Swatch(), 1))
	}
	fmt.Printf("   %s  C: grey out to the pink\n\n", chroma.String())
	say(lines(okPageProse)...)
}

// okCSS is the page as a css sheet: the grey line and the wheel.
func okCSS() string {
	grey := functions.Even(ok.Mix, swatch.Black, swatch.White)
	var wheel []string
	for i := 0; i <= 8; i++ {
		c := ok.OKLCH{L: 0.7, C: 0.12, H: float64(i%8) * 45}
		wheel = append(wheel, fmt.Sprintf("    %s %.0f%%", c,
			float64(i)/8*100))
	}
	return ":root {\n" +
		"  --l: " + grey.String(hexOf) + ";\n" +
		"  --h: linear-gradient(in oklch longer hue,\n" +
		strings.Join(wheel, ",\n") + ");\n" +
		"  --c: linear-gradient(in oklch,\n" +
		"    " + ok.OKLCH{L: 0.7, C: 0, H: 345}.String() + " 0%,\n" +
		"    " + ok.OKLCH{L: 0.7, C: 0.2,
		H: 345}.String() + " 100%);\n" +
		"}\n"
}
