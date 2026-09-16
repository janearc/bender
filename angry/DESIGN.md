# angry -- spec

angry was called uv-h3-tui-client until 2026-09-11.

A proof of concept: H3 cells from kingfisher, drawn in a terminal, driven by
the mouse. Click, drag, scroll.

Module path: `github.com/janearc/angry`.

## What this is for

Two things, and neither is a product.

It answers whether a map is pleasant to operate with a pointer in a terminal.
Nobody here has tried it. The Python client is keyboard-driven and redraws on
a keypress; grabbing the earth and moving it is a different feel and the only
way to know is to hold it.

And it is the clearest demonstration available of what ultraviolet's mouse can
do. DEC 1016 reports which pixel inside a character was clicked, which is
invisible in a form and obvious on a globe.

## What this is NOT

Not daffy. Daffy has a 600-line spec, six build steps and a definition of done
with twenty-two items. This has none of that on purpose. Build it quickly,
find out, and stop. If it turns out to be worth keeping, it gets specified
then.

No save format, no export, no configuration file, no theming system, no
plugin surface, no test coverage target. Tests where they buy confidence in
the maths and nowhere else.

## Follow kingfisher's lead

Read `reference/textmap.py` before writing anything. It is the same pipeline
in Python, already factored, and the factoring is the part to keep:

| object | what it holds |
|---|---|
| `CellSource` | cells for one dataset, re-aggregated to any H3 resolution |
| `Viewport` | the flat window: centre, span, and the grid it fills |
| `Globe` | the same window wrapped round a ball, orthographic |
| `TextMap` | points plus a viewport, out comes a picture |

The design decision worth copying exactly is that `Viewport` and `Globe` are
interchangeable, behind `pixels()`, `bounds()`, `project()`, `unproject()`,
`pan()` and `zoom()`. `TextMap` never learns which one it is holding. In Go
that is an interface with those six methods and two implementations, and it is
what makes a globe cheap rather than a second renderer.

The reason it works, in the original's words: the fill already asks every
pixel which cell contains it, so a projection only has to answer "what
coordinate is this pixel" and "is this pixel even on the planet".

Two rules from that file to carry over rather than rediscover:

- **A space is never a value.** Blank means absent. The density ramp is data
  only. Drawing the bottom of a scale the same as unsurveyed ground makes the
  lowest valley and the ocean beside it identical, and the colour path has
  always kept them apart.
- **H3 tops out at 15, but the real ceiling is per-source.** Nothing renders
  finer than the data was folded. Clamp to the source's `native_res`.

## The data

kingfisher is up and reliable. It is JSON over HTTP, addressed by name:

    KINGFISHER_URL, default http://kingfisher.test

    GET /ingested/                          -> {"directories": [...]}
    GET /ingested/<dataset>/manifest.json   -> the dataset's manifest
    GET /ingested/<dataset>/cells.json      -> {"cells": ...}

Never an address and a port. Exponential backoff with jitter on every call,
as everywhere else.

## What is new here: the mouse

This is the whole point of the exercise.

- **Drag to pan.** Grab the map and move it. On the globe that is rotation,
  on the flat viewport a translation, and both are already `pan()`.
- **Scroll to zoom**, anchored under the pointer, so the cell you are pointing
  at stays under the pointer. Daffy's `SetZoom` does this and the property
  worth asserting is that the cell under the anchor does not move.
- **Click a cell** and the status line names it: its H3 index, its value, its
  resolution.
- **Hover** highlights the cell under the pointer, continuously.
- Sub-cell precision is what makes hover and click land on the cell you meant
  rather than the character you meant. Say so on screen somewhere, since it is
  the thing being demonstrated.

## Ground already measured, do not re-derive

- **Build on ultraviolet, not Bubble Tea.** Bubble Tea has no way to select
  the pixel mouse encoding; ultraviolet has `MouseEncodingSGRPixel`,
  `SetMouseEncoding` and `MousePixelToCell`.
- **Half blocks.** A cell here is 19 by 42 pixels, so a half-block subpixel is
  19 by 21, within ten per cent of square. No aspect correction, and the
  Python file independently reached the same conclusion.
- **H3 resolution follows zoom.** At a full globe on a normal terminal a
  hemisphere covers roughly 8,800 pixels, about 29,000 km² each, so res 2 puts
  about three pixels in a hex. Aim for 4 to 9 pixels per hex and step one
  resolution per 2.65x linear zoom. Verified against `h3-go` v4.5.0.
- **Supersample cells, not colours,** at the limb: sample 2x2 lat/lons per
  subpixel, blend the cells' colours. Cheap, and it fixes the staircase.
- **The three terminal rules** in `~/mesh/external/daffy/DESIGN.md` apply
  unchanged: place an image after the first flush, re-place after any erase or
  resize, and never paint the unpainted half of a cell. Only the third one
  matters if there is no underlay.

## Definition of done

1. It opens, fetches a dataset from kingfisher, and draws it.
2. Drag pans. Scroll zooms, anchored under the pointer. Both on the flat map
   and on the globe.
3. One key swaps flat and globe, and `TextMap` does not know which it has.
4. Hover highlights the cell under the pointer; click names it in a status
   line.
5. It says which terminal capabilities it got, and refuses honestly under
   tmux, where there is no pixel mouse.
6. `README.md` says how to run it. `DESIGN.md` is this file, amended with
   whatever turned out to be wrong.

That is the whole of it. If it is fun to use, we will know. If it is not, we
will also know, and that was the point.

## Acceptance

Branch `poc`, small commits, reviewed by a session other than the author
before it merges. Given the scope, one review at the end is enough; this does
not need a step per feature.

## Amendments, 2026-09-07, from building it

What the port found that this spec, or the Python it follows, had wrong or
had not said.

- **The globe's scale.** `Globe.span_x` in the Python is `360 * w / (2 *
  radius)`. The disc's diameter is a hemisphere, 180 degrees of great circle,
  so the degrees across a window of width `w` are `180 * w / (2 * radius)`.
  In the Python it fed only a status line; here it feeds the resolution
  choice, and the 360 picked a resolution one step too coarse.
- **"Aim for 4 to 9 pixels per hex" is not a band that can be hit.** Adjacent
  resolutions differ sevenfold in area, so any rule that steps by whole
  resolutions sees pixels per hex swing by seven. The rule used is the finest
  resolution with at least four pixels per hex, which puts the value between
  four and twenty-eight and steps once per 2.65x of linear zoom. A full
  globe on 200 columns comes out at res 1, not 2.
- **Drag and anchored zoom on the globe converge, they are not exact.** On
  the flat map, moving the centre by the difference between the grabbed
  coordinate and the one now under the pointer is exact in one step. On the
  globe it is a rotation, so the step is repeated until the error is under a
  millionth of a degree, twelve rounds at most. The tests assert the anchor
  holds to a twentieth of a degree on both.
- **Sampling a mixed-resolution source uses the raw cells**, walking coarser
  on a miss, as the Python sampler does, so a coarse cell keeps its own value
  beside finer ones. A source folded at one resolution uses the rollup table
  at the resolution asked for, one lookup per sample, which is what makes a
  frame fast.
- **The globe's bounds when zoomed in past the frame** come from the frame's
  edge, not the disc's rim. The Python samples the rim always, which reports
  a hemisphere however far in you are. Nothing here fetches by bounds yet, so
  it is a correctness note rather than a bug that showed.
- **Frame cost, measured.** Black Marble, 287k cells folded at res 4,
  drawn as a world view at res 1 on 200 by 100 pixels: 22.6ms a frame flat
  and 17.0ms on the globe after a first frame of about 90ms, which pays for
  the rollup table. A drag repaints on every sample and keeps up. That
  20ms is the ceiling to watch: comfortable for a map that redraws on
  interaction, above a 60fps budget, and the first place to look if a drag
  feels heavy in a real window.
- **The click-versus-drag rule.** A press and release with no movement in
  between names the cell; any movement makes it a drag and no click fires.
- **The three terminal rules held**, and the two ultraviolet defects found
  by daffy applied here too: the probe consumes the terminal's first size
  event so the screen is sized from the probe's measurement, and the program
  settles briefly before `Stop`. Colour is forced to true colour rather than
  read from the environment.
- **The globe's radius cap.** The Python stops the globe at twenty times
  the window, which is fine for a keyboard and useless for a lidar tile: a
  hundredth of a degree needs a radius in the millions of pixels to fill a
  window, and the orthographic maths is fine there. The cap is now only
  against overflow, and swapping projections carries the scale across, so
  `g` on a fitted tile shows the same tile on the globe.
- **Lidar, measured.** Alameda County, 1886 cells at res 12: a view fitted
  to the tile draws at res 12 in about 30ms a layer; a degree-wide view
  rolls it up to res 5 and it is still there. `water_share` is sparse by
  nature, carried only by cells with water.
- **Drag was slow in the first real run, and zoom was not.** Every pointer
  sample re-rendered a full frame, samples arrive faster than frames, and
  the event queue backed up, so the map answered a pointer that had moved
  on. Two changes: the loop takes everything already queued before painting
  once, and a frame drawn mid-drag samples once per pixel, about four times
  faster on the lidar tile, with the supersampled frame drawn on release.
- **A globe against black is invisible where nothing was surveyed.** The
  rule that a space is never a value stands: data still draws in the ramp
  and space is still blank. But a pixel on the planet with nothing surveyed
  is now marked and drawn in a dark tint that is not on the ramp, with a
  graticule whose spacing follows the zoom, so the disc reads as a disc. The
  `b` key turns it off.
- **The data section named the wrong door.** `/ingested/<dataset>/cells.json`
  is a file; kingfisher's contract for a consumer is `/cells?bbox&budget`,
  the priced door the bench uses, where the budget is the frame's pixel
  count and the server folds to fit it and its own memory. The client now
  asks that way by default, fetching on move the way the bench does, and
  `-dataset` remains for reading one dataset whole. The `Bounds()` method
  the spec asked for on the projection is what this needed it for.
- **The server can refuse everything.** After a day of asks kingfisher's
  heap sat at 128Mi of 256Mi with a 96Mi input reserve, which by its own
  formula leaves zero rows, and it refused every fold with 503. The client
  shows the numbers and keeps the last cells. The fix is a restart or a
  larger limit on the server; the commit that added the ceiling says so.
  While refusing every fold, `/health` reported healthy, ready and ok at
  42 hours of uptime, heap at half the limit. A service that cannot do its
  primary work is degraded, and nothing watching `/health` would know; and
  whether that heap is accumulation or an unmeasured working set is a
  question the "restart" answer skips. Both are kingfisher's to take up,
  recorded here because this client is where they showed.
- **Ghostty and Unicode core mode.** ultraviolet asks the terminal for DEC
  2027 at start and enables it on a yes; Ghostty 1.3.1 says yes and then
  never shows the frame, a black alternate screen that still answers `q`.
  Found in daffy the same evening, confirmed by flipping it. The width
  method is pinned before `Start` so the mode is never enabled; the glyphs
  here are all single width.
