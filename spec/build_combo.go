// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"math"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/graph"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

// ComboModel renders a resolved categorical bar/line combo. The embedded
// linechart supplies axes, formatting, canvas and View. Draw and Resize retain
// the resolved data. Panning/zooming are not supported.
type ComboModel struct {
	linechart.Model
	data                CategoricalCombo
	barStyle, lineStyle lipgloss.Style
}

func buildCombo(s Spec) (*ComboModel, error) {
	c, err := ResolveCategoricalCombo(s)
	if err != nil {
		return nil, err
	}
	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ComboSeriesColor(s, c.BarIndex)))
	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ComboSeriesColor(s, c.LineIndex)))
	opts := []linechart.Option{
		linechart.WithXLabelFormatter(func(int, float64) string { return "" }),
		linechart.WithY2Axis(linechart.Axis{Min: c.Min2, Max: c.Max2, LabelFormatter: axisLabelFormatter(s.Y2Axis.Format, ""), LabelStyle: lineStyle}),
	}
	if f := axisLabelFormatter(s.YAxis.Format, ""); f != nil {
		opts = append(opts, linechart.WithYLabelFormatter(f))
	}
	m := &ComboModel{Model: linechart.New(s.Width, s.Height, 0, float64(len(c.Labels)), c.Min, c.Max, opts...), data: c, barStyle: barStyle, lineStyle: lineStyle}
	m.LabelStyle = barStyle
	if s.Theme.Foreground != "" {
		m.AxisStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(s.Theme.Foreground))
	}
	if err := checkGraphSize(m.Model); err != nil {
		return nil, err
	}
	m.Draw()
	return m, nil
}

// Resize recomputes category slots and redraws. Windows too small for the axes
// have an empty canvas and recover when resized larger.
func (m *ComboModel) Resize(w, h int) { m.Model.Resize(max(0, w), max(0, h)); m.Draw() }

// Update handles window resizing without enabling numeric-axis pan/zoom.
func (m ComboModel) Update(msg tea.Msg) (ComboModel, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.Resize(size.Width, size.Height)
	}
	return m, nil
}

// Draw paints bars first, then the line, using the same integer category centers.
// Missing values keep their slots; a missing line value breaks the connection.
func (m *ComboModel) Draw() {
	m.Clear()
	w, h := m.GraphWidth(), m.GraphHeight()
	if w < 1 || h < 1 {
		return
	}
	m.DrawXYAxisAndLabel()
	start := m.Origin().X + 1
	n := len(m.data.Labels)
	centers := categoryCenters(w, n)
	row := func(v float64) int {
		return int(math.Round((m.data.Max - v) / (m.data.Max - m.data.Min) * float64(h)))
	}
	baseline := row(0)
	for x := 0; x < w; x++ {
		m.Canvas.SetCell(canvas.Point{X: start + x, Y: baseline}, canvas.NewCellWithStyle('─', m.AxisStyle))
	}
	drawCategoryLabels(&m.Model, m.data.Labels, centers)
	for i := range m.data.Labels {
		// Partition the entire plot into slots; collisions in dense charts remain
		// deterministic rather than dropping categories or shifting later points.
		left, right := i*w/n, (i+1)*w/n
		center := centers[i]
		width := max(1, (right-left)*3/4)
		if width%2 == 0 {
			width--
		}
		width = min(width, 2*min(center, w-1-center)+1)
		if value := m.data.Bars[i]; value != nil && *value != 0 {
			top := row(math.Max(m.data.Min, math.Min(m.data.Max, *value)))
			for y := max(0, min(top, baseline)); y <= min(h-1, max(top, baseline)); y++ {
				for x := center - (width-1)/2; x <= center+width/2; x++ {
					m.Canvas.SetCell(canvas.Point{X: start + x, Y: y}, canvas.NewCellWithStyle('█', m.barStyle))
				}
			}
		}

	}
	// Work in braille-pixel coordinates, anchored to the exact bar centers.
	g := graph.NewBrailleGrid(w, h, 0, float64(2*w-1), 0, float64(4*h-1))
	point := func(i int) canvas.Float64Point {
		v := *m.data.Line[i]
		y := (v-m.data.Min2)/(m.data.Max2-m.data.Min2)*float64(4*h) - 1
		// The lower axis limit sits on the X axis; keep an in-range point
		// in the last plot pixel instead of losing it beneath that axis.
		if v >= m.data.Min2 && v <= m.data.Max2 {
			y = math.Max(0, math.Min(float64(4*h-1), y))
		}
		return canvas.Float64Point{X: float64(2 * centers[i]), Y: y}
	}
	for i, v := range m.data.Line {
		if v == nil {
			continue
		}
		a, b := point(i), point(i)
		if i+1 < n && m.data.Line[i+1] != nil {
			b = point(i + 1)
		}
		// Clip before interpolation, including segments crossing a pinned range.
		low, high := 0.0, float64(4*h-1)
		if (a.Y < low && b.Y < low) || (a.Y > high && b.Y > high) {
			continue
		}
		originalA, originalB := a, b
		clip := func(p canvas.Float64Point) canvas.Float64Point {
			y := math.Max(low, math.Min(high, p.Y))
			if y != p.Y {
				p.X = originalA.X + (originalB.X-originalA.X)*(y-originalA.Y)/(originalB.Y-originalA.Y)
				p.Y = y
			}
			return p
		}
		a, b = clip(a), clip(b)
		for _, p := range graph.GetLinePoints(g.GridPoint(a), g.GridPoint(b)) {
			g.Set(p)
		}
	}
	for y, row := range g.BraillePatterns() {
		for x, r := range row {
			if r == runes.BrailleBlockOffset {
				continue
			}
			p := canvas.Point{X: start + x, Y: y}
			style := m.lineStyle
			if under := m.Canvas.Cell(p); under.Rune == '█' {
				// A braille rune replaces the entire cell. Carry the solid
				// bar into its background so the line doesn't punch holes.
				style = style.Background(under.Style.GetForeground())
			}
			m.Canvas.SetCell(p, canvas.NewCellWithStyle(r, style))
		}
	}
}
