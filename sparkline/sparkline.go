// Package sparkline implements a canvas that displays time series data
// as a chart with columns moving from right to left.
package sparkline

// File contains a Model using the bubbletea framework
// representing the state of the sparkline
// and options used by the sparkline during initialization with New().

import (
	"math"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/buffer"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/graph"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Option is used to set options when initializing a sparkline. Example:
//
//	sl := New(width, height, WithMaxValue(someValue), WithNoAuto())
type Option func(*Model)

// WithStyle sets the default column style.
func WithStyle(s lipgloss.Style) Option {
	return func(m *Model) {
		m.Style = s
	}
}

// WithKeyMap sets the canvas KeyMap used
// when processing keyboard event messages in Update().
func WithKeyMap(k canvas.KeyMap) Option {
	return func(m *Model) {
		m.Canvas.KeyMap = k
	}
}

// WithUpdateHandler sets the canvas UpdateHandler used
// when processing bubbletea Msg events in Update().
func WithUpdateHandler(h canvas.UpdateHandler) Option {
	return func(m *Model) {
		m.Canvas.UpdateHandler = h
	}
}

// WithNoAuto disables automatically setting the max value
// if new data greater than the current max is added.
func WithNoAuto() Option {
	return func(m *Model) {
		m.Auto = false
	}
}

// WithMaxValue sets the expected maximum data value
// to given float64.
func WithMaxValue(f float64) Option {
	return func(m *Model) {
		m.SetMax(f)
	}
}

// Model contains state of a sparkline
type Model struct {
	Auto   bool           // whether to automatically set max value when adding data
	Style  lipgloss.Style // style applied when drawing columns
	Canvas canvas.Model

	max float64                        // expected maximum data value
	buf *buffer.Float64ScaleRingBuffer // buffer with size as width of canvas
}

// New returns a sparkline Model initialized with given width, height.
// By default, sparkline wil automatically scale to new maximum data values.
// expected data max value and various options.
func New(w, h int, opts ...Option) Model {
	m := Model{
		Auto:   true,
		Style:  lipgloss.NewStyle(),
		Canvas: canvas.New(w, h),
		max:    1,
		buf:    buffer.NewFloat64ScaleRingBuffer(w, 0, float64(h)/1),
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// Width returns sparkline width.
func (m *Model) Width() int {
	return m.Canvas.Width()
}

// Height returns sparkline height.
func (m *Model) Height() int {
	return m.Canvas.Height()
}

// Max returns expected maximum data value.
func (m *Model) Max() float64 {
	return m.max
}

// Scale returns data scaling factor.
func (m *Model) Scale() float64 {
	return m.buf.Scale()
}

// SetMax will update the expected maximum values.
// Existing values will be updated to new scaling.
func (m *Model) SetMax(f float64) {
	m.max = f
	m.buf.SetScale(float64(m.Canvas.Height()) / m.max)
}

// Resize will change sparkline display width and height.
// Existing data values will be updated to new scaling.
// If new width is less than previous width, then
// older data will be lost after resize.
func (m *Model) Resize(w, h int) {
	m.Canvas.Resize(w, h)
	m.Canvas.ViewWidth = w
	m.Canvas.ViewHeight = h
	if m.buf.Size() != w {
		buf := buffer.NewFloat64ScaleRingBuffer(w, 0, float64(h)/m.max)
		for _, f := range m.buf.ReadAllRaw() {
			buf.Push(f)
		}
		m.buf = buf
	} else {
		m.buf.SetScale(float64(h) / m.max)
	}
}

// Clear will reset sparkline canvas and data.
func (m *Model) Clear() {
	m.Canvas.Clear()
	m.buf.Clear()
}

// Push adds float64 data value to sparkline data buffer.
// Negative values will be treated as the value 0.
// Data will be scaled using expected max value and sparkline height.
func (m *Model) Push(f float64) {
	v := math.Max(f, 0)
	if m.Auto && v > m.max {
		m.SetMax(v)
	}
	m.buf.Push(v)
}

// PushAll adds all data values in []float64 to sparkline data buffer.
// Negative values will be treated as the value 0.
// Data will be scaled using expected max value and sparkline height.
func (m *Model) PushAll(f []float64) {
	for _, v := range f {
		m.Push(v)
	}
}

// Draw will display the the scaled data values on to the sparkline canvas
// using columns.
// Sparkline style will be applied across entire canvas.
// Columns representing the data will be displayed going from
// from the bottom to the top and coming from the left to the right of the canvas.
func (m *Model) Draw() {
	m.DrawColumnsOnly()
	m.Canvas.SetStyle(m.Style)
}

// DrawColumnsOnly is the same as Draw except the the style will only be applied
// to the columns and not to the entire canvas.
func (m *Model) DrawColumnsOnly() {
	m.Canvas.Clear()
	d := m.buf.ReadAll()
	graph.DrawColumns(&m.Canvas,
		canvas.Point{m.Canvas.Width() - len(d), m.Canvas.Height() - 1},
		d,
		m.Style)
}

// DrawBraille will display the the scaled data values on to the sparkline canvas
// using braille lines.
// Sparkline style will be applied across entire canvas.
// Braille lines representing the data will be displayed going from
// from the bottom to the top and coming from the left to the right of the canvas.
func (m *Model) DrawBraille() {
	m.Canvas.Clear()
	d := m.buf.ReadAll()
	dLen := len(d)
	grid := graph.NewBrailleGrid(m.Width(), m.Height(),
		0, float64(m.Width()),
		0, float64(m.Height())) // Y values already scaled from buffer
	startX := m.Canvas.Width() - len(d)
	for i := 0; i < dLen; i++ {
		j := i + 1
		if j >= dLen {
			j = i
		}
		gp1 := grid.GridPoint(canvas.Float64Point{X: float64(startX + i), Y: d[i]})
		gp2 := grid.GridPoint(canvas.Float64Point{X: float64(startX + j), Y: d[j]})
		points := graph.GetLinePoints(gp1, gp2)
		for _, p := range points {
			grid.Set(p)
		}
	}
	graph.DrawBraillePatterns(&m.Canvas,
		canvas.Point{X: 0, Y: 0}, grid.BraillePatterns(), m.Style)
	m.Canvas.SetStyle(m.Style)
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

// View returns a string used by the bubbletea framework to display the sparkline.
func (m Model) View() string {
	return m.Canvas.View()
}
