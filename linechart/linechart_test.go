// ntcharts - Copyright (c) 2024 Neomantra Corp.

package linechart

import (
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
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

func TestDrawYLabelDrawsTopLabelWhenGraphHeightNotDivisibleByStep(t *testing.T) {
	lc := New(20, 15, 0, 10, 0, 13,
		WithXYSteps(1, 2),
		WithYLabelFormatter(func(i int, v float64) string {
			if i == 13 {
				return "T"
			}
			return "_"
		}),
	)

	if lc.GraphHeight() != 13 {
		t.Fatalf("unexpected graph height: %d", lc.GraphHeight())
	}

	lc.DrawXYAxisAndLabel()

	top := canvas.Point{X: lc.origin.X - 1, Y: lc.origin.Y - lc.GraphHeight()}
	if got := lc.Canvas.Cell(top).Rune; got != 'T' {
		t.Fatalf("top Y label missing: got %q at %+v", got, top)
	}
}

func TestDrawXLabelDrawsRightmostLabelWhenGraphWidthNotDivisibleByStep(t *testing.T) {
	lc := New(19, 10, 0, 16, 0, 10,
		WithXYSteps(2, 2),
		WithYLabelFormatter(func(i int, v float64) string {
			return "_"
		}),
	)

	last := lc.GraphWidth() - 1
	if last <= 0 {
		t.Fatalf("unexpected graph width: %d", lc.GraphWidth())
	}

	lc.XLabelFormatter = func(i int, v float64) string {
		if i == last {
			return "R"
		}
		return "_"
	}

	lc.DrawXYAxisAndLabel()

	rightmost := canvas.Point{X: lc.origin.X + last, Y: lc.origin.Y + 1}
	if got := lc.Canvas.Cell(rightmost).Rune; got != 'R' {
		t.Fatalf("rightmost X label missing: got %q at %+v", got, rightmost)
	}
}
