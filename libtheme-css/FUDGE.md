# fudge

every number in this library rests on a number somebody picked. this is the
ledger, oldest at the bottom of the stack, newest at the top, so you can see
what your shell's pink is actually standing on.

none of it is a secret and none of it is a scandal; it's just written down here
in one place instead of being implied by four decimal places.

## 1931: seventeen people

the cie fixed the three numbers every colour reduces to, x, y and z, from
experiments by wright (ten observers, 1928-29) and guild (seven, 1931), at two
english labs, matching lamp light in a dark room through a hole two degrees
wide.

that is what "visible" means in this library: their averages, in a table, at one
nanometre. no one has redone it in a way anyone adopted.
`primitives/swatch/data/ciexyz31_1.csv`.

## 1964: daylight

d65 is not a sky. it is the average of a few hundred measurements of noon
daylight over england, smoothed into a curve and declared to be 6500 kelvin. it
is actually 6504, because the constant that turns temperature into a spectrum
was revised after the curve was fixed and nobody moved the label.

it is the white every screen you own assumes, and this library derives it from
the table rather than typing 0.95047. `primitives/swatch/data/d65.csv`.

## 1990, 1996, 1999: phosphors

a television tube made red by hitting a red phosphor, and each phosphor's light
has a fixed place on the 1931 diagram. in april 1990 bt.709 wrote down a red, a
green and a blue for hdtv, to two decimals, as a compromise between the european
and american phosphor sets.

in 1996 hp and microsoft proposed the same three for every computer screen, on
the stated grounds that "most computer monitors are similar in their key color
characteristics - the phosphor chromaticities (primaries)"; in 1999 the iec made
it srgb. the tubes are gone.

every screen in the world is defined relative to their glow. `spaces/srgb`,
where only those three pairs and the curve are typed, and the matrix is derived.

## 2008: a typo

the srgb curve breaks at 0.04045. wcag 2.0 copied it as 0.03928 from an earlier
draft, and every revision since has kept it, because fixing the fourth decimal
of a number in fifteen years of accessibility audits is worse than being
slightly wrong forever. every auditor's tool uses the typo.

so does libreadme, on purpose, because its promise is agreeing with the
auditors. this library uses the srgb constant, because its promise is
correctness and nobody is being audited. both are right.

## 2020: one swede

cielab, the 1976 attempt at a space where distance matches an eye, turns blues
purple when you lighten them. björn ottosson fit twelve numbers to fix that,
checked them against published measurements of what people see, posted them on
his blog, and called the result oklab, as in okay.

the browsers adopted it unchanged. the just-noticeable difference in it is
"roughly 0.02", which is a measurement of people, averaged, and this library's
`ok.Eye`. his name is björn; he goes by ok online. `spaces/ok`, where the twelve
numbers are at the bottom of the file.

## 1999 again: the cube

a terminal that shows 256 colours uses xterm's cube: six levels per lamp, at 0,
95, 135, 175, 215 and 255, plus a grey ramp. those six were chosen by thomas
dickey to look about right on a tube in 1999, and every terminal since copied
them. when a program snaps a colour to "the nearest of 256", those are the 256.

## no date: the sixteen

the sixteen "ansi" colours a terminal names red, green, blue and so on have no
standard values at all. the escape codes are standardised; the colours are not.
every terminal's author picked their own, and yours are in a ghostty config you
wrote, so in this one place the fudge is yours and you can change it.

## today: one reader

on top of all of it sits a profile of one person's eyes, measured by that
person, rating pages by eye on the screen she actually reads on. it is
the only layer in this stack where the sample size is the person it is
for. that is libreadme, and it is the least fudged number here.

so when this library prints `oklch(74% 0.200 345.3)` for your pink, the number
means: relative to seventeen people in 1931, a daylight nobody saw, three
phosphors from a television, one swede's fit, and your own eyes. it is a very
good number. it is not the truth.

nothing here is, and the difference between this library and most is that it
says so.

## the reader's line

`css.Read` turns a colourway's numbered families into ramps, and a ramp here
mixes in oklab. a browser drawing a `linear-gradient` over the same stops mixes
in srgb unless told otherwise, so the ramp this library hands a lamp is not the
ramp the page shows between the same stops; it is the one the eye would prefer.

the stops are exact; the line between them is ours. the gradient rule itself is
not read: this is a container, not a css parser, and a file that wants its
gradient on a lamp promotes the stops to roles.

## the gamut's slack

`srgb.FromSwatch` calls a colour in gamut when every channel is within a
ten-thousandth of 0..1 in linear light, not within a millionth. a display's
darkest step is about three ten-thousandths linear, so a hair past the edge is a
colour it shows exactly as it shows the edge.

the reason it matters: a dark blue's red channel can sit a hair under zero
across a wide band of chroma, and at a millionth the fit walks it back by a
sixth to reach a point the eye cannot tell from where it was. the number is a
display's, not the truth.
