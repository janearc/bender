package mat

import (
	"math"
	"testing"
)

// A matrix times its inverse is the identity, to the float.
func TestInverse(t *testing.T) {
	m := M{{2, 1, 0}, {1, 3, 1}, {0, 1, 4}}
	inv := m.Inverse()
	for i := 0; i < 3; i++ {
		a, b, c := inv.Apply(m[0][i], m[1][i], m[2][i])
		got := [3]float64{a, b, c}
		for j := 0; j < 3; j++ {
			want := 0.0
			if i == j {
				want = 1
			}
			if math.Abs(got[j]-want) > 1e-12 {
				t.Errorf("inverse * m column %d = %v", i, got)
			}
		}
	}
}
