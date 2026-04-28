package chartpicture

import (
	"image/color"
	"testing"

	"github.com/go-analyze/charts"
)

func TestNewWithOptions(t *testing.T) {
	bg := color.RGBA{R: 1, G: 2, B: 3, A: 255}
	m := NewWithOptions(
		WithKittyID(99),
		WithBackground(bg),
		WithCellSize(8, 16),
		WithTheme("dark"),
		WithLineChartOption(charts.NewLineChartOptionWithData([][]float64{{1, 2, 3}})),
	)
	if m.cellW != 8 || m.cellH != 16 {
		t.Errorf("cellSize = %dx%d", m.cellW, m.cellH)
	}
	if m.theme != "dark" {
		t.Errorf("theme = %q", m.theme)
	}
	if m.recipe == nil {
		t.Error("recipe not set by WithLineChartOption")
	}
}
