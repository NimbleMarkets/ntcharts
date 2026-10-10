package chartpicture

import (
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/go-analyze/charts"
)

func TestBarChartOptionFromNT(t *testing.T) {
	bc := barchart.New(20, 10)
	bc.PushAll([]barchart.BarData{
		{Label: "Q1", Values: []barchart.BarValue{{Name: "rev", Value: 100}}},
		{Label: "Q2", Values: []barchart.BarValue{{Name: "rev", Value: 150}}},
		{Label: "Q3", Values: []barchart.BarValue{{Name: "rev", Value: 90}}},
	})

	opt := BarChartOptionFromNT(&bc)

	if opt.Horizontal != bc.Horizontal() {
		t.Errorf("Horizontal mismatch: %v vs %v", opt.Horizontal, bc.Horizontal())
	}
	if len(opt.SeriesList) == 0 {
		t.Fatal("SeriesList empty")
	}
	if got := len(opt.SeriesList[0].Values); got != 3 {
		t.Errorf("series values len = %d, want 3", got)
	}
	wantLabels := []string{"Q1", "Q2", "Q3"}
	if got := opt.CategoryAxis.Labels; !equalStrings(got, wantLabels) {
		t.Errorf("category labels = %v, want %v", got, wantLabels)
	}
}

func TestLineChartOptionFromNT(t *testing.T) {
	lc := linechart.New(40, 12, 0, 10, 0, 100)
	series := [][]float64{
		{1, 5, 10, 7, 3},
		{2, 4, 6, 8, 10},
	}
	names := []string{"alpha", "beta"}

	opt := LineChartOptionFromNT(&lc, series, names)

	if got := len(opt.SeriesList); got != 2 {
		t.Fatalf("series count = %d, want 2", got)
	}
	if opt.SeriesList[0].Name != "alpha" || opt.SeriesList[1].Name != "beta" {
		t.Errorf("names = %v", []string{opt.SeriesList[0].Name, opt.SeriesList[1].Name})
	}
	if got := opt.SeriesList[0].Values; !equalFloats(got, series[0]) {
		t.Errorf("values[0] = %v, want %v", got, series[0])
	}

	if len(opt.YAxis) != 1 {
		t.Fatalf("YAxis len = %d, want 1", len(opt.YAxis))
	}
	if opt.YAxis[0].Min == nil || *opt.YAxis[0].Min != 0 {
		t.Errorf("YAxis Min = %v, want 0", opt.YAxis[0].Min)
	}
	if opt.YAxis[0].Max == nil || *opt.YAxis[0].Max != 100 {
		t.Errorf("YAxis Max = %v, want 100", opt.YAxis[0].Max)
	}
}

func TestBarChartOptionFromNTMultiSeries(t *testing.T) {
	bc := barchart.New(20, 10)
	bc.PushAll([]barchart.BarData{
		{Label: "Q1", Values: []barchart.BarValue{
			{Name: "rev", Value: 100},
			{Name: "cost", Value: 60},
		}},
		{Label: "Q2", Values: []barchart.BarValue{
			{Name: "rev", Value: 150},
			{Name: "cost", Value: 80},
		}},
	})

	opt := BarChartOptionFromNT(&bc)

	if got := len(opt.SeriesList); got != 2 {
		t.Fatalf("series count = %d, want 2", got)
	}
	// Order should be stable: rev (first seen) before cost.
	if opt.SeriesList[0].Name != "rev" || opt.SeriesList[1].Name != "cost" {
		t.Errorf("series names = %q,%q, want rev,cost",
			opt.SeriesList[0].Name, opt.SeriesList[1].Name)
	}
	if !equalFloats(opt.SeriesList[0].Values, []float64{100, 150}) {
		t.Errorf("rev values = %v", opt.SeriesList[0].Values)
	}
	if !equalFloats(opt.SeriesList[1].Values, []float64{60, 80}) {
		t.Errorf("cost values = %v", opt.SeriesList[1].Values)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalFloats(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBarChartOptionFromNTForwardsMinAndMax(t *testing.T) {
	bc := barchart.New(20, 10, barchart.WithNoAutoMaxValue(), barchart.WithMaxValue(50), barchart.WithMinValue(-20))
	bc.Push(barchart.BarData{Label: "Q1", Values: []barchart.BarValue{{Name: "net", Value: -12}}})

	opt := BarChartOptionFromNT(&bc)

	if len(opt.ValueAxis) != 1 || opt.ValueAxis[0].Max == nil || *opt.ValueAxis[0].Max != 50 {
		t.Fatalf("value axis max = %+v, want 50", opt.ValueAxis)
	}
	if opt.ValueAxis[0].Min == nil || *opt.ValueAxis[0].Min != -20 {
		t.Fatalf("value axis min = %v, want -20 so zero sits where the glyph chart puts it", opt.ValueAxis[0].Min)
	}

	// Positive-only data keeps the axis minimum unset, as before.
	pos := barchart.New(20, 10)
	pos.Push(barchart.BarData{Label: "Q1", Values: []barchart.BarValue{{Name: "rev", Value: 10}}})
	if opt := BarChartOptionFromNT(&pos); len(opt.ValueAxis) == 0 || opt.ValueAxis[0].Min != nil {
		t.Fatalf("positive-only chart should leave Min unset, got %+v", opt.ValueAxis)
	}
}

func TestBarChartOptionFromNTStacksMultiValueBars(t *testing.T) {
	bc := barchart.New(20, 10)
	bc.PushAll([]barchart.BarData{
		{Label: "Q1", Values: []barchart.BarValue{{Name: "rev", Value: 100}, {Name: "cost", Value: -60}}},
		{Label: "Q2", Values: []barchart.BarValue{{Name: "rev", Value: 150}, {Name: "cost", Value: -80}}},
	})
	if opt := BarChartOptionFromNT(&bc); opt.StackSeries == nil || !*opt.StackSeries {
		t.Fatal("multi-value bars are stacked in the glyph chart; the image should stack them too")
	}

	single := barchart.New(20, 10)
	single.Push(barchart.BarData{Label: "Q1", Values: []barchart.BarValue{{Name: "rev", Value: 100}}})
	if opt := BarChartOptionFromNT(&single); opt.StackSeries != nil {
		t.Fatal("single-value bars should not request stacking")
	}
}

func TestLineChartOptionFromNTAxes(t *testing.T) {
	series := [][]float64{{1, 5, 10}, {200, 400, 300}}
	names := []string{"load", "temp"}
	sides := []linechart.YAxisSide{linechart.YAxisLeft, linechart.YAxisRight}

	t.Run("two axes", func(t *testing.T) {
		lc := linechart.New(40, 12, 0, 10, 0, 100, linechart.WithY2Axis(linechart.Axis{Min: 100, Max: 500}))
		opt := LineChartOptionFromNTAxes(&lc, series, names, sides)
		if len(opt.YAxis) != 2 {
			t.Fatalf("YAxis len = %d, want 2", len(opt.YAxis))
		}
		if *opt.YAxis[0].Min != 0 || *opt.YAxis[0].Max != 100 {
			t.Errorf("left axis = %v..%v, want 0..100", *opt.YAxis[0].Min, *opt.YAxis[0].Max)
		}
		if *opt.YAxis[1].Min != 100 || *opt.YAxis[1].Max != 500 || opt.YAxis[1].Position != charts.PositionRight {
			t.Errorf("right axis = %v..%v at %q, want 100..500 at right", *opt.YAxis[1].Min, *opt.YAxis[1].Max, opt.YAxis[1].Position)
		}
		if opt.SeriesList[0].YAxisIndex != 0 || opt.SeriesList[1].YAxisIndex != 1 {
			t.Errorf("series axis indices = %d, %d; want 0, 1", opt.SeriesList[0].YAxisIndex, opt.SeriesList[1].YAxisIndex)
		}

		// Longer sides slice is ignored safely
		longSides := append([]linechart.YAxisSide{}, sides...)
		longSides = append(longSides, linechart.YAxisRight)
		opt2 := LineChartOptionFromNTAxes(&lc, series, names, longSides)
		if opt2.SeriesList[0].YAxisIndex != 0 || opt2.SeriesList[1].YAxisIndex != 1 {
			t.Errorf("longer sides: series axis indices = %d, %d; want 0, 1", opt2.SeriesList[0].YAxisIndex, opt2.SeriesList[1].YAxisIndex)
		}
	})
	t.Run("sides ignored without a right axis", func(t *testing.T) {
		lc := linechart.New(40, 12, 0, 10, 0, 100)
		opt := LineChartOptionFromNTAxes(&lc, series, names, sides)
		if len(opt.YAxis) != 1 || opt.SeriesList[1].YAxisIndex != 0 {
			t.Fatalf("YAxis len = %d, series[1] index = %d; want 1, 0", len(opt.YAxis), opt.SeriesList[1].YAxisIndex)
		}
	})
	t.Run("missing sides mean left", func(t *testing.T) {
		lc := linechart.New(40, 12, 0, 10, 0, 100, linechart.WithY2Axis(linechart.Axis{Min: 100, Max: 500}))
		opt := LineChartOptionFromNTAxes(&lc, series, names, nil)
		if opt.SeriesList[0].YAxisIndex != 0 || opt.SeriesList[1].YAxisIndex != 0 {
			t.Fatalf("series axis indices = %d, %d; want 0, 0", opt.SeriesList[0].YAxisIndex, opt.SeriesList[1].YAxisIndex)
		}
	})
	t.Run("nil model", func(t *testing.T) {
		opt := LineChartOptionFromNTAxes(nil, series, names, sides)
		if len(opt.SeriesList) != 2 {
			t.Fatalf("series = %d, want 2", len(opt.SeriesList))
		}
		if len(opt.YAxis) > 1 {
			t.Errorf("YAxis len = %d, want <= 1 (go-chart default, no right axis)", len(opt.YAxis))
		}
		if opt.SeriesList[1].YAxisIndex != 0 {
			t.Errorf("series[1] YAxisIndex = %d, want 0 (default left axis)", opt.SeriesList[1].YAxisIndex)
		}
	})
}

func TestLineChartOptionFromNTKeepsOneAxisWithDualAxisModel(t *testing.T) {
	lc := linechart.New(40, 12, 0, 10, 0, 100, linechart.WithY2Axis(linechart.Axis{Min: 100, Max: 500}))
	opt := LineChartOptionFromNT(&lc, [][]float64{{1, 5, 10}, {200, 400, 300}}, []string{"load", "temp"})
	if len(opt.YAxis) != 1 {
		t.Fatalf("YAxis len = %d, want 1 (the legacy adapter never adds a second axis)", len(opt.YAxis))
	}
	for i, s := range opt.SeriesList {
		if s.YAxisIndex != 0 {
			t.Fatalf("series[%d] YAxisIndex = %d, want 0", i, s.YAxisIndex)
		}
	}
}
