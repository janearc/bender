package swatch

import (
	_ "embed"
	"strconv"
	"strings"
)

// The 1931 observer and the daylight it is most often asked to look at,
// as the CIE published them, so that the numbers below are derived and
// not typed. See data/README.md for where the tables come from.

//go:embed data/ciexyz31_1.csv
var observerCSV string

//go:embed data/d65.csv
var d65CSV string

// table reads "wavelength, v1, v2, ..." lines into a map by wavelength.
func table(csv string) map[int][]float64 {
	out := map[int][]float64{}
	for _, line := range strings.Split(csv, "\n") {
		fields := strings.Split(strings.TrimSpace(line), ",")
		if len(fields) < 2 {
			continue
		}
		nm, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil {
			continue
		}
		row := make([]float64, 0, len(fields)-1)
		for _, f := range fields[1:] {
			v, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
			if err != nil {
				continue
			}
			row = append(row, v)
		}
		out[nm] = row
	}
	return out
}

// observer is the CIE 1931 2-degree standard observer: for each
// wavelength, how much of it counts toward X, Y and Z. It is the
// library's definition of "visible", and it is worth being exact about
// what kind of definition that is.
//
// Visibility is not a property of light. Light has a wavelength; whether anyone
// sees it is a property of the eye looking.
//
// This table is a model of one eye: an average of the matches made by seventeen
// adults with normal colour vision, in 1928-1931, looking at a small patch two
// degrees wide (the width of a thumbnail at arm's length, which lands on the
// fovea), at daylight brightness.
//
// Under those circumstances, and for that average person, wavelengths outside
// roughly 380 to 780 nanometres produce no response, and the table says so by
// tending to zero at its ends and stopping at 360 and 830.
//
// A different observer, a wider field, a dim room, or an eye that is not
// average would give a different table, and this library would accept it in the
// same shape. Every range the library applies downstream is this one,
// inherited, and it is stated once, here.
var observer = table(observerCSV)

// Illuminant is the swatch of a light given as a spectrum: power by wavelength
// in nanometres. Each wavelength's power is weighted by how much the observer
// counts it toward X, Y and Z, and the three weighted sums are taken over every
// wavelength the observer has a row for.
//
// Power at wavelengths the observer does not list contributes nothing, which is
// the visible range being applied, not a limit of the spectrum. The three sums
// are then scaled so that Y is exactly 1: the relative form, in which this
// light is, by definition, the white.
func Illuminant(spectrum map[int]float64) Swatch {
	var x, y, z float64
	for nm, power := range spectrum {
		if w, ok := observer[nm]; ok && len(w) == 3 {
			x += power * w[0]
			y += power * w[1]
			z += power * w[2]
		}
	}
	if y == 0 {
		return Black
	}
	return Swatch{x / y, 1, z / y}
}

// Monochrome is light of one wavelength at unit power, as the observer sees it:
// the observer's own row, as a swatch. Outside the table it is no light.
//
// The spectrum drawn from 380 to 780 through this is the visible range as the
// seventeen saw it, and most of it is outside what any screen can make, which
// every screen shows by clipping.
func Monochrome(nm int) Swatch {
	w, ok := observer[nm]
	if !ok || len(w) != 3 {
		return Black
	}
	return Swatch{w[0], w[1], w[2]}
}

// d65 is CIE standard illuminant D65: not a real sky but the average of noon
// daylight measured in the 1960s, written down as a spectrum, with a nominal
// colour temperature of 6500 kelvin.
//
// It is actually 6504: the spectrum was fixed first, then physicists revised a
// constant in the formula that turns temperature into a spectrum, and the
// number moved under it. Nobody redefined the white; the label is slightly
// wrong forever. sRGB and every screen you own assume this white.
func d65() map[int]float64 {
	out := map[int]float64{}
	for nm, row := range table(d65CSV) {
		if len(row) > 0 {
			out[nm] = row[0]
		}
	}
	return out
}

// White is D65 as seen by the observer above, scaled so Y is 1. It comes
// out as X 0.95047, Z 1.08883, the two numbers most colour libraries
// type in; here they are computed from the two published tables at
// start-up, so the derivation can be checked rather than the typing.
var White = Illuminant(d65())

// Black is no light at all.
var Black = Swatch{}
