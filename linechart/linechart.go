// Package linechart implements a canvas that displays
// (X,Y) Cartesian coordinates as a line chart
package linechart

import (
	"fmt"
	"math"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/graph"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BrailleGrid implements a 2D grid with (X, Y) coordinates
// used to display Braille Pattern runes.
// Since Braille Pattern runes are 4 high and 2 wide,
// the BrailleGrid will internally scale the width and height
// sizes to match those patterns.
// BrailleGrid uses canvas coordinates system with (0,0) being top left.
type BrailleGrid struct {
	cWidth  int // canvas width
	cHeight int // canvas height

	minX float64
	maxX float64
	minY float64
	maxY float64

	gWidth  int // grid width
	gHeight int // grid height
	grid    *runes.PatternDotsGrid
}

// NewBrailleGrid returns new initialized *BrailleGrid
// with given canvas width, canvas height and data value
// minimums and maximums.
func NewBrailleGrid(w, h int, minX, maxX, minY, maxY float64) *BrailleGrid {
	gridW := w * 2
	gridH := h * 4
	g := BrailleGrid{
		cWidth:  w,
		cHeight: h,
		minX:    minX,
		maxX:    maxX,
		minY:    minY,
		maxY:    maxY,
		gWidth:  gridW,
		gHeight: gridH,
		grid:    runes.NewPatternDotsGrid(gridW, gridH),
	}
	g.Clear()
	return &g
}

// Clear will reset the internal grid
func (g *BrailleGrid) Clear() {
	g.grid.Reset()
}

// GridPoint returns a canvas Point representing a BrailleGrid point
// in the canvas coordinates system from a Float64Point data point
// in the Cartesian coordinates system.
func (g *BrailleGrid) GridPoint(f canvas.Float64Point) canvas.Point {
	var sf canvas.Float64Point
	dx := g.maxX - g.minX
	dy := g.maxY - g.minY
	if dx > 0 {
		xs := float64(g.gWidth-1) / dx
		sf.X = (f.X - g.minX) * xs
	}
	if dy > 0 {
		ys := float64(g.gHeight-1) / dy
		sf.Y = (f.Y - g.minY) * ys
	}
	return canvas.CanvasPointFromFloat64Point(canvas.Point{X: 0, Y: g.gHeight - 1}, sf)
}

// Set will set point on grid from given canvas Point.
func (g *BrailleGrid) Set(p canvas.Point) {
	g.grid.Set(p.X, p.Y)
}

// BraillePatterns returns [][]rune containing
// braille pattern runes to draw on to the canvas.
func (g *BrailleGrid) BraillePatterns() [][]rune {
	return g.grid.BraillePatterns()
}

// Model contains state of a linechart with an embedded canvas.Model
type Model struct {
	Canvas     canvas.Model
	AxisStyle  lipgloss.Style // style applied when drawing X and Y axes
	LabelStyle lipgloss.Style // style applied when drawing X and Y number value
	xStep      int            // number of steps when displaying X axis values
	yStep      int            // number of steps when displaying Y axis values

	// data set and the expected min and max values
	minX float64
	maxX float64
	minY float64
	maxY float64

	origin      canvas.Point // start of X and Y axes lines on canvas for graphing area
	graphWidth  int          // width of graphing area - excludes X axis and labels
	graphHeight int          // height of graphing area - excludes Y axis and labels
}

// New returns a linechart Model initialized with given width, height,
// expected data min values and expected data max values.
// Width and height includes area used for chart labeling.
// If xStep is 0, then will not draw X axis or values below X axis.
// If yStep is 0, then will not draw Y axis or values left of Y axis.
func New(w, h int, minX, maxX, minY, maxY float64, xStep, yStep int) Model {
	return NewWithStyle(w, h, minX, maxX, minY, maxY, xStep, yStep, lipgloss.NewStyle(), lipgloss.NewStyle())
}

// NewWithStyle returns a linechart Model initialized with given width, height,
// expected data min values, expected data max values and styles.
// Width and height includes area used for chart labeling.
// If xStep is 0, then will not draw X axis or values below X axis.
// If yStep is 0, then will not draw Y axis or values left of Y axis.
func NewWithStyle(w, h int, minX, maxX, minY, maxY float64, xStep, yStep int, as lipgloss.Style, ls lipgloss.Style) Model {
	// graph width and height exclude area used by axes
	// origin point is canvas coordinates of where axes are drawn
	origin := canvas.Point{X: 0, Y: h - 1}
	gWidth := w
	gHeight := h
	if yStep > 0 {
		// find out how many spaces left of the Y axis
		// to reserve for axis tick value
		nOffset := len(fmt.Sprintf("%.0f", maxY))
		lenMinY := len(fmt.Sprintf("%.0f", minY))
		if lenMinY > nOffset {
			nOffset = lenMinY
		}
		origin.X += nOffset
		gWidth -= (nOffset + 1) // ignore Y axis and tick values
	}
	if xStep > 0 {
		// use last 2 rows of canvas to plot X axis and tick values
		origin.Y -= 1
		gHeight -= 2
	}
	m := Model{
		Canvas:      canvas.New(w, h),
		AxisStyle:   as,
		LabelStyle:  ls,
		yStep:       yStep,
		xStep:       xStep,
		minX:        minX,
		maxX:        maxX,
		minY:        minY,
		maxY:        maxY,
		origin:      origin,
		graphWidth:  gWidth,
		graphHeight: gHeight,
	}
	return m
}

// resetGraphWidthHeight resets the Model GraphWidth and GraphHeight
func (m *Model) resetGraphWidthHeight() {
	m.graphWidth = m.Canvas.Width()
	m.graphHeight = m.Canvas.Height()
	if m.yStep > 0 {
		// find out how many spaces left of the Y axis
		// to reserve for axis tick value
		nOffset := len(fmt.Sprintf("%.0f", m.maxY))
		lenMinY := len(fmt.Sprintf("%.0f", m.minY))
		if lenMinY > nOffset {
			nOffset = lenMinY
		}
		m.graphWidth -= nOffset
	}
	if m.xStep > 0 {
		// use last 2 rows of canvas to plot X axis and tick values
		m.graphHeight -= 1
	}
}

// Width returns linechart width.
func (m *Model) Width() int {
	return m.Canvas.Width()
}

// Height returns linechart height.
func (m *Model) Height() int {
	return m.Canvas.Height()
}

// GraphWidth returns linechart graphing area width.
func (m *Model) GraphWidth() int {
	return m.graphWidth
}

// GraphHeight returns linechart graphing area height.
func (m *Model) GraphHeight() int {
	return m.graphHeight
}

// MinX returns linechart expected minimum X value.
func (m *Model) MinX() float64 {
	return m.minX
}

// MaxX returns linechart expected maximum X value.
func (m *Model) MaxX() float64 {
	return m.maxX
}

// MinY returns linechart expected minimum Y value.
func (m *Model) MinY() float64 {
	return m.minY
}

// MaxY returns linechart expected maximum Y value.
func (m *Model) MaxY() float64 {
	return m.maxY
}

// XStep returns number of steps when displaying Y axis values.
func (m *Model) XStep() int {
	return m.xStep
}

// XStep returns number of steps when displaying Y axis values.
func (m *Model) YStep() int {
	return m.yStep
}

// Origin returns a canvas Point with the coordinates
// of the linechart graph (X,Y) origin.
func (m *Model) Origin() canvas.Point {
	return m.origin
}

// Clear will reset linechart canvas including axes and labels.
func (m *Model) Clear() {
	m.Canvas.Clear()
}

// SetXStep updates the number of steps when displaying X axis values.
func (m *Model) SetXStep(xStep int) {
	m.xStep = xStep
	m.resetGraphWidthHeight()
}

// SetYStep updates the number of steps when displaying Y axis values.
func (m *Model) SetYStep(yStep int) {
	m.yStep = yStep
	m.resetGraphWidthHeight()
}

// SetXRange updates the minimum and maximum expected X values.
func (m *Model) SetXRange(min, max float64) {
	m.minX = min
	m.maxX = max
}

// SetYRange updates the minimum and maximum expected Y values.
func (m *Model) SetYRange(min, max float64) {
	m.minY = min
	m.maxY = max
}

// Resize will change linechart display width and height.
// Existing runes on the linechart will not be redrawn.
func (m *Model) Resize(w, h int) {
	m.Canvas.Resize(w, h)
	m.Canvas.ViewWidth = w
	m.Canvas.ViewHeight = h
	m.resetGraphWidthHeight()
}

// drawYLabel draws Y axis values left of the Y axis every n step.
// Repeating values will be hidden.
// Does nothing if n <= 0.
func (m *Model) drawYLabel(n int) {
	// from origin going up, draw data value left of the Y axis every n steps
	// origin X coordinates already set such that there is space available
	if n <= 0 {
		return
	}
	lastVal := fmt.Sprintf("%.0f", m.minY-1)
	rangeSz := m.maxY - m.minY // number of possible expected values
	increment := rangeSz / float64(m.graphHeight)
	for i := 0; i <= m.graphHeight; {
		v := m.minY + (increment * float64(i)) // value to set left of Y axis
		s := fmt.Sprintf("%.0f", v)
		if lastVal != s {
			m.Canvas.SetStringWithStyle(canvas.Point{m.origin.X - len(s), m.origin.Y - i}, s, m.LabelStyle)
			lastVal = s
		}
		i += n
	}
}

// drawXLabel draws X axis values below the X axis every n step.
// Repeating values will be hidden.
// Does nothing if n <= 0.
func (m *Model) drawXLabel(n int) {
	// from origin going right, draw data value left of the Y axis every n steps
	if n <= 0 {
		return
	}
	lastVal := fmt.Sprintf("%.0f", m.minX-1)
	rangeSz := m.maxX - m.minX // number of possible expected values
	increment := rangeSz / float64(m.graphWidth)
	for i := 0; i < m.graphWidth; {
		// can only set if rune to the left of target coordinates is empty
		if c := m.Canvas.Cell(canvas.Point{m.origin.X + i - 1, m.origin.Y + 1}); c.Rune == runes.Null {
			v := m.minX + (increment * float64(i)) // value to set under X axis
			s := fmt.Sprintf("%.0f", v)
			// dont display if number will be cut off or value repeats
			if (lastVal != s) && ((len(s) + i) < m.graphWidth) {
				m.Canvas.SetStringWithStyle(canvas.Point{m.origin.X + i, m.origin.Y + 1}, s, m.LabelStyle)
				lastVal = s
			}
		}
		i += n
	}
}

// DrawXYAxisAndLabel draws the X, Y axes.
func (m *Model) DrawXYAxisAndLabel() {
	drawY := m.yStep > 0
	drawX := m.xStep > 0
	if drawY && drawX {
		graph.DrawXYAxis(&m.Canvas, m.origin, m.AxisStyle)
	} else {
		if drawY { // draw Y axis
			graph.DrawVerticalLineUp(&m.Canvas, m.origin, m.AxisStyle)
		}
		if drawX { // draw X axis
			graph.DrawHorizonalLineRight(&m.Canvas, m.origin, m.AxisStyle)
		}
	}
	m.drawYLabel(m.yStep)
	m.drawXLabel(m.xStep)
}

// scalePoint returns a Float64Point scaled to the graph size
// of the linechart from a Float64Point data point, width and height.
func (m *Model) scalePoint(f canvas.Float64Point, w, h int) (r canvas.Float64Point) {
	// scale factor is graph height/width over its value range
	dx := m.maxX - m.minX
	dy := m.maxY - m.minY
	if dx > 0 {
		xs := float64(w) / dx
		r.X = (f.X - m.minX) * xs
	}
	if dy > 0 {
		ys := float64(h) / dy
		r.Y = (f.Y - m.minY) * ys
	}
	return
}

// ScaleFloat64Point returns a Float64Point scaled to the graph size
// of the linechart from a Float64Point data point.
func (m *Model) ScaleFloat64Point(f canvas.Float64Point) (r canvas.Float64Point) {
	// if rounded to the nearest integer would be between 0 to graph width/height,
	// and indexing the full graph width/height would be outside of the canvas
	return m.scalePoint(f, m.graphWidth-1, m.graphHeight-1)
}

// ScaleFloat64PointForLine returns a Float64Point scaled to the graph size
// of the linechart from a Float64Point data point.  Used when drawing line runes
// with line styles that can combine with the axes.
func (m *Model) ScaleFloat64PointForLine(f canvas.Float64Point) (r canvas.Float64Point) {
	// full graph height and can be used since LineStyle runes
	// can be combined with axes instead of overriding them
	return m.scalePoint(f, m.graphWidth, m.graphHeight)
}

// DrawRune draws the rune with style on to the linechart
// from a given Float64Point data point.
func (m *Model) DrawRune(f canvas.Float64Point, r rune, s lipgloss.Style) {
	sf := m.ScaleFloat64Point(f) // scale Cartesian coordinates data point to graphing area
	p := canvas.CanvasPointFromFloat64Point(m.origin, sf)
	// draw rune avoiding the axes
	if m.yStep > 0 {
		p.X++
	}
	if m.xStep > 0 {
		p.Y--
	}
	m.Canvas.SetCell(p, canvas.NewCellWithStyle(r, s))
}

// DrawRuneLine draws the rune with style on to the linechart
// such that there is an approximate straight line between the two given
// Float64Point data points.
func (m *Model) DrawRuneLine(f1 canvas.Float64Point, f2 canvas.Float64Point, r rune, s lipgloss.Style) {
	// scale Cartesian coordinates data point to graphing area
	sf1 := m.ScaleFloat64Point(f1)
	sf2 := m.ScaleFloat64Point(f2)

	// convert scaled points to canvas points
	p1 := canvas.CanvasPointFromFloat64Point(m.origin, sf1)
	p2 := canvas.CanvasPointFromFloat64Point(m.origin, sf2)

	// draw rune on all canvas coordinates between
	// the two canvas points that approximates a line
	points := graph.GetLinePoints(p1, p2)
	for _, p := range points {
		if m.yStep > 0 {
			p.X++
		}
		if m.xStep > 0 {
			p.Y--
		}
		m.Canvas.SetCell(p, canvas.NewCellWithStyle(r, s))
	}
}

// DrawRuneCircle draws the rune with style on to the linechart
// such that there is an approximate circle of float64 radious around
// the center of a circle at Float64Point data point.
func (m *Model) DrawRuneCircle(c canvas.Float64Point, f float64, r rune, s lipgloss.Style) {
	center := canvas.Point{int(math.Round(c.X)), int(math.Round(c.Y))} // round center to nearest integer
	radius := int(math.Round(f))                                       // round radius to nearest integer

	points := graph.GetCirclePoints(center, radius)
	for _, v := range points {
		// scale Cartesian coordinates data point to graphing area
		sf := m.ScaleFloat64Point(canvas.NewFloat64PointFromPoint(v))
		// convert scaled points to canvas points
		p := canvas.CanvasPointFromFloat64Point(m.origin, sf)
		// draw rune while avoiding drawing outside of graphing area
		// or on the X and Y axes
		ok := (p.X >= m.origin.X) && (p.Y <= m.origin.Y)
		if (m.yStep > 0) && (p.X == m.origin.X) {
			ok = false
		}
		if (m.xStep > 0) && (p.Y == m.origin.Y) {
			ok = false
		}
		if ok {
			m.Canvas.SetCell(p, canvas.NewCellWithStyle(r, s))
		}
	}
}

// DrawLine draws line runes of a given LineStyle and style on to the linechart
// such that there is an approximate straight line between the two given Float64Point data points.
func (m *Model) DrawLine(f1 canvas.Float64Point, f2 canvas.Float64Point, ls runes.LineStyle, s lipgloss.Style) {
	// scale Cartesian coordinates data points to graphing area
	sf1 := m.ScaleFloat64PointForLine(f1)
	sf2 := m.ScaleFloat64PointForLine(f2)

	// convert scaled points to canvas points
	p1 := canvas.CanvasPointFromFloat64Point(m.origin, sf1)
	p2 := canvas.CanvasPointFromFloat64Point(m.origin, sf2)

	// draw line runes on all canvas coordinates between
	// the two canvas points that approximates a line
	points := graph.GetLinePoints(p1, p2)
	if len(points) <= 0 {
		return
	}
	graph.DrawLinePoints(&m.Canvas, points, ls, s)
}

// DrawBrailleLine draws braille line runes of a given LineStyle and style on to the linechart
// such that there is an approximate straight line between the two given Float64Point data points.
// Braille runes will not overlap the axes.
func (m *Model) DrawBrailleLine(f1 canvas.Float64Point, f2 canvas.Float64Point, s lipgloss.Style) {
	bGrid := NewBrailleGrid(m.graphWidth, m.graphHeight, m.minX, m.maxX, m.minY, m.maxY)

	// get braille grid points from two Float64Point data points
	p1 := bGrid.GridPoint(f1)
	p2 := bGrid.GridPoint(f2)

	// set all points in the braille grid between two points that approximates a line
	points := graph.GetLinePoints(p1, p2)
	for _, p := range points {
		bGrid.Set(p)
	}

	// get all rune patterns for braille grid and draw them on to the canvas
	startX := 0
	if m.yStep > 0 {
		startX = m.origin.X + 1
	}
	patterns := bGrid.BraillePatterns()
	graph.DrawBraillePatterns(&m.Canvas, canvas.Point{X: startX, Y: 0}, patterns, s)
}

// DrawBrailleCircle draws braille line runes of a given LineStyle and style on to the linechart
// such that there is an approximate circle of given float64 radius
// around the center of a circle at Float64Point data point.
// Braille runes will not overlap the axes.
func (m *Model) DrawBrailleCircle(p canvas.Float64Point, f float64, s lipgloss.Style) {
	c := canvas.Point{int(math.Round(p.X)), int(math.Round(p.Y))} // round center to nearest integer
	r := int(math.Round(f))                                       // round radius to nearest integer

	// set braille grid points from computed circle points around center
	bGrid := NewBrailleGrid(m.graphWidth, m.graphHeight, m.minX, m.maxX, m.minY, m.maxY)
	points := graph.GetCirclePoints(c, r)
	for _, p := range points {
		bGrid.Set(bGrid.GridPoint(canvas.NewFloat64PointFromPoint(p)))
	}

	// get all rune patterns for braille grid and draw them on to the canvas
	startX := 0
	if m.yStep > 0 {
		startX = m.origin.X + 1
	}
	patterns := bGrid.BraillePatterns()
	graph.DrawBraillePatterns(&m.Canvas, canvas.Point{X: startX, Y: 0}, patterns, s)
}

// getBraillePoint returns a Point for a braille map from a given Point for a canvas.
func (m *Model) getBraillePoint(f canvas.Point) canvas.Point {
	return canvas.Point{X: f.X * 2, Y: f.Y * 4} // braille runes have 4 height and 2 width
}

// Init initializes the linechart.
func (m Model) Init() tea.Cmd {
	return m.Canvas.Init()
}

// Update processes tea.Msg.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Canvas, cmd = m.Canvas.Update(msg)
	return m, cmd
}

// View returns a string used by the bubbletea framework to display the linechart.
func (m Model) View() string {
	return m.Canvas.View()
}
