package main

import (
	"fmt"

	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// rampPageProse is what the page says.
const rampPageProse = `
a ramp is stops, each a swatch at a position from 0 to 1, and a
mixer, a named way of blending two swatches that each space supplies.
give it a t, get a swatch. where t comes from is not this library's
business: a canvas, a strip of lamps and a bar of hours each compute
their own and hand it over, which is what lets a sun, a sky and a
stripe share one ramp. containers, not form. sampled at n, a ramp is
what a strip with n lamps or a palette with n entries gets.`

var (
	rampA = srgb.MustHex("#160d2b").Swatch()
	rampB = srgb.MustHex("#ffa2ff").Swatch()
	three = functions.Even(ok.Mix,
		srgb.MustHex("#2a0f4d").Swatch(),
		srgb.MustHex("#d94a8c").Swatch(),
		srgb.MustHex("#fff4c0").Swatch())
)

// rampPage paints the page: a function from a number to a colour.
func rampPage() {
	title("ramp: a function from a number to a colour")
	fmt.Printf("   %s  in oklab\n", bar(functions.Even(ok.Mix, rampA,
		rampB), 48))
	fmt.Printf("   %s  in srgb, the same two stops\n",
		bar(functions.Even(srgb.Mix, rampA, rampB), 48))
	fmt.Printf("   %s  three stops, even\n", bar(three, 48))
	fmt.Printf("   %s  the same three, sampled at eight\n\n", bar(three, 8))
	say(lines(rampPageProse)...)
}

// rampCSS is the page as a css sheet: the same stops in both mixers.
func rampCSS() string {
	return ":root {\n" +
		"  --in-oklab: " +
		functions.Even(ok.Mix, rampA, rampB).String(hexOf) + ";\n" +
		"  --in-srgb: " +
		functions.Even(srgb.Mix, rampA, rampB).String(hexOf) + ";\n" +
		"  --three: " + three.String(hexOf) + ";\n}\n"
}
