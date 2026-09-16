package textmap

import (
	"math"
	"testing"
)

// limbRow is the first row down a column that lands on the planet, or -1.
func limbRow(h *Horizon, x float64) int {
	_, ht := h.Pixels()
	for y := 0; y < ht; y++ {
		if _, _, ok := h.Unproject(x, float64(y)+0.5); ok {
			return y
		}
	}
	return -1
}

// The limb sits where Sky says, with space above it and ground below: the
// sky is not drawn, it is the rays that miss.
func TestHorizonLimbWhereAsked(t *testing.T) {
	h := NewHorizon(-30, 20, 126, 20)
	w, ht := h.Pixels()
	got := limbRow(h, float64(w)/2)
	want := h.Sky * float64(ht)
	if math.Abs(float64(got)-want) > 1.5 {
		t.Errorf("limb at row %d of %d, want about %.1f", got, ht, want)
	}
	if _, _, ok := h.Unproject(float64(w)/2, 0.5); ok {
		t.Error("top of the middle column should be space")
	}
	if _, _, ok := h.Unproject(float64(w)/2, float64(ht)-0.5); !ok {
		t.Error("bottom of the middle column should be ground")
	}
}

// The limb is an arc, highest in the middle, which is the whole reason for
// the view: a straight line would be a flat map with a sky painted on.
func TestHorizonLimbCurves(t *testing.T) {
	h := NewHorizon(-30, 20, 126, 20)
	w, _ := h.Pixels()
	mid, edge := limbRow(h, float64(w)/2), limbRow(h, 0.5)
	if mid < 0 || edge <= mid {
		t.Errorf(
			"limb at the middle row %d, at the edge row %d; want the edge lower",
			mid,
			edge,
		)
	}
}

// A zoom changes height and the camera tilts to keep the limb in place, so
// coming down to the ground does not tip the horizon off the top.
func TestHorizonZoomKeepsTheLimb(t *testing.T) {
	h := NewHorizon(-30, 20, 126, 20)
	w, _ := h.Pixels()
	before := limbRow(h, float64(w)/2)
	for _, f := range []float64{0.25, 8} {
		h.Zoom(f)
		if got := limbRow(h, float64(w)/2); math.Abs(
			float64(got-before),
		) > 1 {
			t.Errorf(
				"after zoom %v the limb moved from row %d to %d",
				f,
				before,
				got,
			)
		}
	}
}

// SetCentre puts the coordinate at the middle of the window, and SetScale
// changes the width without losing it.
func TestHorizonCentreAndScale(t *testing.T) {
	h := NewHorizon(0, 0, 126, 20)
	h.SetCentre(10, 20)
	lon, lat := h.Centre()
	if !near(lon, 10, 1e-4) || !near(lat, 20, 1e-4) {
		t.Fatalf("centre %.5f,%.5f, want 10,20", lon, lat)
	}
	for _, span := range []float64{2, 30} {
		h.SetScale(span)
		if got := h.Scale(); math.Abs(got-span)/span > 0.01 {
			t.Errorf("SetScale(%v) gave %.3f", span, got)
		}
		lon, lat = h.Centre()
		if !near(lon, 10, 1e-3) || !near(lat, 20, 1e-3) {
			t.Errorf(
				"SetScale(%v) moved the centre to %.4f,%.4f",
				span,
				lon,
				lat,
			)
		}
	}
}

// Bounds holds every pixel on the planet, since it is what kingfisher is
// asked for, and anything outside it would be drawn from nothing.
func TestHorizonBoundsHoldTheView(t *testing.T) {
	h := NewHorizon(-30, 20, 126, 20)
	bw, bs, be, bn := h.Bounds()
	w, ht := h.Pixels()
	const slack = 0.5
	for y := 0; y < ht; y++ {
		for x := 0; x < w; x += 5 {
			lon, lat, ok := h.Unproject(
				float64(x)+0.5,
				float64(y)+0.5,
			)
			if !ok {
				continue
			}
			if lon < bw-slack || lon > be+slack || lat < bs-slack ||
				lat > bn+slack {
				t.Fatalf(
					"pixel %d,%d at %.2f,%.2f is outside %.2f,%.2f,%.2f,%.2f",
					x,
					y,
					lon,
					lat,
					bw,
					bs,
					be,
					bn,
				)
			}
		}
	}
	// Facing across the antimeridian, a box cannot say it, so the request
	// widens to every longitude rather than asking for the wrong side.
	h.SetCentre(179.5, 0)
	h.Heading = 90
	h.SetCentre(179.5, 0)
	if w, _, e, _ := h.Bounds(); w != -180 || e != 180 {
		t.Errorf(
			"across the antimeridian bounds are %.2f..%.2f, want the whole circle",
			w,
			e,
		)
	}
}

// Forward is up the window and a turn is across it; a flight along a
// heading keeps going the same way.
func TestHorizonPan(t *testing.T) {
	h := NewHorizon(0, 0, 126, 20)
	h.Pan(0, 0.5)
	if !(h.Lat > 0) || !near(h.Lon, 0, 1e-9) {
		t.Errorf(
			"flying north from 0,0 went to %.4f,%.4f",
			h.Lon,
			h.Lat,
		)
	}
	h.Pan(0.5, 0)
	if !near(h.Heading, h.FOV/2, 1e-9) {
		t.Errorf(
			"heading %.3f after half a window's turn, want %.3f",
			h.Heading,
			h.FOV/2,
		)
	}
	h = NewHorizon(0, 0, 126, 20)
	h.Heading = 90
	for i := 0; i < 8; i++ {
		h.Pan(0, 1)
	}
	if !near(h.Lat, 0, 1e-6) || !near(h.Heading, 90, 1e-6) {
		t.Errorf(
			"flying east along the equator drifted to lat %.6f heading %.6f",
			h.Lat,
			h.Heading,
		)
	}
}

// Air glows just above the limb and fades out; the top of the sky and the
// ground itself do not glow, and the render marks only space as air.
func TestGlow(t *testing.T) {
	h := NewHorizon(-30, 20, 126, 20)
	w, _ := h.Pixels()
	x := float64(w) / 2
	limb := limbRow(h, x)
	above := float64(limb) - 0.5
	if g := h.Glow(x, above); g <= 0.5 {
		t.Errorf("just above the limb glows %.2f, want bright", g)
	}
	if g := h.Glow(x, 0.5); g != 0 {
		t.Errorf("the top of the sky glows %.2f, want none", g)
	}
	if h.Glow(x, above-2) >= h.Glow(x, above) {
		t.Error("the glow should fade away from the limb")
	}
	g := NewGlobe(0, 0, 80, 24)
	cx, cy := g.centre()
	if g.Glow(cx+g.Radius+0.5, cy) <= 0 || g.Glow(cx, cy) != 0 ||
		g.Glow(cx+g.Radius+10, cy) != 0 {
		t.Error("the globe's air is a ring just outside the disc")
	}
	f := Render(nil, h, "value", 0)
	air := 0
	for _, px := range f.Pixels {
		if px.Glow > 0 {
			air++
			if px.Planet || px.Set {
				t.Fatal("only space glows")
			}
		}
	}
	if air == 0 {
		t.Error("the render drew no air at all")
	}
}
