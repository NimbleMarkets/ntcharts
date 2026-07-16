// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

func scatterSpec() Spec {
	return Spec{
		Type: ChartTypeScatter, Width: 40, Height: 12,
		XAxis: XAxis{Format: Format{Kind: "si"}},
		Data: Data{Series: []Series{
			{Name: "japan", Color: "#22aadd", Values: []DataPoint{
				{X: 2100.0, Y: 31.5}, {X: 1980.0, Y: 33.1},
			}},
			{Name: "usa", Color: "#dd8822", Values: []DataPoint{
				{X: 2875.0, Y: 24.0}, {X: 3200.0, Y: 19.2}, {X: 3600.0, Y: 16.5},
			}},
		}},
	}
}

func TestBuildScatter(t *testing.T) {
	got, err := Build(scatterSpec())
	if err != nil {
		t.Fatalf("Build(scatter): %v", err)
	}
	m, ok := got.(*linechart.Model)
	if !ok {
		t.Fatalf("Build(scatter) returned %T, want *linechart.Model", got)
	}
	view := m.View()
	if !strings.Contains(view, "•") {
		t.Fatalf("scatter view has no point markers:\n%s", view)
	}
}

func TestBuildScatterEmptySeriesErrors(t *testing.T) {
	s := scatterSpec()
	for i := range s.Data.Series {
		s.Data.Series[i].Values = nil
	}
	if _, err := Build(s); err == nil {
		t.Fatal("expected error for scatter with no points")
	}
}

func TestBuildScatterPinnedY(t *testing.T) {
	s := scatterSpec()
	s.YAxis.Min = f64(0)
	s.YAxis.Max = f64(40)
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(scatter, pinned Y): %v", err)
	}
}
