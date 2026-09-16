// angry draws H3 cells from kingfisher in a terminal and lets
// the mouse drive them: drag to pan, scroll to zoom, click a cell to name
// it, flat map or globe.
//
//	angry [-dataset name] [-layer name] [-globe] [-probe]
//
// It needs SGR-pixel mouse reporting, which tmux does not forward; it says
// so and exits rather than pretending.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/janearc/angry-dist/kingfisher"
	"github.com/janearc/angry-dist/textmap"
)

type options struct {
	region  string
	bbox    string
	dataset string
	layer   string
	globe   bool
	probe   bool
	list    bool
	frame   bool
	cols    int
	rows    int
	span    float64
	look    string
	repl    bool
	palette string
	floor   string
	fps     int
	live    bool
}

// main reads the flags, opens the terminal, and returns what the view exited
// with. every flag is a window on the same question: which cells to ask for,
// which layer to draw, and whether to start on the globe.
func main() {
	var opts options
	flag.StringVar(
		&opts.region,
		"region",
		"sfbay",
		"a named window to ask the shelf for: sfbay, california, "+
			"cascadia, us, world",
	)
	flag.StringVar(
		&opts.bbox,
		"bbox",
		"",
		"w,s,e,n in degrees; overrides -region",
	)
	flag.StringVar(
		&opts.dataset,
		"dataset",
		"",
		"one ingested dataset by name, read whole from its"+
			" file rather than priced through the shelf",
	)
	flag.StringVar(
		&opts.layer,
		"layer",
		"",
		"layer to draw; the first when empty",
	)
	flag.BoolVar(
		&opts.globe,
		"globe",
		false,
		"start on the globe rather than the flat map",
	)
	flag.BoolVar(
		&opts.probe,
		"probe",
		false,
		"report what the terminal can do and exit",
	)
	flag.BoolVar(
		&opts.list,
		"list",
		false,
		"list kingfisher's datasets and exit",
	)
	flag.BoolVar(
		&opts.frame,
		"frame",
		false,
		"print one frame to stdout and exit; works in any terminal, "+
			"tmux included",
	)
	flag.IntVar(
		&opts.cols,
		"cols",
		0,
		"frame width in cells for -frame; the terminal's when zero",
	)
	flag.IntVar(
		&opts.rows,
		"rows",
		0,
		"frame height in cells for -frame; the terminal's when zero",
	)
	flag.Float64Var(
		&opts.span,
		"span",
		0,
		"width of the view in degrees for -frame; fitted to the data "+
			"when zero, 360 is the world",
	)
	flag.BoolVar(
		&opts.repl,
		"repl",
		false,
		"read commands on stdin and write a frame per command;"+
			" globe, flat, horizon, look, span, fly, spin,"+
			" palette, floor, layer, dataset, quit",
	)
	flag.IntVar(
		&opts.fps,
		"fps",
		0,
		"frames a second while -repl flies; 12 when zero",
	)
	flag.BoolVar(
		&opts.live,
		"live",
		false,
		"with -repl, redraw each frame in place so a pane plays it "+
			"like video",
	)
	flag.StringVar(
		&opts.palette,
		"palette",
		"heat",
		"colours and scale: heat, or night for lights on a dark planet",
	)
	flag.StringVar(
		&opts.floor,
		"floor",
		"",
		"a value at or below which a cell is drawn as bare ground "+
			"rather than data",
	)
	flag.StringVar(
		&opts.look,
		"look",
		"",
		"lon,lat to look at, put at the middle of the view; fitted to "+
			"the data when empty",
	)
	flag.Usage = func() {
		fmt.Fprint(
			os.Stderr,
			"usage: angry [-region name | -bbox w,s,e,n |"+
				" -dataset name] [-layer name] [-globe]"+
				" [-frame] [-repl] [-list] [-probe]\n\n",
		)
		flag.PrintDefaults()
		fmt.Fprintf(
			os.Stderr,
			"\nkingfisher at $KINGFISHER_URL, now %s\n",
			kingfisher.URL(),
		)
	}
	flag.Parse()
	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "angry:", err)
		os.Exit(1)
	}
}

// run fetches the data first, since a terminal with nothing to draw is
// worse than a pause on the command line, then hands over to the app.
func run(opts options) error {
	if opts.probe {
		return probe(os.Stdout)
	}
	// Parsed before any fetch, so a bad -look costs nothing.
	view, err := parseLook(opts.look)
	if err != nil {
		return err
	}
	st, err := parseStyle(opts.palette, opts.floor)
	if opts.palette != "" && opts.palette != "heat" {
		st = st.named()
	}
	if err != nil {
		return err
	}
	// A take reads its data once, before the first frame, so it can run at
	// any frame rate without asking kingfisher anything mid-take.
	//
	// The shelf fetches as the view moves, which is the thing that has
	// knocked kingfisher over before, so the repl refuses it rather than
	// quietly starting the interactive viewer instead.
	if opts.repl && opts.dataset == "" {
		return fmt.Errorf(
			"-repl needs -dataset: a take reads one" +
				" dataset whole, once, and never fetches" +
				" mid-take",
		)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	kf := kingfisher.New(kingfisher.URL())
	if opts.list {
		names, err := kf.Datasets(ctx)
		if err != nil {
			return err
		}
		for _, n := range names {
			fmt.Println(n)
		}
		return nil
	}
	if opts.dataset == "" {
		return runShelf(ctx, kf, opts, view, st)
	}
	name := opts.dataset
	fmt.Fprintf(
		os.Stderr,
		"fetching %s whole from %s, unpriced\n",
		name,
		kf.Base,
	)
	m, err := kf.Manifest(ctx, name)
	if err != nil {
		return err
	}
	cells, err := kf.Cells(ctx, name)
	if err != nil {
		return err
	}
	src := textmap.NewCellSource(name, cells.Cells, m.LayerKinds())
	if src.Len() == 0 {
		return fmt.Errorf("%s: no valid cells", name)
	}
	layers := src.Layers()
	layer := opts.layer
	if layer == "" {
		layer = layers[0]
	}
	found := false
	for _, l := range layers {
		if l == layer {
			found = true
		}
	}
	if !found {
		return fmt.Errorf(
			"%s has no layer %q; it has %s",
			name,
			layer,
			strings.Join(layers, ", "),
		)
	}
	fmt.Fprintf(
		os.Stderr,
		"%d cells at res %d, layers %s\n",
		src.Len(),
		src.NativeRes(),
		strings.Join(layers, ", "),
	)
	if opts.repl {
		return runRepl(
			ctx,
			kf,
			os.Stdin,
			os.Stdout,
			os.Stderr,
			src,
			m,
			layer,
			opts,
			view,
			st,
		)
	}
	if opts.frame {
		return frameOnce(
			os.Stdout,
			src,
			m,
			layer,
			opts.globe,
			opts.cols,
			opts.rows,
			opts.span,
			view,
			st,
		)
	}
	// the door goes with it: a view opened on one dataset can still ask
	// kingfisher what else it has, which is what d does
	return runApp(start{src: src, m: m, layer: layer, globe: opts.globe,
		kf: kf, style: st})
}

// runShelf asks the priced door for a window sized to the screen. The
// quote comes first, so a window with nothing on it is a message on the
// command line rather than an empty screen.
func runShelf(
	ctx context.Context,
	kf *kingfisher.Client,
	opts options,
	view *aim,
	st style,
) error {
	window, ok := kingfisher.Regions[opts.region]
	if opts.bbox != "" {
		var b kingfisher.BBox
		_, err := fmt.Sscanf(
			opts.bbox, "%f,%f,%f,%f", &b[0], &b[1], &b[2], &b[3])
		if err != nil {
			return fmt.Errorf("-bbox wants w,s,e,n: %v", err)
		}
		window, ok = b, true
	}
	if !ok {
		return fmt.Errorf(
			"unknown region %q; known: sfbay, california, "+
				"cascadia, us, world",
			opts.region,
		)
	}
	cols, rows := opts.cols, opts.rows
	if cols <= 0 || rows <= 0 {
		cols, rows = 120, 40
	}
	budget := cols * rows * 2
	q, err := kf.Shelf(ctx, window, budget)
	if err != nil {
		return err
	}
	fmt.Fprintf(
		os.Stderr,
		"shelf: window %s holds %d datasets, %d cells at res %d; "+
			"budget %d serves res %d\n",
		window,
		q.Datasets,
		q.CellsNative,
		q.ResNative,
		budget,
		q.Res,
	)
	if q.Datasets == 0 {
		return fmt.Errorf("no shelved dataset overlaps that window")
	}
	if opts.frame {
		w, err := kf.Window(ctx, window.Grown(1.6), budget)
		if err != nil {
			return err
		}
		src := textmap.NewCellSource(
			"shelf",
			w.Cells,
			map[string]string{
				"point_density": "LAYER_KIND_EXTENSIVE",
			},
		)
		layer := opts.layer
		if layer == "" && len(src.Layers()) > 0 {
			layer = src.Layers()[0]
		}
		m := &kingfisher.Manifest{Title: "shelf"}
		m.Pipeline.BBox = window[:]
		return frameOnce(
			os.Stdout,
			src,
			m,
			layer,
			opts.globe,
			opts.cols,
			opts.rows,
			opts.span,
			view,
			st,
		)
	}
	return runApp(
		start{
			layer:  opts.layer,
			globe:  opts.globe,
			kf:     kf,
			window: window,
			style:  st,
		},
	)
}
