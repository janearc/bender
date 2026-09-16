# angry

H3 cells from kingfisher, drawn in a terminal, driven by the mouse. Drag to
pan, scroll to zoom, click a cell to name it. Flat map, globe, or the
horizon from low orbit.

It was called uv-h3-tui-client until 2026-09-11.

A proof of concept. `DESIGN.md` is the spec, amended with what turned out to
be wrong; `reference/` holds the Python client it is ported from.

## Requirements

A terminal with SGR-pixel mouse reporting: Ghostty, kitty, foot. tmux does not
forward it, so this says so and exits rather than pretending.

kingfisher reachable at `$KINGFISHER_URL`, default `http://kingfisher.test`.

Go 1.26 and a C compiler: `h3-go` is cgo, so `CGO_ENABLED=1` is required
and there is no trivial cross-compile.

## Run

    go build -o angry ./cmd/angry
    ./angry                                                   # the shelf, San Francisco Bay
    ./angry -region california -globe                         # a named window on the globe
    ./angry -bbox=-122.05,37.03,-121.97,37.07                 # any window
    ./angry -list                                             # what kingfisher has
    ./angry -dataset gibs-VIIRS_Black_Marble-r4               # one dataset, read whole
    ./angry -probe                                            # what the terminal answered
    ./angry -dataset usgs-64390f6bd34ee8d4ade0b21b -globe -frame
                                                              # one frame to stdout; any terminal, tmux included
    ./angry -dataset usgs-64390f6bd34ee8d4ade0b21b -globe -frame -span 360
                                                              # the whole sphere, the tile a dot on it

| flag | what it does |
|---|---|
| `-region name` | the window to ask the shelf for: `sfbay` (the default), `california`, `cascadia`, `us`, `world` |
| `-bbox w,s,e,n` | any window, overriding `-region` |
| `-dataset name` | one ingested dataset read whole from its file, unpriced; the shelf is not consulted |
| `-layer name` | the layer to draw; the first when empty |
| `-globe` | start on the globe |
| `-list` | list datasets and exit |
| `-frame` | print one frame fitted to the data and exit; no mouse, no alternate screen, works under tmux |
| `-cols`, `-rows` | the frame's size for `-frame`; the terminal's when zero |
| `-span deg` | the view's width for `-frame`; fitted to the data when zero, 360 is the world |
| `-look lon,lat` | what to look at, put at the middle of the view; fitted to the data when empty |
| `-repl` | demo mode: read commands on stdin, write a frame per command, below |
| `-fps n` | frames a second while the repl flies; 12 when zero |
| `-live` | with `-repl`, redraw each frame in place, so a pane plays it like video |
| `-probe` | report the terminal's capabilities and exit |

## The budget

By default the client asks kingfisher's priced door, `/cells`, for a window
and a budget, never a resolution. The budget is the one exact figure a
terminal has: the map's pixel count, columns times rows times two, since
every cell past that averages into a bin that already holds one. The server
picks the finest resolution that fits the budget and its own memory, folds
on its side, and says what it served; the header shows the resolution and
cell count against the budget, and `clipped by memory` when the server had
to serve coarser than asked.

The fetched window is grown 1.6 times past the screen so a small pan is
free. A fetch happens when the view leaves what is held, or zooms in past
half the fetched span so finer cells are worth asking for. It runs on a drag
release, a zoom, an arrow key or a resize, not per pointer sample, and it
blocks the loop for the request. It is never sent more often than every
two seconds: a move inside that gap draws from what is held, the status
line says a fetch is waiting, and the fetch follows when the gap is up. The colour ramp is pinned to everything
fetched, so the same ground keeps its colour as the view moves.

A refusal is shown with the server's numbers. Kingfisher refuses every fold
with 503 when its heap sits too close to its limit for any fold to fit, and
the fix for that is a restart or a larger limit on the server, not a smaller
ask. `-dataset` reads one dataset's file whole and does not go through the
budget; it is there for looking at one dataset, not for browsing.

## Driving it

| input | what it does |
|---|---|
| drag with the left button | pan: a translation on the flat map, a rotation on the globe |
| wheel | zoom, anchored under the pointer, so the cell you point at stays put |
| click | name the cell: H3 index, resolution, value, centre |
| hover | light the cell under the pointer and name it in the status line |
| `g` | swap flat map and globe, facing the same place at the same scale |
| `h` | the horizon from low orbit: up flies forward, left and right turn, `+` and `-` change height |
| `b` | toggle the background: unsurveyed planet in a dark tint with a graticule, so the globe shows against space |
| `l` | next layer |
| `r` | reset the view to the data |
| arrows, `+`, `-` | pan and zoom from the keyboard |
| `?` | help |
| `q` | quit |

While dragging, frames are drawn with one sample per pixel and the queued
pointer samples are taken together before each frame, so the map follows
the pointer rather than a backlog; the full supersampled frame is drawn when
the button is released.

The H3 resolution follows the zoom: the finest resolution with at least four
pixels per hex, clamped to the dataset's own. The header shows the resolution
drawn against the dataset's ceiling. A blank is unsurveyed ground; the colour
ramp is data only.

## Demo mode

`-repl` reads one command per line on stdin and writes one frame per
command to stdout, in exactly the format `-frame` writes, so a console pane
can aim the view while something else records it. A bad command goes to
stderr and the last good frame stays.

It needs `-dataset`, and refuses without one. The dataset is read whole,
once, before the first frame; after that nothing is fetched, so a take can
run as fast as frames can be drawn -- about 12 ms each at 126 by 20 --
without asking kingfisher anything. The shelf fetches as the view moves,
and this client has knocked kingfisher over before by doing that too fast.

    printf 'look 10 48\nhorizon\nspan 10\n' | ./angry -dataset gibs-VIIRS_Black_Marble-r4 -repl -cols 126 -rows 20 -palette night -floor 30

To watch a take rather than decode it, add `-live` and give it something to
do over time:

    printf 'look 10 40\nhorizon\nspan 10\nfly 10\n' | ./angry -dataset gibs-VIIRS_Black_Marble-r4 -repl -live -cols 126 -rows 20 -palette night -floor 30

Without `-live` the frames are the same stream `-frame` writes, one after
another, which is what a compositor reads.

| command | what it does |
|---|---|
| `look lon lat` | put a coordinate at the middle of the view |
| `span deg` | how wide the view is; on the horizon, lower means closer to the ground |
| `fly secs [rate]` | move forward for that long, a frame each tick; `rate` is a fraction of the view a second, 0.1 by default |
| `spin secs [rate]` | turn for that long, the same way; a negative `rate` turns the other way |
| `flat`, `globe`, `horizon` | switch projection, keeping the aim |
| `palette name` | `heat`, or `night` for lights on a dark planet: log scale, top at the 99th percentile |
| `floor v`, `floor off` | draw values at or below `v` as bare ground; 30 suits Black Marble at night |
| `layer name` | draw another layer |
| `dataset name` | load another dataset and fit the view to it |
| `quit` | stop |

What the two motions mean depends on the projection:

| | `fly` | `spin` |
|---|---|---|
| horizon | forward along the heading | turn the heading |
| globe | tilt toward a pole, stopping at 89 degrees | turn the planet under the camera |
| flat | north, stopping at 85 degrees | east |

So a globe that should turn in a take wants `spin`; `fly` on the globe
climbs toward the pole and then stops.

The horizon is the view from low orbit: a camera above the ground looking
at the limb, which comes out as an arc across the top of the frame. Rays
that miss the planet are space and are left empty, so whatever is
composited behind the frame shows through as sky. Just outside the limb,
on the globe and the horizon, a band of air is drawn from deep blue to
cyan; it is thicker than the real atmosphere, because showing off.

## Tests

    go test ./...

The maths in `textmap` is tested. `cmd` has one test that drives the program
through a fake console with pixel-mouse bytes, and one that renders the real
Black Marble dataset and times it, skipped when kingfisher is not reachable.
