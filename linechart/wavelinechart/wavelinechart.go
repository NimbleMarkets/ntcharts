// bubbletea-charts - Copyright (c) 2024 Neomantra Corp.

// Package wavelinechart implements a linechart that draws wave lines on the graph
package wavelinechart

import (
	"math"
	"sort"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/buffer"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/graph"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"
	"github.com/NimbleMarkets/bubbletea-charts/linechart"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const DefaultDataSetName = "default"

type dataSet struct {
	LineStyle runes.LineStyle // type of line runes to draw
	Style     lipgloss.Style

	// stores data points from Plot() and contains scaled data points
	pBuf *buffer.Float64PointScaleBuffer
}

// Model contains state of a wavelinechart with an embedded linechart.Model
// A data set is a list of (X,Y) Cartesian coordinates.
// For each data set, wavelinecharts can only plot a single rune in each column
// of the graph canvas by mapping (X,Y) data points values in Cartesian coordinates
// to the (X,Y) canvas coordinates of the graph.
// If multiple data points map to the same column, then the latest data point
// will be used for that column.
// By default, there is a line through the graph X axis without any plotted points.
// Uses linechart Model UpdateHandler() for processing keyboard and mouse messages.
type Model struct {
	linechart.Model
	dLineStyle runes.LineStyle     // default data set LineStyletype
	dStyle     lipgloss.Style      // default data set Style
	dSets      map[string]*dataSet // maps names to data sets
}

// New returns a wavelinechart Model initialized
// with given linechart Model and various options.
// By default, the chart will auto set X and Y value ranges,
// and only enable moving viewport on X axis.
func New(w, h int, opts ...Option) Model {
	m := Model{
		Model: linechart.New(w, h, 0, 1, 0, 1,
			linechart.WithAutoXYRange(),                                   // automatically adjust value ranges
			linechart.WithUpdateHandler(linechart.XAxisUpdateHandler(1))), // only scroll on X axis
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
	xs := float64(m.GraphWidth()) / (m.ViewMaxX() - m.ViewMinX()) // X scale factor
	ys := float64(m.Origin().Y) / (m.ViewMaxY() - m.ViewMinY())   // y scale factor
	ds := &dataSet{
		LineStyle: m.dLineStyle,
		Style:     m.dStyle,
		pBuf: buffer.NewFloat64PointScaleBuffer(
			canvas.Float64Point{X: m.ViewMinX(), Y: m.ViewMinY()},
			canvas.Float64Point{X: xs, Y: ys}),
	}
	return ds
}

// getLineSequence returns a sequence of Y values
// to draw line runes from a given set of scaled []FloatPoint64.
func (m *Model) getLineSequence(points []canvas.Float64Point) (seqY []int) {
	// Create a []int storing canvas coordinates to draw line runes.
	// Each index of the []int corresponds to a canvas column
	// and the value of each index is the canvas row
	// I.E. (X,seqY[X]) coorindates will be used to draw the line runes
	width := m.Width() - m.Origin().X // lines can draw on Y axis
	seqY = make([]int, width, width)

	// initialize every index to the value such that
	// a horizontal line at Y = 0 will be drawn
	f := m.ScaleFloat64Point(canvas.Float64Point{X: 0.0, Y: 0.0})
	for i := range seqY {
		seqY[i] = canvas.CanvasYCoordinate(m.Origin().Y, int(math.Round(f.Y)))
		// avoid drawing below X axis
		if (m.XStep() > 0) && (seqY[i] > m.Origin().Y) {
			seqY[i] = m.Origin().Y
		}
	}
	// map data set containing scaled Float64Point data points
	// onto graph row and column
	for _, p := range points {
		m.setLineSequencePoint(seqY, p)
	}
	return
}

// setLineSequencePoint will map a scaled Float64Point data point
// on to a sequence of graph Y values.  Points mapping onto
// existing indices of the sequence will override the existing value.
func (m *Model) setLineSequencePoint(seqY []int, f canvas.Float64Point) {
	x := int(math.Round(f.X))
	// avoid drawing outside graphing area
	if (x >= 0) && (x < len(seqY)) {
		// avoid drawing below X axis
		seqY[x] = canvas.CanvasYCoordinate(m.Origin().Y, int(math.Round(f.Y)))
		if (m.XStep() > 0) && (seqY[x] > m.Origin().Y) {
			seqY[x] = m.Origin().Y
		}
	}
}

// rescaleData will scale all internally stored data with new scale factor.
func (m *Model) rescaleData() {
	// rescale all data set graph points
	xs := float64(m.GraphWidth()) / (m.ViewMaxX() - m.ViewMinX()) // X scale factor
	ys := float64(m.Origin().Y) / (m.ViewMaxY() - m.ViewMinY())   // y scale factor
	for _, ds := range m.dSets {
		ds.pBuf.SetOffset(canvas.Float64Point{X: m.ViewMinX(), Y: m.ViewMinY()})
		ds.pBuf.SetScale(canvas.Float64Point{X: xs, Y: ys}) // buffer rescales all raw data points
	}
}

// ClearAllData will reset stored data values in all data sets.
func (m *Model) ClearAllData() {
	for n := range m.dSets {
		m.ClearDataSet(n)
	}
	m.dSets[DefaultDataSetName] = m.newDataSet()
}

// ClearDataSet will erase stored data set given by name string.
func (m *Model) ClearDataSet(n string) {
	if _, ok := m.dSets[n]; ok {
		delete(m.dSets, n)
	}
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

// Resize will change wavelinechart display width and height.
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

// Plot will map a Float64Point data value to a canvas coordinates
// to be displayed with Draw. Uses default data set.
func (m *Model) Plot(f canvas.Float64Point) {
	m.PlotDataSet(DefaultDataSetName, f)
}

// PlotDataSet will map a Float64Point data value to a canvas coordinates
// to be displayed with Draw. Uses given data set by name string.
func (m *Model) PlotDataSet(n string, f canvas.Float64Point) {
	if m.AutoAdjustRange(f) { // auto adjust x and y ranges if enabled
		m.UpdateGraphSizes()
		m.rescaleData()
	}
	if _, ok := m.dSets[n]; !ok {
		m.dSets[n] = m.newDataSet()
	}
	ds := m.dSets[n]
	ds.pBuf.Push(f)
}

// Draw will draw lines runes for each column
// of the graphing area of the canvas. Uses default data set.
func (m *Model) Draw() {
	m.DrawDataSets([]string{DefaultDataSetName})
}

// DrawAll will draw lines runes for each column
// of the graphing area of the canvas for all data sets.
// Will always draw default data set.
func (m *Model) DrawAll() {
	names := make([]string, 0, len(m.dSets))
	for n, ds := range m.dSets {
		if (n == DefaultDataSetName) || (ds.pBuf.Length() > 0) {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	m.DrawDataSets(names)
}

// DrawDataSets will draw lines runes for each column
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
			startX := m.Origin().X
			seqY := m.getLineSequence(ds.pBuf.ReadAll())
			graph.DrawLineSequence(&m.Canvas,
				true,
				startX,
				seqY,
				ds.LineStyle,
				ds.Style)
		}
	}
}

// Update processes bubbletea Msg to by invoking
// UpdateMsgHandlerFunc callback if wavelinechart is focused.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.Focused() {
		return m, nil
	}
	m.UpdateHandler(&m.Model, msg)
	m.rescaleData() // rescale data points to new viewing window
	return m, nil
}
