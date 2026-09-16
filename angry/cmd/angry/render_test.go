package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/janearc/angry-dist/kingfisher"
	"github.com/janearc/angry-dist/textmap"
)

// A frame of the real Black Marble dataset at a terminal's size, timed,
// because a drag repaints on every sample and has to keep up. Skipped when
// kingfisher is not reachable.
func TestRenderRealDataset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	kf := kingfisher.New(kingfisher.URL())
	m, err := kf.Manifest(ctx, "gibs-VIIRS_Black_Marble-r4")
	if err != nil {
		t.Skipf("kingfisher not reachable: %v", err)
	}
	cells, err := kf.Cells(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	src := textmap.NewCellSource(m.ID, cells.Cells, m.LayerKinds())
	if src.Len() < 200000 || src.NativeRes() != 4 {
		t.Fatalf("%d cells at res %d", src.Len(), src.NativeRes())
	}
	for _, p := range []textmap.Projection{
		textmap.NewViewport(0, 20, 360, 200, 50),
		textmap.NewGlobe(-100, 40, 200, 50),
	} {
		res := textmap.ResForScale(p.Scale(), 200, src.NativeRes())
		start := time.Now()
		f := textmap.Render(src, p, "brightness", res)
		first := time.Since(start)
		start = time.Now()
		textmap.Render(src, p, "brightness", res)
		again := time.Since(start)
		t.Logf(
			"%T res %d: %d of %d pixels filled, first frame %v, next %v",
			p,
			res,
			f.Filled,
			f.W*f.H,
			first,
			again,
		)
		if f.Filled < f.W*f.H/10 {
			t.Errorf("%T: too little of the world drawn", p)
		}
		if again > 400*time.Millisecond {
			t.Errorf(
				"%T: a frame takes %v, too slow to drag",
				p,
				again,
			)
		}
		if res < 1 || res > 4 {
			t.Errorf("world view res %d", res)
		}
	}
}

// A lidar tile: 1886 cells at res 12 over a hundredth of a degree. The
// view fitted to its bounding box must reach the native resolution and
// fill, and a coarse rollup of it must still draw; both are how a person
// would first see it. Skipped when kingfisher is not reachable.
func TestRenderLidarTile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	kf := kingfisher.New(kingfisher.URL())
	m, err := kf.Manifest(ctx, "usgs-628dbf12d34ef70cdba3c7c1")
	if err != nil {
		t.Skipf("kingfisher not reachable: %v", err)
	}
	cells, err := kf.Cells(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	src := textmap.NewCellSource(m.ID, cells.Cells, m.LayerKinds())
	if src.NativeRes() != 12 || src.Len() < 1000 {
		t.Fatalf("%d cells at res %d", src.Len(), src.NativeRes())
	}
	if !src.IsExtensive("point_density") &&
		m.LayerKinds()["point_density"] != "" &&
		strings.Contains(m.LayerKinds()["point_density"], "EXTENSIVE") {
		t.Error("kind")
	}
	w, s, e, n := m.Pipeline.BBox[0], m.Pipeline.BBox[1], m.Pipeline.BBox[2], m.Pipeline.BBox[3]
	v := textmap.NewViewport((w+e)/2, (s+n)/2, (e-w)*1.06, 200, 50)
	res := textmap.ResForScale(v.Scale(), 200, src.NativeRes())
	if res != 12 {
		t.Errorf(
			"fitted to the tile the view should draw at the native res 12, got %d",
			res,
		)
	}
	for _, layer := range src.Layers() {
		start := time.Now()
		f := textmap.Render(src, v, layer, res)
		took := time.Since(start)
		start = time.Now()
		q := textmap.RenderQuick(src, v, layer, res)
		quick := time.Since(start)
		t.Logf(
			"%s at res %d: %d of %d pixels, full %v, quick %v (%d pixels), values %.2f to %.2f",
			layer,
			res,
			f.Filled,
			f.W*f.H,
			took,
			quick,
			q.Filled,
			f.Lo,
			f.Hi,
		)
		if quick > took {
			t.Errorf(
				"%s: the quick frame took longer than the full one",
				layer,
			)
		}
		// water_share is carried only by cells with water in them, so it is
		// sparse by nature; the elevation layers cover the tile.
		if layer == "elevation_ground" && f.Filled < f.W*f.H/4 {
			t.Errorf(
				"%s: the tile should fill most of a view fitted to it",
				layer,
			)
		}
		if f.Hi <= f.Lo {
			t.Errorf("%s: no range", layer)
		}
	}
	// Zoomed out to the county the tile is a dot at a coarse resolution,
	// rolled up, and still there.
	county := textmap.NewViewport((w+e)/2, (s+n)/2, 1, 200, 50)
	cres := textmap.ResForScale(county.Scale(), 200, src.NativeRes())
	f := textmap.Render(src, county, "elevation_ground", cres)
	if cres >= 12 || f.Filled == 0 {
		t.Errorf("county view: res %d, %d pixels", cres, f.Filled)
	}
	// Hover on the tile names a res-12 cell.
	c, _, ok := textmap.CellAt(src, v, "elevation_ground", res, 100.5, 50.5)
	if !ok || c.Resolution() != 12 {
		t.Errorf("hover %v %v", c, ok)
	}
}
