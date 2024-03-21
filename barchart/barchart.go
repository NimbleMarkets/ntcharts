// Package barchart implements a canvas that displays a bar chart
// with bars going either horizontally or vertically.
package barchart

import (
	"math"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/buffer"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/graph"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// BarValue contain bar segment name, value and style for drawing.
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

	max  float64    // expected maximum data value
	sf   float64    // scale factor
	data []*dataSet // each index is a unique bar

	zoneManager *zone.Manager // provides mouse functionality
	zoneID      string
}

// New returns a barchart Model initialized with given width, height
// and various options.
// By default, barchart will automatically scale bar to new maximum data values,
// bar width to fill up the canvas and have one gap between bars, and display
// bars vertically.
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

// resetScale will recompute scale factor and scale
// all existing data sets
func (m *Model) resetScale() {
	if m.horizontal {
		m.sf = float64(m.Canvas.Width()-m.origin.X) / m.max
	} else {
		m.sf = float64(m.origin.Y) / m.max
	}
	for _, ds := range m.data {
		ds.buf.SetScale(m.sf)
	}
}

// resetOrigin will set origin for axis and labels
func (m *Model) resetOrigin() {
	if m.showAxis {
		if m.horizontal {
			var maxLen int
			for _, ds := range m.data {
				lw := len(ds.bd.Label)
				if lw > maxLen {
					maxLen = lw
				}
			}
			m.origin = canvas.Point{X: maxLen, Y: 0}
		} else {
			m.origin = canvas.Point{X: 0, Y: m.Canvas.Height() - 2}
		}
	} else {
		if m.horizontal {
			m.origin = canvas.Point{X: 0, Y: 0}
		} else {
			m.origin = canvas.Point{X: 0, Y: m.Canvas.Height()}
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
		gaps := (len(m.data) - 1) * m.barGap // total space used by gaps
		size := graphSize - gaps             // total available space for bars
		m.barWidth = size / len(m.data)      // each bar width
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
		m.sf = 1
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

// MaxValue returns expected maximum data value.
func (m *Model) MaxValue() float64 {
	return m.max
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
// at the given Point.
func (m *Model) BarDataFromPoint(p canvas.Point) (r BarData) {
	bIdx := p.X                  // which bar is selected
	vIdx := m.origin.Y - p.Y - 1 // which bar rune is selected
	if m.horizontal {
		bIdx = p.Y
		vIdx = p.X - m.origin.X
		if m.showAxis {
			vIdx -= 1
		}
	}
	if bIdx >= len(m.barIndices) {
		return
	}
	// get bar data and return all values that
	// can be drawn onto that point on the canvas
	if idx := m.barIndices[bIdx]; idx != -1 {
		r.Label = m.data[idx].bd.Label
		v := m.data[idx].bd.Values
		// can use scaled data to check if which
		// values are drawn on the canvas since
		sv := m.data[idx].buf.ReadAll()
		var sum float64
		var oLen int
		for i, f := range sv {
			newSum := sum + f
			nLen := int(math.Floor(newSum))
			if oLen <= vIdx && vIdx <= nLen {
				r.Values = append(r.Values, v[i])
			}
			sum = newSum
			oLen = nLen
		}
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

// Push adds given BarData to barchart data set.
// Negative values will be treated as the value 0.
// Data will be scaled using expected max value and barchart size.
func (m *Model) Push(lv BarData) {
	var sum float64 // assumes no overflow
	for _, v := range lv.Values {
		v.Value = math.Max(v.Value, 0)
		sum += v.Value
	}
	if m.AutoMaxValue && sum > m.max {
		m.SetMax(sum)
	}
	m.data = append(m.data, m.newDataSet(lv))
	m.resetBarIndices()
}

// PushAll adds all data values in []BarData to barchart data set.
// Negative values will be treated as the value 0.
// Data will be scaled using expected max value and barchart size.
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

// drawBars will draw columns from bottom to top
// for each data values, from left to right of the graph
// for the data set for vertical bars.
// The function will draw rows from left to right
// for each data values, from top to bottom of the graph
// for the data set for horizontal bars.
func (m *Model) drawBars() {
	// value bounded by bar length
	startX := m.origin.X
	barLen := float64(m.origin.Y)
	if m.horizontal {
		barLen = float64(m.Canvas.Width() - m.origin.X)
		if m.showAxis {
			startX += 1
			barLen -= 1
		}
	}
	dLen := len(m.data)
	for i, b := range m.barIndices {
		if b >= 0 && b < dLen {
			v := m.data[b].buf.ReadAll()
			s := m.data[b].bd.Values
			var sum float64
			for _, f := range v {
				sum += f
			}
			startIdx := len(v) - 1
			for j := startIdx; j >= 0; j-- {
				style := s[j].Style.Copy()
				if j+1 < startIdx {
					// in case of edge cases where column top runes
					// are replaced, use the previous style's colors
					// as the background to avoid a gap in the column
					style.Background(s[j+1].Style.GetForeground())
				}
				if m.horizontal {
					graph.DrawRowLeftToRight(&m.Canvas,
						canvas.Point{startX, i},
						math.Min(sum, barLen),
						style)
				} else {
					graph.DrawColumnBottomToTop(&m.Canvas,
						canvas.Point{i, m.origin.Y - 1},
						math.Min(sum, barLen), style)
				}
				sum -= v[j]
			}
		}
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
				p := canvas.Point{i, m.origin.Y + 1}
				if m.horizontal {
					if len(l) > m.origin.X {
						l = l[:m.origin.X]
					}
					p = canvas.Point{0, i}
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
