package spec

import (
	"image/color"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
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

func TestBuildHeatmapOneSidedPins(t *testing.T) {
	for _, matrix := range []bool{false, true} {
		for _, tc := range []struct {
			min, max         *float64
			wantMin, wantMax float64
		}{
			{f64(-5), nil, -5, 9},
			{nil, f64(20), 1, 20},
		} {
			s := heatSpec()
			if matrix {
				s.Heat = &HeatData{Matrix: [][]float64{{1, 5, 9}, {3, 7, 2}}}
			}
			s.Heat.MinValue, s.Heat.MaxValue = tc.min, tc.max
			got, err := Build(s)
			if err != nil {
				t.Fatal(err)
			}
			s.Heat.MinValue, s.Heat.MaxValue = &tc.wantMin, &tc.wantMax
			want, err := Build(s)
			if err != nil {
				t.Fatal(err)
			}
			assertHeatCellsEqual(t, got.(*heatmap.Model), want.(*heatmap.Model))
		}
	}
	for _, axis := range []HeatData{
		{Cells: []HeatCell{{Z: 1}, {Z: 5}}, MinValue: f64(6)},
		{Cells: []HeatCell{{Z: 1}, {Z: 5}}, MaxValue: f64(0)},
		{Cells: []HeatCell{{Z: 1}}, MinValue: f64(5), MaxValue: f64(1)},
	} {
		s := heatSpec()
		s.Heat = &axis
		if _, err := Build(s); err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("expected inverted range error, got %v", err)
		}
	}
}

func assertHeatCellsEqual(t *testing.T, got, want *heatmap.Model) {
	t.Helper()
	colours := 0
	for y := 0; y < got.Height(); y++ {
		for x := 0; x < got.Width(); x++ {
			p := canvas.Point{X: x, Y: y}
			a := got.Canvas.Cell(p).Style.GetBackground()
			b := want.Canvas.Cell(p).Style.GetBackground()
			if (a == nil) != (b == nil) {
				t.Fatalf("different background presence at %v", p)
			}
			if a != nil {
				colours++
				if color.RGBAModel.Convert(a) != color.RGBAModel.Convert(b) {
					t.Fatalf("different cell colours at %v: %v vs %v", p, a, b)
				}
			}
		}
	}
	if colours == 0 {
		t.Fatal("no heat cells rendered")
	}
}

func TestBuildHeatmapMatrixRowsMatchCellY(t *testing.T) {
	s := heatSpec()
	want, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	// A non-square matrix makes a transposition observable. Matrix takes
	// precedence over the conflicting sparse cells, including its value range.
	s.Heat = &HeatData{Matrix: [][]float64{{1, 5, 9}, {3, 7, 2}}, Cells: []HeatCell{{X: 50, Y: 50, Z: 1000}}}
	got, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	assertHeatCellsEqual(t, got.(*heatmap.Model), want.(*heatmap.Model))
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
