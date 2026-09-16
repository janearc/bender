package field

import (
	"math"
	"testing"
)

// about is equality to within floating error.
func about(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Linear is 0 at A, 1 at B, half way in the middle, and held beyond.
func TestLinear(t *testing.T) {
	l := Linear{Point{0, 0}, Point{1, 0}}
	if !about(l.T(0, 0.3), 0) || !about(l.T(1, 0.7), 1) ||
		!about(l.T(0.5, 0), 0.5) {
		t.Errorf(
			"along the axis: %v %v %v",
			l.T(0, 0.3),
			l.T(1, 0.7),
			l.T(0.5, 0),
		)
	}
	if !about(l.T(-1, 0), 0) || !about(l.T(2, 0), 1) {
		t.Errorf("beyond the ends should hold")
	}
	if !about((Linear{Point{0.5, 0.5}, Point{0.5, 0.5}}).T(0.2, 0.9), 0) {
		t.Errorf("a zero axis is 0 everywhere")
	}
}

// Radial is 0 at the centre, 1 at the radius, and held beyond.
func TestRadial(t *testing.T) {
	r := Radial{Point{0.5, 0.5}, 0.25}
	if !about(r.T(0.5, 0.5), 0) || !about(r.T(0.75, 0.5), 1) ||
		!about(r.T(0.625, 0.5), 0.5) {
		t.Errorf(
			"radial: %v %v %v",
			r.T(0.5, 0.5),
			r.T(0.75, 0.5),
			r.T(0.625, 0.5),
		)
	}
	if !about(r.T(0, 0), 1) {
		t.Errorf("beyond the radius should hold at 1")
	}
}

// Conic is 0 straight up, a quarter to the right, half straight down.
func TestConic(t *testing.T) {
	c := Conic{Point{0.5, 0.5}}
	if !about(c.T(0.5, 0), 0) || !about(c.T(1, 0.5), 0.25) ||
		!about(c.T(0.5, 1), 0.5) ||
		!about(c.T(0, 0.5), 0.75) {
		t.Errorf(
			"conic: up %v right %v down %v left %v",
			c.T(0.5, 0),
			c.T(1, 0.5),
			c.T(0.5, 1),
			c.T(0, 0.5),
		)
	}
}

// Through a matrix that halves u, a circle's t at the rim comes out as
// half: the same field, seen squashed.
func TestThrough(t *testing.T) {
	r := Radial{Point{0.5, 0.5}, 0.5}
	squash := Through{[2][2]float64{{0.5, 0}, {0, 1}}, r}
	if !about(squash.T(1, 0.5), 0.5) || !about(squash.T(0.5, 1), 1) {
		t.Errorf("through: %v %v", squash.T(1, 0.5), squash.T(0.5, 1))
	}
	identity := Through{[2][2]float64{{1, 0}, {0, 1}}, r}
	if !about(identity.T(0.9, 0.3), r.T(0.9, 0.3)) {
		t.Errorf("identity changed the field")
	}
}

// Func is a field written inline: an egg, a radial whose radius is
// larger below the centre than above, is a few lines and needs no type.
// At the same distance, a point below the centre gets a smaller t than a
// point above, because the egg is bigger there.
func TestFunc(t *testing.T) {
	egg := Func(func(u, v float64) float64 {
		dx, dy := u-0.5, v-0.5
		radius := 0.3
		if dy > 0 {
			radius = 0.45
		}
		return math.Hypot(dx, dy) / radius
	})
	above, below := egg.T(0.5, 0.3), egg.T(0.5, 0.7)
	if !(below < above) {
		t.Errorf("below the centre the egg is bigger, "+
			"so t should be smaller: above %v below %v",
			above, below)
	}
	if egg.T(0.5, 0.5) != 0 || egg.T(0, 0) != 1 {
		t.Errorf(
			"centre and far corner: %v %v",
			egg.T(0.5, 0.5),
			egg.T(0, 0),
		)
	}
}
