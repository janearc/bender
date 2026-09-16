package swatch

import "testing"

// A swatch gives back exactly what it was given; storage is not arithmetic.
func TestXYZRoundTrip(t *testing.T) {
	s := FromXYZ(0.25, 0.5, 0.75)
	if x, y, z := s.XYZ(); x != 0.25 || y != 0.5 || z != 0.75 {
		t.Errorf("got %v %v %v", x, y, z)
	}
}

// Y is how much light: white is 1, black is 0.
func TestLuminanceEnds(t *testing.T) {
	if _, y, _ := White.XYZ(); y != 1 {
		t.Errorf("white Y = %v", y)
	}
	if _, y, _ := Black.XYZ(); y != 0 {
		t.Errorf("black Y = %v", y)
	}
}
