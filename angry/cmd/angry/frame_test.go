package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// The frame mode prints a header, rows of half blocks in true colour with
// a reset per line, and a footer, on both projections, with no terminal.
func TestPrintFrame(t *testing.T) {
	src, m := synthetic()
	for _, globe := range []bool{false, true} {
		var out bytes.Buffer
		if err := frameOnce(&out, src, m, "v", globe, 60, 20, 0, nil, style{}); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(
			strings.TrimSuffix(out.String(), "\n"),
			"\n",
		)
		if len(lines) != 22 {
			t.Fatalf("globe %v: %d lines", globe, len(lines))
		}
		if !strings.Contains(lines[0], "res ") ||
			!strings.Contains(lines[21], "unsurveyed") {
			t.Errorf("header or footer: %q %q", lines[0], lines[21])
		}
		body := strings.Join(lines[1:21], "\n")
		if !strings.Contains(body, "▀") &&
			!strings.Contains(body, "▄") {
			t.Error("no half blocks")
		}
		if !strings.Contains(body, "\x1b[38;2;") ||
			!strings.Contains(body, "\x1b[48;2;") {
			t.Error("no true colour")
		}
		for i, l := range lines[1:21] {
			if !strings.HasSuffix(l, "\x1b[0m") {
				t.Errorf("row %d does not reset", i)
			}
		}
		if globe != strings.Contains(lines[0], "globe") {
			t.Error("header names the projection")
		}
	}
}

// A width given without a height is honoured, and the other way round,
// counted in characters, not bytes: a half block is three bytes.
func TestPrintFrameSize(t *testing.T) {
	src, m := synthetic()
	var out bytes.Buffer
	if err := frameOnce(&out, src, m, "v", true, 40, 0, 0, nil, style{}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(out.String(), "\n")
	row := lines[1]
	plain := stripSGR(row)
	if n := len([]rune(plain)); n != 40 {
		t.Errorf("-cols 40 gave %d characters", n)
	}
	if len(lines) < 4 {
		t.Error("rows should have been derived")
	}
	out.Reset()
	frameOnce(&out, src, m, "v", false, 0, 5, 0, nil, style{})
	if got := strings.Count(out.String(), "\n"); got != 7 {
		t.Errorf("-rows 5 gave %d lines", got)
	}
}

func stripSGR(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// At world scale the globe is a disc against blank space: tinted planet
// pixels inside the rim, nothing outside, and graticule lines through it.
func TestPrintFrameWorldGlobe(t *testing.T) {
	src, m := synthetic()
	var out bytes.Buffer
	if err := frameOnce(&out, src, m, "v", true, 80, 30, 360, nil, style{}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(out.String(), "\n")
	middle := stripSGR(lines[16])
	top := stripSGR(lines[1])
	if strings.TrimSpace(top) != "" {
		t.Errorf("above the disc should be space: %q", top)
	}
	if !strings.HasPrefix(middle, "  ") || !strings.Contains(middle, "▀") ||
		strings.TrimSpace(middle) == "" {
		t.Errorf(
			"the middle row should be space, then disc: %q",
			middle,
		)
	}
	if !strings.Contains(out.String(), sgr(38, gridTint)) &&
		!strings.Contains(out.String(), sgr(48, gridTint)) {
		t.Error("no graticule drawn")
	}
	if !strings.Contains(out.String(), sgr(38, planetTint)) &&
		!strings.Contains(out.String(), sgr(48, planetTint)) {
		t.Error("no planet tint drawn")
	}
}

// A look is optional, and a bad one is an error at the command line
// rather than a frame drawn quietly from somewhere else.
func TestParseLook(t *testing.T) {
	if c, err := parseLook(""); c != nil || err != nil {
		t.Errorf("empty: %v %v", c, err)
	}
	c, err := parseLook("-122.4,37.8")
	if err != nil {
		t.Fatal(err)
	}
	if c.lon != -122.4 || c.lat != 37.8 {
		t.Errorf("got %v", *c)
	}
	for _, bad := range []string{"122.4", "here,there", "400,0", "0,91", "-181,0"} {
		if _, err := parseLook(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

// Two centres give two different pictures. This is the regression for the
// measured defect: in dataset mode the view was fitted to the data, so
// three different windows produced one identical frame and there was no
// way to turn the ball from outside the process.
func TestFrameLookTurns(t *testing.T) {
	src, m := synthetic()
	seen := map[string]bool{}
	for _, c := range []*aim{{lon: -120, lat: 30}, {lon: 0, lat: 0}, {lon: 100, lat: -40}} {
		var out bytes.Buffer
		if err := frameOnce(&out, src, m, "v", true, 60, 20, 90, c, style{}); err != nil {
			t.Fatal(err)
		}
		header := strings.SplitN(out.String(), "\n", 2)[0]
		want := fmt.Sprintf("centre %.3f, %.3f", c.lon, c.lat)
		if !strings.Contains(header, want) {
			t.Errorf("asked for %q, header says %q", want, header)
		}
		if seen[header] {
			t.Errorf("centre %v drew a frame already seen", *c)
		}
		seen[header] = true
	}
}
