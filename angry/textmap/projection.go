package textmap

import "math"

// PixelRowsPerChar is the half-block convention: two map rows per
// character row, and those pixels are roughly square.
const PixelRowsPerChar = 2

// Projection is the window on the world, whichever shape it takes. TextMap
// never learns which one it is holding, which is what makes a globe cheap: the
// fill asks every pixel which cell contains it, so a projection only has to
// answer what coordinate a pixel is and whether it is on the planet at all.
type Projection interface {
	// Pixels is the pixel grid behind the character grid.
	Pixels() (w, h int)
	// Bounds is west, south, east, north in real degrees.
	Bounds() (w, s, e, n float64)
	// Project is a coordinate's pixel, or false when it is outside the
	// window or behind the planet.
	Project(lon, lat float64) (x, y int, ok bool)
	// Unproject is the coordinate at a point in pixel space, or false for
	// space. Fractional pixels are allowed, so a subpixel can be sampled.
	Unproject(x, y float64) (lon, lat float64, ok bool)
	// Pan moves by a fraction of the window: a translation on the flat map,
	// a rotation on the globe.
	Pan(fx, fy float64)
	// Zoom by a factor about the centre: below one moves in.
	Zoom(factor float64)
	// Resize the character grid.
	Resize(cols, rows int)
	// Scale is the width of the window in degrees, for choosing a
	// resolution and for a status line.
	Scale() float64
	// Centre is the coordinate at the middle of the window.
	Centre() (lon, lat float64)
	// SetCentre moves the window to a coordinate.
	SetCentre(lon, lat float64)
}

// Viewport is the flat window: a centre, a span, and the grid it fills.
// Work happens in projected units, x = lon * cos(centre latitude), y = lat:
// equirectangular with the aspect corrected for where you are standing,
// so a degree of x and a degree of y are the same size on screen.
type Viewport struct {
	Lon, Lat float64
	SpanX    float64 // width in projected degrees
	Cols     int
	Rows     int
}

// NewViewport makes a flat window.
func NewViewport(lon, lat, spanX float64, cols, rows int) *Viewport {
	v := &Viewport{Lon: lon, Lat: lat, SpanX: math.Max(1e-9, spanX)}
	v.Resize(cols, rows)
	return v
}

// Pixels is the grid: two map rows per character row.
func (v *Viewport) Pixels() (int, int) {
	return v.Cols, v.Rows * PixelRowsPerChar
}

// Resize sets the character grid.
func (v *Viewport) Resize(
	cols, rows int,
) {
	v.Cols, v.Rows = max(1, cols), max(1, rows)
}

// kx is the longitude foreshortening at this latitude, floored just above
// zero so a viewport at the pole narrows rather than dividing by nothing.
func (v *Viewport) kx() float64 {
	return math.Max(1e-6, math.Cos(v.Lat*math.Pi/180))
}

// SpanY is the window's height in projected degrees, from the grid's aspect.
func (v *Viewport) SpanY() float64 {
	w, h := v.Pixels()
	return v.SpanX * float64(h) / float64(w)
}

// Bounds in real degrees.
func (v *Viewport) Bounds() (float64, float64, float64, float64) {
	hx := v.SpanX / 2 / v.kx()
	hy := v.SpanY() / 2
	return v.Lon - hx, v.Lat - hy, v.Lon + hx, v.Lat + hy
}

// Project a coordinate to a pixel; clipping is the point.
func (v *Viewport) Project(lon, lat float64) (int, int, bool) {
	w, h := v.Pixels()
	x := (lon-v.Lon)*v.kx()/v.SpanX*float64(w) + float64(w)/2
	y := float64(h)/2 - (lat-v.Lat)/v.SpanY()*float64(h)
	if x < 0 || y < 0 || x >= float64(w) || y >= float64(h) {
		return 0, 0, false
	}
	return int(x), int(y), true
}

// Unproject a pixel-space point to a coordinate.
func (v *Viewport) Unproject(x, y float64) (float64, float64, bool) {
	w, h := v.Pixels()
	lon := v.Lon + (x-float64(w)/2)*v.SpanX/float64(w)/v.kx()
	lat := v.Lat - (y-float64(h)/2)*v.SpanY()/float64(h)
	return lon, lat, true
}

// Pan by a fraction of the window. Latitude stops short of the pole.
func (v *Viewport) Pan(fx, fy float64) {
	v.Lon += fx * v.SpanX / v.kx()
	v.Lat = math.Max(-85, math.Min(85, v.Lat+fy*v.SpanY()))
}

// Zoom about the centre.
func (v *Viewport) Zoom(factor float64) {
	v.SpanX = math.Max(1e-7, math.Min(360, v.SpanX*factor))
}

// Scale is the span.
func (v *Viewport) Scale() float64 { return v.SpanX }

// Centre is the middle of the window.
func (v *Viewport) Centre() (float64, float64) { return v.Lon, v.Lat }

// SetCentre moves the window.
func (v *Viewport) SetCentre(lon, lat float64) {
	v.Lon = lon
	v.Lat = math.Max(-85, math.Min(85, lat))
}

// Globe is the same window wrapped round a ball: orthographic, one
// hemisphere, the far side genuinely hidden rather than squashed round the
// edge. Radius is in pixels, so zoom is a scale and pan is an angle.
type Globe struct {
	Lon, Lat float64
	Cols     int
	Rows     int
	Radius   float64
}

// NewGlobe makes a globe filling the grid.
func NewGlobe(lon, lat float64, cols, rows int) *Globe {
	g := &Globe{Lon: lon, Lat: lat}
	g.Resize(cols, rows)
	w, h := g.Pixels()
	g.Radius = float64(min(w, h)) / 2 * 0.98
	return g
}

// Pixels is the grid.
func (g *Globe) Pixels() (int, int) { return g.Cols, g.Rows * PixelRowsPerChar }

// Resize sets the character grid, keeping the radius.
func (g *Globe) Resize(
	cols, rows int,
) {
	g.Cols, g.Rows = max(1, cols), max(1, rows)
}

// centre is the middle of the frame in pixels, where the globe's disc sits.
func (g *Globe) centre() (float64, float64) {
	w, h := g.Pixels()
	return float64(w) / 2, float64(h) / 2
}

// Project a coordinate; the cosine of the angular distance decides
// visibility, and negative means the far side.
func (g *Globe) Project(lon, lat float64) (int, int, bool) {
	p := lat * math.Pi / 180
	p0 := g.Lat * math.Pi / 180
	dl := (lon - g.Lon) * math.Pi / 180
	cosc := math.Sin(p0)*math.Sin(p) + math.Cos(p0)*math.Cos(p)*math.Cos(dl)
	if cosc < 0 {
		return 0, 0, false
	}
	cx, cy := g.centre()
	x := cx + g.Radius*math.Cos(p)*math.Sin(dl)
	tilt := math.Cos(p0)*math.Sin(p) -
		math.Sin(p0)*math.Cos(p)*math.Cos(dl)
	y := cy - g.Radius*tilt
	w, h := g.Pixels()
	if x < 0 || y < 0 || x >= float64(w) || y >= float64(h) {
		return 0, 0, false
	}
	return int(x), int(y), true
}

// Unproject a pixel-space point, or false for space: off the disc is not
// off the map, it is not the planet at all.
func (g *Globe) Unproject(x, y float64) (float64, float64, bool) {
	cx, cy := g.centre()
	dx, dy := x-cx, y-cy
	rho := math.Hypot(dx, dy)
	if rho > g.Radius {
		return 0, 0, false
	}
	if rho < 1e-9 {
		return g.Lon, g.Lat, true
	}
	c := math.Asin(math.Min(1, rho/g.Radius))
	p0 := g.Lat * math.Pi / 180
	sinc, cosc := math.Sin(c), math.Cos(c)
	lat := math.Asin(cosc*math.Sin(p0) + (-dy*sinc*math.Cos(p0))/rho)
	lon := g.Lon*math.Pi/180 + math.Atan2(
		dx*sinc,
		rho*cosc*math.Cos(p0)+dy*sinc*math.Sin(p0),
	)
	return math.Mod(
		lon*180/math.Pi+540,
		360,
	) - 180, lat * 180 / math.Pi, true
}

// Glow is how far outside the disc a point is, as a fraction of the air's
// thickness. The disc's rim is the limb seen face on, so the air is a ring
// around it; at least a pixel and a half thick, or at a small radius it
// would not be drawn at all.
func (g *Globe) Glow(x, y float64) float64 {
	cx, cy := g.centre()
	rho := math.Hypot(x-cx, y-cy)
	if rho <= g.Radius {
		return 0
	}
	thick := math.Max(1.5, g.Radius*AtmosphereRadii)
	return math.Max(0, 1-(rho-g.Radius)/thick)
}

// Bounds is what to ask for: the world when the disc wraps the antimeridian
// or holds a pole, else the rim's extent.
func (g *Globe) Bounds() (float64, float64, float64, float64) {
	cx, cy := g.centre()
	var lons, lats []float64
	const steps = 64
	for i := 0; i < steps; i++ {
		a := 2 * math.Pi * float64(i) / steps
		lon, lat, ok := g.Unproject(
			cx+g.Radius*0.999*math.Cos(a),
			cy+g.Radius*0.999*math.Sin(a),
		)
		if ok {
			lons = append(lons, lon)
			lats = append(lats, lat)
		}
	}
	// Zoomed in past the frame, the frame's edge bounds the view, not the
	// disc's rim: sample the border and keep what is on the planet.
	w, h := g.Pixels()
	if g.Radius > float64(min(w, h))/2 {
		lons, lats = nil, nil
		for i := 0; i < steps; i++ {
			f := float64(i) / steps
			fw, fh := f*float64(w), f*float64(h)
			edge := [][2]float64{
				{fw, 0.5},
				{fw, float64(h) - 0.5},
				{0.5, fh},
				{float64(w) - 0.5, fh},
			}
			for _, pt := range edge {
				lon, lat, ok := g.Unproject(pt[0], pt[1])
				if ok {
					lons = append(lons, lon)
					lats = append(lats, lat)
				}
			}
		}
	}
	if len(lons) == 0 {
		return -180, -90, 180, 90
	}
	minLon, maxLon, minLat, maxLat := lons[0], lons[0], lats[0], lats[0]
	for i := range lons {
		minLon, maxLon = math.Min(
			minLon,
			lons[i],
		), math.Max(
			maxLon,
			lons[i],
		)
		minLat, maxLat = math.Min(
			minLat,
			lats[i],
		), math.Max(
			maxLat,
			lats[i],
		)
	}
	wrapped := maxLon-minLon > 180
	_, _, poleN := g.Project(0, 89.9)
	_, _, poleS := g.Project(0, -89.9)
	pole := poleN || poleS
	if wrapped || pole {
		if pole {
			return -180, -90, 180, 90
		}
		return -180, minLat, 180, maxLat
	}
	return minLon, minLat, maxLon, maxLat
}

// Pan spins and tilts: the step is in degrees, not a fraction of the
// window, because the disc does not change size with the data.
func (g *Globe) Pan(fx, fy float64) {
	g.Lon = math.Mod(g.Lon+fx*90+540, 360) - 180
	g.Lat = math.Max(-89, math.Min(89, g.Lat+fy*60))
}

// maxRadius caps the globe's radius in pixels. The Python stopped at
// twenty times the window; a lidar tile a hundredth of a degree wide
// needs a radius in the millions of pixels to fill a window, and the
// orthographic maths is fine there, so the cap is only against overflow.
const maxRadius = 1e8

// Zoom scales the radius.
func (g *Globe) Zoom(factor float64) {
	g.Radius = math.Max(2, math.Min(g.Radius/factor, maxRadius))
}

// SetScale sets the radius so the window covers a span in degrees, the
// inverse of Scale, clamped to the same limits as Zoom.
func (g *Globe) SetScale(span float64) {
	w, _ := g.Pixels()
	if span <= 0 {
		return
	}
	g.Radius = math.Max(2, math.Min(180*float64(w)/(2*span), maxRadius))
}

// Scale is the flat viewport's span equivalent: the degrees a window this
// wide covers at the disc's scale, where the disc's diameter is half a
// great circle, 180 degrees.
func (g *Globe) Scale() float64 {
	w, _ := g.Pixels()
	return 180 * float64(w) / (2 * g.Radius)
}

// Centre is the point facing the viewer.
func (g *Globe) Centre() (float64, float64) { return g.Lon, g.Lat }

// SetCentre turns the globe to face a coordinate.
func (g *Globe) SetCentre(lon, lat float64) {
	g.Lon = math.Mod(lon+540, 360) - 180
	g.Lat = math.Max(-89, math.Min(89, lat))
}

// settle moves the window until the coordinate lon,lat sits under the
// pixel x,y, by shifting the centre by the remaining error a few times. On
// the flat map one round is exact; on the globe the shift is a rotation and
// the rounds converge.
func settle(p Projection, x, y, lon, lat float64) {
	for i := 0; i < 12; i++ {
		lon1, lat1, ok := p.Unproject(x, y)
		if !ok {
			return
		}
		dlon := lon - lon1
		if dlon > 180 {
			dlon -= 360
		} else if dlon < -180 {
			dlon += 360
		}
		dlat := lat - lat1
		if math.Abs(dlon) < 1e-6 && math.Abs(dlat) < 1e-6 {
			return
		}
		clon, clat := p.Centre()
		p.SetCentre(clon+dlon, clat+dlat)
	}
}

// ZoomAt zooms about a pixel so the coordinate under it stays under it.
// An anchor in space just zooms.
func ZoomAt(p Projection, factor float64, x, y float64) {
	lon0, lat0, ok := p.Unproject(x, y)
	p.Zoom(factor)
	if !ok {
		return
	}
	settle(p, x, y, lon0, lat0)
}

// Drag moves the window so the coordinate that was under one pixel is now
// under another, which is what grabbing the map means. A drag from space
// does nothing.
func Drag(p Projection, fromX, fromY, toX, toY float64) {
	lon0, lat0, ok := p.Unproject(fromX, fromY)
	if !ok {
		return
	}
	if _, _, ok := p.Unproject(toX, toY); !ok {
		return
	}
	settle(p, toX, toY, lon0, lat0)
}
