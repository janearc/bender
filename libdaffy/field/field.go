package field

import "math"

// How the maths becomes a picture.
//
// Think of the surface being painted as a square sheet, one unit on a
// side. Every cell on the screen is a point on that sheet: u says how far
// across, v how far down, both from 0 to 1. Painting the sheet is two
// separate questions, asked once per cell:
//
//	1. What number does this point get?      (a Field: position to t)
//	2. What colour does that number get?      (a Ramp: t to a swatch)
//
// A Field answers the first. It is any rule that takes u and v and gives back a
// t between 0 and 1. Nothing more: no colour, no cells, no idea which ramp it
// will feed. The Ramp answers the second and never learns where the number came
// from.
//
// The cell is painted with ramp.At(field.T(u, v)), and that expression is what
// a gradient is. There is no gradient type, because a gradient is those two
// things put together, and keeping them apart is what lets one ramp paint a
// sun, a sky and a stripe.
//
// The rules that come with the package are the three shapes everyone
// knows, and they are examples, not the limit:
//
//	Linear   the number grows along a direction, 0 at one end, 1 at the
//	         other. bands.
//	Radial   the number is the distance from a centre, 0 there and 1 at a
//	         radius. rings. a sun is this.
//	Conic    the number is the angle around a centre, 0 at the top and
//	         back to 1 after a full turn. a colour wheel.
//	Through  any of the above seen through a matrix: the sheet is
//	         squashed or turned before the rule reads it, so rings
//	         become ellipses and bands tilt.
//
// A Field is a container for a rule, not an enforcer of a geometry. Any
// function of u and v is one; Func below lets you write one inline. So a shape
// none of the three describe is still a field. An egg is a radial whose radius
// depends on the direction from the centre, larger below than above.
//
// Ripples on its lower half are the same egg with a small wave added to t as
// the distance grows, only where v is past the centre. Both are a few lines
// with math.Atan2 and math.Sin, both are still position-to-t, and the ramp that
// colours them is the same ramp that coloured the sun.
//
// If a shape needs a new rule, write the rule;
//
// nothing here has to change.
//
// Time is not a coordinate of the sheet. An animation is one ramp and a
// field per frame, where the frame's time was folded into the field
// when it was built: a centre that moves, a wave that has advanced.

// Field is a rule from a position on the surface, u and v in 0..1, to a
// t in 0..1.
type Field interface {
	T(u, v float64) float64
}

// Func is a field written as a plain function, for a rule that does not
// need a type of its own.
type Func func(u, v float64) float64

// T is the function.
func (f Func) T(u, v float64) float64 { return clamp(f(u, v)) }

// Point is a position on the surface, 0..1 each way, v downward like a
// screen.
type Point struct{ U, V float64 }

// Linear is the number growing along the axis from A to B: 0 at A, 1 at
// B, held beyond either end. The maths is a projection, a dot product
// with the axis divided by its length squared, which is the same thing
// as asking "how far along the arrow is the shadow of this point".
type Linear struct{ A, B Point }

// T is the position along the axis.
func (l Linear) T(u, v float64) float64 {
	dx, dy := l.B.U-l.A.U, l.B.V-l.A.V
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return 0
	}
	return clamp(((u-l.A.U)*dx + (v-l.A.V)*dy) / l2)
}

// Radial is the distance from a centre divided by a radius: 0 at the centre, 1
// at the radius and beyond. Rings. A sun is this with the centre low on the
// sheet. The sheet is square; if the cells it is drawn on are not, the caller
// corrects the position first or the rings are ovals.
//
// For rings that are not circles on purpose, see the note at the top of the
// file.
type Radial struct {
	Centre Point
	Radius float64
}

// T is the distance from the centre, in radii.
func (r Radial) T(u, v float64) float64 {
	if r.Radius <= 0 {
		return 0
	}
	return clamp(math.Hypot(u-r.Centre.U, v-r.Centre.V) / r.Radius)
}

// Conic is the angle around a centre as a fraction of a full turn: 0
// straight up, increasing clockwise, back to 1 at the top. A colour
// wheel is this in front of a ramp whose ends are the same colour.
type Conic struct{ Centre Point }

// T is the fraction of a turn.
func (c Conic) T(u, v float64) float64 {
	a := math.Atan2(u-c.Centre.U, -(v - c.Centre.V))
	if a < 0 {
		a += 2 * math.Pi
	}
	return a / (2 * math.Pi)
}

// Through is a field seen through a matrix. The position is moved before
// the inner field reads it, about the centre of the sheet, so a circle
// becomes an ellipse, an axis turns, a shape leans. M is two rows of two:
//
// the new u is M[0][0]*x + M[0][1]*y and the new v is M[1][0]*x +
// M[1][1]*y, for x and y measured from the centre. The identity matrix,
// ones on the diagonal, changes nothing; halving M[0][0] squashes the
// sheet to half its width before the rule sees it.
type Through struct {
	M     [2][2]float64
	Field Field
}

// T is the inner field at the transformed position.
func (t Through) T(u, v float64) float64 {
	x, y := u-0.5, v-0.5
	return t.Field.T(
		t.M[0][0]*x+t.M[0][1]*y+0.5,
		t.M[1][0]*x+t.M[1][1]*y+0.5,
	)
}

// clamp holds a value to 0..1.
func clamp(t float64) float64 { return math.Max(0, math.Min(1, t)) }
