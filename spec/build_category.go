// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"fmt"
	"math"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/graph"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

// CategoricalModel renders line or scatter data on ordered category slots.
// The embedded linechart supplies Y scales and formatting. Missing categories
// retain their slots and break lines. Numeric pan/zoom is disabled.
type CategoricalModel struct {
	linechart.Model
	labels []string
	data   Spec // copied, with each point's X replaced by its category index
}

func buildCategorical(s Spec) (*CategoricalModel, error) {
	if s.XAxis.Scale == ScaleLog {
		return nil, fmt.Errorf("spec: categorical X axis cannot be logarithmic")
	}
	labels := append([]string(nil), s.XAxis.Labels...)
	if len(labels) == 0 && len(s.Data.XAxisData) > 0 {
		for _, v := range s.Data.XAxisData {
			label, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("spec: categorical x_axis_data must contain strings")
			}
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		seen := map[string]bool{}
		for _, ser := range s.Data.Series {
			for _, p := range ser.Values {
				label, ok := p.X.(string)
				if !ok {
					return nil, fmt.Errorf("spec: categorical points require string X values or explicit labels")
				}
				if !seen[label] {
					labels = append(labels, label)
					seen[label] = true
				}
			}
		}
	}
	if len(labels) == 0 {
		return nil, fmt.Errorf("spec: categorical chart requires category labels")
	}
	indexes := map[string]int{}
	for i, label := range labels {
		if _, ok := indexes[label]; ok {
			return nil, fmt.Errorf("spec: duplicate category %q", label)
		}
		indexes[label] = i
	}
	data := s
	data.XAxis = s.XAxis
	data.XAxis.Type = XAxisValue
	data.XAxis.Labels = nil
	data.XAxis.Format = Format{}
	data.Data = Data{Series: make([]Series, len(s.Data.Series))}
	for i, ser := range s.Data.Series {
		data.Data.Series[i] = ser
		points := make([]DataPoint, len(ser.Values))
		used := map[int]bool{}
		for j, p := range ser.Values {
			if math.IsNaN(p.Y) || math.IsInf(p.Y, 0) {
				return nil, fmt.Errorf("spec: categorical Y values must be finite; omit missing points")
			}
			idx := j
			if x := PointX(p, j, s.Data.XAxisData); x != nil {
				label, ok := x.(string)
				if !ok {
					return nil, fmt.Errorf("spec: categorical point X must be a string")
				}
				idx, ok = indexes[label]
				if !ok {
					return nil, fmt.Errorf("spec: unknown category %q", label)
				}
			}
			if idx >= len(labels) {
				return nil, fmt.Errorf("spec: more positional points than categories")
			}
			if s.Type == ChartTypeLine && used[idx] {
				return nil, fmt.Errorf("spec: duplicate line point for category %q in series %q", labels[idx], ser.Name)
			}
			used[idx] = true
			points[j] = p
			points[j].X = float64(idx)
		}
		if s.Type == ChartTypeLine {
			sort.SliceStable(points, func(a, b int) bool { return points[a].X.(float64) < points[b].X.(float64) })
		}
		data.Data.Series[i].Values = points
	}
	var base *linechart.Model
	var err error
	if s.Type == ChartTypeLine {
		base, err = buildLine(data)
	} else {
		var model any
		model, err = buildScatter(data)
		if err == nil {
			base = model.(*linechart.Model)
		}
	}
	if err != nil {
		return nil, err
	}
	base.XLabelFormatter = func(int, float64) string { return "" }
	base.SetXRange(0, float64(len(labels)))
	base.SetViewXRange(0, float64(len(labels)))
	m := &CategoricalModel{Model: *base, labels: labels, data: data}
	m.Draw()
	return m, nil
}

// Resize redraws category slots and labels without discarding the data.
func (m *CategoricalModel) Resize(w, h int) { m.Model.Resize(max(0, w), max(0, h)); m.Draw() }

// Update accepts window sizes without allowing numeric category-axis pan/zoom.
func (m CategoricalModel) Update(msg tea.Msg) (CategoricalModel, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.Resize(size.Width, size.Height)
	}
	return m, nil
}

// Draw recomputes slot centers from the current plot width. Declared empty
// categories consume space; dense labels are truncated or skipped, never moved.
func (m *CategoricalModel) Draw() {
	m.Clear()
	w, h := m.GraphWidth(), m.GraphHeight()
	if w < 1 || h < 1 {
		return
	}
	m.DrawXYAxisAndLabel()
	centers := categoryCenters(w, len(m.labels))
	drawCategoryLabels(&m.Model, m.labels, centers)
	for i, ser := range m.data.Data.Series {
		style := seriesStyle(ser, i, m.data.Theme)
		y := func(v float64) float64 {
			if ser.YAxis == YAxisRight {
				return m.Y2ToY(v)
			}
			return v
		}
		if m.data.Type == ChartTypeScatter {
			for _, p := range ser.Values {
				center := centers[int(p.X.(float64))]
				x := float64(len(m.labels)) * float64(center) / float64(max(1, w-1))
				m.DrawRuneWithStyle(canvas.Float64Point{X: x, Y: y(p.Y)}, '•', style)
			}
			continue
		}
		g := graph.NewBrailleGrid(w, h, 0, float64(2*w-1), 0, float64(4*h-1))
		point := func(p DataPoint) canvas.Float64Point {
			scaled := m.ScaleFloat64Point(canvas.Float64Point{Y: y(p.Y)})
			return canvas.Float64Point{X: float64(2 * centers[int(p.X.(float64))]), Y: 4 * scaled.Y}
		}
		for j, p := range ser.Values {
			a, b := point(p), point(p)
			if j+1 < len(ser.Values) && ser.Values[j+1].X.(float64) == p.X.(float64)+1 {
				b = point(ser.Values[j+1])
			}
			low, high := 0.0, float64(4*h-1)
			if (a.Y < low && b.Y < low) || (a.Y > high && b.Y > high) {
				continue
			}
			origA, origB := a, b
			clip := func(p canvas.Float64Point) canvas.Float64Point {
				v := math.Max(low, math.Min(high, p.Y))
				if v != p.Y {
					p.X = origA.X + (origB.X-origA.X)*(v-origA.Y)/(origB.Y-origA.Y)
					p.Y = v
				}
				return p
			}
			a, b = clip(a), clip(b)
			for _, p := range graph.GetLinePoints(g.GridPoint(a), g.GridPoint(b)) {
				g.Set(p)
			}
		}
		graph.DrawBraillePatterns(&m.Canvas, canvas.Point{X: m.Origin().X + 1, Y: 0}, g.BraillePatterns(), style)
	}
}
