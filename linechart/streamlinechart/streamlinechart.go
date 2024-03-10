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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const DefaultDataSetName = "default"

type dataSet struct {
	LineStyle runes.LineStyle // type of line runes to draw
	Style     lipgloss.Style

	// stores Y data values used to draw line runes
	sBuf *buffer.Float64ScaleRingBuffer
}

// Option is used to set options when initializing a streamlinechart. Example:
//
//	slc := New(width, height, minY, maxY, WithStyles(someLineStyle, someLipglossStyle))
type Option func(*Model)

// WithLineChart sets internal linechart to given linechart.
func WithLineChart(lc *linechart.Model) Option {
	return func(m *Model) {
		m.Model = *lc
	}
}

// WithXYSteps sets the number of steps when drawing X and Y axes values.
// If X steps 0, then X axis will be hidden.
// If Y steps 0, then Y axis will be hidden.
func WithXYSteps(x int, y int) Option {
	return func(m *Model) {
		m.SetXStep(x)
		m.SetYStep(y)
	}
}

// WithStyles sets the default line style and lipgloss style of data sets.
func WithStyles(ls runes.LineStyle, s lipgloss.Style) Option {
	return func(m *Model) {
		m.SetStyles(ls, s)
	}
}

// WithAxesStyles sets the axes line and line label styles.
func WithAxesStyles(as lipgloss.Style, ls lipgloss.Style) Option {
	return func(m *Model) {
		m.AxisStyle = as
		m.LabelStyle = ls
	}
}

// WithDataSetStyles sets the line style and lipgloss style
// of the data set given by name.
func WithDataSetStyles(n string, ls runes.LineStyle, s lipgloss.Style) Option {
	return func(m *Model) {
		m.SetDataSetStyles(n, ls, s)
	}
}

// WithStream adds []float64 data points to the default data set.
func WithStream(f []float64) Option {
	return func(m *Model) {
		for _, v := range f {
			m.Push(v)
		}
	}
}

// WithDataSetStream adds []float64 data points to the data set given by name.
func WithDataSetStream(n string, f []float64) Option {
	return func(m *Model) {
		for _, v := range f {
			m.PushDataSet(n, v)
		}
	}
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
}

// New returns a streamlinechart Model initialized from
// width, height, Y value range and various options.
// By default, the chart will hide the X axis,
// auto set Y value ranges, and only enable moving viewport on Y axis.
func New(w, h int, minY, maxY float64, opts ...Option) Model {
	m := Model{
		Model: linechart.New(w, h, 0, 1, minY, maxY,
			linechart.WithXYSteps(0, 2),                                  // hide X axis
			linechart.WithAutoYRange(),                                   // automatically adjust Y value range
			linechart.WithUpdateHandler(linechart.YAxisUpdateHandler())), // only scroll on Y axis
		dLineStyle: runes.ArcLineStyle,
		dStyle:     lipgloss.NewStyle(),
		dSets:      make(map[string]*dataSet),
	}
	m.dSets[DefaultDataSetName] = m.newDataSet()
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// newDataSet returns a new initialize *dataSet.
func (m *Model) newDataSet() *dataSet {
	// note that graph width is not used since lines are able to overlap onto Y axis
	ys := float64(m.Origin().Y) / (m.ViewMaxY() - m.ViewMinY()) // y scale factor
	return &dataSet{
		LineStyle: m.dLineStyle,
		Style:     m.dStyle,
		sBuf:      buffer.NewFloat64ScaleRingBuffer(m.Width()-m.Origin().X, m.ViewMinY(), ys),
	}
}

// rescaleData will scale all internally stored data with new scale factor.
func (m *Model) rescaleData() {
	// rescale stream buffer
	ys := float64(m.Origin().Y) / (m.ViewMaxY() - m.ViewMinY()) // y scale factor
	for _, ds := range m.dSets {
		width := m.Width() - m.Origin().X // width of graphing area includes Y axis
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
	if m.AutoAdjustRange(canvas.Float64Point{X: m.MinX(), Y: f}) {
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
