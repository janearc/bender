// Package kingfisher reads datasets from the librarian: JSON over HTTP,
// addressed by name, never an address and a port. Every call backs off
// exponentially with jitter, as everywhere else in the estate.
package kingfisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultURL is where kingfisher is when nothing says otherwise.
const DefaultURL = "http://kingfisher.test"

// URL is kingfisher's base, from the environment or the default.
func URL() string {
	if u := os.Getenv("KINGFISHER_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return DefaultURL
}

// Client talks to one kingfisher.
type Client struct {
	Base    string
	HTTP    *http.Client
	Retries int
	// Sleep is what backoff waits with; tests replace it.
	Sleep func(time.Duration)
}

// New makes a client for a base URL.
func New(base string) *Client {
	return &Client{
		Base:    strings.TrimRight(base, "/"),
		HTTP:    &http.Client{Timeout: 60 * time.Second},
		Retries: 5,
		Sleep:   time.Sleep,
	}
}

// Manifest is the part of a dataset's manifest this client reads.
type Manifest struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
	Layers      map[string]struct {
		Kind string `json:"kind"`
		Unit string `json:"unit"`
	} `json:"layers"`
	Pipeline struct {
		Res   int       `json:"res"`
		Cells int       `json:"cells"`
		BBox  []float64 `json:"bbox"`
	} `json:"pipeline"`
}

// LayerKinds is the layer to kind table the cell source wants.
func (m *Manifest) LayerKinds() map[string]string {
	out := map[string]string{}
	for name, l := range m.Layers {
		out[name] = l.Kind
	}
	if len(out) == 0 && m.Kind != "" {
		out["value"] = m.Kind
	}
	return out
}

// Cells is a dataset's cell table as served: the resolution and a map of
// id to either a number or a map of layer to number.
type Cells struct {
	Res   int            `json:"res"`
	Cells map[string]any `json:"cells"`
}

// Datasets lists what is ingested.
func (c *Client) Datasets(ctx context.Context) ([]string, error) {
	var listing struct {
		Directories []string `json:"directories"`
	}
	if err := c.get(ctx, "/ingested/", &listing); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(listing.Directories))
	for _, d := range listing.Directories {
		out = append(out, strings.TrimSuffix(d, "/"))
	}
	return out, nil
}

// Manifest fetches a dataset's manifest.
func (c *Client) Manifest(
	ctx context.Context,
	dataset string,
) (*Manifest, error) {
	var m Manifest
	at := "/ingested/" + dataset + "/manifest.json"
	if err := c.get(ctx, at, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Cells fetches a dataset's cells.
func (c *Client) Cells(ctx context.Context, dataset string) (*Cells, error) {
	var cells Cells
	at := "/ingested/" + dataset + "/cells.json"
	if err := c.get(ctx, at, &cells); err != nil {
		return nil, err
	}
	if cells.Cells == nil {
		return nil, fmt.Errorf("kingfisher: %s has no cells", dataset)
	}
	return &cells, nil
}

// get fetches JSON with exponential backoff and jitter. A 4xx is final; a
// 5xx or a transport error is retried.
func (c *Client) get(ctx context.Context, path string, into any) error {
	var last error
	wait := 200 * time.Millisecond
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			jitter := time.Duration(rand.Float64() * float64(wait))
			c.Sleep(wait + jitter)
			wait = min(wait*2, 10*time.Second)
		}
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			c.Base+path,
			nil,
		)
		if err != nil {
			return err
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			last = err
			if ctx.Err() != nil {
				return ctx.Err()
			}
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck // read to the end
		if resp.StatusCode == 503 || resp.StatusCode == 413 {
			// The priced door saying no, with numbers. Not retried:
			// a smaller ask or a restart is the fix, and both are
			// the caller's to decide.
			r := &Refused{Status: resp.StatusCode}
			if json.Unmarshal(body, r) == nil && r.Msg != "" {
				return r
			}
		}
		switch {
		case resp.StatusCode >= 500:
			last = fmt.Errorf(
				"kingfisher: %s: %s",
				path,
				resp.Status,
			)
			continue
		case resp.StatusCode >= 400:
			return fmt.Errorf(
				"kingfisher: %s: %s",
				path,
				resp.Status,
			)
		case err != nil:
			last = err
			continue
		}
		if err := json.Unmarshal(body, into); err != nil {
			return fmt.Errorf("kingfisher: %s: %w", path, err)
		}
		return nil
	}
	return fmt.Errorf(
		"kingfisher: gave up after %d tries: %w",
		c.Retries+1,
		last,
	)
}

// --- the priced door
// ------------------------------------------------------------ /cells is how a
// consumer is meant to ask: a window and a budget, never a resolution.
//
// The server picks the finest resolution that fits the budget and its own
// memory, folds on its side, and says what it served. The budget a terminal can
// state exactly: cols times rows times two, the pixel count of the frame,
// because every cell past that averages into a bin that already holds one.

// BBox is west, south, east, north in degrees.
type BBox [4]float64

// String is the query form.
func (b BBox) String() string {
	return fmt.Sprintf("%.6f,%.6f,%.6f,%.6f", b[0], b[1], b[2], b[3])
}

// Covers reports whether this box holds another.
func (b BBox) Covers(inner BBox) bool {
	return b[0] <= inner[0] && b[1] <= inner[1] && b[2] >= inner[2] &&
		b[3] >= inner[3]
}

// Grown is the box scaled about its centre, so a fetch is bigger than the
// screen and a small pan is not a request.
func (b BBox) Grown(factor float64) BBox {
	cx, cy := (b[0]+b[2])/2, (b[1]+b[3])/2
	hw, hh := (b[2]-b[0])/2*factor, (b[3]-b[1])/2*factor
	return BBox{cx - hw, cy - hh, cx + hw, cy + hh}
}

// Regions are the named windows the bench knows.
var Regions = map[string]BBox{
	"sfbay":      {-123.10, 36.90, -121.20, 38.50},
	"california": {-124.48, 32.53, -114.13, 42.01},
	"cascadia":   {-124.80, 41.99, -116.90, 49.00},
	"us":         {-125.00, 24.40, -66.90, 49.40},
	"world":      {-180.00, -85.00, 180.00, 85.00},
}

// Limits is what the server says a request may ask for and what fits now.
type Limits struct {
	MaxBudget        int     `json:"max_budget"`
	BytesPerRow      int     `json:"bytes_per_row"`
	MemoryLimitBytes int64   `json:"memory_limit_bytes"`
	HeapBytes        int64   `json:"heap_bytes"`
	RowsThatFitNow   int     `json:"rows_that_fit_now"`
	Headroom         float64 `json:"headroom"`
}

// Quote is /shelf's answer: what is held in a window and what a budget
// would be served at.
type Quote struct {
	Datasets      int  `json:"datasets"`
	CellsNative   int  `json:"cells_native"`
	ResNative     int  `json:"res_native"`
	Res           int  `json:"res"`
	CellsEstimate int  `json:"cells_estimate"`
	Budget        int  `json:"budget"`
	Folded        bool `json:"folded"`
	Shelf         []struct {
		ID     string    `json:"id"`
		Title  string    `json:"title"`
		Res    int       `json:"res"`
		Cells  int       `json:"cells"`
		BBox   []float64 `json:"bbox"`
		Layers []string  `json:"layers"`
	} `json:"shelf"`
}

// Window is /cells' answer: the cells of a window folded to a budget, and
// what the server did to fit them.
type Window struct {
	Res          int            `json:"res"`
	Datasets     int            `json:"datasets"`
	CellsRead    int            `json:"cells_read"`
	Points       int64          `json:"points"`
	Budget       int            `json:"budget"`
	BudgetServed int            `json:"budget_served"`
	ClippedBy    string         `json:"clipped_by"`
	BBox         []float64      `json:"bbox"`
	Cells        map[string]any `json:"cells"`
}

// Refused is the server saying no with its numbers: 503 when no fold fits
// in its memory, 413 when the budget is above the ceiling.
type Refused struct {
	Status int
	Msg    string `json:"error"`
	Heap   int64  `json:"heap_bytes"`
	Limit  int64  `json:"memory_limit_bytes"`
	Limits Limits `json:"limits"`
}

// Error says what the librarian refused and why. a 503 carries its heap and
// what would fit now, because "refused" alone leaves the reader guessing
// whether to wait, ask for less, or restart it.
func (r *Refused) Error() string {
	if r.Status == 503 {
		return fmt.Sprintf(
			"kingfisher refused: %s (heap %dMi of %dMi, rows that "+
				"fit now %d)",
			r.Msg,
			r.Heap>>20,
			r.Limit>>20,
			r.Limits.RowsThatFitNow,
		)
	}
	return fmt.Sprintf("kingfisher refused (%d): %s", r.Status, r.Msg)
}

// Shelf asks what a window holds and what a budget buys.
func (c *Client) Shelf(
	ctx context.Context,
	bbox BBox,
	budget int,
) (*Quote, error) {
	var q Quote
	at := fmt.Sprintf("/shelf?bbox=%s&budget=%d", bbox, budget)
	if err := c.get(ctx, at, &q); err != nil {
		return nil, err
	}
	return &q, nil
}

// Window fetches a window's cells folded to a budget. A refusal comes back
// as *Refused with the server's numbers rather than as a bare status.
func (c *Client) Window(
	ctx context.Context,
	bbox BBox,
	budget int,
) (*Window, error) {
	var w Window
	err := c.get(
		ctx,
		fmt.Sprintf("/cells?bbox=%s&budget=%d", bbox, budget),
		&w,
	)
	if err != nil {
		return nil, err
	}
	if w.Cells == nil {
		w.Cells = map[string]any{}
	}
	return &w, nil
}

// Directory is what the server publishes about a window, with its limits.
func (c *Client) Limits(ctx context.Context) (*Limits, error) {
	var body struct {
		Limits Limits `json:"limits"`
	}
	if err := c.get(ctx, "/directory", &body); err != nil {
		return nil, err
	}
	return &body.Limits, nil
}
