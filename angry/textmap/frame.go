package textmap

import (
	"math"
	"sort"

	"github.com/uber/h3-go/v4"
)

// RGB is a colour as bytes.
type RGB struct{ R, G, B uint8 }

// Palette is a named run of colours a normalised value is drawn in.
// Linear between stops; nobody needs a colour science library to make a
// terminal look good.
type Palette struct {
	Name  string
	Stops []RGB
}

// Heat runs cold to hot: blue, cyan, green, amber, red. It is the default,
// and right for anything where low and high are both worth seeing.
var Heat = Palette{
	"heat",
	[]RGB{
		{0x2b, 0x4b, 0xd9},
		{0x00, 0xc8, 0xe0},
		{0x3f, 0xd0, 0x6b},
		{0xe8, 0xa3, 0x3d},
		{0xe0, 0x4f, 0x4f},
	},
}

// Night is lights on a dark planet: ember, amber, gold, white. It starts
// at a dim ember rather than at black, because the darkest thing on
// screen should be the planet between the lights, not the faintest light.
var Night = Palette{
	"night",
	[]RGB{
		{0x5a, 0x24, 0x08},
		{0xc0, 0x60, 0x10},
		{0xff, 0xb0, 0x30},
		{0xff, 0xe6, 0x9a},
		{0xff, 0xff, 0xf4},
	},
}

// Palettes are the ones a flag or a command can name.
var Palettes = []Palette{Heat, Night}

// At is the colour for a normalised value in 0..1.
func (pl Palette) At(t float64) RGB {
	st := pl.Stops
	if len(st) == 0 {
		// A zero palette is the default one, so a style nobody set
		// draws the way everything drew before there were palettes.
		st = Heat.Stops
	}
	t = math.Max(0, math.Min(1, t))
	x := t * float64(len(st)-1)
	i := int(x)
	if i >= len(st)-1 {
		return st[len(st)-1]
	}
	a, b, f := st[i], st[i+1], x-float64(i)
	lerp := func(p, q uint8) uint8 {
		return uint8(float64(p) + (float64(q)-float64(p))*f)
	}
	return RGB{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B)}
}

// Ramp is the default palette's colour for a normalised value, kept so
// every caller that predates palettes reads the same.
func Ramp(t float64) RGB { return Heat.At(t) }

// Pixel is one map pixel: whether anything was surveyed there, its normalised
// value, and the cell that answered for it. A space is never a value: blank
// means absent.
//
// Planet says the pixel is on the world even though nothing was surveyed there,
// which is what lets a globe read as a globe against space; Grid marks a
// graticule line through such a pixel. Neither is data and neither is drawn in
// the ramp.
type Pixel struct {
	Set    bool
	T      float64 // normalised 0..1
	Value  float64 // the raw value
	Cell   h3.Cell // the cell under the pixel's centre, when Set
	Planet bool
	Grid   bool
	// Glow is atmosphere: how close to the limb a pixel of space passes,
	// 0 for none and 1 against the planet. Only space glows.
	Glow float64
}

// Glower is a projection that knows where its atmosphere is. It is
// optional, so a projection without a limb -- the flat map -- has no glow
// rather than a wrong one.
type Glower interface {
	// Glow is the atmosphere at a point in pixel space that misses the
	// planet: 1 at the limb, falling to 0 at the top of the air.
	Glow(x, y float64) float64
}

// AtmosphereRadii is how thick the drawn atmosphere is, in planet radii.
// The real one is thinner; this is for showing off, and at terminal
// resolution anything thinner than a pixel or two is not there at all.
const AtmosphereRadii = 0.02

// Atmosphere is the colour of a glow: deep blue at the top of the air,
// bright cyan against the limb.
func Atmosphere(g float64) RGB {
	return Palette{
		"air",
		[]RGB{
			{0x12, 0x2a, 0x6e},
			{0x2f, 0x7f, 0xd8},
			{0x9a, 0xec, 0xff},
		},
	}.At(
		g,
	)
}

// Graticule is the spacing of the grid lines drawn on unsurveyed planet,
// in degrees, chosen for the window's span so there are a few lines
// across it whatever the zoom.
func Graticule(spanDeg float64) float64 {
	steps := []float64{
		90, 30, 10, 5, 1, 0.5, 0.1, 0.05, 0.01, 0.005, 0.001,
	}
	for _, step := range steps {
		if spanDeg/step >= 3 {
			return step
		}
	}
	return 0.001
}

// Frame is one picture: a pixel grid with its scale.
type Frame struct {
	W, H   int
	Pixels []Pixel
	Lo, Hi float64
	Res    int // the resolution drawn
	Filled int // pixels that found a cell
}

// At is the pixel at x,y, or an unset one off the frame.
func (f *Frame) At(x, y int) Pixel {
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return Pixel{}
	}
	return f.Pixels[y*f.W+x]
}

// ResForScale chooses the H3 resolution for a window scale: the finest
// resolution that still puts a few pixels in a hex. The width of the window in
// degrees over its width in pixels gives kilometres per pixel; a hex's average
// area over that squared gives pixels per hex.
//
// Take the finest resolution with at least four pixels per hex, which steps one
// resolution per 2.65x of linear zoom, the square root of the sevenfold area
// ratio between resolutions. Verified against h3-go v4.5.0.
func ResForScale(scaleDeg float64, widthPx int, native int) int {
	if widthPx < 1 || scaleDeg <= 0 {
		return 0
	}
	const kmPerDeg = 111.32
	kmPerPx := scaleDeg * kmPerDeg / float64(widthPx)
	pxArea := kmPerPx * kmPerPx
	const target = 4.0 // pixels per hex, the floor of the 4 to 9 window
	best := 0
	for r := 0; r <= min(native, MaxRes); r++ {
		area, err := h3.HexagonAreaAvgKm2(r)
		if err != nil {
			break
		}
		if area/pxArea >= target {
			best = r
		} else {
			break
		}
	}
	return best
}

// Options shape a render. Quick samples once per pixel, for the frames drawn
// while the map is being dragged. Pin fixes the colour scale to a range instead
// of the frame's own, so the same ground keeps its colour as the view moves and
// two frames can be compared, which is the point of a map.
//
// Scale says how values become 0..1; see Scale.
type Options struct {
	Quick bool
	Pin   *[2]float64
	Scale Scale
}

// Scale is how raw values are laid onto a palette. The zero value is the
// old behaviour: linear, minimum to maximum, every value drawn.
//
// It exists because some data is mostly one value. Half of Black Marble's
// cells hold the same dark floor and a few city cores hold ten times what
// everything else does, so a linear scale from minimum to maximum puts
// ninety-nine per cent of the planet in the bottom fifth of the ramp.
type Scale struct {
	// Floor, when set, is a value at or below which a cell is not drawn as
	// data: the pixel falls through to unsurveyed planet. For night lights
	// that is the dark ground between them, which reads better as the
	// planet's own silhouette than as the faintest light.
	Floor *float64
	// Clip is the quantile the top of the scale is taken at, 0.99 for the
	// ninety-ninth percentile; zero means the maximum. Values above it
	// draw at the top colour.
	Clip float64
	// Log spaces the scale logarithmically, so the faint end gets room.
	Log bool
}

// Norm lays a value onto 0..1 between lo and hi under the scale.
func (sc Scale) Norm(v, lo, hi float64) float64 {
	if hi-lo < 1e-12 {
		return 1
	}
	if sc.Log {
		return math.Max(
			0,
			math.Min(
				1,
				math.Log1p(math.Max(0, v-lo))/math.Log1p(hi-lo),
			),
		)
	}
	return math.Max(0, math.Min(1, (v-lo)/(hi-lo)))
}

// Bounds is the scale's lo and hi over a set of values: the floor or the
// least value at the bottom, the clip quantile or the greatest at the top.
// It reports false when no value survives the floor.
func (sc Scale) Bounds(vals []float64) (lo, hi float64, ok bool) {
	kept := make([]float64, 0, len(vals))
	for _, v := range vals {
		if sc.Floor != nil && v <= *sc.Floor {
			continue
		}
		kept = append(kept, v)
	}
	if len(kept) == 0 {
		return 0, 0, false
	}
	sort.Float64s(kept)
	lo, hi = kept[0], kept[len(kept)-1]
	if sc.Floor != nil {
		lo = *sc.Floor
	}
	if sc.Clip > 0 && sc.Clip < 1 {
		hi = kept[int(sc.Clip*float64(len(kept)-1))]
	}
	return lo, hi, true
}

// Render paints one frame: every pixel asks which cell contains it, with a 2x2
// supersample so the limb and the cell edges are blended rather than stepped.
// The scale is taken over the whole frame after the fill, so a cell whose
// centre is off screen still contributes to it.
//
// Supersampling blends cells, not colours: the value is the mean of the cells
// found, and the pixel names the first cell it found.
func Render(src *CellSource, p Projection, layer string, res int) *Frame {
	return RenderWith(src, p, layer, res, Options{})
}

// RenderQuick is Render with one sample per pixel, for a drag in progress.
func RenderQuick(src *CellSource, p Projection, layer string, res int) *Frame {
	return RenderWith(src, p, layer, res, Options{Quick: true})
}

// RenderWith paints one frame with options.
func RenderWith(
	src *CellSource,
	p Projection,
	layer string,
	res int,
	o Options,
) *Frame {
	return render(src, p, layer, res, o.Quick, o.Pin, o.Scale)
}

// render fills every pixel, then lays the filled ones onto the scale.
func render(
	src *CellSource,
	p Projection,
	layer string,
	res int,
	quick bool,
	pin *[2]float64,
	sc Scale,
) *Frame {
	w, h := p.Pixels()
	f := &Frame{W: w, H: h, Pixels: make([]Pixel, w*h), Res: res}
	at := func(lat, lon float64) (h3.Cell, float64, bool) {
		return 0, 0, false
	}
	if src != nil {
		at = src.Sampler(res, layer)
	}
	offsets := [][2]float64{
		{0.25, 0.25},
		{0.75, 0.25},
		{0.25, 0.75},
		{0.75, 0.75},
	}
	if quick {
		offsets = [][2]float64{{0.5, 0.5}}
	}
	step := Graticule(p.Scale())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sum, n := 0.0, 0
			var cell h3.Cell
			for _, o := range offsets {
				lon, lat, ok := p.Unproject(
					float64(x)+o[0],
					float64(y)+o[1],
				)
				if !ok || lat < -90 || lat > 90 {
					continue
				}
				lon = math.Mod(lon+540, 360) - 180
				c, v, ok := at(lat, lon)
				if !ok {
					continue
				}
				sum += v
				if n == 0 {
					// the first sample found
					// stands for the pixel
					cell = c
				}
				n++
			}
			if n == 0 {
				// Unsurveyed. On the planet or in space, and if
				// on the planet, on a graticule line or not: a
				// line is where the grid index changes between
				// this pixel and the next.
				lon, lat, ok := p.Unproject(
					float64(x)+0.5,
					float64(y)+0.5,
				)
				if !ok || lat < -90 || lat > 90 {
					// Space, but perhaps air: a projection
					// with a limb says how close to it this
					// pixel passes.
					if gl, is := p.(Glower); is && !ok {
						gx := float64(x) + 0.5
						gy := float64(y) + 0.5
						if g := gl.Glow(gx, gy); g > 0 {
							f.Pixels[y*w+x] = Pixel{
								Glow: g,
							}
						}
					}
					continue
				}
				px := Pixel{Planet: true}
				lonR, _, okR := p.Unproject(
					float64(x)+1.5,
					float64(y)+0.5,
				)
				_, latD, okD := p.Unproject(
					float64(x)+0.5,
					float64(y)+1.5,
				)
				if okR &&
					math.Floor(
						lonR/step,
					) != math.Floor(
						lon/step,
					) ||
					okD &&
						math.Floor(
							latD/step,
						) != math.Floor(
							lat/step,
						) {
					px.Grid = true
				}
				f.Pixels[y*w+x] = px
				continue
			}
			v := sum / float64(n)
			if sc.Floor != nil && v <= *sc.Floor {
				// At or under the floor is ground, not data:
				// the pixel is planet, and draws as the
				// planet's silhouette.
				f.Pixels[y*w+x] = Pixel{Planet: true}
				continue
			}
			f.Pixels[y*w+x] = Pixel{Set: true, Value: v, Cell: cell}
			f.Filled++
		}
	}
	if f.Filled == 0 {
		return f
	}
	vals := make([]float64, 0, f.Filled)
	for _, px := range f.Pixels {
		if px.Set {
			vals = append(vals, px.Value)
		}
	}
	lo, hi, _ := sc.Bounds(vals)
	if pin != nil {
		lo, hi = pin[0], pin[1]
	}
	f.Lo, f.Hi = lo, hi
	// A flat source is a coverage map, not an empty one: every bin is data,
	// and Norm draws it at the top.
	for i := range f.Pixels {
		px := &f.Pixels[i]
		if px.Set {
			px.T = sc.Norm(px.Value, lo, hi)
		}
	}
	return f
}

// CellAt is the cell under a pixel-space point at a resolution, for hover
// and click, which need the cell under the pointer rather than under the
// pixel's centre. False when the point is off the planet or unsurveyed.
func CellAt(
	src *CellSource,
	p Projection,
	layer string,
	res int,
	x, y float64,
) (h3.Cell, float64, bool) {
	lon, lat, ok := p.Unproject(x, y)
	if !ok || lat < -90 || lat > 90 || src == nil {
		return 0, 0, false
	}
	lon = math.Mod(lon+540, 360) - 180
	return src.Sampler(res, layer)(lat, lon)
}
