package main

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

// capture runs f with stdout redirected and returns what it printed.
func capture(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var b bytes.Buffer
	io.Copy(&b, r)
	return b.String()
}

var escape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Every page paints something, and every line fits in eighty columns
// once the escapes are stripped, because the person this is for reads at
// that width.
func TestPagesFitTheScreen(t *testing.T) {
	for _, p := range pages {
		out := escape.ReplaceAllString(capture(t, p.show), "")
		if !strings.HasPrefix(out, "== "+p.name) {
			t.Errorf("%s: does not open with its title", p.name)
		}
		for n, line := range strings.Split(out, "\n") {
			if len(line) > 80 {
				t.Errorf("%s: line %d is %d columns", p.name,
					n+1, len(line))
			}
		}
	}
}

// The only escapes the docs write are 24-bit backgrounds and resets:
// no foreground, no dim, no bold. That is the whole of what they assume
// about the terminal, and it is why they read on a light one too.
func TestOnlyBackgroundsArePainted(t *testing.T) {
	allowed := regexp.MustCompile(`^\x1b\[(48;2;\d+;\d+;\d+|0)m$`)
	for _, p := range pages {
		for _, e := range escape.FindAllString(capture(t, p.show), -1) {
			if !allowed.MatchString(e) {
				t.Errorf("%s writes %q, which assumes "+
					"something about the terminal", p.name,
					e)
			}
		}
	}
}

// With NO_COLOR set, no escape is written at all and the pages still
// say everything in words and hashes.
func TestNoColour(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, p := range pages {
		out := capture(t, p.show)
		if escape.MatchString(out) {
			t.Errorf("%s writes escapes under NO_COLOR", p.name)
		}
		if !strings.Contains(out, "#") {
			t.Errorf("%s shows no swatch shape under NO_COLOR",
				p.name)
		}
	}
}

// Every value in every css page is one of the productions the library
// promises to emit, and nothing else: a six-digit hex, oklch(), oklab(),
// or a linear-gradient over those.
func TestCSSSpeaksOnlyWhatIsPromised(t *testing.T) {
	value := regexp.MustCompile(
		`^(#[0-9a-f]{6}|oklch\([^)]*\)|oklab\([^)]*\)|` +
			`linear-gradient\((.|\n)*\))$`)
	decl := regexp.MustCompile(`(?s)--[a-z-]+:\s*(.*?);`)
	for _, p := range pages {
		sheet := p.css()
		if !strings.HasPrefix(sheet, ":root {\n") ||
			!strings.HasSuffix(sheet, "}\n") {
			t.Errorf("%s: css is not one :root block", p.name)
		}
		for _, m := range decl.FindAllStringSubmatch(sheet, -1) {
			if !value.MatchString(strings.TrimSpace(m[1])) {
				t.Errorf("%s: value %q is not a promised "+
					"production", p.name, m[1])
			}
		}
		for n, line := range strings.Split(sheet, "\n") {
			if len(line) > 80 {
				t.Errorf("%s: css line %d is %d columns",
					p.name, n+1, len(line))
			}
		}
	}
}

// The embedded source for each page is the file on disk, byte for byte,
// so --go shows what actually ran.
func TestGoIsTheFile(t *testing.T) {
	for _, p := range pages {
		embedded, err := sources.ReadFile("page_" + p.name + ".go")
		if err != nil {
			t.Fatal(err)
		}
		disk, err := os.ReadFile("page_" + p.name + ".go")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(embedded, disk) {
			t.Errorf("%s: the embedded source differs from the "+
				"file", p.name)
		}
	}
}
