// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/barchart"
)

func barSpec(nSeries int) Spec {
	s := Spec{
		Type: ChartTypeBar, Width: 40, Height: 10,
		XAxis: XAxis{Labels: []string{"Q1", "Q2", "Q3"}},
	}
	names := []string{"rev", "cost"}
	for i := 0; i < nSeries; i++ {
		s.Data.Series = append(s.Data.Series, Series{
			Name:   names[i],
			Values: []DataPoint{{Y: 3}, {Y: 5}, {Y: 2}},
		})
	}
	return s
}

func TestBuildBarMultiSeriesRequiresStacked(t *testing.T) {
	s := barSpec(2) // Stacked defaults to false
	_, err := Build(s)
	if err == nil || !strings.Contains(err.Error(), "stacked") {
		t.Fatalf("expected grouped-bars error mentioning stacked, got %v", err)
	}
	s.Options.Stacked = true
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(bar, stacked): %v", err)
	}
}

func TestBuildBarSingleSeriesNeedsNoStacked(t *testing.T) {
	if _, err := Build(barSpec(1)); err != nil {
		t.Fatalf("Build(bar, single series): %v", err)
	}
}

func TestBuildBarForwardsYAxisPins(t *testing.T) {
	s := barSpec(1)
	s.YAxis.Min = f64(-4)
	s.YAxis.Max = f64(12)
	m, err := Build(s)
	if err != nil {
		t.Fatalf("Build(bar, pinned): %v", err)
	}
	bc := m.(*barchart.Model)
	if got := bc.MinValue(); got != -4 {
		t.Fatalf("MinValue() = %v, want -4 (y_axis.min must reach barchart.WithMinValue)", got)
	}
	if got := bc.MaxValue(); got != 12 {
		t.Fatalf("MaxValue() = %v, want 12", got)
	}
}

func TestBuildBarNegativeValuesAutoScaleBelowZero(t *testing.T) {
	s := barSpec(1)
	s.Data.Series[0].Values = []DataPoint{{Y: 3}, {Y: -5}, {Y: 2}}
	m, err := Build(s)
	if err != nil {
		t.Fatalf("Build(bar, negative): %v", err)
	}
	if got := m.(*barchart.Model).MinValue(); got > -5 {
		t.Fatalf("MinValue() = %v, want <= -5 (negative bars auto-scale the floor)", got)
	}
}

func TestBuildBarRejectsInvertedPins(t *testing.T) {
	s := barSpec(1)
	s.YAxis.Min = f64(10)
	s.YAxis.Max = f64(2)
	if _, err := Build(s); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected inverted-pin error, got %v", err)
	}
}

func TestBuildBarHorizontal(t *testing.T) {
	s := barSpec(1)
	s.Options.Orientation = OrientationHorizontal
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(bar, horizontal): %v", err)
	}
}
