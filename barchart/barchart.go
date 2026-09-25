// ntcharts - Copyright (c) 2024 Neomantra Corp.

// Package barchart implements a canvas that displays a bar chart
// with bars going either horizontally or vertically.
package barchart

import (
	"math"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/buffer"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/graph"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// BarValue contain bar segment name, value and style for drawing.
// A negative Value draws away from the axis in the opposite direction
// of positive values: within one bar, positive segments stack up (or
// right) from zero and negative segments stack down (or left) from zero.
type BarValue struct {
	Name  string
	Value float64
	Style lipgloss.Style
}

// BarData contains a label for the bar and a list of BarValues.
// If displaying vertical bars, the length of the label string will be
// bounded by the width of each bar.
// If displaying horizontal bars, the length of the label string
// will not be bounded and will affect where the vertical axis line starts.
type BarData struct {
	Label  string
	Values []BarValue
}

type dataSet struct {
	bd  BarData
	buf *buffer.Float64ScaleBuffer // contains bar values
}

// Model contains state of a barchart
type Model struct {
	Canvas       canvas.Model
	AxisStyle    lipgloss.Style // style applied when drawing axis
	LabelStyle   lipgloss.Style // style applied when drawing axis labels
	AutoMaxValue bool           // whether to automatically set max value when adding data
	AutoBarWidth bool           // whether to automatically set bar width

	showAxis   bool         // whether to display axis and labels
	horizontal bool         // whether to display bars horizontally
	origin     canvas.Point //  start of axis line on canvas for graphing area

	barWidth   int   // width of each bar on the canvas
	barGap     int   // number of empty spaces between each bar on the canvas
	barIndices []int // size of graphing area, index value is which bar will be drawn
	labelWidth int   // columns reserved for labels when horizontal with axis

	max      float64    // expected maximum data value (>= 0)
	min      float64    // expected minimum data value (<= 0); 0 unless data goes negative
	posCells int        // cells of graph length for values above zero
	negCells int        // cells of graph length for values below zero
	sf       float64    // scale factor
	data     []*dataSet // each index is a unique bar

	zoneManager *zone.Manager // provides mouse functionality
	zoneID      string
}

// New returns a barchart Model initialized with given width, height
// and various options.
// By default, barchart will automatically scale bar to new maximum
// (and minimum, when values are negative) data values,
// bar width to fill up the canvas and have one gap between bars, and display
// bars vertically. When any value is negative the axis moves away from the
// edge so that bars can extend both above and below (or right and left of) it.
// If the given data values are too small compared to the max value,
// then it is possible that the rendering of the bars will not be accurate
// in terms of proportions due to the limitations of the block element runes.
func New(w, h int, opts ...Option) Model {
	m := Model{
		AutoMaxValue: true,
		AutoBarWidth: true,
		Canvas:       canvas.New(w, h),
		showAxis:     true,
		horizontal:   false,
		origin:       canvas.Point{X: 0, Y: h - 2},
		barWidth:     1,
		barGap:       1,
		barIndices:   make([]int, w),
		max:          1,
		sf:           1,
		data:         []*dataSet{},
	}
	m.resetScale()
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// newDataSet returns a new initialize *dataSet.
func (m *Model) newDataSet(lv BarData) *dataSet {
	ds := &dataSet{
		bd:  lv,
		buf: buffer.NewFloat64ScaleBuffer(0, m.sf),
	}
	for _, v := range lv.Values {
		ds.buf.Push(v.Value)
	}
	return ds
}

// graphLength returns the number of cells available for bars along the
// value direction: canvas height less the axis and label rows for vertical
// bars, or canvas width less the label columns and axis column for
// horizontal bars.
func (m *Model) graphLength() int {
	var l int
	if m.horizontal {
		l = m.Canvas.Width() - m.labelWidth
		if m.showAxis {
			l--
		}
	} else {
		l = m.Canvas.Height()
		if m.showAxis {
			l -= 2
		}
	}
	return max(l, 0)
}

// axisOffset returns 1 when an axis line occupies a cell between the
// positive and negative bar regions, otherwise 0.
func (m *Model) axisOffset() int {
	if m.showAxis {
		return 1
	}
	return 0
}

// resetScale splits the graph length between values above and below zero,
// recomputes the scale factor so both extremes fit, places the origin
// (the axis cell, or the first positive cell when no axis is shown), and
// rescales all existing data sets.
// With only one drawable cell, the larger side receives it (positive wins
// ties); the other side is clipped without reducing the shared scale to zero.
func (m *Model) resetScale() {
	length := m.graphLength()
	m.posCells, m.negCells = length, 0
	if rng := m.max - m.min; m.min < 0 && rng > 0 {
		m.posCells = int(math.Round(float64(length) * m.max / rng))
		if m.max > 0 && m.posCells == 0 && length > 1 {
			m.posCells = 1
		}
		m.negCells = length - m.posCells
		if m.negCells == 0 && length > 1 {
			m.negCells = 1
			m.posCells--
		}
	}
	switch {
	case m.min < 0 && m.max > 0 && m.posCells > 0 && m.negCells > 0:
		m.sf = math.Min(float64(m.posCells)/m.max, float64(m.negCells)/-m.min)
	case m.min < 0 && m.negCells > 0:
		m.sf = float64(m.negCells) / -m.min
	case m.max > 0:
		m.sf = float64(m.posCells) / m.max
	default:
		m.sf = 0
	}
	if m.horizontal {
		m.origin = canvas.Point{X: m.labelWidth + m.negCells, Y: 0}
	} else {
		m.origin = canvas.Point{X: 0, Y: m.posCells}
	}
	for _, ds := range m.data {
		ds.buf.SetScale(m.sf)
	}
}

// resetOrigin recomputes the space reserved for labels. The origin itself
// is placed by resetScale, which callers invoke next.
func (m *Model) resetOrigin() {
	m.labelWidth = 0
	if m.showAxis && m.horizontal {
		for _, ds := range m.data {
			if lw := len(ds.bd.Label); lw > m.labelWidth {
				m.labelWidth = lw
			}
		}
	}
}

// resetBarWidth will recalculate the width of each bar such that
// the bars will fill up the graph if AutoBarWidth is enabled.
func (m *Model) resetBarWidth() {
	if m.AutoBarWidth {
		graphSize := m.Canvas.Width()
		if m.horizontal {
			graphSize = m.Canvas.Height()
		}
		dLen := len(m.data)
		gaps := (dLen - 1) * m.barGap // total space used by gaps
		size := graphSize - gaps      // total available space for bars
		if dLen > 0 {
			m.barWidth = size / dLen // each bar width
		}
	}
}

// resetBarIndices will reset the barIndices
// containing with bar to display for that row or column.
// If AutoBarWidth is enabled, then will recompute
// the bar widths such that the bars are as wide as possible
// to fit the graph with the given bar gap.
// It is possible that the bar widths and bar gap is such
// that the bars do not all fit on the graph.
func (m *Model) resetBarIndices() {
	m.resetBarWidth()
	if m.horizontal {
		if len(m.barIndices) != m.Canvas.Height() {
			m.barIndices = make([]int, m.Canvas.Height())
		}
	} else {
		if len(m.barIndices) != m.Canvas.Width() {
			m.barIndices = make([]int, m.Canvas.Width())
		}
	}
	for i := range m.barIndices { // reset indices
		m.barIndices[i] = -1 //indicates no bar
	}
	barIdx := 0
	barCount := len(m.data) // number of bars to display
	end := len(m.barIndices)
	for i := 0; i < end; i += m.barWidth {
		j := i
		jEnd := i + m.barWidth
		for j < jEnd {
			m.barIndices[j] = barIdx
			j++
			if j >= end {
				break
			}
		}
		barIdx++
		if barIdx >= barCount {
			break
		}
		i += m.barGap
	}
}

// Resize will change barchart display width and height.
// Existing data values will be updated to new scaling.
func (m *Model) Resize(w, h int) {
	m.Canvas.Resize(w, h)
	m.Canvas.ViewWidth = w
	m.Canvas.ViewHeight = h
	m.resetOrigin()
	m.resetScale()
	m.resetBarIndices()
}

// Clear will reset barchart canvas and data.
func (m *Model) Clear() {
	m.Canvas.Clear()
	m.data = []*dataSet{}
	if m.AutoMaxValue {
		m.max = 1
		m.min = 0
		m.resetScale()
	}
}

// SetZoneManager enables mouse functionality
// by setting a bubblezone.Manager to the linechart.
// To disable mouse functionality after enabling, call SetZoneManager on nil.
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.zoneManager = zm
	if (zm != nil) && (m.zoneID == "") {
		m.zoneID = zm.NewPrefix()
	}
}

// ZoneManager will return linechart zone Manager.
func (m *Model) ZoneManager() *zone.Manager {
	return m.zoneManager
}

// ZoneID will return linechart zone ID used by zone Manager.
func (m *Model) ZoneID() string {
	return m.zoneID
}

// Width returns barchart width.
func (m *Model) Width() int {
	return m.Canvas.Width()
}

// Height returns barchart height.
func (m *Model) Height() int {
	return m.Canvas.Height()
}

// Data returns a copy of the bar data currently stored in the model, in the
// order the bars were pushed. Modifying the returned slice does not affect
// the model's internal state.
func (m *Model) Data() []BarData {
	out := make([]BarData, len(m.data))
	for i, ds := range m.data {
		out[i] = ds.bd
		out[i].Values = append([]BarValue(nil), ds.bd.Values...)
	}
	return out
}

// MaxValue returns expected maximum data value.
func (m *Model) MaxValue() float64 {
	return m.max
}

// MinValue returns expected minimum data value.
// It is 0 unless negative values were pushed or SetMin was called.
func (m *Model) MinValue() float64 {
	return m.min
}

// Scale returns data scaling factor.
func (m *Model) Scale() float64 {
	return m.sf
}

// BarGap returns number of empty spaces between bars.
func (m *Model) BarGap() int {
	return m.barGap
}

// BarWidth returns the bar width drawn for each bar.
func (m *Model) BarWidth() int {
	return m.barWidth
}

// ShowAxis returns whether drawing axis and labels
// on to the barchart canvas.
func (m *Model) ShowAxis() bool {
	return m.showAxis
}

// Horizontal returns whether displaying bars horizontally.
func (m *Model) Horizontal() bool {
	return m.horizontal
}

// BarDataFromPoint returns a possible BarData containing
// all BarValues drawn for the rune on the barchart canvas
// at the given Point. Points on the positive side of the axis
// match positive segments; points on the negative side match
// negative segments.
func (m *Model) BarDataFromPoint(p canvas.Point) (r BarData) {
	if p.X < 0 || p.X >= m.Canvas.Width() || p.Y < 0 || p.Y >= m.Canvas.Height() {
		return
	}
	var bIdx, posIdx, negIdx int // which bar; distance into the positive / negative stack
	if m.horizontal {
		bIdx = p.Y
		posIdx = p.X - (m.origin.X + m.axisOffset())
		negIdx = (m.origin.X - 1) - p.X
	} else {
		bIdx = p.X
		posIdx = (m.origin.Y - 1) - p.Y
		negIdx = p.Y - (m.origin.Y + m.axisOffset())
	}
	if bIdx < 0 || bIdx >= len(m.barIndices) {
		return
	}
	idx := m.barIndices[bIdx]
	if idx < 0 || idx >= len(m.data) {
		return
	}
	r.Label = m.data[idx].bd.Label
	negative := posIdx < 0
	want := posIdx
	limit := m.posCells
	if negative {
		want = negIdx
		limit = m.negCells
	}
	if want < 0 || want >= limit { // axis, labels, or outside the drawable region
		return
	}
	// walk the scaled values on the selected side of the axis and
	// return all values that can be drawn onto that point on the canvas
	v := m.data[idx].bd.Values
	sv := m.data[idx].buf.ReadAll()
	var sum float64
	for i, f := range sv {
		if (f < 0) != negative {
			continue
		}
		newSum := sum + math.Abs(f)
		// Match the drawers' nearest-eighth rounding and exclude the cell
		// immediately beyond an exact integer endpoint.
		start := math.Round(sum*8) / 8
		end := math.Round(newSum*8) / 8
		if start < end && start < float64(want+1) && float64(want) < end {
			r.Values = append(r.Values, v[i])
		}
		sum = newSum
	}
	return
}

// SetHorizontal will either enable or disable drawing
// bars horizontally on to the barchart canvas.
func (m *Model) SetHorizontal(b bool) {
	m.horizontal = b
	m.resetOrigin()
	m.resetScale()
	m.resetBarIndices()
}

// SetShowAxis will either enable or disable drawing
// axis and labels on to the barchart canvas.
func (m *Model) SetShowAxis(b bool) {
	m.showAxis = b
	m.resetOrigin()
	m.resetScale()
}

// SetBarWidth will set the bar width drawn of each bar.
// If AutoBarWidth is enabled, then this method will do nothing.
func (m *Model) SetBarWidth(w int) {
	if m.AutoBarWidth {
		return
	}
	m.barWidth = w
	if m.barWidth < 1 {
		m.barWidth = 1
	}
	m.resetBarIndices()
}

// SetBarGap will set the number of empty spaces between each bar.
func (m *Model) SetBarGap(g int) {
	m.barGap = g
	if m.barGap < 0 {
		m.barGap = 0
	}
	m.resetBarIndices()
}

// SetMax will update the expected maximum values and scale factor.
// Existing values will be updated to new scaling.
func (m *Model) SetMax(f float64) {
	m.max = f
	m.resetScale()
}

// SetMin will update the expected minimum value and scale factor.
// Values above 0 are clamped to 0 so the axis always sits at zero.
// Existing values will be updated to new scaling.
func (m *Model) SetMin(f float64) {
	m.min = math.Min(f, 0)
	m.resetScale()
}

// Push adds given BarData to barchart data set.
// Positive values stack away from the axis in one direction and negative
// values in the other. If AutoMaxValue is enabled, the expected maximum
// and minimum grow to fit the bar's positive and negative sums.
// Data will be scaled using the expected range and barchart size.
func (m *Model) Push(lv BarData) {
	var posSum, negSum float64 // assumes no overflow
	for _, v := range lv.Values {
		if v.Value < 0 {
			negSum += v.Value
		} else {
			posSum += v.Value
		}
	}
	if m.AutoMaxValue {
		rescale := false
		if posSum > m.max {
			m.max = posSum
			rescale = true
		}
		if negSum < m.min {
			m.min = negSum
			rescale = true
		}
		if rescale {
			m.resetScale()
		}
	}
	m.data = append(m.data, m.newDataSet(lv))
	m.resetBarIndices()
}

// PushAll adds all data values in []BarData to barchart data set.
// See Push for how negative values are handled.
// Data will be scaled using the expected range and barchart size.
func (m *Model) PushAll(lv []BarData) {
	for _, v := range lv {
		m.Push(v)
	}
}

// Draw will display the the scaled data values as bars on to the barchart canvas.
// The order of the bars will be displayed from left to right, or from top to bottom
// in the same order as the data was inserted.
func (m *Model) Draw() {
	m.Canvas.Clear()
	m.drawAxisAndLabels()
	m.drawBars()
}

// drawBars draws each bar's positive segments away from the axis in one
// direction (up, or right) and its negative segments in the other (down,
// or left). Vertical bars are laid out left to right and horizontal bars
// top to bottom, in insertion order.
func (m *Model) drawBars() {
	dLen := len(m.data)
	for i, b := range m.barIndices {
		if b < 0 || b >= dLen {
			continue
		}
		v := m.data[b].buf.ReadAll()
		var pos, neg []int // indices of segments on each side of the axis
		for j, f := range v {
			if f < 0 {
				neg = append(neg, j)
			} else {
				pos = append(pos, j)
			}
		}
		m.drawStack(i, pos, v, m.data[b].bd.Values, false)
		m.drawStack(i, neg, v, m.data[b].bd.Values, true)
	}
}

// drawStack draws one side of a bar. Segments are drawn from the outermost
// (the full stacked length, in the last segment's style) to the innermost,
// so each shorter draw overlays the previous one and the boundary cells
// blend the two adjacent colors.
func (m *Model) drawStack(i int, idx []int, v []float64, s []BarValue, negative bool) {
	limit := float64(m.posCells)
	if negative {
		limit = float64(m.negCells)
	}
	var sum float64
	for _, j := range idx {
		sum += math.Abs(v[j])
	}
	last := len(idx) - 1
	for k := last; k >= 0; k-- {
		j := idx[k]
		style := s[j].Style.Copy()
		if !negative && k+1 < last {
			// in case of edge cases where column top runes
			// are replaced, use the previous style's colors
			// as the background to avoid a gap in the column
			style.Background(s[idx[k+1]].Style.GetForeground())
		}
		length := math.Min(sum, limit)
		switch {
		case m.horizontal && negative:
			graph.DrawRowRightToLeft(&m.Canvas, canvas.Point{X: m.origin.X - 1, Y: i}, length, style)
		case m.horizontal:
			graph.DrawRowLeftToRight(&m.Canvas, canvas.Point{X: m.origin.X + m.axisOffset(), Y: i}, length, style)
		case negative:
			graph.DrawColumnTopToBottom(&m.Canvas, canvas.Point{X: i, Y: m.origin.Y + m.axisOffset()}, length, style)
		default:
			graph.DrawColumnBottomToTop(&m.Canvas, canvas.Point{X: i, Y: m.origin.Y - 1}, length, style)
		}
		sum -= math.Abs(v[j])
	}
}

// drawAxisAndLabels will draw axis and labels
// on to the barchart for horizontal or vertical bars
func (m *Model) drawAxisAndLabels() {
	if !m.showAxis {
		return
	}
	if m.horizontal {
		graph.DrawVerticalLineDown(&m.Canvas, m.origin, m.AxisStyle)
	} else {
		graph.DrawHorizonalLineRight(&m.Canvas, m.origin, m.AxisStyle)
	}
	// attempt to draw the label under the axis for each bar
	// the label string width will be bound by the bar width
	lastIdx := -1
	dLen := len(m.data)
	for i, b := range m.barIndices {
		if b >= 0 && b < dLen {
			if b != lastIdx {
				l := m.data[b].bd.Label
				p := canvas.Point{X: i, Y: m.Canvas.Height() - 1} // labels sit on the bottom row
				if m.horizontal {
					if len(l) > m.labelWidth {
						l = l[:m.labelWidth]
					}
					p = canvas.Point{X: 0, Y: i}
				} else {
					if len(l) > m.barWidth {
						l = l[:m.barWidth]
					}
				}
				m.Canvas.SetStringWithStyle(p, l, m.LabelStyle)
				lastIdx = b
			}
		}
	}
}

func (m Model) Init() tea.Cmd {
	return m.Canvas.Init()
}

// Update forwards bubbletea Msg to underlying canvas.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Canvas, cmd = m.Canvas.Update(msg)
	return m, cmd
}

// View returns a string used by the bubbletea framework to display the barchart.
func (m Model) View() (r string) {
	r = m.Canvas.View()
	if m.zoneManager != nil {
		r = m.zoneManager.Mark(m.zoneID, r)
	}
	return
}
