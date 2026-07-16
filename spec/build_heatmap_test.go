package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
)

func heatSpec() Spec {
	return Spec{
		Type: ChartTypeHeatmap, Width: 30, Height: 10,
		Heat: &HeatData{Cells: []HeatCell{
			{X: 0, Y: 0, Z: 1}, {X: 1, Y: 0, Z: 5}, {X: 2, Y: 0, Z: 9},
			{X: 0, Y: 1, Z: 3}, {X: 1, Y: 1, Z: 7}, {X: 2, Y: 1, Z: 2},
		}},
		Theme: Theme{Gradient: []string{"#000044", "#ff4400"}},
	}
}

func TestBuildHeatmapCells(t *testing.T) {
	got, err := Build(heatSpec())
	if err != nil {
		t.Fatalf("Build(heatmap): %v", err)
	}
	m, ok := got.(*heatmap.Model)
	if !ok {
		t.Fatalf("Build(heatmap) returned %T, want *heatmap.Model", got)
	}
	if strings.TrimSpace(m.View()) == "" {
		t.Fatal("heatmap view is empty")
	}
}

func TestBuildHeatmapMatrix(t *testing.T) {
	s := heatSpec()
	s.Heat = &HeatData{Matrix: [][]float64{{1, 5, 9}, {3, 7, 2}}}
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(heatmap, matrix): %v", err)
	}
}

func TestBuildHeatmapPinnedValueRange(t *testing.T) {
	s := heatSpec()
	s.Heat.MinValue = f64(0)
	s.Heat.MaxValue = f64(10)
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(heatmap, pinned range): %v", err)
	}
}

func TestGradientScale(t *testing.T) {
	cs, err := gradientScale([]string{"#000000", "#ff0000"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 32 {
		t.Fatalf("gradient scale has %d steps, want 32", len(cs))
	}
	if _, err := gradientScale([]string{"not-a-color"}); err == nil {
		t.Fatal("expected error for invalid hex stop")
	}
	cs, err = gradientScale(nil)
	if err != nil || cs != nil {
		t.Fatalf("nil stops should give (nil, nil), got (%v, %v)", cs, err)
	}
}
