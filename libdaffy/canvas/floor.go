package canvas

// CellRow is the cell row holding a pixel row. It is floor division, not
// Go's truncation, because pixel rows above the canvas are negative and
// reachable: the view may show space before the canvas.
func CellRow(py int) int { return floorDiv(py, 2) }

// floorDiv rounds toward minus infinity, so a pixel above the origin
// lands on the cell above it and never on the one below.
func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// floorMod is the remainder that goes with floorDiv: never negative.
func floorMod(a, b int) int {
	m := a % b
	if m != 0 && ((m < 0) != (b < 0)) {
		m += b
	}
	return m
}
