package textmap

import (
	"math"
	"testing"

	"github.com/uber/h3-go/v4"
)

// The zero scale is the old behaviour: minimum to maximum, linear.
func TestScaleZeroIsLinear(t *testing.T) {
	var sc Scale
	lo, hi, ok := sc.Bounds([]float64{3, 1, 2})
	if !ok || lo != 1 || hi != 3 {
		t.Fatalf("bounds %v %v %v", lo, hi, ok)
	}
	if got := sc.Norm(2, lo, hi); got != 0.5 {
		t.Errorf("norm %v, want 0.5", got)
	}
	if sc.Norm(5, 5, 5) != 1 {
		t.Error("a flat source is a coverage map and draws at the top")
	}
}

// Black Marble's shape: most of it one dark value, a long tail, a few
// huge cores. Under the night scale the median light sits well up the
// ramp; under the old one it is crushed at the bottom.
func TestScaleUncrushesALongTail(t *testing.T) {
	vals := make([]float64, 0, 1000)
	for i := 0; i < 600; i++ {
		vals = append(vals, 5.72) // the dark floor
	}
	for i := 0; i < 395; i++ {
		vals = append(
			vals,
			6+float64(i)*0.12,
		) // ordinary lights, 6 to 53
	}
	for i := 0; i < 2; i++ {
		vals = append(
			vals,
			255,
		) // city cores, half a per cent of the lights
	}
	floor := 5.72
	night := Scale{Floor: &floor, Clip: 0.99, Log: true}
	lo, hi, ok := night.Bounds(vals)
	if !ok || lo != floor || hi >= 255 {
		t.Fatalf(
			"night bounds %v..%v, want the floor to below the cores",
			lo,
			hi,
		)
	}
	var old Scale
	olo, ohi, _ := old.Bounds(vals)
	median := 29.0
	if n, o := night.Norm(median, lo, hi), old.Norm(median, olo, ohi); n < 0.6 ||
		o > 0.15 {
		t.Errorf(
			"median light at %.2f under night, %.2f under the old scale",
			n,
			o,
		)
	}
}

// Nothing survives a floor above everything, and that is reported rather
// than invented.
func TestScaleFloorAboveEverything(t *testing.T) {
	f := 100.0
	if _, _, ok := (Scale{Floor: &f}).Bounds([]float64{1, 2, 3}); ok {
		t.Error("no value survives, so no bounds")
	}
}

// Log gives the faint end room and still reaches both ends.
func TestScaleLog(t *testing.T) {
	sc := Scale{Log: true}
	if sc.Norm(0, 0, 100) != 0 || math.Abs(sc.Norm(100, 0, 100)-1) > 1e-12 {
		t.Error("ends")
	}
	if !(sc.Norm(5, 0, 100) > (Scale{}).Norm(5, 0, 100)) {
		t.Error("log should lift the faint end")
	}
}

// Under a floor, the ground between the lights is planet, not data: it
// draws in the planet's tint and so shows the silhouette against space.
func TestRenderFloorIsGround(t *testing.T) {
	c, _ := h3.LatLngToCell(h3.LatLng{Lat: 37.7, Lng: -122.4}, 3)
	disk, _ := c.GridDisk(4)
	cells := map[string]any{}
	for _, d := range disk {
		cells[d.String()] = 5.72
	}
	src := NewCellSource("floor", cells, nil)
	p := NewViewport(-122.4, 37.7, 4, 20, 10)
	floor := 5.72
	f := RenderWith(
		src,
		p,
		"value",
		3,
		Options{Scale: Scale{Floor: &floor}},
	)
	planet := 0
	for _, px := range f.Pixels {
		if px.Set {
			t.Fatal("a value at the floor was drawn as data")
		}
		if px.Planet {
			planet++
		}
	}
	if planet == 0 {
		t.Error("the floored ground should be planet")
	}
}

// The night palette starts at an ember rather than black, so the faintest
// light is never mistaken for the ground; a zero palette draws as heat.
func TestPalettes(t *testing.T) {
	if Night.At(0) == (RGB{}) {
		t.Error("night's faint end must not be black")
	}
	if (Palette{}).At(0.3) != Heat.At(0.3) {
		t.Error("the zero palette is heat")
	}
}
