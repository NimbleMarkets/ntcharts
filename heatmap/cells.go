// ntcharts - Copyright (c) 2026 Neomantra Corp.

package heatmap

// File contains filled cells and axis labels for index-grid heatmaps: data
// whose X and Y are cell indices, drawn as a grid of coloured blocks with a
// name for each row and column.

import (
	"image/color"
	"math"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

// graphBox returns the canvas cell of the graphing area's left column and
// bottom row. The area is GraphWidth() columns by GraphHeight() rows: it
// runs right from left and up from bottom. The Y axis column and the X axis
// rows of the underlying linechart are not part of it.
func (m *Model) graphBox() (left, bottom int) {
	left, bottom = m.Origin().X, m.Origin().Y
	if m.YStep() > 0 {
		left++ // the column the Y axis would occupy
	}
	if m.XStep() > 0 {
		bottom-- // the row the X axis would occupy
	}
	return left, bottom
}

// span returns the half-open range of graph cells [from, to) that the data
// interval [lo, hi] covers on an axis showing min..max over n cells. Both
// edges round to the nearest cell boundary, so adjacent data cells tile with
// neither gap nor overlap, and every data cell covers at least one cell.
func span(lo, hi, min, max float64, n int) (from, to int) {
	if max <= min || n <= 0 || hi <= min || lo >= max {
		return 0, 0
	}
	f := func(v float64) int {
		return clamp(int(math.Floor((v-min)/(max-min)*float64(n)+0.5)), 0, n)
	}
	from, to = f(lo), f(hi)
	if to <= from {
		to = min1(from+1, n)
		from = to - 1
	}
	return from, to
}

func min1(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// cellSpans returns the graph columns and rows, counted from the left and
// from the bottom, covered by the data cell centred on (x, y) with the
// given data-unit size.
func (m *Model) cellSpans(x, y, w, h float64) (c0, c1, r0, r1 int) {
	c0, c1 = span(x-w/2, x+w/2, m.ViewMinX(), m.ViewMaxX(), m.GraphWidth())
	r0, r1 = span(y-h/2, y+h/2, m.ViewMinY(), m.ViewMaxY(), m.GraphHeight())
	return
}

// fillCell colours every canvas cell covered by the data cell at pt.
func (m *Model) fillCell(pt HeatPoint, c color.Color) {
	left, bottom := m.graphBox()
	c0, c1, r0, r1 := m.cellSpans(pt.X, pt.Y, m.cellW, m.cellH)
	for r := r0; r < r1; r++ {
		for col := c0; col < c1; col++ {
			cp := canvas.Point{X: left + col, Y: bottom - r}
			old := m.Canvas.GetCellStyle(cp)
			if old == nil { // out of bounds
				continue
			}
			m.Canvas.SetCellStyle(cp, (*old).Background(c))
		}
	}
}

// unit returns the data-unit cell size labels are placed with: the cell
// size, or one data unit per cell when drawing single-cell points.
func (m *Model) unit() (w, h float64) {
	w, h = m.cellW, m.cellH
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}
	return w, h
}

// drawLabels writes the row labels right-aligned in the margin left of the
// graph, each on the middle row of its cells, and the column labels on the
// row under the graph, each starting at the left edge of its cells. A label
// that would land on the same row as one already drawn, or run into the
// previous column label, is left out rather than overdrawn.
func (m *Model) drawLabels() {
	w, h := m.unit()
	left, bottom := m.graphBox()

	used := map[int]bool{}
	for i, label := range m.yLabels {
		if label == "" {
			continue
		}
		_, _, r0, r1 := m.cellSpans(0, float64(i), w, h)
		if r0 == r1 {
			continue
		}
		row := bottom - (r0+r1-1)/2
		x := m.Origin().X - len(label)
		if used[row] || x < 0 || row < 0 {
			continue
		}
		used[row] = true
		m.Canvas.SetStringWithStyle(canvas.Point{X: x, Y: row}, label, m.LabelStyle)
	}

	// column labels need the row under the graph, which the X axis rows provide
	if m.XStep() == 0 {
		return
	}
	row := bottom + 1
	end := left - 1 // last column used by a drawn label, plus the gap before the next
	for i, label := range m.xLabels {
		if label == "" {
			continue
		}
		c0, c1, _, _ := m.cellSpans(float64(i), 0, w, h)
		if c0 == c1 {
			continue
		}
		x := left + c0
		if x <= end || x+len(label) > m.Width() {
			continue
		}
		m.Canvas.SetStringWithStyle(canvas.Point{X: x, Y: row}, label, m.LabelStyle)
		end = x + len(label)
	}
}

// SetXLabels names the cell columns; see WithXLabels.
func (m *Model) SetXLabels(labels []string) {
	m.xLabels = labels
	if len(labels) == 0 {
		return
	}
	m.AutoMinX, m.AutoMaxX = false, false
	m.SetXRange(-0.5, float64(len(labels))-0.5)
	m.Model.SetViewXRange(-0.5, float64(len(labels))-0.5)
	m.UpdateGraphSizes()
}

// SetYLabels names the cell rows; see WithYLabels.
func (m *Model) SetYLabels(labels []string) {
	m.yLabels = labels
	if len(labels) == 0 {
		return
	}
	m.AutoMinY, m.AutoMaxY = false, false
	m.SetYRange(-0.5, float64(len(labels))-0.5)
	m.Model.SetViewYRange(-0.5, float64(len(labels))-0.5)
	// The linechart samples its formatter to size the margin. Heatmap row
	// labels are placed independently of numeric ticks, so reserve the longest
	// label at every sample, including after resizing or changing axis steps.
	longest := ""
	for _, label := range labels {
		if len(label) > len(longest) {
			longest = label
		}
	}
	m.YLabelFormatter = func(_ int, _ float64) string { return longest }
	m.UpdateGraphSizes()
}

// XLabels returns the cell column labels.
func (m *Model) XLabels() []string { return m.xLabels }

// YLabels returns the cell row labels.
func (m *Model) YLabels() []string { return m.yLabels }
