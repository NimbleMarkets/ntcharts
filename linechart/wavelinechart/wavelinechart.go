// Package wavelinechart implements a linechart that draws wave lines on the graph
package wavelinechart

import (
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

	// []int stores canvas coordinates to draw line runes
	// Each index of the []int corresponds to a canvas column
	// and the value of each index is the canvas row
	// I.E. (X,seqY[X]) coorindates will be used to draw the line runes
	seqY []int

	// stores data points from Plot() and contains scaled data points
	pBuf *buffer.Float64PointScaleBuffer
}

// Model contains state of a wavelinechart with an embedded linechart.Model
// A data set is a list of (X,Y) Cartesian coordinates.
// For each data set, wavelinecharts can only plot a single rune in each column
// of the graph canvas by mapping (X,Y) data points values in Cartesian coordinates
// to the (X,Y) canvas coordinates of the graph.
// By default, there is a line through the graph X axis without any plotted points.
type Model struct {
	linechart.Model
	dLineStyle runes.LineStyle     // default data set LineStyletype
	dStyle     lipgloss.Style      // default data set Style
	dSets      map[string]*dataSet // maps names to data sets
}

// New returns a wavelinechart Model initialized with given linechart.Model.
func New(lc linechart.Model) Model {
	return NewWithStyle(lc, runes.ArcLineStyle, lipgloss.NewStyle())
}

// NewWithStyle returns a wavelinechart Model initialized with
// given linechart.Model and styles as the default data set styles.
func NewWithStyle(lc linechart.Model, ls runes.LineStyle, s lipgloss.Style) Model {
	m := Model{
		Model:      lc,
		dLineStyle: ls,
		dStyle:     s,
		dSets:      make(map[string]*dataSet),
	}
	m.dSets[DefaultDataSetName] = m.newDataSet()
	return m
}

// newDataSet returns a new initialize *dataSet.
func (m *Model) newDataSet() *dataSet {
	xs := float64(m.GraphWidth()) / (m.ViewMaxX() - m.ViewMinX()) // X scale factor
	ys := float64(m.Origin().Y) / (m.ViewMaxY() - m.ViewMinY())   // y scale factor
	ds := &dataSet{
		LineStyle: m.dLineStyle,
		Style:     m.dStyle,
		seqY:      make([]int, m.Width(), m.Width()),
		pBuf: buffer.NewFloat64PointScaleBuffer(
			canvas.Float64Point{X: m.ViewMinX(), Y: m.ViewMinY()},
			canvas.Float64Point{X: xs, Y: ys}),
	}
	m.resetDataSetSeqY(ds)
	return ds
}

// resetPoints will set graph sequence of Y coordinates of a data set
// such that Draw will display each Y coordinate on Y = 0
func (m *Model) resetDataSetSeqY(ds *dataSet) {
	f := m.ScaleFloat64Point(canvas.Float64Point{X: 0.0, Y: 0.0})
	ds.seqY = make([]int, m.Width(), m.Width())
	for i, _ := range ds.seqY {
		// avoid drawing below X axis
		ds.seqY[i] = canvas.CanvasPointFromFloat64Point(m.Origin(), f).Y
		if (m.XStep() > 0) && (ds.seqY[i] > m.Origin().Y) {
			ds.seqY[i] = m.Origin().Y
		}
	}
}

// setDataSetSeqY will set a data set canvas row and column
// from given scaled Float64Point data point
func (m *Model) setDataSetSeqY(ds *dataSet, f canvas.Float64Point) {
	p := canvas.CanvasPointFromFloat64Point(m.Origin(), f)
	// avoid drawing outside graphing area
	if (p.X >= 0) && (p.X < len(ds.seqY)) {
		// avoid drawing below X axis
		ds.seqY[p.X] = p.Y
		if (m.XStep() > 0) && (ds.seqY[p.X] > m.Origin().Y) {
			ds.seqY[p.X] = m.Origin().Y
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
		m.resetDataSetSeqY(ds)
		for _, v := range ds.pBuf.ReadAll() {
			m.setDataSetSeqY(ds, v)
		}
	}
}

// ClearAllData will reset stored data values in all data sets.
func (m *Model) ClearAllData() {
	for n, _ := range m.dSets {
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

// SetDataSetStyle will set the default styles of data sets.
func (m *Model) SetStyle(ls runes.LineStyle, s lipgloss.Style) {
	m.dLineStyle = ls
	m.dStyle = s
	m.SetDataSetStyle(DefaultDataSetName, ls, s)
}

// SetDataSetStyle will set the styles of the given data set by name string.
func (m *Model) SetDataSetStyle(n string, ls runes.LineStyle, s lipgloss.Style) {
	if _, ok := m.dSets[n]; !ok {
		m.dSets[n] = m.newDataSet()
	}
	ds := m.dSets[n]
	ds.LineStyle = ls
	ds.Style = s
}

// Plot will map a Float64Point data value to a canvas coordinates
// to be displayed with Draw. Uses "default" data set.
func (m *Model) Plot(f canvas.Float64Point) {
	m.PlotDataSet(DefaultDataSetName, f)
}

// Plot will map a Float64Point data value to a canvas coordinates
// to be displayed with Draw. Uses given data set by name string.
func (m *Model) PlotDataSet(n string, f canvas.Float64Point) {
	if _, ok := m.dSets[n]; !ok {
		m.dSets[n] = m.newDataSet()
	}
	ds := m.dSets[n]
	ds.pBuf.Push(f)
	s := ds.pBuf.At(ds.pBuf.Length() - 1)
	m.setDataSetSeqY(ds, s)
}

// Draw will draw lines runes for each column
// of the graphing area of the canvas. Uses "default" data set.
func (m *Model) Draw() {
	m.DrawDataSets([]string{DefaultDataSetName})
}

// DrawAll will draw lines runes for each column
// of the graphing area of the canvas for all data sets.
func (m *Model) DrawAll() {
	names := make([]string, 0, len(m.dSets))
	for n, _ := range m.dSets {
		names = append(names, n)
	}
	sort.Strings(names)
	m.DrawDataSets(names)
}

// DrawDataSets will draw lines runes for each column
// of the graphing area of the canvas for each data set given
// by name strings.
func (m *Model) DrawDataSets(names []string) {
	m.Clear()
	m.DrawXYAxisAndLabel()
	for _, n := range names {
		if ds, ok := m.dSets[n]; ok {
			startX := m.Origin().X
			seqY := ds.seqY[startX:]
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
	if !m.Model.Focused() {
		return m, nil
	}
	m.Model, _ = m.Model.Update(msg)
	m.rescaleData() // rescale data points to new viewing window
	return m, nil
}
