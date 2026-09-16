package textmap

import "math"

// Horizon is the view from low orbit: a camera a few hundred kilometres up,
// looking ahead at the horizon rather than down at the ground.
//
// The globe is a camera infinitely far away, so it is always a whole disc.
// This one is close enough for perspective: the limb comes out as a shallow
// arc with the ground foreshortening up into it. Nothing about the data
// changes.
//
// Every pixel still asks what coordinate it is, and the answer comes from a ray
// cast from the camera instead of from the orthographic formula, so kingfisher
// sees the same window-and-budget request it sees for any other view.
type Horizon struct {
	Lon, Lat float64 // the point directly beneath the camera
	Heading  float64 // degrees clockwise from north the camera faces
	Alt      float64 // height above the surface, in planet radii
	FOV      float64 // horizontal field of view, degrees
	// fraction of the height above the limb, at the middle column
	Sky  float64
	Cols int
	Rows int
}

// Where a new horizon view starts. 0.063 radii is about 400 km, the height
// of the station in the photographs this view is meant to look like.
const (
	horizonAlt = 0.063
	horizonFOV = 90
	horizonSky = 0.25
	minAlt     = 0.0005
	maxAlt     = 4
)

// NewHorizon makes a view with the camera above a coordinate, facing north.
func NewHorizon(lon, lat float64, cols, rows int) *Horizon {
	h := &Horizon{
		Lon: lon,
		Lat: lat,
		Alt: horizonAlt,
		FOV: horizonFOV,
		Sky: horizonSky,
	}
	h.Resize(cols, rows)
	return h
}

// Pixels is the grid.
func (h *Horizon) Pixels() (int, int) {
	return h.Cols, h.Rows * PixelRowsPerChar
}

// Resize sets the character grid, keeping the camera where it is.
func (h *Horizon) Resize(
	cols, rows int,
) {
	h.Cols, h.Rows = max(1, cols), max(1, rows)
}

// vec is a point or a direction in planet-centred space, where the surface
// is the unit sphere. The globe gets away with trigonometry on angles; a
// camera that is not at infinity needs rays, and rays want vectors.
type vec [3]float64

// dot is the dot product.
func (a vec) dot(b vec) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

// add is the sum.
func (a vec) add(
	b vec,
) vec {
	return vec{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

// scale multiplies by a number.
func (a vec) scale(k float64) vec { return vec{a[0] * k, a[1] * k, a[2] * k} }

// cross is the cross product, used once, to get the camera's right from its
// forward and up.
func (a vec) cross(b vec) vec {
	return vec{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

// unit is the same direction at length one.
func (a vec) unit() vec { return a.scale(1 / math.Sqrt(a.dot(a))) }

// rad converts degrees, which the Projection interface speaks, to radians.
func rad(deg float64) float64 { return deg * math.Pi / 180 }

// deg converts back.
func deg(r float64) float64 { return r * 180 / math.Pi }

// wrapLon keeps a longitude in -180..180, so a flight across the
// antimeridian does not accumulate a longitude of 400.
func wrapLon(lon float64) float64 { return math.Mod(lon+540, 360) - 180 }

// clampLat stops short of the poles, where north and east stop meaning
// anything and the camera's frame would spin.
func clampLat(lat float64) float64 { return math.Max(-89, math.Min(89, lat)) }

// surface is a coordinate as a point on the unit sphere.
func surface(lon, lat float64) vec {
	p, l := rad(lat), rad(lon)
	return vec{
		math.Cos(p) * math.Cos(l),
		math.Cos(p) * math.Sin(l),
		math.Sin(p),
	}
}

// coord is the inverse of surface.
func coord(p vec) (float64, float64) {
	return wrapLon(
			deg(math.Atan2(p[1], p[0])),
		), deg(
			math.Asin(math.Max(-1, math.Min(1, p[2]))),
		)
}

// tangent is the half-width of the window in the camera's image plane. One
// tangent unit per half-width in both axes, because the pixels are square.
func (h *Horizon) tangent() float64 { return math.Tan(rad(h.FOV) / 2) }

// pitch is how far below level the camera looks. It is derived rather than
// stored so the limb stays where Sky puts it through a zoom: the horizon
// sits lower the higher you are, and the camera tilts to follow it.
func (h *Horizon) pitch() float64 {
	w, ht := h.Pixels()
	dip := math.Acos(1 / (1 + h.Alt))
	above := (0.5 - h.Sky) * 2 * h.tangent() * float64(ht) / float64(w)
	return dip + math.Atan(above)
}

// local is the ground frame at a coordinate: straight up, east and north.
// Heading is measured in it, which is why it is the one place a compass
// direction turns into a vector.
func local(lon, lat float64) (up, east, north vec) {
	l, p := rad(lon), rad(lat)
	up = surface(lon, lat)
	east = vec{-math.Sin(l), math.Cos(l), 0}
	north = vec{
		-math.Sin(p) * math.Cos(l),
		-math.Sin(p) * math.Sin(l),
		math.Cos(p),
	}
	return up, east, north
}

// ahead is the level direction the camera faces.
func (h *Horizon) ahead() (n, hd vec) {
	n, east, north := local(h.Lon, h.Lat)
	return n, north.scale(math.Cos(rad(h.Heading))).
		add(east.scale(math.Sin(rad(h.Heading))))
}

// camera is the eye and its three axes in planet-centred space: forward,
// right and up in the image.
func (h *Horizon) camera() (eye, fwd, right, up vec) {
	n, hd := h.ahead()
	t := h.pitch()
	fwd = hd.scale(math.Cos(t)).add(n.scale(-math.Sin(t)))
	up = hd.scale(math.Sin(t)).add(n.scale(math.Cos(t)))
	right = fwd.cross(up)
	return n.scale(1 + h.Alt), fwd, right, up
}

// perPixel is tangent units per pixel.
func (h *Horizon) perPixel() float64 {
	w, _ := h.Pixels()
	return h.tangent() / (float64(w) / 2)
}

// Unproject casts the pixel's ray at the planet. A miss is space, which is
// what puts sky above the limb without anything having to draw it.
func (h *Horizon) Unproject(x, y float64) (float64, float64, bool) {
	w, ht := h.Pixels()
	eye, fwd, right, up := h.camera()
	k := h.perPixel()
	d := fwd.add(right.scale((x - float64(w)/2) * k)).
		add(up.scale((float64(ht)/2 - y) * k)).
		unit()
	b := eye.dot(d)
	disc := b*b - (eye.dot(eye) - 1)
	if disc < 0 {
		return 0, 0, false
	}
	t := -b - math.Sqrt(disc)
	if t <= 0 {
		return 0, 0, false
	}
	lon, lat := coord(eye.add(d.scale(t)))
	return lon, lat, true
}

// Glow is how close the pixel's ray passes to the planet, as a fraction
// of the atmosphere's thickness: a ray that grazes the limb is lit, one
// that clears the air is not. A ray pointing away from the planet never
// comes closer than the camera, so it does not glow.
func (h *Horizon) Glow(x, y float64) float64 {
	w, ht := h.Pixels()
	eye, fwd, right, up := h.camera()
	k := h.perPixel()
	d := fwd.add(right.scale((x - float64(w)/2) * k)).
		add(up.scale((float64(ht)/2 - y) * k)).
		unit()
	b := eye.dot(d)
	if b >= 0 {
		return 0
	}
	closest := math.Sqrt(math.Max(0, eye.dot(eye)-b*b))
	if closest <= 1 {
		return 0 // this ray hits the planet; it is ground, not air
	}
	g := 1 - (closest-1)/AtmosphereRadii
	return math.Max(0, g)
}

// Project is the pixel a coordinate lands on, or false when it is past the
// horizon, behind the camera, or outside the window.
func (h *Horizon) Project(lon, lat float64) (int, int, bool) {
	eye, fwd, right, up := h.camera()
	p := surface(lon, lat)
	// A point faces the camera when the eye is above its horizon plane.
	if eye.dot(p) <= 1 {
		return 0, 0, false
	}
	v := p.add(eye.scale(-1))
	z := v.dot(fwd)
	if z <= 0 {
		return 0, 0, false
	}
	w, ht := h.Pixels()
	k := h.perPixel()
	x := float64(w)/2 + v.dot(right)/z/k
	y := float64(ht)/2 - v.dot(up)/z/k
	if x < 0 || y < 0 || x >= float64(w) || y >= float64(ht) {
		return 0, 0, false
	}
	return int(x), int(y), true
}

// Bounds is what to ask kingfisher for: the extent of every pixel that
// lands on the planet. Sampled on a grid because the visible ground is a
// foreshortened cap with no tidy edge to walk, and widened to the world the
// same way the globe does when it crosses the antimeridian or sees a pole.
func (h *Horizon) Bounds() (float64, float64, float64, float64) {
	w, ht := h.Pixels()
	const nx, ny = 48, 24
	first := true
	var minLon, maxLon, minLat, maxLat float64
	take := func(x, y float64) {
		lon, lat, ok := h.Unproject(x, y)
		if !ok {
			return
		}
		if first {
			minLon, maxLon, minLat, maxLat = lon, lon, lat, lat
			first = false
			return
		}
		minLon, maxLon = math.Min(minLon, lon), math.Max(maxLon, lon)
		minLat, maxLat = math.Min(minLat, lat), math.Max(maxLat, lat)
	}
	for i := 0; i <= nx; i++ {
		x := math.Min(
			float64(w)-0.5,
			math.Max(0.5, float64(i)*float64(w)/nx),
		)
		for j := 0; j <= ny; j++ {
			take(
				x,
				math.Min(
					float64(ht)-0.5,
					math.Max(
						0.5,
						float64(j)*float64(ht)/ny,
					),
				),
			)
		}
		// The farthest ground in a column is a sliver just under the
		// limb, thinner than any grid row, and it is the part of the
		// view that covers the most planet. So find the limb itself and
		// sample there.
		top, bottom := 0.0, float64(ht)
		if _, _, ok := h.Unproject(x, top); ok {
			continue
		}
		if _, _, ok := h.Unproject(x, bottom); !ok {
			continue
		}
		for k := 0; k < 40; k++ {
			mid := (top + bottom) / 2
			if _, _, ok := h.Unproject(x, mid); ok {
				bottom = mid
			} else {
				top = mid
			}
		}
		take(x, bottom)
	}
	if first {
		return -180, -90, 180, 90
	}
	_, _, poleN := h.Project(0, 89.9)
	_, _, poleS := h.Project(0, -89.9)
	if poleN || poleS {
		return -180, -90, 180, 90
	}
	if maxLon-minLon > 180 {
		return -180, minLat, 180, maxLat
	}
	return minLon, minLat, maxLon, maxLat
}

// Pan flies and turns: up the window is forward along the heading, across
// it is a turn. A fraction of the window is a fraction of the field of view
// for a turn, and of the distance to the horizon for a step forward, so
// one press means about the same amount of motion at any height.
func (h *Horizon) Pan(fx, fy float64) {
	h.Heading = math.Mod(h.Heading+fx*h.FOV+360, 360)
	h.fly(fy * math.Acos(1/(1+h.Alt)))
}

// fly moves the camera along its heading by an arc in radians, carrying the
// heading along the great circle so a long flight stays straight.
func (h *Horizon) fly(arc float64) {
	if arc == 0 {
		return
	}
	n, hd := h.ahead()
	n2 := n.scale(math.Cos(arc)).add(hd.scale(math.Sin(arc)))
	hd2 := hd.scale(math.Cos(arc)).add(n.scale(-math.Sin(arc)))
	h.Lon, h.Lat = coord(n2)
	h.Lat = clampLat(h.Lat)
	_, east, north := local(h.Lon, h.Lat)
	h.Heading = math.Mod(
		deg(math.Atan2(hd2.dot(east), hd2.dot(north)))+360,
		360,
	)
}

// Zoom changes height, keeping the ground at the middle of the window where
// it was: below one comes down.
func (h *Horizon) Zoom(factor float64) {
	lon, lat := h.Centre()
	h.Alt = math.Max(minAlt, math.Min(maxAlt, h.Alt*factor))
	h.SetCentre(lon, lat)
}

// Scale is the width of ground the window spans across its middle row, in
// degrees of arc. The foreground is narrower and the far field wider; the
// middle is the honest single number for choosing a resolution.
func (h *Horizon) Scale() float64 {
	w, ht := h.Pixels()
	for _, y := range []float64{float64(ht) / 2, float64(ht) - 0.5} {
		lon0, lat0, ok0 := h.Unproject(0.5, y)
		lon1, lat1, ok1 := h.Unproject(float64(w)-0.5, y)
		if ok0 && ok1 {
			a, b := surface(lon0, lat0), surface(lon1, lat1)
			return deg(
				math.Acos(math.Max(-1, math.Min(1, a.dot(b)))),
			)
		}
	}
	return 180
}

// SetScale finds the height at which the window spans a width, keeping the
// same ground in the middle. The span grows with height, so a bisection on
// the logarithm of the height finds it.
func (h *Horizon) SetScale(span float64) {
	if span <= 0 {
		return
	}
	lon, lat := h.Centre()
	lo, hi := math.Log(minAlt), math.Log(maxAlt)
	for i := 0; i < 48; i++ {
		mid := (lo + hi) / 2
		h.Alt = math.Exp(mid)
		if h.Scale() < span {
			lo = mid
		} else {
			hi = mid
		}
	}
	h.SetCentre(lon, lat)
}

// Centre is the ground at the middle of the window, or the point beneath
// the camera if the middle is sky.
func (h *Horizon) Centre() (float64, float64) {
	w, ht := h.Pixels()
	if lon, lat, ok := h.Unproject(float64(w)/2, float64(ht)/2); ok {
		return lon, lat
	}
	return h.Lon, h.Lat
}

// SetCentre moves the camera until a coordinate is at the middle of the
// window. The camera sits some way back from what it looks at, so moving it
// by the remaining error a few times converges, as settle does for a drag.
func (h *Horizon) SetCentre(lon, lat float64) {
	lat = clampLat(lat)
	for i := 0; i < 16; i++ {
		clon, clat := h.Centre()
		dlon := wrapLon(lon - clon)
		dlat := lat - clat
		if math.Abs(dlon) < 1e-7 && math.Abs(dlat) < 1e-7 {
			return
		}
		h.Lon = wrapLon(h.Lon + dlon)
		h.Lat = clampLat(h.Lat + dlat)
	}
}
