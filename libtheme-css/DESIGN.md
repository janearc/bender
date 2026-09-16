# libtheme-css

one colour, complete, and the operations on it. a theme is a sheet of named
colours; this library reads such a sheet, converts each colour between the
spaces a screen, an eye and a lamp use, blends between colours where the eye
would, and writes the sheet back out. it owns colour and nothing else.

## requirements

- the same colour is the same colour in a terminal, an editor, a web page
  and a lamp, without a person matching hex codes by eye.
- a colour is stored once, in a form every device space is defined against,
  so a device can be added without touching a stored value.
- distance and blending are measured where the eye agrees, not where the
  lamps do.
- every number in the library derives from a published table or standard,
  and the ones that cannot are listed in one place (FUDGE.md).
- a sheet is css custom properties, so any tool that reads css can read it.
- the documentation runs: every page paints its own examples through the
  library, so the picture, the sheet and the code cannot drift apart.

## the shape

    primitives/swatch      one colour as cie xyz (1931); the observer and
                           the d65 tables it is derived from, in data/
    primitives/functions   a ramp: stops and a mixer, t to a swatch
    spaces/ok              oklab and oklch; distance, tolerance, fit, reach
    spaces/srgb            the screen's lamps: hex, hsl, hsv, bytes
    css                    the container: a sheet of named swatches, read
                           and written as custom properties
    dialects/hue           how a hue lamp names a colour, and back
    dialects/grafana       five shades of a hue, for a tool that takes hex
    cmd/libtheme           the console end: show, ramp, read, known
    cmd/visualdocs         the documentation, shown, one page per idea
    internal/mat           the matrix arithmetic the spaces share

each layer imports only what is below it: functions import the swatch,
spaces import both, css imports the spaces, dialects import css.

## decisions

- store xyz, work in oklab.
- the swatch's fields are unexported; nothing averages xyz by accident.
- derive, do not type: the white is integrated from the tables, srgb's
  matrix from the standard's chromaticities and that white.
- the only typed numbers are the two cie tables, srgb's four pairs and
  its curve, and oklab's fit.
- one tolerance for arithmetic noise, one for what an eye can see; every
  equality in the library uses one of the two.
- the mixer is a parameter of a ramp, since spaces disagree about what a
  straight line is and css lets a sheet say which.
- no form: no fields, shapes or animation. a ramp answers what colour t
  is and never asks where t came from.
- a hue on a grey is `none`, as css spells it.
- `make visualdocs` and `make visualtest` are the acceptance tests.

## what it is not

- not a css engine. it reads custom properties whose values are colours
  and refuses everything else, out loud.
- not a renderer. it draws nothing.
- not a device. a lamp's brightness, a screen's gamut and a reader's eye
  are facts about the device and the reader, and belong to their own
  libraries: the dialects, and libreadme.

## see also

README.md for the first screen, FUDGE.md for where every number comes
from, and the package comments, which `go doc` prints.

jane michelle arc, 2026.
