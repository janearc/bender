package kingfisher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	var fails atomic.Int32
	fails.Store(2)
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/ingested/":
				w.Write([]byte(`{"directories": ["a/", "b/"]}`))
			case "/ingested/a/manifest.json":
				w.Write(
					[]byte(
						`{"id": "a", "kind": "LAYER_KIND_INTENSIVE", "layers": {"brightness": {"kind": "LAYER_KIND_INTENSIVE"}}, "pipeline": {"res": 4, "cells": 2}}`,
					),
				)
			case "/ingested/a/cells.json":
				if fails.Add(-1) >= 0 {
					http.Error(w, "busy", http.StatusServiceUnavailable)
					return
				}
				w.Write(
					[]byte(
						`{"res": 4, "cells": {"8428309ffffffff": {"brightness": 3}}}`,
					),
				)
			case "/ingested/b/manifest.json":
				w.Write(
					[]byte(
						`{"id": "b", "kind": "LAYER_KIND_EXTENSIVE"}`,
					),
				)
			case "/ingested/b/cells.json":
				w.Write([]byte(`{"res": 4}`))
			case "/ingested/bad/cells.json":
				w.Write([]byte(`not json`))
			default:
				http.NotFound(w, r)
			}
		}),
	)
	defer srv.Close()
	c := New(srv.URL + "/")
	var slept []time.Duration
	c.Sleep = func(d time.Duration) { slept = append(slept, d) }
	ctx := context.Background()

	ds, err := c.Datasets(ctx)
	if err != nil || len(ds) != 2 || ds[0] != "a" {
		t.Fatalf("datasets %v %v", ds, err)
	}
	m, err := c.Manifest(ctx, "a")
	if err != nil || m.Pipeline.Res != 4 ||
		m.LayerKinds()["brightness"] != "LAYER_KIND_INTENSIVE" {
		t.Fatalf("manifest %+v %v", m, err)
	}
	mb, _ := c.Manifest(ctx, "b")
	if mb.LayerKinds()["value"] != "LAYER_KIND_EXTENSIVE" {
		t.Error("a manifest with no layers maps its kind to value")
	}
	cells, err := c.Cells(ctx, "a")
	if err != nil || cells.Res != 4 || len(cells.Cells) != 1 {
		t.Fatalf("cells %+v %v", cells, err)
	}
	// Two 503s were retried with growing waits.
	if len(slept) != 2 || slept[1] < slept[0] ||
		slept[0] < 200*time.Millisecond ||
		slept[0] > 400*time.Millisecond {
		t.Errorf("backoff %v", slept)
	}
	if _, err := c.Cells(ctx, "b"); err == nil ||
		!strings.Contains(err.Error(), "no cells") {
		t.Error("empty cells")
	}
	if _, err := c.Cells(ctx, "bad"); err == nil {
		t.Error("bad json")
	}
	if _, err := c.Manifest(ctx, "nope"); err == nil ||
		!strings.Contains(err.Error(), "404") {
		t.Error("404 is final")
	}
	// A dead server gives up after the retries.
	dead := New("http://127.0.0.1:1")
	dead.Sleep = func(time.Duration) {}
	dead.Retries = 2
	if _, err := dead.Datasets(ctx); err == nil ||
		!strings.Contains(err.Error(), "gave up after 3") {
		t.Errorf("dead server: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := dead.Datasets(cancelled); err == nil {
		t.Error("cancelled context")
	}
	t.Setenv("KINGFISHER_URL", "http://x/")
	if URL() != "http://x" {
		t.Error("URL from env")
	}
	t.Setenv("KINGFISHER_URL", "")
	if URL() != DefaultURL {
		t.Error("default URL")
	}
}
