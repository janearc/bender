package canvas

import "testing"

// the bayer pattern's density matches its mix over a large area.
func TestBayer(t *testing.T) {
	// Every threshold in a 4x4 tile is distinct and covers 0-1 evenly.
	seen := map[float64]bool{}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			seen[Threshold(x, y, 4)] = true
		}
	}
	if len(seen) != 16 {
		t.Errorf("4x4 has %d distinct thresholds", len(seen))
	}
	seen = map[float64]bool{}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			seen[Threshold(x, y, 8)] = true
		}
	}
	if len(seen) != 64 {
		t.Errorf("8x8 has %d distinct thresholds", len(seen))
	}
	if Threshold(-1, -1, 4) != Threshold(3, 3, 4) {
		t.Error("negative coordinates should tile")
	}
	if Threshold(1, 1, 5) != Threshold(1, 1, 4) {
		t.Error("an unknown size falls back to 4")
	}
	// The mix fraction is honoured: half the pixels of a tile take B.
	p := Pattern{Kind: Dither4, Mix: 0.5}
	n := 0
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if p.UseB(x, y) {
				n++
			}
		}
	}
	if n != 8 {
		t.Errorf("dither at half mixes %d of 16", n)
	}
	p8 := Pattern{Kind: Dither8, Mix: 0.25}
	n = 0
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if p8.UseB(x, y) {
				n++
			}
		}
	}
	if n != 16 {
		t.Errorf("dither8 at a quarter mixes %d of 64", n)
	}
}

// a stipple's density matches its mix, and its seed makes it repeatable.
func TestStipple(t *testing.T) {
	p := Pattern{Kind: Stipple, Mix: 0.3, Seed: 7}
	n := 0
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if p.UseB(x, y) {
				n++
			}
		}
	}
	if n < 2500 || n > 3500 {
		t.Errorf("stipple at 0.3 set %d of 10000", n)
	}
	if Noise(3, 4, 1) != Noise(3, 4, 1) ||
		Noise(3, 4, 1) == Noise(3, 4, 2) ||
		Noise(3, 4, 1) == Noise(4, 3, 1) {
		t.Error("noise should be stable per pixel " +
			"and vary with seed and position")
	}
	if v := Noise(-7, -9, 3); v < 0 || v >= 1 {
		t.Error("noise range")
	}
}

// checker alternates every pixel; hatch alternates by row or by column.
func TestCheckerAndHatch(t *testing.T) {
	c := Pattern{Kind: Checker, Period: 2}
	if c.UseB(0, 0) || !c.UseB(2, 0) || !c.UseB(0, 2) || c.UseB(2, 2) ||
		c.UseB(1, 1) {
		t.Error("checker period 2")
	}
	if c.UseB(-1, 0) == c.UseB(0, 0) {
		t.Error("checker across zero")
	}
	c0 := Pattern{Kind: Checker}
	if c0.UseB(0, 0) || !c0.UseB(1, 0) {
		t.Error("checker period defaults to 1")
	}
	h := Pattern{Kind: Hatch, Period: 4, Weight: 1, Style: Horizontal}
	if !h.UseB(5, 0) || h.UseB(5, 1) || !h.UseB(5, 4) {
		t.Error("horizontal hatch")
	}
	h.Style = Vertical
	if !h.UseB(0, 5) || h.UseB(1, 5) {
		t.Error("vertical hatch")
	}
	h.Style = Diagonal
	h.Weight = 2
	if !h.UseB(0, 0) || !h.UseB(1, 0) || h.UseB(2, 0) || !h.UseB(3, 1) {
		t.Error("diagonal hatch with weight 2")
	}
	h.Weight = 0
	if !h.UseB(0, 0) || h.UseB(1, 0) {
		t.Error("weight defaults to 1")
	}
	if (Pattern{Kind: Flat}).UseB(3, 3) {
		t.Error("flat never uses B")
	}
	if (Pattern{Kind: Checker, Period: 2}).PartAt(2, 0, 1, 2) != 2 ||
		(Pattern{Kind: Flat}).PartAt(2, 0, 1, 2) != 1 {
		t.Error("PartAt")
	}
	kinds := map[PatternKind]string{
		Flat:    "flat",
		Dither4: "dither 4x4",
		Dither8: "dither 8x8",
		Stipple: "stipple",
		Checker: "checker",
		Hatch:   "hatch",
	}
	for k, want := range kinds {
		if k.String() != want {
			t.Errorf("%v", k)
		}
	}
	hatches := map[HatchStyle]string{
		Diagonal:   "diagonal",
		Horizontal: "horizontal",
		Vertical:   "vertical",
	}
	for s, want := range hatches {
		if s.String() != want {
			t.Errorf("%v", s)
		}
	}
}

// the flat form is the textbook 4x4 matrix, and the 8x8 the recurrence
// builds from it: four times the inner tile plus the 2x2 base by quadrant.
func TestBayerIsTheMatrix(t *testing.T) {
	table := [4][4]int{
		{0, 8, 2, 10},
		{12, 4, 14, 6},
		{3, 11, 1, 9},
		{15, 7, 13, 5},
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			got := int(Threshold(x, y, 4)*16 - 0.5)
			if got != table[y][x] {
				t.Fatalf(
					"4x4 at %d,%d: %d, want %d",
					x,
					y,
					got,
					table[y][x],
				)
			}
		}
	}
	base := [2][2]int{{0, 2}, {3, 1}}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			want := 4*table[y%4][x%4] + base[y/4][x/4]
			got := int(Threshold(x, y, 8)*64 - 0.5)
			if got != want {
				t.Fatalf(
					"8x8 at %d,%d: %d, want %d",
					x,
					y,
					got,
					want,
				)
			}
		}
	}
}
