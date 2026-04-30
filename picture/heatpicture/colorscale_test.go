package heatpicture

import (
	"image/color"
	"testing"
)

// scale used in tests: black -> red -> white. Three stops, one midpoint each.
// Pre-converted []color.RGBA matches what SetColorScale stores internally.
var testScale = []color.RGBA{
	{R: 0, G: 0, B: 0, A: 255},
	{R: 255, G: 0, B: 0, A: 255},
	{R: 255, G: 255, B: 255, A: 255},
}

func nearRGBA(t *testing.T, got color.Color, wantR, wantG, wantB, wantA uint8, tol uint8) {
	t.Helper()
	r, g, b, a := got.RGBA()
	gr, gg, gb, ga := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
	if absDiff(gr, wantR) > tol || absDiff(gg, wantG) > tol || absDiff(gb, wantB) > tol || absDiff(ga, wantA) > tol {
		t.Errorf("RGBA mismatch: got (%d,%d,%d,%d), want (%d,%d,%d,%d) tol=%d",
			gr, gg, gb, ga, wantR, wantG, wantB, wantA, tol)
	}
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

// TestColorAt_BelowRange clamps to the first stop. The renderer normalizes
// value→[0,1] via (val-min)/(max-min); a value < min produces t<0, which the
// scale should treat as 0.
func TestColorAt_BelowRange(t *testing.T) {
	c := colorAt(testScale, -0.5)
	nearRGBA(t, c, 0, 0, 0, 255, 1)
}

// TestColorAt_AboveRange clamps to the last stop.
func TestColorAt_AboveRange(t *testing.T) {
	c := colorAt(testScale, 1.5)
	nearRGBA(t, c, 255, 255, 255, 255, 1)
}

// TestColorAt_FirstStop_Exact returns the first stop for t=0.
func TestColorAt_FirstStop_Exact(t *testing.T) {
	c := colorAt(testScale, 0)
	nearRGBA(t, c, 0, 0, 0, 255, 1)
}

// TestColorAt_LastStop_Exact returns the last stop for t=1.
func TestColorAt_LastStop_Exact(t *testing.T) {
	c := colorAt(testScale, 1)
	nearRGBA(t, c, 255, 255, 255, 255, 1)
}

// TestColorAt_MidStop_Exact returns the middle stop for t=0.5 in a 3-stop
// scale (so t maps to scale index 1.0 = exact second stop).
func TestColorAt_MidStop_Exact(t *testing.T) {
	c := colorAt(testScale, 0.5)
	nearRGBA(t, c, 255, 0, 0, 255, 1)
}

// TestColorAt_QuarterPoint_Interpolates blends 50/50 between black and red,
// since t=0.25 in a 3-stop scale lands at index 0.5 (halfway between stops 0
// and 1).
func TestColorAt_QuarterPoint_Interpolates(t *testing.T) {
	c := colorAt(testScale, 0.25)
	nearRGBA(t, c, 128, 0, 0, 255, 2)
}

// TestColorAt_ThreeQuarterPoint_Interpolates blends 50/50 between red and
// white, since t=0.75 in a 3-stop scale lands at index 1.5.
func TestColorAt_ThreeQuarterPoint_Interpolates(t *testing.T) {
	c := colorAt(testScale, 0.75)
	nearRGBA(t, c, 255, 128, 128, 255, 2)
}

// TestColorAt_EmptyScale falls back to a sentinel color so the renderer
// doesn't crash if the consumer hasn't set a scale yet. Sentinel is
// transparent so the "no scale" condition is distinguishable from any real
// color choice.
func TestColorAt_EmptyScale(t *testing.T) {
	c := colorAt(nil, 0.5)
	_, _, _, a := c.RGBA()
	if a != 0 {
		t.Errorf("expected fully transparent fallback for empty scale, got alpha=%d", a)
	}
}

// TestColorAt_SingleStop returns that stop for any t.
func TestColorAt_SingleStop(t *testing.T) {
	scale := []color.RGBA{{R: 0, G: 128, B: 255, A: 255}}
	c := colorAt(scale, 0.42)
	nearRGBA(t, c, 0, 128, 255, 255, 1)
}

// TestToRGBA_ConvertsAndShifts verifies the lipgloss/Color → RGBA pre-conversion
// produces the right 8-bit values from a known hex input.
func TestToRGBA_ConvertsAndShifts(t *testing.T) {
	src := []color.Color{
		color.RGBA{R: 255, G: 128, B: 64, A: 255},
		color.RGBA{R: 0, G: 0, B: 0, A: 255},
	}
	got := toRGBA(src)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].R != 255 || got[0].G != 128 || got[0].B != 64 || got[0].A != 255 {
		t.Errorf("got[0] = %+v, want {255,128,64,255}", got[0])
	}
}
