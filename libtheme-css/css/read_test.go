package css

import (
	"math"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"
)

const vaporwaveish = `/* a colourway */
:root {
  --bg: #160d2b;
  --accent: #ff6ec7;
  --ink: #fff;
}
.scanlines { background: repeating-linear-gradient(180deg,
    rgba(255,110,199,.09) 0 2px, transparent 2px 5px); }
.sun { background: radial-gradient(38% 26% at 50% 62%, #ff2d95 0%,
    #ff6ec7 30%, rgba(255,110,199,0) 72%); }
.sunset { background: linear-gradient(180deg, #2a0f4d 0%, #4a1361 34%,
    #8c2472 56%, #d94a8c 66%, #2b1250 80%, #160d2b 100%); }
:root {
  --sunset-1: #2a0f4d;
  --sunset-2: #4a1361;
  --sunset-3: #8c2472;
  --sunset-4: #d94a8c;
  --sunset-5: #2b1250;
  --sunset-6: #160d2b;
  --sun-1: #ff2d95;
  --sun-2: #ff6ec7;
}
`

// roles come out in file order with their dashes gone; a 3-digit hex
// reads; the numbered families are roles too.
func TestReadRoles(t *testing.T) {
	cw := Read(vaporwaveish)
	names := cw.Roles.Names()
	if len(names) != 11 || names[0] != "bg" || names[1] != "accent" ||
		names[2] != "ink" {
		t.Fatalf("roles: %v", names)
	}
	ink, _ := cw.Roles.Get("ink")
	if ink != srgb.MustHex("#ffffff").Swatch() {
		t.Error("#fff did not read as white")
	}
}

// ramps come only from numbered families, in file order and with even
// stops; the gradient rules in the file are not read, so scanlines is
// nothing and sunset is the six promoted roles, not the rule.
func TestReadRamps(t *testing.T) {
	cw := Read(vaporwaveish)
	if _, ok := cw.Ramps["scanlines"]; ok {
		t.Error("a gradient rule was read")
	}
	sunset := cw.Ramps["sunset"]
	if len(sunset.Stops) != 6 {
		t.Fatalf("sunset has %d stops", len(sunset.Stops))
	}
	for i, want := range []float64{0, .2, .4, .6, .8, 1} {
		if math.Abs(sunset.Stops[i].At-want) > 1e-9 {
			t.Errorf("sunset stop %d at %v, want %v", i,
				sunset.Stops[i].At, want)
		}
	}
	if sunset.Stops[3].Swatch != srgb.MustHex("#d94a8c").Swatch() {
		t.Error("sunset stop 4 is not #d94a8c")
	}
	if sun := cw.Ramps["sun"]; len(sun.Stops) != 2 {
		t.Errorf("sun: %v", sun.Stops)
	}
	if len(cw.Order) != 2 || cw.Order[0] != "sunset" {
		t.Errorf("order: %v", cw.Order)
	}
}

// a family alone, with no gradient of that name, is an even ramp.
func TestReadFamilyAlone(t *testing.T) {
	cw := Read(":root { --dusk-1: #000000; --dusk-2: #808080; --dusk-3: " +
		"#ffffff; }")
	d := cw.Ramps["dusk"]
	if len(d.Stops) != 3 || d.Stops[1].At != 0.5 {
		t.Fatalf("dusk: %v", d.Stops)
	}
}
