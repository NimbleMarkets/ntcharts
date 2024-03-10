package linechart

import (
	"testing"
)

func TestNew(t *testing.T) {
	w := 30
	h := 15
	minX := -5.0
	maxX := 10.0
	minY := -5.0
	maxY := 10.0
	xStep := 1
	yStep := 2

	lc := New(w, h, minX, maxX, minY, maxY, WithXYSteps(xStep, yStep))

	if lc.Width() != w {
		t.Errorf("Width not initialized:%d", lc.Width())
	}
	if lc.Height() != h {
		t.Errorf("Height not initialized:%d", lc.Height())
	}
	if lc.GraphWidth() != w-3 {
		t.Errorf("GraphWidth not initialized:%d", lc.GraphWidth())
	}
	if lc.GraphHeight() != h-2 {
		t.Errorf("GraphHeight not initialized:%d", lc.GraphHeight())
	}

	if lc.MinX() != minX {
		t.Errorf("MinX not initialized:%f", lc.MinX())
	}
	if lc.MaxX() != maxX {
		t.Errorf("MaxX not initialized:%f", lc.MaxX())
	}
	if lc.MinY() != minY {
		t.Errorf("MinY not initialized:%f", lc.MinY())
	}
	if lc.MaxY() != maxY {
		t.Errorf("MaxY not initialized:%f", lc.MaxY())
	}
}
