package main

import (
	"fmt"

	"github.com/janearc/libtheme-css/spaces/ok"
)

// eyePageProse is what the page says.
const eyePageProse = `
there are two tolerances in the library and no others. Exact, a
billionth, is arithmetic noise: a round trip, a sum walked in a
different order. Eye, 0.02, is the smallest difference most people
notice side by side, a measurement of people, averaged, and the
reason oklab exists is that 0.02 means the same for a dark blue as
for a pale yellow. every 'are these the same' compares against one
of the two, so precision is a decision made once, not per call.`

var (
	eyeBase     = ok.OKLab{L: 0.6, A: 0.1, B: -0.05}
	eyeSame     = ok.OKLab{L: 0.61, A: 0.1, B: -0.05}
	eyeTellable = ok.OKLab{L: 0.62, A: 0.1, B: -0.05}
	eyeApart    = ok.OKLab{L: 0.7, A: 0.1, B: -0.05}
)

// eyePage paints the page: precision, set once.
func eyePage() {
	title("eye: precision, set once")
	pair := func(other ok.OKLab, note string) {
		fmt.Printf("   %s %s  %.2f apart in oklab: %s\n",
			paint(eyeBase.Swatch(), 6), paint(other.Swatch(), 6),
			ok.Distance(eyeBase, other), note)
	}
	pair(eyeSame, "the same to a person")
	pair(eyeTellable, "just about tellable")
	pair(eyeApart, "different")
	fmt.Println()
	say(lines(eyePageProse)...)
}

// eyeCSS is the page as a css sheet: the two tolerances as swatches.
func eyeCSS() string {
	line := func(name string, c ok.OKLab) string {
		return fmt.Sprintf("  --%s: oklab(%.3f %.3f %.3f);\n",
			name, c.L, c.A, c.B)
	}
	return ":root {\n" +
		line("base", eyeBase) +
		line("same-to-a-person", eyeSame) +
		line("just-tellable", eyeTellable) +
		line("different", eyeApart) +
		fmt.Sprintf("  /* %.2g apart in oklab: where a person starts "+
			"to see it */\n",
			ok.Eye) +
		"}\n"
}
