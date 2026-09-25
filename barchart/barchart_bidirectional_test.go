// ntcharts - Copyright (c) 2026 Neomantra Corp.

package barchart

import (
	"math"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"

	"charm.land/lipgloss/v2"
)

var (
	posStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	negStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestNegativeValuesTrackMin(t *testing.T) {
	bc := New(6, 10) // axis shown: graph length is 8 rows
	if bc.MinValue() != 0 {
		t.Fatalf("MinValue default = %v, want 0", bc.MinValue())
	}
	bc.Push(BarData{Label: "A", Values: []BarValue{
		{Value: 30, Style: posStyle},
		{Value: -20, Style: negStyle},
	}})
	if bc.MaxValue() != 30 || bc.MinValue() != -20 {
		t.Fatalf("range = [%v, %v], want [-20, 30]", bc.MinValue(), bc.MaxValue())
	}
	// 8 rows split 5 positive / 3 negative; scale is the tighter of 5/30 and 3/20.
	if !near(bc.Scale(), 0.15) {
		t.Fatalf("Scale = %v, want 0.15", bc.Scale())
	}
	if bc.origin.Y != 5 {
		t.Fatalf("origin.Y = %d, want 5 (axis row below 5 positive rows)", bc.origin.Y)
	}
}

func TestNoAutoMaxValueLeavesMinAlone(t *testing.T) {
	bc := New(6, 10, WithNoAutoMaxValue(), WithMaxValue(50), WithMinValue(-30))
	bc.Push(BarData{Label: "A", Values: []BarValue{{Value: -100, Style: negStyle}}})
	if bc.MinValue() != -30 {
		t.Fatalf("MinValue = %v, want -30 unchanged", bc.MinValue())
	}
	// 8 rows split 5/3 for [-30, 50]; scale is min(5/50, 3/30) = 0.1
	if !near(bc.Scale(), 0.1) {
		t.Fatalf("Scale = %v, want 0.1", bc.Scale())
	}
}

func TestSetMinClampsToZero(t *testing.T) {
	bc := New(6, 10, WithMinValue(10))
	if bc.MinValue() != 0 {
		t.Fatalf("MinValue = %v, want 0 (min cannot exceed zero)", bc.MinValue())
	}
}

func TestPositiveOnlyDataIsUnchanged(t *testing.T) {
	// The existing contract: with only positive data the axis sits at the
	// bottom and scale is (h-2)/max.
	bc := New(6, 10, WithMaxValue(50))
	bc.Push(BarData{Label: "A", Values: []BarValue{{Value: 25, Style: posStyle}}})
	if !near(bc.Scale(), 8.0/50) || bc.origin.Y != 8 || bc.MinValue() != 0 {
		t.Fatalf("scale=%v origin.Y=%d min=%v", bc.Scale(), bc.origin.Y, bc.MinValue())
	}
}

func TestDrawVerticalBidirectional(t *testing.T) {
	bc := New(1, 10, WithMaxValue(50), WithMinValue(-30), WithNoAutoMaxValue())
	bc.Push(BarData{Label: "A", Values: []BarValue{
		{Value: 20, Style: posStyle},
		{Value: -14, Style: negStyle},
	}})
	bc.Draw()
	// scale 0.1: +20 -> 2 rows above the axis (rows 4,3); -14 -> 1.4 rows below (row 6 full, row 7 partial)
	x := 0
	if c := bc.Canvas.Cell(canvas.Point{X: x, Y: 5}); c.Rune != runes.LineHorizontal {
		t.Errorf("axis row 5 rune = %q, want horizontal line", c.Rune)
	}
	for _, y := range []int{4, 3} {
		if c := bc.Canvas.Cell(canvas.Point{X: x, Y: y}); c.Rune != runes.FullBlock {
			t.Errorf("positive row %d rune = %q, want full block", y, c.Rune)
		}
	}
	if c := bc.Canvas.Cell(canvas.Point{X: x, Y: 2}); runes.IsLowerBlockElement(c.Rune) {
		t.Errorf("row 2 should be empty, got %q", c.Rune)
	}
	if c := bc.Canvas.Cell(canvas.Point{X: x, Y: 6}); c.Rune != runes.FullBlock {
		t.Errorf("negative row 6 rune = %q, want full block", c.Rune)
	}
	if c := bc.Canvas.Cell(canvas.Point{X: x, Y: 7}); !c.Style.GetReverse() || c.Rune != runes.InverseLowerBlockElement(runes.LowerBlockElementFromFloat64(0.4)) {
		t.Errorf("negative row 7 = %q reverse=%v, want inverse 0.4 block in reverse video", c.Rune, c.Style.GetReverse())
	}
	if c := bc.Canvas.Cell(canvas.Point{X: x, Y: 8}); runes.IsLowerBlockElement(c.Rune) {
		t.Errorf("row 8 should be empty, got %q", c.Rune)
	}
	if c := bc.Canvas.Cell(canvas.Point{X: x, Y: 9}); c.Rune != 'A' {
		t.Errorf("label row 9 rune = %q, want 'A' (labels stay on the bottom row)", c.Rune)
	}
}

func TestDrawVerticalNoAxisBidirectional(t *testing.T) {
	bc := New(1, 8, WithNoAxis(), WithMaxValue(50), WithMinValue(-30), WithNoAutoMaxValue())
	bc.Push(BarData{Label: "A", Values: []BarValue{
		{Value: 50, Style: posStyle},
		{Value: -30, Style: negStyle},
	}})
	bc.Draw()
	// 8 rows split 5/3 with no axis row: rows 0-4 positive, rows 5-7 negative.
	for y := 0; y < 8; y++ {
		if c := bc.Canvas.Cell(canvas.Point{X: 0, Y: y}); c.Rune != runes.FullBlock {
			t.Errorf("row %d rune = %q, want full block", y, c.Rune)
		}
	}
	if c := bc.Canvas.Cell(canvas.Point{X: 0, Y: 4}); c.Style.GetForeground() != posStyle.GetForeground() {
		t.Errorf("row 4 should be positive color")
	}
	if c := bc.Canvas.Cell(canvas.Point{X: 0, Y: 5}); c.Style.GetForeground() != negStyle.GetForeground() {
		t.Errorf("row 5 should be negative color")
	}
}

func TestDrawHorizontalBidirectional(t *testing.T) {
	bc := New(12, 1, WithHorizontalBars(), WithMaxValue(50), WithMinValue(-30), WithNoAutoMaxValue())
	bc.Push(BarData{Label: "A", Values: []BarValue{
		{Value: 25, Style: posStyle},
		{Value: -20, Style: negStyle},
	}})
	bc.Resize(12, 1) // recompute label width from data, as callers do today
	bc.Draw()
	// label col 0, graph length 10 split 6 positive / 4 negative, axis at x=5, scale 0.12
	if !near(bc.Scale(), 0.12) || bc.origin.X != 5 {
		t.Fatalf("scale=%v origin.X=%d, want 0.12 and 5", bc.Scale(), bc.origin.X)
	}
	if c := bc.Canvas.Cell(canvas.Point{X: 0, Y: 0}); c.Rune != 'A' {
		t.Errorf("label cell = %q, want 'A'", c.Rune)
	}
	if c := bc.Canvas.Cell(canvas.Point{X: 5, Y: 0}); c.Rune != runes.LineVertical {
		t.Errorf("axis cell = %q, want vertical line", c.Rune)
	}
	for _, x := range []int{6, 7, 8} { // +25 -> 3 cells right of the axis
		if c := bc.Canvas.Cell(canvas.Point{X: x, Y: 0}); c.Rune != runes.FullBlock {
			t.Errorf("positive col %d = %q, want full block", x, c.Rune)
		}
	}
	for _, x := range []int{4, 3} { // -20 -> 2.4 cells left of the axis
		if c := bc.Canvas.Cell(canvas.Point{X: x, Y: 0}); c.Rune != runes.FullBlock {
			t.Errorf("negative col %d = %q, want full block", x, c.Rune)
		}
	}
	if c := bc.Canvas.Cell(canvas.Point{X: 2, Y: 0}); !c.Style.GetReverse() || c.Rune != runes.InverseLeftBlockElement(runes.LeftBlockElementFromFloat64(0.4)) {
		t.Errorf("negative col 2 = %q reverse=%v, want inverse 0.4 block in reverse video", c.Rune, c.Style.GetReverse())
	}
}

func TestBarDataFromPointBothSides(t *testing.T) {
	bc := New(1, 10, WithMaxValue(50), WithMinValue(-30), WithNoAutoMaxValue())
	bc.Push(BarData{Label: "A", Values: []BarValue{
		{Name: "up", Value: 20, Style: posStyle},
		{Name: "down", Value: -14, Style: negStyle},
	}})
	bc.Draw()
	if r := bc.BarDataFromPoint(canvas.Point{X: 0, Y: 4}); len(r.Values) != 1 || r.Values[0].Name != "up" {
		t.Errorf("row 4 (above axis) = %+v, want the positive segment", r.Values)
	}
	if r := bc.BarDataFromPoint(canvas.Point{X: 0, Y: 7}); len(r.Values) != 1 || r.Values[0].Name != "down" {
		t.Errorf("row 7 (below axis) = %+v, want the negative segment", r.Values)
	}
	if r := bc.BarDataFromPoint(canvas.Point{X: 0, Y: 8}); len(r.Values) != 0 {
		t.Errorf("row 8 (beyond bar) = %+v, want nothing", r.Values)
	}

	hc := New(12, 1, WithHorizontalBars(), WithMaxValue(50), WithMinValue(-30), WithNoAutoMaxValue())
	hc.Push(BarData{Label: "A", Values: []BarValue{
		{Name: "right", Value: 25, Style: posStyle},
		{Name: "left", Value: -20, Style: negStyle},
	}})
	hc.Resize(12, 1)
	hc.Draw()
	if r := hc.BarDataFromPoint(canvas.Point{X: 7, Y: 0}); len(r.Values) != 1 || r.Values[0].Name != "right" {
		t.Errorf("col 7 (right of axis) = %+v, want the positive segment", r.Values)
	}
	if r := hc.BarDataFromPoint(canvas.Point{X: 3, Y: 0}); len(r.Values) != 1 || r.Values[0].Name != "left" {
		t.Errorf("col 3 (left of axis) = %+v, want the negative segment", r.Values)
	}
}

func TestBidirectionalOneDrawableCell(t *testing.T) {
	for _, horizontal := range []bool{false, true} {
		for _, axis := range []bool{false, true} {
			for _, tc := range []struct {
				name     string
				max, min float64
				negative bool
			}{
				{"positive", 2, -1, false},
				{"negative", 1, -2, true},
				{"tie", 1, -1, false},
			} {
				bc := New(1, 1, WithMaxValue(tc.max), WithMinValue(tc.min), WithNoAutoMaxValue())
				bc.SetHorizontal(horizontal)
				bc.SetShowAxis(axis)
				w, h := 1, 1
				if axis {
					if horizontal {
						w = 2
					} else {
						h = 3
					}
				}
				bc.Resize(w, h)
				bc.Push(BarData{Values: []BarValue{{Value: tc.max, Style: posStyle}, {Value: tc.min, Style: negStyle}}})
				bc.Draw()
				if !near(bc.Scale(), 1/math.Max(tc.max, -tc.min)) {
					t.Fatalf("%s horizontal=%t axis=%t scale=%v", tc.name, horizontal, axis, bc.Scale())
				}
				p := canvas.Point{}
				if axis {
					if horizontal && !tc.negative {
						p.X = 1
					}
					if !horizontal && tc.negative {
						p.Y = 1
					}
				}
				want := posStyle.GetForeground()
				if tc.negative {
					want = negStyle.GetForeground()
				}
				c := bc.Canvas.Cell(p)
				if c.Rune != runes.FullBlock || c.Style.GetForeground() != want {
					t.Errorf("%s horizontal=%t axis=%t: cell %v = %+v", tc.name, horizontal, axis, p, c)
				}
				if hit := bc.BarDataFromPoint(p); len(hit.Values) != 1 || (hit.Values[0].Value < 0) != tc.negative {
					t.Errorf("%s: hit = %+v", tc.name, hit)
				}
			}
		}
	}
}

func TestBidirectionalHitBounds(t *testing.T) {
	for _, horizontal := range []bool{false, true} {
		bc := New(1, 7, WithMaxValue(2), WithMinValue(-2), WithNoAutoMaxValue())
		bc.SetHorizontal(horizontal)
		bc.Push(BarData{Label: "A", Values: []BarValue{{Value: 20}, {Value: -20}}})
		w, h := 1, 7
		if horizontal {
			w, h = 7, 1
		}
		bc.Resize(w, h)
		bc.Draw()
		for y := -1; y <= h; y++ {
			for x := -1; x <= w; x++ {
				p := canvas.Point{X: x, Y: y}
				c := bc.Canvas.Cell(p)
				want := 0
				if c.Rune == runes.FullBlock {
					want = 1
				}
				if hit := bc.BarDataFromPoint(p); len(hit.Values) != want {
					t.Errorf("horizontal=%t point=%v rune=%q hit=%+v", horizontal, p, c.Rune, hit)
				}
			}
		}
		bc.Clear()
		if hit := bc.BarDataFromPoint(canvas.Point{}); len(hit.Values) != 0 {
			t.Errorf("hit after clear = %+v", hit)
		}
	}
}

func TestBidirectionalHitRoundedEndpoints(t *testing.T) {
	for _, value := range []float64{1, 1.01, 1.0625, 1.9375, 2} {
		for _, horizontal := range []bool{false, true} {
			bc := New(1, 10, WithMaxValue(4), WithMinValue(-4), WithNoAutoMaxValue())
			bc.SetHorizontal(horizontal)
			bc.Push(BarData{Values: []BarValue{{Value: value}, {Value: -value}}})
			if horizontal {
				bc.Resize(9, 1)
			}
			bc.Draw()
			for y := 0; y < bc.Height(); y++ {
				for x := 0; x < bc.Width(); x++ {
					p := canvas.Point{X: x, Y: y}
					r := bc.Canvas.Cell(p).Rune
					want := 0
					if runes.IsLowerBlockElement(r) || runes.IsLeftBlockElement(r) {
						want = 1
					}
					if hit := bc.BarDataFromPoint(p); len(hit.Values) != want {
						t.Errorf("value=%v horizontal=%t point=%v rune=%q hit=%+v", value, horizontal, p, r, hit)
					}
				}
			}
		}
	}
}
