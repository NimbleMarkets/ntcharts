// ntcharts - Copyright (c) 2026 Neomantra Corp.

package heatmap

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"

	"charm.land/lipgloss/v2"
)

// a 24-step ramp so each distinct value in the tests below (at most 24 cells,
// value 0..cols*rows) lands on its own colour
var testScale = func() []color.Color {
	var cs []color.Color
	for i := 0; i < 24; i++ {
		cs = append(cs, lipgloss.Color(fmt.Sprintf("#%02x0000", i*10)))
	}
	return cs
}()

// bg returns the background colour of a canvas cell, or nil if it has none
// (lipgloss reports an unset background as NoColor, not nil).
func bg(m *Model, x, y int) color.Color {
	c := m.Canvas.Cell(canvas.Point{X: x, Y: y}).Style.GetBackground()
	if _, none := c.(lipgloss.NoColor); none {
		return nil
	}
	return c
}

// rowText returns the runes drawn on canvas row y, with unset cells as spaces.
func rowText(m *Model, y int) string {
	var b strings.Builder
	for x := 0; x < m.Width(); x++ {
		r := m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune
		if r == 0 {
			r = ' '
		}
		b.WriteRune(r)
	}
	return b.String()
}

// grid fills a cols x rows index grid whose value is x+y*cols, Y up.
func grid(cols, rows int) []HeatPoint {
	var pts []HeatPoint
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pts = append(pts, NewHeatPointInt(x, y, float64(x+y*cols)))
		}
	}
	return pts
}

func gridModel(w, h, cols, rows int, opts ...Option) Model {
	opts = append([]Option{
		WithColorScale(testScale),
		WithValueRange(0, float64(cols*rows)),
		WithCellSize(1, 1),
	}, opts...)
	m := New(w, h, opts...)
	// expected range first, then the displayed range (a SetView*Range outside the expected one fails)
	m.SetXYRange(-0.5, float64(cols)-0.5, -0.5, float64(rows)-0.5)
	m.SetViewXYRange(-0.5, float64(cols)-0.5, -0.5, float64(rows)-0.5)
	m.PushAll(grid(cols, rows))
	m.Draw()
	return m
}

// Without a cell size each point is a single canvas cell, as before.
func TestPointsStaySingleCellsWithoutCellSize(t *testing.T) {
	m := New(40, 12, WithColorScale(testScale), WithValueRange(0, 20))
	m.PushAll(grid(5, 4))
	m.Draw()
	coloured := 0
	for y := 0; y < m.Height(); y++ {
		for x := 0; x < m.Width(); x++ {
			if bg(&m, x, y) != nil {
				coloured++
			}
		}
	}
	if coloured != 20 {
		t.Fatalf("%d coloured cells, want exactly one per point (20)", coloured)
	}
}

// With a cell size the cells tile the graph area: no gaps inside it,
// nothing outside it, and every canvas cell of a block shares its colour.
func TestCellSizeTilesTheGraph(t *testing.T) {
	m := gridModel(40, 12, 5, 4)
	left := m.Origin().X + 1
	if m.YStep() == 0 {
		left = m.Origin().X
	}
	bottom := m.Origin().Y - 1
	top := bottom - m.GraphHeight() + 1
	for y := 0; y < m.Height(); y++ {
		for x := 0; x < m.Width(); x++ {
			inside := x >= left && x < left+m.GraphWidth() && y >= top && y <= bottom
			if got := bg(&m, x, y) != nil; got != inside {
				t.Fatalf("cell (%d,%d): coloured=%v, want %v (graph x %d..%d, y %d..%d)",
					x, y, got, inside, left, left+m.GraphWidth()-1, top, bottom)
			}
		}
	}
	// five columns across graphWidth cells: block widths differ by at most one
	widths := map[int]int{}
	run := 1
	for x := left + 1; x <= left+m.GraphWidth(); x++ {
		if x < left+m.GraphWidth() && bg(&m, x, bottom) == bg(&m, x-1, bottom) {
			run++
			continue
		}
		widths[run]++
		run = 1
	}
	total := 0
	for w, n := range widths {
		total += n
		if w < m.GraphWidth()/5 || w > m.GraphWidth()/5+1 {
			t.Fatalf("block width %d outside %d..%d: %v", w, m.GraphWidth()/5, m.GraphWidth()/5+1, widths)
		}
	}
	if total != 5 {
		t.Fatalf("found %d blocks across the bottom row, want 5: %v", total, widths)
	}
}

// Y=0 is the bottom row of blocks, X=0 the left column.
func TestCellOrientation(t *testing.T) {
	m := gridModel(40, 12, 5, 4)
	left := m.Origin().X + 1
	bottom := m.Origin().Y - 1
	top := bottom - m.GraphHeight() + 1
	right := left + m.GraphWidth() - 1
	// values: (0,0)=0 lowest -> first colour; (4,3)=19 highest -> int(19/20*24) = 22
	if got, want := bg(&m, left, bottom), color.Color(testScale[0]); got != want {
		t.Fatalf("bottom-left cell is %v, want the lowest colour %v", got, want)
	}
	if got, want := bg(&m, right, top), color.Color(testScale[22]); got != want {
		t.Fatalf("top-right cell is %v, want the highest colour %v", got, want)
	}
}

func TestYLabelsAreCentredOnTheirRowsAndSizeTheMargin(t *testing.T) {
	m := gridModel(40, 12, 3, 3, WithYLabels([]string{"Mon", "Tuesday", "Wed"}))
	// the margin is the longest label
	if got := m.Origin().X; got != len("Tuesday") {
		t.Fatalf("origin X = %d, want %d (the longest label)", got, len("Tuesday"))
	}
	_, bottom := m.graphBox()
	for i, label := range []string{"Mon", "Tuesday", "Wed"} {
		// Y=i counts up from the bottom; the label sits on a row of its own block
		_, _, r0, r1 := m.cellSpans(0, float64(i), 1, 1)
		found := false
		for r := r0; r < r1; r++ {
			line := rowText(&m, bottom-r)
			want := strings.Repeat(" ", m.Origin().X-len(label)) + label
			if strings.HasPrefix(line, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("label %q (Y=%d, block rows %d..%d from the bottom) not drawn right-aligned in its block", label, i, r0, r1-1)
		}
	}
}

func TestXLabelsSitUnderTheirColumns(t *testing.T) {
	m := gridModel(40, 12, 3, 2, WithXLabels([]string{"09", "12", "15"}))
	left, bottom := m.graphBox()
	line := rowText(&m, bottom+1)
	for i, label := range []string{"09", "12", "15"} {
		c0, _, _, _ := m.cellSpans(float64(i), 0, 1, 1)
		if got := line[left+c0 : left+c0+len(label)]; got != label {
			t.Errorf("under column %d (canvas x %d) found %q, want %q; row is %q", i, left+c0, got, label, line)
		}
	}
}

// More categories than rows: labels that would share a row are dropped, never overdrawn.
func TestDenseYLabelsDoNotCollide(t *testing.T) {
	var labels []string
	for i := 0; i < 30; i++ {
		labels = append(labels, string(rune('a'+i%26))+string(rune('0'+i/26)))
	}
	m := gridModel(30, 8, 3, 30, WithYLabels(labels))
	seen := 0
	for y := 0; y < m.Height(); y++ {
		if strings.TrimSpace(rowText(&m, y)[:m.Origin().X]) != "" {
			seen++
		}
	}
	if seen == 0 || seen > m.GraphHeight() {
		t.Fatalf("%d labelled rows for %d graph rows", seen, m.GraphHeight())
	}
}

func TestLabelsDefineTheGridRange(t *testing.T) {
	m := New(40, 12, WithXLabels([]string{"a", "b", "c", "d"}), WithYLabels([]string{"p", "q"}))
	if min, max := m.ViewMinX(), m.ViewMaxX(); min != -0.5 || max != 3.5 {
		t.Fatalf("X range %v..%v, want -0.5..3.5", min, max)
	}
	if min, max := m.ViewMinY(), m.ViewMaxY(); min != -0.5 || max != 1.5 {
		t.Fatalf("Y range %v..%v, want -0.5..1.5", min, max)
	}
}

func TestFilledCellsOutsideViewAreNotDrawn(t *testing.T) {
	for _, pt := range []HeatPoint{
		NewHeatPoint(-100, 1, 1), NewHeatPoint(100, 1, 1),
		NewHeatPoint(1, -100, 1), NewHeatPoint(1, 100, 1),
		NewHeatPoint(-1, 1, 1), NewHeatPoint(3, 1, 1),
	} {
		m := New(20, 8, WithCellSize(1, 1), WithColorScale(testScale),
			WithValueRange(0, 1), WithXLabels([]string{"A", "B", "C"}),
			WithYLabels([]string{"a", "b", "c"}))
		m.DrawPoint(pt)
		for y := 0; y < m.Height(); y++ {
			for x := 0; x < m.Width(); x++ {
				if bg(&m, x, y) != nil {
					t.Fatalf("offscreen point %v painted cell (%d,%d)", pt, x, y)
				}
			}
		}
	}
}

func TestPartiallyVisibleFilledCellIsClipped(t *testing.T) {
	m := New(20, 8, WithCellSize(1, 1), WithColorScale(testScale),
		WithValueRange(0, 1), WithXLabels([]string{"A", "B", "C"}),
		WithYLabels([]string{"a", "b", "c"}))
	m.DrawPoint(NewHeatPoint(-0.75, 1, 1))
	left, bottom := m.graphBox()
	coloured := 0
	for y := 0; y < m.Height(); y++ {
		for x := 0; x < m.Width(); x++ {
			if bg(&m, x, y) == nil {
				continue
			}
			coloured++
			// A quarter data unit remains visible: at this chart size it
			// rounds to the first two canvas columns.
			if x < left || x >= left+2 || y > bottom || y <= bottom-m.GraphHeight() {
				t.Fatalf("partially visible cell painted outside its clipped columns: (%d,%d)", x, y)
			}
		}
	}
	if coloured == 0 {
		t.Fatal("partially visible cell was discarded")
	}
}

func TestShortHeatmapReservesLongestRowLabel(t *testing.T) {
	m := New(30, 12, WithCellSize(1, 1), WithXLabels([]string{"A"}),
		WithYLabels([]string{"a", "Tuesday", "c"}))
	m.Resize(30, 5)
	m.Draw()
	if got := m.Origin().X; got != len("Tuesday") {
		t.Fatalf("margin = %d, want %d", got, len("Tuesday"))
	}
	if !strings.Contains(rowText(&m, 1), "Tuesday") {
		t.Fatalf("middle row label missing: %q", rowText(&m, 1))
	}
}
