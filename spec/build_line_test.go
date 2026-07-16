// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/linechart/wavelinechart"
)

func lineSpec() Spec {
	return Spec{
		Type: ChartTypeLine, Width: 40, Height: 10,
		Data: Data{Series: []Series{
			{Name: "a", Color: "#ff0000", Values: []DataPoint{
				{X: 0.0, Y: 1}, {X: 1.0, Y: 3}, {X: 2.0, Y: 2}, {X: 3.0, Y: 5},
			}},
			{Name: "b", Values: []DataPoint{
				{X: 0.0, Y: 4}, {X: 1.0, Y: 2}, {X: 2.0, Y: 4}, {X: 3.0, Y: 1},
			}},
		}},
	}
}

func TestBuildLine(t *testing.T) {
	got, err := Build(lineSpec())
	if err != nil {
		t.Fatalf("Build(line): %v", err)
	}
	m, ok := got.(*wavelinechart.Model)
	if !ok {
		t.Fatalf("Build(line) returned %T, want *wavelinechart.Model", got)
	}
	view := m.View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("line chart view is empty")
	}
}

func TestBuildLineIndexXWhenOmitted(t *testing.T) {
	s := lineSpec()
	for i := range s.Data.Series {
		for j := range s.Data.Series[i].Values {
			s.Data.Series[i].Values[j].X = nil // rely on index-as-X
		}
	}
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(line, index X): %v", err)
	}
}

func TestBuildLinePinnedYRange(t *testing.T) {
	s := lineSpec()
	s.YAxis.Min = f64(0)
	s.YAxis.Max = f64(10)
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(line, pinned Y): %v", err)
	}
}

// TestBuildLineOneSidedYPin exercises the one-sided pin rule: a lone Min or
// Max pins that bound while the other is derived from the series' Y values
// (lineSpec's Y values range from 1 to 5).
func TestBuildLineOneSidedYPin(t *testing.T) {
	t.Run("min only", func(t *testing.T) {
		s := lineSpec()
		s.YAxis.Min = f64(-10)
		if _, err := Build(s); err != nil {
			t.Fatalf("Build(line, one-sided min): %v", err)
		}
	})
	t.Run("max only", func(t *testing.T) {
		s := lineSpec()
		s.YAxis.Max = f64(100)
		if _, err := Build(s); err != nil {
			t.Fatalf("Build(line, one-sided max): %v", err)
		}
	})
}

// TestBuildLineInvertedYPinErrors pins Min above the series' actual data max
// (5): the data-derived Max is below the pinned Min, so Build must reject
// the resulting inverted range rather than silently constructing it.
func TestBuildLineInvertedYPinErrors(t *testing.T) {
	s := lineSpec()
	s.YAxis.Min = f64(50)
	_, err := Build(s)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected inverted y_axis range error, got %v", err)
	}
}
