// ntcharts - Copyright (c) 2024 Neomantra Corp.

// Package heatmap implements a canvas that displays a heatmap,
// color-mapped data over a grid.
package heatmap

// File contains a Model using the BubbleTea framework
// representing the state of the heatmap
// and options used by the heatmap during initialization with New().

import (
	"math"

	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/NimbleMarkets/ntcharts/linechart"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

///////////////////////////////////////////////////////////////////////////////

// default color scale is a 0->1 gradient from black to white
var defaultColorScale = []lipgloss.Color{
	lipgloss.Color("#000000"), // black
	lipgloss.Color("#111111"),
	lipgloss.Color("#222222"),
	lipgloss.Color("#333333"),
	lipgloss.Color("#444444"),
	lipgloss.Color("#555555"),
	lipgloss.Color("#666666"),
	lipgloss.Color("#777777"),
	lipgloss.Color("#888888"),
	lipgloss.Color("#999999"),
	lipgloss.Color("#AAAAAA"),
	lipgloss.Color("#BBBBBB"),
	lipgloss.Color("#CCCCCC"),
	lipgloss.Color("#DDDDDD"),
	lipgloss.Color("#EEEEEE"),
	lipgloss.Color("#FFFFFFF"), // white
}

// GetDefaultColorScale returns the current default color scale used by all heatmaps.
func GetDefaultColorScale() []lipgloss.Color {
	return defaultColorScale
}

// SetDefaultColorScale sets the default color scale used by all heatmaps.
// Returns the previous color scale.
func SetDefaultColorScale(cs []lipgloss.Color) []lipgloss.Color {
	old := defaultColorScale
	defaultColorScale = cs
	return old
}

///////////////////////////////////////////////////////////////////////////////

// HeatPoint has a cartesian X,Y and a Value
type HeatPoint struct {
	X, Y, V float64
}

func NewHeatPoint(x, y, value float64) HeatPoint {
	return HeatPoint{X: x, Y: y, V: value}
}

func NewHeatPointInt(x, y int, value float64) HeatPoint {
	return HeatPoint{X: float64(x), Y: float64(y), V: value}
}

func (p HeatPoint) AsFloat64Point() canvas.Float64Point {
	return canvas.Float64Point{X: p.X, Y: p.Y}
}

///////////////////////////////////////////////////////////////////////////////

// Model contains state of a heatmap
// It embeds a linechart.Model, for the graph view, but renders maps instead
type Model struct {
	linechart.Model

	ColorScale []lipgloss.Color // Color gradient user for heatmap
	cellStyle  lipgloss.Style   // default style for heatmap cells

	points []HeatPoint // data points

	AutoMinValue bool    // AutoMinValue true will automatically adjust minimum data value
	AutoMaxValue bool    // AutoMaxValue true will automatically adjust minimum data value
	minValue     float64 // minimum data value
	maxValue     float64 // expected maximum data value
}

// New returns a heatmap Model initialized with given width, height
// and various options.
// By default, heatmap will automatically scale bars to new maximum data values.
func New(w, h int, opts ...Option) Model {
	m := Model{
		Model: linechart.New(w, h, 0, 1, 0, 1,
			linechart.WithAutoXYRange()), // automatically adjust value ranges
		ColorScale: GetDefaultColorScale(),
		cellStyle:  lipgloss.NewStyle(),
		points:     nil,
		minValue:   +math.MaxFloat64,
		maxValue:   -math.MaxFloat64,
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
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

// SetValueRange updates the displayed minimum and maximum Y values.
// Existing data will be rescaled.
func (m *Model) SetValueRange(minVal, maxVal float64) {
	m.minValue = minVal
	m.maxValue = maxVal
	m.rescaleData()
}

// Resize will change wavelinechart display width and height.
// Existing data will be rescaled.
func (m *Model) Resize(w, h int) {
	m.Model.Resize(w, h)
	m.rescaleData()
}

// rescaleData will scale all internally stored data with new scale factor.
func (m *Model) rescaleData() {
	// rescale all data set graph points
	// xs := float64(m.GraphWidth()) / (m.ViewMaxX() - m.ViewMinX()) // X scale factor
	// ys := float64(m.Origin().Y) / (m.ViewMaxY() - m.ViewMinY())   // y scale factor
	// TODO
	// for _, ds := range m.dSets {
	// 	ds.pBuf.SetOffset(canvas.Float64Point{X: m.ViewMinX(), Y: m.ViewMinY()})
	// 	ds.pBuf.SetScale(canvas.Float64Point{X: xs, Y: ys}) // buffer rescales all raw data points
	// }
}

// AutoAdjustValueRange automatically adjusts the heatmap's value range based on the passed value.
// It returns whether or not the display range has been adjusted.
func (m *Model) AutoAdjustValueRange(value float64) (b bool) {
	if m.AutoMinValue && value < m.minValue {
		m.minValue = value
		b = true
	}
	if m.AutoMaxValue && value > m.maxValue {
		m.maxValue = value
		b = true
	}
	return
}

// Clear will clear heatmap data.
func (m *Model) ClearData() {
	m.Canvas.Clear()
	m.points = nil
}

// Push adds float64 data value to the heat data buffer.
// Data will be scaled using expected max value and sparkline height.
func (m *Model) Push(p HeatPoint) {
	m.AutoAdjustRange(p.AsFloat64Point())
	m.AutoAdjustValueRange(p.V)
	m.points = append(m.points, p)
}

// PushAll adds all data values the heatmap.
func (m *Model) PushAll(pts []HeatPoint) {
	for _, pt := range pts {
		m.Push(pt)
	}
}

// PushAllMatrixRow adds all data in a matrix to the heatmap, using its X,Y indices
// Matrix is row-major, with rows representing the data.
func (m *Model) PushAllMatrixRow(dataRows [][]float64) {
	for x, row := range dataRows {
		for y, val := range row {
			m.Push(NewHeatPoint(float64(x), float64(y), val))
		}
	}
}

// DrawPoint draws a HeatPoint on the heatmap Canvas.
// It does so by adjusting the background.
func (m *Model) DrawPoint(pt HeatPoint) {
	if len(m.ColorScale) == 0 {
		return
	}

	// linear map value to color
	rangeV := m.maxValue - m.minValue
	s := (pt.V - m.minValue) / rangeV
	csi := clamp(int(s*float64(len(m.ColorScale))), 0, len(m.ColorScale)-1)
	color := m.ColorScale[csi]

	// plot on canvas
	sf := m.ScaleFloat64PointForLine(pt.AsFloat64Point())
	cp := canvas.CanvasPointFromFloat64Point(m.Origin(), sf)

	// we just assign the background
	oldStyle := m.Model.Canvas.GetCellStyle(cp)
	if oldStyle == nil { // out of bounds
		oldStyle = &m.cellStyle
	}
	newStyle := (*oldStyle).Background(color)
	m.Model.Canvas.SetCellStyle(cp, newStyle)
}

// Draw will display the data on the canvas.
// Columns representing the data will be displayed going from
// from the bottom to the top and coming from the left to the right of the canvas.
func (m *Model) Draw() {
	for _, pt := range m.points {
		m.DrawPoint(pt)
	}
}

///////////////////////////////////////////////////////////////////////////////
// bubbletea.Model Interface

func (m Model) Init() tea.Cmd {
	return m.Canvas.Init()
}

// Update forwards bubbletea Msg to underlying canvas.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Canvas, cmd = m.Canvas.Update(msg)
	return m, cmd
}

// View returns a string used by the bubbletea framework to display the sparkline.
func (m Model) View() string {
	return m.Canvas.View()
}

///////////////////////////////////////////////////////////////////////////////

func clamp(x, min, max int) int {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}
