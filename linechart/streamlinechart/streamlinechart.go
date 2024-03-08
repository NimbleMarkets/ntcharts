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

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UpdateHandler callback invoked during an Update()
// and passes in the wavelinechart Model and bubbletea Msg.
type UpdateHandler func(*Model, tea.Msg)

// DefaultUpdateHandler is used by steamlinechart to enable
// zooming in and out with the mouse wheels,
// moving the viewing window with mouse left hold and movement,
// and moving the viewing window with the arrow keys.
// There is only movement along the Y axis.
// Uses linechart.Canvas Keymap for keyboard messages.
func DefaultUpdateHandler() UpdateHandler {
	var lastPos canvas.Point
	return func(m *Model, tm tea.Msg) {
		switch msg := tm.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.Canvas.KeyMap.Up):
				m.MoveUp(1)
			case key.Matches(msg, m.Canvas.KeyMap.Down):
				m.MoveDown(1)
			}
		case tea.MouseMsg:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				// zoom in limited values cannot cross
				m.ZoomIn(0, 1)
			case tea.MouseButtonWheelDown:
				// zoom out limited by max values
				m.ZoomOut(0, 1)
			}
			switch msg.Action {
			case tea.MouseActionPress:
				zInfo := m.GetZoneManager().Get(m.GetZoneID())
				if zInfo.InBounds(msg) {
					x, y := zInfo.Pos(msg)
					lastPos = canvas.Point{X: x, Y: y} // set position of last click
				}
			case tea.MouseActionMotion: // event occurs when mouse is pressed
				zInfo := m.GetZoneManager().Get(m.GetZoneID())
				if zInfo.InBounds(msg) {
					x, y := zInfo.Pos(msg)
					if y > lastPos.Y {
						m.MoveDown(1)
					} else if y < lastPos.Y {
						m.MoveUp(1)
					}
					lastPos = canvas.Point{X: x, Y: y} // update last mouse position
				}
			}
		}
	}
}

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
	UpdateHandler UpdateHandler       // handlers update events
	dLineStyle    runes.LineStyle     // default data set LineStyletype
	dStyle        lipgloss.Style      // default data set Style
	dSets         map[string]*dataSet // maps names to data sets
}

// New returns a streamlinechart Model initialized with given linechart.Model.
func New(lc linechart.Model) Model {
	return NewWithStyle(lc, runes.ArcLineStyle, lipgloss.NewStyle())
}

// NewWithStyle returns a streamlinechart Model initialized with
// given linechart.Model and styles as the default data set styles.
func NewWithStyle(lc linechart.Model, ls runes.LineStyle, s lipgloss.Style) Model {
	m := Model{
		Model:         lc,
		UpdateHandler: DefaultUpdateHandler(),
		dLineStyle:    ls,
		dStyle:        s,
		dSets:         make(map[string]*dataSet),
	}
	m.dSets[DefaultDataSetName] = m.newDataSet()
	return m
}

// newDataSet returns a new initialize *dataSet.
func (m *Model) newDataSet() *dataSet {
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
	sf := float64(m.Origin().Y) / (m.ViewMaxY() - m.ViewMinY()) // scale factor
	for _, ds := range m.dSets {
		ds.sBuf.SetScale(sf)
		ds.sBuf.SetOffset(m.ViewMinY())
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
	m.UpdateHandler(&m, msg)
	m.rescaleData()
	return m, nil
}
