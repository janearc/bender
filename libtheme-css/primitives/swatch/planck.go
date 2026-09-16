package swatch

import "math"

// Planckian is the swatch of a black body at a temperature in kelvin: the
// colour a lamp means when it gives a colour temperature.
//
// The spectrum is Planck's law taken at every wavelength the observer has a row
// for, and Illuminant does the rest, so the locus is derived from the same
// tables as White and is never typed in. Below a few hundred kelvin there is no
// visible light to speak of and the result is black.
func Planckian(kelvin float64) Swatch {
	if kelvin <= 0 {
		return Black
	}
	const (
		h = 6.62607015e-34 // planck, J·s
		c = 2.99792458e8   // light, m/s
		k = 1.380649e-23   // boltzmann, J/K
	)
	spectrum := make(map[int]float64, len(observer))
	for nm := range observer {
		wl := float64(nm) * 1e-9
		spectrum[nm] = 1 / (math.Pow(wl,
			5) * (math.Exp(h*c/(wl*k*kelvin)) - 1))
	}
	return Illuminant(spectrum)
}
