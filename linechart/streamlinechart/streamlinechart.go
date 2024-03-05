// Package streamlinechart implements a linechart that draws lines
// going from the right of the chart to the left of the chart
package streamlinechart

import (
	"math"
	"sort"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/buffer"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/graph"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"
	"github.com/NimbleMarkets/bubbletea-charts/linechart"
	"github.com/charmbracelet/lipgloss"
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
type Model struct {
	linechart.Model
	dLineStyle runes.LineStyle     // default data set LineStyletype
	dStyle     lipgloss.Style      // default data set Style
	dSets      map[string]*dataSet // maps names to data sets
}

// New returns a streamlinechart Model initialized with given linechart.Model.
func New(lc linechart.Model) Model {
	return NewWithStyle(lc, runes.ArcLineStyle, lipgloss.NewStyle())
}

// NewWithStyle returns a streamlinechart Model initialized with
// given linechart.Model and styles as the default data set styles.
func NewWithStyle(lc linechart.Model, ls runes.LineStyle, s lipgloss.Style) Model {
	m := Model{
		Model:      lc,
		dLineStyle: ls,
		dStyle:     s,
		dSets:      make(map[string]*dataSet),
	}
	return m
}

// newDataSet returns a new initialize *dataSet.
func (m *Model) newDataSet() *dataSet {
	ys := float64(m.Origin().Y) / (m.MaxY() - m.MinY()) // y scale factor
	return &dataSet{
		LineStyle: m.dLineStyle,
		Style:     m.dStyle,
		sBuf:      buffer.NewFloat64ScaleRingBuffer(m.Width()-m.Origin().X, m.MinY(), ys),
	}
}

// rescaleData will scale all internally stored data with new scale factor.
func (m *Model) rescaleData() {
	// rescale stream buffer
	sf := float64(m.Origin().Y) / (m.MaxY() - m.MinY()) // scale factor
	for _, ds := range m.dSets {
		ds.sBuf.SetScale(sf)
		ds.sBuf.SetOffset(m.MinY())
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

// Resize will change streamlinechart display width and height.
// Existing data will be rescaled.
func (m *Model) Resize(w, h int) {
	// data buffers does not change since the graphing area
	// remains the same and X,Y coordinates are still valid
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

// Push will push a float64 Y data value to the "default" data set
// to be displayed with Draw.
func (m *Model) Push(f float64) {
	m.PushDataSet(DefaultDataSetName, f)
}

// Push will push a float64 Y data value to a data set
// to be displayed with Draw. Using given data set by name string.
func (m *Model) PushDataSet(n string, f float64) {
	if _, ok := m.dSets[n]; !ok {
		m.dSets[n] = m.newDataSet()
	}
	m.dSets[n].sBuf.Push(f)
}

// Draw will draw lines runes displayed from right to left
// of the graphing area of the canvas. Uses "default" data set.
func (m *Model) Draw() {
	m.DrawDataSets([]string{DefaultDataSetName})
}

// DrawAll will draw lines runes for all data sets from right
// to left of the graphing area of the canvas.
func (m *Model) DrawAll() {
	names := make([]string, 0, len(m.dSets))
	for n, _ := range m.dSets {
		names = append(names, n)
	}
	sort.Strings(names)
	m.DrawDataSets(names)
}

// DrawDataSets will draw lines runes from right to left
// of the graphing area of the canvas for each data set given
// by name strings.
func (m *Model) DrawDataSets(names []string) {
	m.Clear()
	m.DrawXYAxisAndLabel()
	for _, n := range names {
		if ds, ok := m.dSets[n]; ok {
			s := ds.sBuf.ReadAll()
			startX := m.Canvas.Width() - len(s)
			// round float64 to nearest integer to fit onto the canvas
			l := make([]int, 0, len(s))
			for _, v := range s {
				l = append(l, int(math.Round(v)))
			}
			graph.DrawLineSequence(&m.Canvas,
				(startX == m.Origin().X),
				startX,
				canvas.CanvasYCoordinates(m.Origin().Y, l), // convert to canvas coordinates
				ds.LineStyle,
				ds.Style)
		}
	}
}
