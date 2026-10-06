// ntcharts - Copyright (c) 2024 Neomantra Corp.

package linechart

import (
	"fmt"
	"math"
	"testing"

	"github.com/NimbleMarkets/ntcharts/canvas"
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

// TestDrawXLabelFinalTickUsesTrueAxisMaximum guards the final tick's value,
// not just its position. The per-column interpolation drawXLabel uses for
// every column (v = viewMinX + increment*i) is always exactly one increment
// short of viewMaxX at i == last, because the rightmost data point is
// actually plotted using graphWidth-1 as its denominator (one less than the
// denominator behind that per-column increment). When the resulting short
// label is small enough to fit left-anchored (the common case for a
// single-digit axis), drawXLabel must still print the true axis maximum
// (viewMaxX) rather than silently rendering a value one increment low.
func TestDrawXLabelFinalTickUsesTrueAxisMaximum(t *testing.T) {
	lc := New(19, 10, 0, 9, 0, 10,
		WithXYSteps(2, 2),
		WithYLabelFormatter(func(i int, v float64) string {
			return "_"
		}),
	)

	last := lc.GraphWidth() - 1
	if last <= 0 {
		t.Fatalf("unexpected graph width: %d", lc.GraphWidth())
	}

	lc.DrawXYAxisAndLabel()

	want := fmt.Sprintf("%.0f", lc.ViewMaxX())
	rightmost := canvas.Point{X: lc.origin.X + last, Y: lc.origin.Y + 1}
	if got := string(lc.Canvas.Cell(rightmost).Rune); got != want {
		t.Fatalf("final X label (left-anchored fits path) = %q, want axis maximum %q (viewMaxX=%v)", got, want, lc.ViewMaxX())
	}
}

func TestDrawXLabelFallbackPreservesEarlierLabels(t *testing.T) {
	m := New(20, 5, 0, 100, 0, 10, WithXYSteps(4, 0),
		WithXLabelFormatter(func(i int, v float64) string {
			if v == 100 {
				return "FINAL_MAXIMUM"
			}
			return fmt.Sprint(i)
		}))
	m.DrawXYAxisAndLabel()
	for _, x := range []int{8, 12, 16} {
		if got := m.Canvas.Cell(canvas.Point{X: x, Y: 4}).Rune; got != rune(fmt.Sprint(x)[0]) {
			t.Fatalf("fallback overwrote label at %d: %q", x, got)
		}
	}
}

func TestDrawXLabelEdgeCases(t *testing.T) {
	for _, tc := range []struct {
		name     string
		width    int
		min, max float64
		format   LabelFormatter
		want     string
	}{
		{"one column", 1, 0, 9, DefaultLabelFormatter(), "9"},
		{"negative maximum", 10, -100, -10, DefaultLabelFormatter(), "-100   -10"},
		{"repeat", 10, 0, 10, func(int, float64) string { return "same" }, "same      "},
		{"too wide", 3, 0, 10, func(int, float64) string { return "oversized" }, "   "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := New(tc.width, 5, tc.min, tc.max, 0, 10, WithXYSteps(100, 0), WithXLabelFormatter(tc.format))
			m.DrawXYAxisAndLabel()
			var row []rune
			for x := 0; x < tc.width; x++ {
				r := m.Canvas.Cell(canvas.Point{X: x, Y: 4}).Rune
				if r == 0 {
					r = ' '
				}
				row = append(row, r)
			}
			if string(row) != tc.want {
				t.Fatalf("got %q, want %q", string(row), tc.want)
			}
		})
	}
}

func TestDefaultLabelFormatterNoNegativeZero(t *testing.T) {
	f := DefaultLabelFormatter()
	tests := []struct {
		v    float64
		want string
	}{
		{0, "0"},
		{math.Copysign(0, -1), "0"},
		{-0.3, "0"},
		{-0.5, "0"},
		{0.3, "0"},
		{-0.6, "-1"},
		{-2, "-2"},
		{2.4, "2"},
	}
	for _, tt := range tests {
		if got := f(0, tt.v); got != tt.want {
			t.Errorf("DefaultLabelFormatter()(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}
