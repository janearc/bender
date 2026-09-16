package textmap

import (
	"math"
	"testing"

	"github.com/uber/h3-go/v4"
)

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// Project and Unproject are inverses on both projections, across the
// window and across the disc.
func TestProjectionRoundTrip(t *testing.T) {
	for _, p := range []Projection{
		NewViewport(10, 45, 40, 80, 24),
		NewGlobe(-30, 20, 80, 24),
		NewHorizon(-30, 20, 80, 24),
	} {
		w, h := p.Pixels()
		checked := 0
		for y := 0; y < h; y += 3 {
			for x := 0; x < w; x += 3 {
				lon, lat, ok := p.Unproject(
					float64(x)+0.5,
					float64(y)+0.5,
				)
				if !ok {
					continue
				}
				px, py, ok := p.Project(lon, lat)
				if !ok || px != x || py != y {
					t.Errorf(
						"%T: pixel %d,%d -> %.4f,%.4f -> %d,%d %v",
						p,
						x,
						y,
						lon,
						lat,
						px,
						py,
						ok,
					)
				}
				checked++
			}
		}
		if checked == 0 {
			t.Errorf("%T: nothing to check", p)
		}
	}
}

// The far side of the globe is hidden and space is space.
func TestGlobeHidesFarSide(t *testing.T) {
	g := NewGlobe(0, 0, 80, 24)
	if _, _, ok := g.Project(180, 0); ok {
		t.Error("antipode should be hidden")
	}
	if _, _, ok := g.Project(0, 0); !ok {
		t.Error("centre should show")
	}
	if _, _, ok := g.Unproject(0, 0); ok {
		t.Error("the corner is space")
	}
	w, s, e, n := g.Bounds()
	if w != -180 || e != 180 || s != -90 || n != 90 {
		t.Errorf(
			"a full hemisphere from the equator holds both poles' longitudes: %v %v %v %v",
			w,
			s,
			e,
			n,
		)
	}
	g.Zoom(0.1) // in
	w, s, e, n = g.Bounds()
	if e-w > 180 || n-s > 90 {
		t.Errorf(
			"zoomed in, the bounds are a small box: %v %v %v %v",
			w,
			s,
			e,
			n,
		)
	}
	g.Pan(0.5, 0)
	if !near(g.Lon, 45, 1e-9) {
		t.Errorf("pan spins: %v", g.Lon)
	}
	g.SetCentre(200, 95)
	if !near(g.Lon, -160, 1e-9) || g.Lat != 89 {
		t.Errorf("SetCentre wraps and clamps: %v %v", g.Lon, g.Lat)
	}
}

func TestGlobeSetScale(t *testing.T) {
	g := NewGlobe(0, 0, 200, 50)
	g.SetScale(10)
	if !near(g.Scale(), 10, 1e-9) {
		t.Errorf("SetScale then Scale: %v", g.Scale())
	}
	g.SetScale(0)
	if !near(g.Scale(), 10, 1e-9) {
		t.Error("a zero span is ignored")
	}
	g.SetScale(1e-12)
	if g.Radius > maxRadius {
		t.Error("clamped like Zoom")
	}
	// A lidar tile's width fits on the globe and round-trips a pixel.
	g.SetScale(0.011)
	lon, lat, ok := g.Unproject(120.5, 30.5)
	x, y, ok2 := g.Project(lon, lat)
	if !ok || !ok2 || x != 120 || y != 30 {
		t.Errorf("tiny globe round trip: %v %v %d %d", ok, ok2, x, y)
	}
}

func TestViewportBoundsAndPan(t *testing.T) {
	v := NewViewport(0, 0, 90, 90, 45)
	w, s, e, n := v.Bounds()
	if !near(w, -45, 1e-9) || !near(e, 45, 1e-9) || !near(n-s, 90, 1e-9) {
		t.Errorf("bounds %v %v %v %v", w, s, e, n)
	}
	v.Pan(0.5, 0)
	if !near(v.Lon, 45, 1e-9) {
		t.Errorf("pan %v", v.Lon)
	}
	v.Pan(0, 100)
	if v.Lat != 85 {
		t.Errorf("latitude clamps short of the pole: %v", v.Lat)
	}
	v.Zoom(2)
	if v.SpanX != 180 {
		t.Errorf("zoom out %v", v.SpanX)
	}
	v.Zoom(100)
	if v.SpanX != 360 {
		t.Error("span caps at the world")
	}
	if _, _, ok := NewViewport(0, 0, 10, 80, 24).Project(0, -89); ok {
		t.Error("clipping is the point")
	}
	v.Resize(0, 0)
	if v.Cols != 1 || v.Rows != 1 {
		t.Error("resize floor")
	}
}

// Zooming about a pixel keeps the coordinate under it under it, on both
// projections, in both directions.
func TestZoomAtKeepsAnchor(t *testing.T) {
	for _, p := range []Projection{
		NewViewport(10, 45, 40, 80, 24),
		NewGlobe(-30, 20, 80, 24),
		NewHorizon(-30, 20, 80, 24),
	} {
		for _, factor := range []float64{0.5, 2} {
			x, y := 46.5, 28.5
			lon0, lat0, ok := p.Unproject(x, y)
			if !ok {
				t.Fatalf("%T: anchor off the planet", p)
			}
			ZoomAt(p, factor, x, y)
			lon1, lat1, ok := p.Unproject(x, y)
			if !ok || !near(lon0, lon1, 0.05) ||
				!near(lat0, lat1, 0.05) {
				t.Errorf(
					"%T factor %v: anchor moved from %.3f,%.3f to %.3f,%.3f (%v)",
					p,
					factor,
					lon0,
					lat0,
					lon1,
					lat1,
					ok,
				)
			}
		}
		// Anchored in space, the globe just zooms.
		if g, ok := p.(*Globe); ok {
			lon, lat := g.Lon, g.Lat
			ZoomAt(g, 2, 0, 0)
			if g.Lon != lon || g.Lat != lat {
				t.Error(
					"zoom anchored in space should not turn the globe",
				)
			}
		}
	}
}

// Dragging moves the coordinate that was under the pointer to where the
// pointer went.
func TestDrag(t *testing.T) {
	for _, p := range []Projection{
		NewViewport(10, 45, 40, 80, 24),
		NewGlobe(-30, 20, 80, 24),
		NewHorizon(-30, 20, 80, 24),
	} {
		lon0, lat0, _ := p.Unproject(40.5, 24.5)
		Drag(p, 40.5, 24.5, 50.5, 20.5)
		lon1, lat1, ok := p.Unproject(50.5, 20.5)
		if !ok || !near(lon0, lon1, 0.5) || !near(lat0, lat1, 0.5) {
			t.Errorf(
				"%T: grabbed %.3f,%.3f, now %.3f,%.3f under the pointer",
				p,
				lon0,
				lat0,
				lon1,
				lat1,
			)
		}
		// A drag from space does nothing.
		if g, ok := p.(*Globe); ok {
			lon, lat := g.Lon, g.Lat
			Drag(g, 0, 0, 10, 10)
			if g.Lon != lon || g.Lat != lat {
				t.Error(
					"drag from space should not move the globe",
				)
			}
		}
	}
}

func TestResForScale(t *testing.T) {
	// A full world on a normal terminal is coarse; zooming in steps finer,
	// one resolution per about 2.65x, and never past the source.
	world := ResForScale(360, 200, 15)
	if world < 1 || world > 3 {
		t.Errorf("world res %d", world)
	}
	// The globe's scale is the disc's: a full globe on 200 columns is a
	// hemisphere across about 98 pixels, coarser than the flat world.
	g := NewGlobe(0, 0, 200, 50)
	if g.Scale() < 360 || ResForScale(g.Scale(), 200, 15) < 1 {
		t.Errorf(
			"globe scale %.0f res %d",
			g.Scale(),
			ResForScale(g.Scale(), 200, 15),
		)
	}
	last := world
	for scale := 360.0; scale > 0.01; scale /= 2.65 {
		r := ResForScale(scale, 200, 15)
		if r < last || r > last+1 {
			t.Errorf(
				"scale %.3f: res %d after %d, should step by at most one",
				scale,
				r,
				last,
			)
		}
		last = r
	}
	if ResForScale(0.001, 200, 4) != 4 {
		t.Error("clamps to native")
	}
	if ResForScale(0, 200, 4) != 0 || ResForScale(1, 0, 4) != 0 {
		t.Error("degenerate inputs")
	}
}

func TestRamp(t *testing.T) {
	stops := Heat.Stops
	if Ramp(-1) != stops[0] || Ramp(2) != stops[len(stops)-1] ||
		Ramp(0) != stops[0] {
		t.Error("ends")
	}
	mid := Ramp(0.5)
	if mid != stops[2] {
		t.Errorf("middle stop %v", mid)
	}
}

// A source rolls up: intensive layers take the mean, extensive add; the
// sampler finds the cell under a point, and walks coarser only when the
// source is mixed.
func TestCellSource(t *testing.T) {
	ll := h3.LatLng{Lat: 37.7, Lng: -122.4}
	fine, _ := h3.LatLngToCell(ll, 6)
	parent, _ := fine.Parent(4)
	children, _ := parent.Children(6)
	cells := map[string]any{}
	for i, c := range children {
		cells[c.String()] = map[string]any{
			"count":  float64(i),
			"height": 10.0,
		}
	}
	cells["not-a-cell"] = 1.0
	cells[children[0].String()+"x"] = map[string]any{"count": "text"}
	src := NewCellSource(
		"t",
		cells,
		map[string]string{"count": "LAYER_KIND_EXTENSIVE"},
	)
	if src.Len() != len(children) || src.NativeRes() != 6 {
		t.Fatalf("len %d native %d", src.Len(), src.NativeRes())
	}
	if l := src.Layers(); len(l) != 2 || l[0] != "count" {
		t.Errorf("layers %v", l)
	}
	if !src.IsExtensive("count") || src.IsExtensive("height") {
		t.Error("kinds")
	}
	up := src.Table(4, "count")
	sum := 0.0
	for i := range children {
		sum += float64(i)
	}
	if up[parent] != sum {
		t.Errorf("extensive rollup %v want %v", up[parent], sum)
	}
	if src.Table(4, "height")[parent] != 10 {
		t.Error("intensive rollup takes the mean")
	}
	if len(src.Table(9, "height")) != len(children) {
		t.Error("finer than native clamps to native")
	}
	at := src.Sampler(6, "height")
	if c, v, ok := at(ll.Lat, ll.Lng); !ok || c != fine || v != 10 {
		t.Errorf("sampler %v %v %v", c, v, ok)
	}
	if _, _, ok := at(0, 0); ok {
		t.Error("nothing at the equator")
	}
	// Coverage: a bare number per cell is a value layer.
	cov := NewCellSource(
		"c",
		map[string]any{children[0].String(): 1.0},
		nil,
	)
	if cov.Layers()[0] != "value" {
		t.Error("bare number layer")
	}
	lon, lat := src.Centre()
	if !near(lon, ll.Lng, 0.5) || !near(lat, ll.Lat, 0.5) {
		t.Errorf("centre %v %v", lon, lat)
	}
	// A mixed source walks coarser on a miss.
	mixed := NewCellSource(
		"m",
		map[string]any{parent.String(): 5.0, children[0].String(): 7.0},
		nil,
	)
	at = mixed.Sampler(6, "value")
	far, _ := children[len(children)-1].LatLng()
	if _, v, ok := at(far.Lat, far.Lng); !ok || v != 5 {
		t.Errorf("mixed walk %v %v", v, ok)
	}
}

func TestRender(t *testing.T) {
	ll := h3.LatLng{Lat: 37.7, Lng: -122.4}
	c, _ := h3.LatLngToCell(ll, 2)
	src := NewCellSource("t", map[string]any{c.String(): 3.0}, nil)
	v := NewViewport(ll.Lng, ll.Lat, 4, 20, 10)
	f := Render(src, v, "value", 2)
	if f.Filled == 0 {
		t.Fatal("nothing filled")
	}
	if f.Filled == f.W*f.H {
		t.Error("a single cell should not fill a 4-degree window")
	}
	// One value is a coverage map: everything set is at T 1.
	for _, px := range f.Pixels {
		if px.Set && (px.T != 1 || px.Cell != c) {
			t.Fatalf("flat frame pixel %+v", px)
		}
	}
	if got := f.At(-1, 0); got.Set {
		t.Error("off frame")
	}
	if _, _, ok := CellAt(src, v, "value", 2, float64(f.W)/2, float64(f.H)/2); !ok {
		t.Error("CellAt the centre")
	}
	if _, _, ok := CellAt(nil, v, "value", 2, 0, 0); ok {
		t.Error("nil source")
	}
	empty := Render(nil, v, "value", 2)
	if empty.Filled != 0 || len(empty.Pixels) != 20*20 {
		t.Error("empty frame is still a full rectangle")
	}
	// Two cells with different values normalise to the ends.
	ring, _ := c.GridDisk(1)
	d := ring[len(ring)-1]
	if d == c {
		d = ring[0]
	}
	two := NewCellSource(
		"t",
		map[string]any{c.String(): 3.0, d.String(): 9.0},
		nil,
	)
	f = Render(two, NewViewport(ll.Lng, ll.Lat, 8, 20, 10), "value", 2)
	if f.Lo != 3 || f.Hi != 9 {
		t.Errorf("scale %v %v", f.Lo, f.Hi)
	}
}

func TestGraticule(t *testing.T) {
	cases := map[float64]float64{
		360:    90,
		100:    30,
		20:     5,
		2:      0.5,
		0.044:  0.01,
		0.0001: 0.001,
	}
	for span, want := range cases {
		if got := Graticule(span); got != want {
			t.Errorf("span %v: step %v, want %v", span, got, want)
		}
	}
}

// Unsurveyed pixels on the planet are marked, with graticule lines where
// the grid index changes; space is neither.
func TestPlanetAndGrid(t *testing.T) {
	g := NewGlobe(0, 0, 40, 20)
	f := Render(nil, g, "v", 0)
	planet, grid, space := 0, 0, 0
	for _, px := range f.Pixels {
		switch {
		case px.Planet && px.Grid:
			grid++
			planet++
		case px.Planet:
			planet++
		default:
			space++
		}
	}
	if planet == 0 || grid == 0 || space == 0 {
		t.Errorf("planet %d grid %d space %d", planet, grid, space)
	}
	if f.At(0, 0).Planet {
		t.Error("the corner is space")
	}
	if !f.At(20, 20).Planet {
		t.Error("the centre is planet")
	}
	// The quick render marks them too.
	q := RenderQuick(nil, g, "v", 0)
	if !q.At(20, 20).Planet {
		t.Error("quick render planet")
	}
}
