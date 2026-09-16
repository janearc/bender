// Package mat is the one piece of linear algebra the spaces share: a 3x3
// matrix, applied to a column of three, and inverted.
package mat

// M is a 3x3 matrix, rows first.
type M [3][3]float64

// Apply is the matrix times a column of three.
func (m M) Apply(a, b, c float64) (float64, float64, float64) {
	return m[0][0]*a + m[0][1]*b + m[0][2]*c,
		m[1][0]*a + m[1][1]*b + m[1][2]*c,
		m[2][0]*a + m[2][1]*b + m[2][2]*c
}

// Inverse is the matrix that undoes this one, by the adjugate over the
// determinant, which for a 3x3 is short enough to write out.
//
// Published inverses are rounded to ten places and do not quite undo their
// forward matrices; computing the inverse makes a round trip exact to the
// float, so the only numbers anyone types are the forward ones.
func (m M) Inverse() M {
	det := m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
	return M{
		{(m[1][1]*m[2][2] - m[1][2]*m[2][1]) / det,
			(m[0][2]*m[2][1] - m[0][1]*m[2][2]) / det,
			(m[0][1]*m[1][2] - m[0][2]*m[1][1]) / det},
		{(m[1][2]*m[2][0] - m[1][0]*m[2][2]) / det,
			(m[0][0]*m[2][2] - m[0][2]*m[2][0]) / det,
			(m[0][2]*m[1][0] - m[0][0]*m[1][2]) / det},
		{(m[1][0]*m[2][1] - m[1][1]*m[2][0]) / det,
			(m[0][1]*m[2][0] - m[0][0]*m[2][1]) / det,
			(m[0][0]*m[1][1] - m[0][1]*m[1][0]) / det},
	}
}

// Scale is the matrix with each column multiplied by the matching value.
func (m M) Scale(a, b, c float64) M {
	return M{
		{m[0][0] * a, m[0][1] * b, m[0][2] * c},
		{m[1][0] * a, m[1][1] * b, m[1][2] * c},
		{m[2][0] * a, m[2][1] * b, m[2][2] * c},
	}
}
