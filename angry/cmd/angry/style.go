package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/janearc/angry-dist/textmap"
)

// style is how values look: which palette, and how they are laid onto it.
// It is chosen for showing a dataset off rather than for measuring it, so
// a preset can change the scale as well as the colours.
type style struct {
	palette textmap.Palette
	scale   textmap.Scale
	// chosen is whether a person named this palette. one that was named
	// is kept across a change of map; one that was inferred follows the
	// data.
	chosen bool
}

// defaultStyle is the one every view had before styles existed.
func defaultStyle() style { return style{palette: textmap.Heat} }

// withPalette applies a named palette, and with it the scale that makes that
// palette look right. night is lights on a dark planet: the scale goes
// logarithmic so faint lights get room, and tops out at the ninety-ninth
// percentile so a few city cores do not flatten the rest.
//
// The floor is left alone, because it belongs to the dataset, not to the
// palette. Named is the palette as an explicit choice, which a later change of
// map will not overrule.
func (s style) named() style { s.chosen = true; return s }

// withPalette returns the style wearing a named palette, and the scale it
// implies: night is logarithmic because city lights span four decades and a
// linear ramp shows one bright coast and nothing else.
func (s style) withPalette(name string) (style, error) {
	for _, p := range textmap.Palettes {
		if p.Name == name {
			s.palette = p
			s.scale.Log = name == "night"
			s.scale.Clip = 0
			if name == "night" {
				s.scale.Clip = 0.99
			}
			return s, nil
		}
	}
	names := make([]string, len(textmap.Palettes))
	for i, p := range textmap.Palettes {
		names[i] = p.Name
	}
	return s, fmt.Errorf(
		"no palette %q; there is %s",
		name,
		strings.Join(names, ", "),
	)
}

// withFloor sets the value at or below which a cell is ground rather than
// data, or clears it for "off" or an empty string.
func (s style) withFloor(v string) (style, error) {
	if v == "" || v == "off" {
		s.scale.Floor = nil
		return s, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return s, fmt.Errorf("floor wants a number or off, not %q", v)
	}
	s.scale.Floor = &f
	return s, nil
}

// parseStyle reads the two flags at the command line, so a bad one is an
// error before anything is fetched.
func parseStyle(palette, floor string) (style, error) {
	s, err := defaultStyle().withPalette(palette)
	if err != nil {
		return s, fmt.Errorf("-palette: %v", err)
	}
	if s, err = s.withFloor(floor); err != nil {
		return s, fmt.Errorf("-%v", err)
	}
	return s, nil
}

// note is what the frame's last line says about the style, so a picture
// that hides everything under a floor says so under itself.
func (s style) note() string {
	var parts []string
	if s.palette.Name != "" && s.palette.Name != textmap.Heat.Name {
		parts = append(parts, s.palette.Name)
	}
	if s.scale.Log {
		parts = append(parts, "log")
	}
	if s.scale.Clip > 0 {
		parts = append(
			parts,
			fmt.Sprintf("top at p%g", s.scale.Clip*100),
		)
	}
	if s.scale.Floor != nil {
		parts = append(
			parts,
			fmt.Sprintf(
				"floor %.3f drawn as ground",
				*s.scale.Floor,
			),
		)
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// styleFor is the look a map asks for. A palette belongs to the data rather
// than to the session: night lights are unreadable in heat, where the low end
// is blue and the scale is linear, and a cloud radius is unreadable in night,
// where everything above the floor is fire.
//
// So changing map changes palette, unless the person said which one they
// wanted, in which case theirs stands.
func styleFor(dataset, layer string, held style) style {
	if held.chosen {
		return held
	}
	name := strings.ToLower(dataset + " " + layer)
	dark := []string{
		"black_marble", "blackmarble", "night_lights", "nightlights",
		"daynightband", "radiance", "lights",
	}
	for _, w := range dark {
		if strings.Contains(name, w) {
			out, err := held.withPalette("night")
			if err == nil {
				return out
			}
		}
	}
	out, err := held.withPalette("heat")
	if err != nil {
		return held
	}
	return out
}
