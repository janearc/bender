package srgb

import (
	"image/color"
	"math"
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
)

// near is whether two values agree within a tolerance.
func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// The matrix derived from four chromaticities lands on the nine numbers
// Bruce Lindbloom derives from the same four with the same white, to five
// places; his white is rounded to five digits and ours is not, which
// moves the last entry by a millionth. It does not land on the standard's
// own printed matrix past the third place, and that is a finding, not a
// bug: the standard rounded
// its white to four digits (0.9505, 1, 1.0890) before deriving, and
// that rounding shifts the third decimal of three entries. The swatch's
// white is the 1 nm integration, so the derivation here is the more
// exact of the two.
func TestDerivedMatrix(t *testing.T) {
	lindbloom := [3][3]float64{
		{0.4124564, 0.3575761, 0.1804375},
		{0.2126729, 0.7151522, 0.0721750},
		{0.0193339, 0.1191920, 0.9503041},
	}
	for i := range lindbloom {
		for j := range lindbloom[i] {
			if !near(rgbToXYZ[i][j], lindbloom[i][j], 5e-6) {
				t.Errorf("matrix[%d][%d] = %.7f, lindbloom "+
					"%.7f", i, j, rgbToXYZ[i][j],
					lindbloom[i][j])
			}
		}
	}
}

// All three lamps at full are exactly the white, which is the constraint
// the matrix was derived under, so this is the derivation checking
// itself. Measured to a trillionth on darwin/arm64.
func TestWhiteIsWhite(t *testing.T) {
	x, y, z := White.Swatch().XYZ()
	wx, wy, wz := swatch.White.XYZ()
	if !near(x, wx, 1e-12) || !near(y, wy, 1e-12) || !near(z, wz, 1e-12) {
		t.Errorf("white = %v %v %v, want %v %v %v", x, y, z, wx, wy, wz)
	}
}

// The red lamp at full, through the swatch into oklab, is where the
// author of oklab says it is: L 0.628, a 0.225, b 0.126. No number in
// between was typed from memory.
func TestRedInOKLab(t *testing.T) {
	got := ok.FromSwatch(Red.Swatch())
	if !near(got.L, 0.628, 1e-3) || !near(got.A, 0.225, 1e-3) ||
		!near(got.B, 0.126, 1e-3) {
		t.Errorf("red in oklab = %+v", got)
	}
}

// Every hex code survives the trip through the swatch and back.
func TestHexRoundTrip(t *testing.T) {
	for _, h := range []string{"#000000", "#ffffff", "#ff0000", "#00ff00",
		"#0000ff", "#ff6ec7", "#ffa2ff",
		"#160d2b", "#7f7f7f", "#01fe80"} {
		c, err := FromHex(h)
		if err != nil {
			t.Fatal(err)
		}
		back, in := FromSwatch(c.Swatch())
		if !in {
			t.Errorf("%s reported out of gamut", h)
		}
		if got := back.Hex(); got != h {
			t.Errorf("%s came back as %s", h, got)
		}
	}
	if _, err := FromHex("#12345"); err == nil {
		t.Errorf("five digits accepted")
	}
	short, _ := FromHex("#f6c")
	long, _ := FromHex("#ff66cc")
	if short != long {
		t.Errorf("#f6c read as %+v, #ff66cc as %+v", short, long)
	}
}

// A colour the lamps cannot make says so: a green more colourful than the
// green lamp, built in oklab, comes back clipped and not in gamut.
func TestGamutIsHonest(t *testing.T) {
	loud := ok.OKLCH{L: 0.7, C: 0.4, H: 145}.Rect().Swatch()
	if _, in := FromSwatch(loud); in {
		t.Errorf("an impossible green claimed to be in gamut")
	}
}

// The cylinders agree with the textbook on the corners, and undo
// themselves.
func TestCylinders(t *testing.T) {
	if h := Red.HSL(); !near(h.H, 0, 1e-9) || !near(h.S, 1, 1e-9) ||
		!near(h.L, 0.5, 1e-9) {
		t.Errorf("red hsl = %+v", h)
	}
	if v := Red.HSV(); !near(v.H, 0, 1e-9) || !near(v.S, 1, 1e-9) ||
		!near(v.V, 1, 1e-9) {
		t.Errorf("red hsv = %+v", v)
	}
	if h := White.HSL(); !near(h.S, 0, 1e-9) || !near(h.L, 1, 1e-9) {
		t.Errorf("white hsl = %+v", h)
	}
	if v := Green.HSV(); !near(v.H, 120, 1e-9) {
		t.Errorf("green hue = %v", v.H)
	}
	if v := Blue.HSV(); !near(v.H, 240, 1e-9) {
		t.Errorf("blue hue = %v", v.H)
	}
	for _, c := range []RGB{Red, Green, Blue, White, Black, {0.2, 0.5, 0.9},
		{0.9, 0.3, 0.1}} {
		if r := c.HSL().RGB(); !near(r.R, c.R, 1e-9) || !near(r.G, c.G,
			1e-9) || !near(r.B, c.B, 1e-9) {
			t.Errorf("%+v via hsl came back %+v", c, r)
		}
		if r := c.HSV().RGB(); !near(r.R, c.R, 1e-9) || !near(r.G, c.G,
			1e-9) || !near(r.B, c.B, 1e-9) {
			t.Errorf("%+v via hsv came back %+v", c, r)
		}
	}
}

// a channel a hair outside is in gamut, since no display could show
// the difference; a whole step outside is not.
func TestGamutHasSlack(t *testing.T) {
	hair := swatch.FromXYZ(xyzOf(-5e-5, 0, 0.5))
	if _, in := FromSwatch(hair); !in {
		t.Error("a hair under zero was called out of gamut")
	}
	step := swatch.FromXYZ(xyzOf(-1e-3, 0, 0.5))
	if _, in := FromSwatch(step); in {
		t.Error("a whole step under zero passed as in gamut")
	}
}

// xyzOf is the tristimulus of a linear-light srgb triple, for a probe.
func xyzOf(r, g, b float64) (x, y, z float64) {
	return rgbToXYZ.Apply(r, g, b)
}

// bytes in and out, a colour is an image/color.Color, and equal means
// within a byte per channel.
func TestBytesColorAndEqual(t *testing.T) {
	c := RGB8(0x14, 0x14, 0x18)
	if r, g, b := c.Bytes(); r != 0x14 || g != 0x14 || b != 0x18 {
		t.Fatalf("bytes: %d %d %d", r, g, b)
	}
	var _ color.Color = c
	if r, _, _, a := RGB8(255, 0, 0).RGBA(); r != 0xffff || a != 0xffff {
		t.Errorf("rgba: %d %d", r, a)
	}
	if !c.Equal(RGB{c.R + 0.001, c.G, c.B}) {
		t.Error("a difference below a byte was unequal")
	}
	if c.Equal(RGB8(0x15, 0x14, 0x18)) {
		t.Error("a byte of difference was equal")
	}
	if !In(Red.Swatch()) || In(ok.OKLCH{L: 0.7, C: 0.4,
		H: 145}.Rect().Swatch()) {
		t.Error("In does not agree with FromSwatch")
	}
}
