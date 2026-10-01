package spec

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
	"github.com/charmbracelet/x/ansi"
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

// ---- labels and filled cells ----

// plain strips the styling from a model's view, one string per row.
func plainRows(m *heatmap.Model) []string {
	return strings.Split(ansi.Strip(m.View()), "\n")
}

func labelledHeatSpec() Spec {
	return Spec{
		Type: ChartTypeHeatmap, Width: 40, Height: 14,
		XAxis: XAxis{Labels: []string{"09", "12", "15"}},
		YAxis: YAxis{Labels: []string{"Mon", "Tuesday", "Wed"}},
		Heat: &HeatData{Cells: []HeatCell{
			{X: 0, Y: 0, Z: 1}, {X: 1, Y: 0, Z: 5}, {X: 2, Y: 0, Z: 9},
			{X: 0, Y: 1, Z: 3}, {X: 1, Y: 1, Z: 7}, {X: 2, Y: 1, Z: 2},
			{X: 0, Y: 2, Z: 4}, {X: 1, Y: 2, Z: 6}, {X: 2, Y: 2, Z: 8},
		}},
		Theme: Theme{Gradient: []string{"#000044", "#ff4400"}},
	}
}

// y_axis.labels[0] is the first row of the chart, the one at the top, as in
// flint's own heatmaps: row labels read downwards in label order.
func TestBuildHeatmapRowLabelsReadTopDown(t *testing.T) {
	got, err := Build(labelledHeatSpec())
	if err != nil {
		t.Fatal(err)
	}
	m := got.(*heatmap.Model)
	rows := plainRows(m)
	order := []string{"Mon", "Tuesday", "Wed"}
	last := -1
	for _, label := range order {
		found := -1
		for i, r := range rows {
			if strings.Contains(r, label) {
				found = i
				break
			}
		}
		if found < 0 {
			t.Fatalf("row label %q missing from the view:\n%s", label, strings.Join(rows, "\n"))
		}
		if found <= last {
			t.Fatalf("row label %q on row %d, not below the previous label (row %d):\n%s", label, found, last, strings.Join(rows, "\n"))
		}
		last = found
	}
	// the column labels sit under the grid
	under := rows[len(rows)-2]
	for _, label := range []string{"09", "12", "15"} {
		if !strings.Contains(under, label) {
			t.Fatalf("column label %q missing from the row under the grid %q", label, under)
		}
	}
}

// Y=0 is the first label's row, so the first row of cells is the top one,
// whatever the labels: the data cell at Y=0 is drawn above the cell at Y=1.
func TestBuildHeatmapFirstRowIsOnTop(t *testing.T) {
	s := heatSpec() // Y=0 holds 1, 5, 9; Y=1 holds 3, 7, 2
	s.Theme.Gradient = []string{"#000000", "#ffffff"}
	s.Heat.MinValue, s.Heat.MaxValue = f64(0), f64(10)
	got, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := got.(*heatmap.Model)
	topRow, bottomRow := -1, -1
	for y := 0; y < m.Height(); y++ {
		for x := 0; x < m.Width(); x++ {
			if c := m.Canvas.Cell(canvas.Point{X: x, Y: y}).Style.GetBackground(); isColour(c) {
				if topRow < 0 {
					topRow = y
				}
				bottomRow = y
			}
		}
	}
	// the cell at X=2 is 9 (Y=0) over 2 (Y=1): the lighter colour is in the upper block
	x := m.Width() - 1
	for ; x >= 0; x-- {
		if isColour(m.Canvas.Cell(canvas.Point{X: x, Y: topRow}).Style.GetBackground()) {
			break
		}
	}
	upper := m.Canvas.Cell(canvas.Point{X: x, Y: topRow}).Style.GetBackground()
	lower := m.Canvas.Cell(canvas.Point{X: x, Y: bottomRow}).Style.GetBackground()
	if lum(upper) <= lum(lower) {
		t.Fatalf("Y=0 (value 9) should be drawn above Y=1 (value 2); upper %v, lower %v", upper, lower)
	}
}

func isColour(c color.Color) bool {
	_, none := c.(lipgloss.NoColor)
	return c != nil && !none
}

func lum(c color.Color) uint32 {
	r, g, b, _ := c.RGBA()
	return r + g + b
}

// A heatmap is a grid of filled blocks that tile the plot, not isolated cells.
func TestBuildHeatmapCellsTileThePlot(t *testing.T) {
	s := heatSpec()
	got, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := got.(*heatmap.Model)
	coloured := 0
	for y := 0; y < m.Height(); y++ {
		for x := 0; x < m.Width(); x++ {
			if isColour(m.Canvas.Cell(canvas.Point{X: x, Y: y}).Style.GetBackground()) {
				coloured++
			}
		}
	}
	if want := m.GraphWidth() * m.GraphHeight(); coloured != want {
		t.Fatalf("%d coloured cells, want the whole %dx%d plot (%d)", coloured, m.GraphWidth(), m.GraphHeight(), want)
	}
}

// Without labels the plot has no margin or label rows beyond the cells' own.
func TestBuildHeatmapUnlabelledHasNoLabelText(t *testing.T) {
	got, err := Build(heatSpec())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range plainRows(got.(*heatmap.Model)) {
		if strings.TrimSpace(r) != "" {
			t.Fatalf("an unlabelled heatmap drew text %q", r)
		}
	}
}

func TestBuildHeatmapLabelsMatchMatrixRows(t *testing.T) {
	s := labelledHeatSpec()
	want, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	s.Heat = &HeatData{Matrix: [][]float64{{1, 5, 9}, {3, 7, 2}, {4, 6, 8}}}
	got, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if a, b := strings.Join(plainRows(got.(*heatmap.Model)), "\n"), strings.Join(plainRows(want.(*heatmap.Model)), "\n"); a != b {
		t.Fatalf("matrix and cell forms render differently:\n%s\n---\n%s", a, b)
	}
}

func TestBuildHeatmapNegativeIndexIsAnError(t *testing.T) {
	s := heatSpec()
	s.Heat.Cells = append(s.Heat.Cells, HeatCell{X: -1, Y: 0, Z: 1})
	if _, err := Build(s); err == nil || !strings.Contains(err.Error(), "non-negative") {
		t.Fatalf("expected a non-negative index error, got %v", err)
	}
}

// Labels beyond the data still get their rows: a label with no cells is an empty row.
func TestBuildHeatmapLabelsLongerThanData(t *testing.T) {
	s := labelledHeatSpec()
	s.YAxis.Labels = append(s.YAxis.Labels, "Thu")
	got, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if rows := strings.Join(plainRows(got.(*heatmap.Model)), "\n"); !strings.Contains(rows, "Thu") {
		t.Fatalf("label with no data missing:\n%s", rows)
	}
}
