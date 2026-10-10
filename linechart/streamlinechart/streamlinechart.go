// ntcharts - Copyright (c) 2024 Neomantra Corp.

// Package streamlinechart implements a linechart that draws lines
// going from the right of the chart to the left of the chart
package streamlinechart

import (
	"math"
	"sort"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/buffer"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/graph"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const DefaultDataSetName = "default"

type dataSet struct {
	LineStyle runes.LineStyle // type of line runes to draw
	Style     lipgloss.Style

	// stores Y data values used to draw line runes
	sBuf *buffer.Float64ScaleRingBuffer
}

// Model contains state of a streamlinechart with an embedded linechart.Model
// A data set consists of a sequence of Y data values.
// For each data set, streamlinecharts can only plot a single rune in each column
// of the graph canvas from right to left.
// Uses linechart Model UpdateHandler() for processing keyboard and mouse messages.
type Model struct {
	linechart.Model
	dLineStyle runes.LineStyle     // default data set LineStyletype
	dStyle     lipgloss.Style      // default data set Style
	dSets      map[string]*dataSet // maps names to data sets
	y2Sets     map[string]bool     // names of data sets on the right Y axis; nil until used
}

// New returns a streamlinechart Model initialized from
// width, height and various options.
// By default, the chart will hide the X axis,
// auto set Y value ranges, and only enable moving viewport on Y axis.
func New(w, h int, opts ...Option) Model {
	m := Model{
		Model: linechart.New(w, h, 0, 1, 0, 1,
			linechart.WithXYSteps(0, 2),                                   // hide X axis
			linechart.WithAutoYRange(),                                    // automatically adjust Y value range
			linechart.WithUpdateHandler(linechart.YAxisUpdateHandler(1))), // only scroll on Y axis
		dLineStyle: runes.ArcLineStyle,
		dStyle:     lipgloss.NewStyle(),
		dSets:      make(map[string]*dataSet),
	}
	for _, opt := range opts {
		opt(&m)
	}
	m.UpdateGraphSizes()
	if _, ok := m.dSets[DefaultDataSetName]; !ok {
		m.dSets[DefaultDataSetName] = m.newDataSet()
	}
	return m
}

// newDataSet returns a new initialize *dataSet.
func (m *Model) newDataSet() *dataSet {
	// use GraphHeight() for y scale factor to align with y axis ticks
	// note that graph width is not used since lines are able to overlap onto Y axis
	ys := float64(m.GraphHeight()) / (m.ViewMaxY() - m.ViewMinY()) // y scale factor
	return &dataSet{
		LineStyle: m.dLineStyle,
		Style:     m.dStyle,
		sBuf:      buffer.NewFloat64ScaleRingBuffer(m.GraphRight()-m.Origin().X, m.ViewMinY(), ys),
	}
}

// rescaleData will scale all internally stored data with new scale factor.
func (m *Model) rescaleData() {
	// rescale stream buffer
	ys := float64(m.GraphHeight()) / (m.ViewMaxY() - m.ViewMinY()) // y scale factor
	for _, ds := range m.dSets {
		width := m.GraphRight() - m.Origin().X // width of graphing area includes Y axis
		// create new buffer with new size if the graphing area size has changed
		if ds.sBuf.Size() != width {
			buf := buffer.NewFloat64ScaleRingBuffer(width, m.ViewMinY(), ys)
			for _, f := range ds.sBuf.ReadAllRaw() {
				buf.Push(f)
			}
			ds.sBuf = buf
		} else {
			ds.sBuf.SetScale(ys)
			ds.sBuf.SetOffset(m.ViewMinY())
		}
	}
}

// ClearAllData will reset stored data values in all data sets.
func (m *Model) ClearAllData() {
	for _, ds := range m.dSets {
		ds.sBuf.Clear()
	}
	m.dSets[DefaultDataSetName] = m.newDataSet()
}

// ClearDataSet will erase stored data set given by name string.
func (m *Model) ClearDataSet(n string) {
	if ds, ok := m.dSets[n]; ok {
		ds.sBuf.Clear()
	}
}

// SetYScale sets the scale of the Y axis. See linechart.Model.SetYScale.
// On a log Y axis, values that are not greater than zero are not drawn:
// the line breaks around them. Existing data will be rescaled.
func (m *Model) SetYScale(s linechart.Scale) {
	m.Model.SetYScale(s)
	m.rescaleData()
}

// SetYAxis sets the left Y axis. See linechart.Model.SetYAxis.
// Existing data will be rescaled.
func (m *Model) SetYAxis(a linechart.Axis) {
	m.Model.SetYAxis(a)
	m.rescaleData()
}

// SetY2Axis adds a right Y axis, or replaces it. See
// linechart.Model.SetY2Axis. Put data sets on it with SetDataSetYAxis.
// Data pushed to right-assigned sets before this call was auto-ranged
// against the left axis, and that left range is not undone.
// Existing data will be rescaled.
func (m *Model) SetY2Axis(a linechart.Axis) {
	m.Model.SetY2Axis(a)
	m.rescaleData()
}

// ClearY2Axis removes the right Y axis. Data sets keep their axis
// assignment, and those on the right are drawn against the left axis until
// a right axis is set again. Existing data will be rescaled.
func (m *Model) ClearY2Axis() {
	m.Model.ClearY2Axis()
	m.rescaleData()
}

// SetDataSetYAxis puts the data set given by name string on the left or
// right Y axis, creating the data set if needed. All data sets start on
// the left axis. A data set on the right axis is drawn and auto-ranged as
// a left one while the chart has no right axis (see SetY2Axis).
func (m *Model) SetDataSetYAxis(n string, side linechart.YAxisSide) {
	if _, ok := m.dSets[n]; !ok {
		m.dSets[n] = m.newDataSet()
	}
	if side == linechart.YAxisRight {
		if m.y2Sets == nil {
			m.y2Sets = make(map[string]bool)
		}
		m.y2Sets[n] = true
	} else {
		delete(m.y2Sets, n)
	}
}

// y2Scale returns the right Y axis's scale and true if the data set given
// by name is drawn against the right axis: it is assigned to it and the
// chart has one.
func (m *Model) y2Scale(n string) (linechart.Scale, bool) {
	if !m.y2Sets[n] {
		return linechart.ScaleLinear, false
	}
	a, ok := m.Y2Axis()
	return a.Scale, ok
}

// autoAdjustRange auto-ranges f against the Y axis the data set given by
// name is drawn against.
func (m *Model) autoAdjustRange(n string, f canvas.Float64Point) bool {
	if _, right := m.y2Scale(n); right {
		return m.AutoAdjustY2Range(f)
	}
	return m.AutoAdjustRange(f)
}

// SetXRange updates the minimum and maximum expected X values.
// Existing data will be rescaled.
func (m *Model) SetXRange(min, max float64) {
	m.Model.SetXRange(min, max)
	m.rescaleData()
}

// SetYRange updates the minimum and maximum expected Y values.
// Existing data will be rescaled.
func (m *Model) SetYRange(min, max float64) {
	m.Model.SetYRange(min, max)
	m.rescaleData()
}

// SetViewXRange updates the displayed minimum and maximum X values.
// Existing data will be rescaled.
func (m *Model) SetViewXRange(min, max float64) {
	m.Model.SetViewXRange(min, max)
	m.rescaleData()
}

// SetViewYRange updates the displayed minimum and maximum Y values.
// Existing data will be rescaled.
func (m *Model) SetViewYRange(min, max float64) {
	m.Model.SetViewYRange(min, max)
	m.rescaleData()
}

// SetViewXYRange updates the displayed minimum and maximum X and Y values.
// Existing data will be rescaled.
func (m *Model) SetViewXYRange(minX, maxX, minY, maxY float64) {
	m.Model.SetViewXRange(minX, maxX)
	m.Model.SetViewYRange(minY, maxY)
	m.rescaleData()
}

// Resize will change streamlinechart display width and height.
// Existing data will be rescaled.
func (m *Model) Resize(w, h int) {
	m.Model.Resize(w, h)
	m.rescaleData()
}

// SetStyles will set the default styles of data sets.
func (m *Model) SetStyles(ls runes.LineStyle, s lipgloss.Style) {
	m.dLineStyle = ls
	m.dStyle = s
	m.SetDataSetStyles(DefaultDataSetName, ls, s)
}

// SetDataSetStyles will set the styles of the given data set by name string.
func (m *Model) SetDataSetStyles(n string, ls runes.LineStyle, s lipgloss.Style) {
	if _, ok := m.dSets[n]; !ok {
		m.dSets[n] = m.newDataSet()
	}
	ds := m.dSets[n]
	ds.LineStyle = ls
	ds.Style = s
}

// Push will push a float64 Y data value to the default data set
// to be displayed with Draw.
func (m *Model) Push(f float64) {
	m.PushDataSet(DefaultDataSetName, f)
}

// Push will push a float64 Y data value to a data set
// to be displayed with Draw. Using given data set by name string.
func (m *Model) PushDataSet(n string, f float64) {
	// auto adjust x and y ranges if enabled
	if m.autoAdjustRange(n, canvas.Float64Point{X: m.MinX(), Y: f}) {
		m.UpdateGraphSizes()
		m.rescaleData()
	}
	if _, ok := m.dSets[n]; !ok {
		m.dSets[n] = m.newDataSet()
	}
	m.dSets[n].sBuf.Push(f)
}

// Draw will draw lines runes displayed from right to left
// of the graphing area of the canvas. Uses default data set.
func (m *Model) Draw() {
	m.DrawDataSets([]string{DefaultDataSetName})
}

// DrawAll will draw lines runes for all data sets from right
// to left of the graphing area of the canvas.
func (m *Model) DrawAll() {
	names := make([]string, 0, len(m.dSets))
	for n, ds := range m.dSets {
		if ds.sBuf.Length() > 0 {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	m.DrawDataSets(names)
}

// DrawDataSets will draw lines runes from right to left
// of the graphing area of the canvas for each data set given
// by name strings.
func (m *Model) DrawDataSets(names []string) {
	if len(names) == 0 {
		return
	}
	m.Clear()
	m.DrawXYAxisAndLabel()
	for _, n := range names {
		if ds, ok := m.dSets[n]; ok {
			if _, right := m.y2Scale(n); right || m.YScale() == linechart.ScaleLog {
				m.drawRawDataSet(n, ds)
				continue
			}
			s := ds.sBuf.ReadAll()
			startX := m.GraphRight() - len(s)
			// round float64 data value to nearest integer to fit onto the canvas
			l := make([]int, 0, len(s))
			for _, v := range s {
				l = append(l, int(math.Round(v)))
			}
			// convert to canvas coordinates and avoid drawing below X axis
			yCoords := canvas.CanvasYCoordinates(m.Origin().Y, l)
			if m.XStep() > 0 {
				for i, v := range yCoords {
					if v > m.Origin().Y {
						yCoords[i] = m.Origin().Y
					}
				}
			}
			graph.DrawLineSequence(&m.Canvas,
				(startX == m.Origin().X),
				startX,
				yCoords,
				ds.LineStyle,
				ds.Style)
		}
	}
}

// drawRawDataSet draws the data set's line from its raw values, for a log
// Y axis or a data set on the right Y axis: the buffer scales linearly
// against the left axis, so its raw values are scaled here instead. A
// value with no place on its axis breaks the line: each run of drawable
// values is drawn on its own.
func (m *Model) drawRawDataSet(n string, ds *dataSet) {
	scale, right := m.y2Scale(n)
	if !right {
		scale = m.YScale()
	}
	raw := ds.sBuf.ReadAllRaw()
	startX := m.GraphRight() - len(raw)
	for from := 0; from < len(raw); from++ {
		var yCoords []int
		to := from
		for ; to < len(raw) && scale.Valid(raw[to]); to++ {
			y := raw[to]
			if right {
				y = m.Y2ToY(y)
			}
			f := m.ScaleFloat64PointForLine(canvas.Float64Point{X: m.ViewMinX(), Y: y})
			// convert to canvas coordinates and avoid drawing below X axis
			cy := canvas.CanvasYCoordinate(m.Origin().Y, int(math.Round(f.Y)))
			if m.XStep() > 0 && cy > m.Origin().Y {
				cy = m.Origin().Y
			}
			yCoords = append(yCoords, cy)
		}
		if len(yCoords) > 0 {
			graph.DrawLineSequence(&m.Canvas,
				(startX+from == m.Origin().X),
				startX+from,
				yCoords,
				ds.LineStyle,
				ds.Style)
		}
		from = to
	}
}

// Update processes bubbletea Msg to by invoking
// UpdateHandlerFunc callback if linechart is focused.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.Focused() {
		return m, nil
	}
	m.UpdateHandler(&m.Model, msg)
	m.rescaleData()
	return m, nil
}
