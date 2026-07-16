// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"fmt"
	"math"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/wavelinechart"

	"charm.land/lipgloss/v2"
)

// Build converts a Spec into the appropriate ntcharts terminal model.
//
// The concrete return type depends on Spec.Type:
//
//   - ChartTypeBar         -> *barchart.Model
//   - ChartTypeLine        -> *wavelinechart.Model
//   - ChartTypeTimeSeries  -> *timeserieslinechart.Model
//   - ChartTypeScatter     -> *linechart.Model
//   - ChartTypeHeatmap     -> *heatmap.Model
//
// Callers should type-assert the result. Unsupported chart types return an
// error rather than panicking, so future additions to ChartType do not break
// existing callers.
//
// Build does not mutate s.
func Build(s Spec) (any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	switch s.Type {
	case ChartTypeBar:
		return buildBar(s)
	case ChartTypeTimeSeries:
		return buildTimeSeries(s)
	case ChartTypeLine:
		return buildLine(s)
	case ChartTypeScatter:
		return buildScatter(s)
	case ChartTypeHeatmap:
		return buildHeatmap(s)
	case ChartTypeStreamline,
		ChartTypeSparkline,
		ChartTypeOHLC,
		ChartTypeCanvas:
		return nil, fmt.Errorf("spec: Build for chart type %q is not yet implemented", s.Type)
	default:
		return nil, fmt.Errorf("spec: unknown chart type %q", s.Type)
	}
}

// buildBar constructs a *barchart.Model from s.
//
// The bar chart is built along the following mapping:
//
//	spec.Series              -> stacked BarValue segments within each bar
//	spec.XAxis.Labels        -> per-bar Label
//	spec.YAxis.Max           -> WithMaxValue (if set; otherwise auto-max)
//	spec.Series[i].Color     -> BarValue.Style foreground
//	spec.Theme.Palette       -> fallback colour for series without Color
//
// Each X-axis label becomes a single bar whose stacked segments are the
// Y value of every series at the matching index. When XAxis.Labels is empty,
// labels are derived from the first series' DataPoint.X values.
func buildBar(s Spec) (*barchart.Model, error) {
	labels := s.XAxis.Labels
	if len(labels) == 0 {
		labels = deriveBarLabels(s.Data.Series)
	}
	if len(labels) == 0 {
		return nil, fmt.Errorf("spec: bar chart requires XAxisLabels or per-series X values")
	}

	data := make([]barchart.BarData, len(labels))
	for i, label := range labels {
		values := make([]barchart.BarValue, 0, len(s.Data.Series))
		for j, ser := range s.Data.Series {
			if i >= len(ser.Values) {
				continue
			}
			values = append(values, barchart.BarValue{
				Name:  ser.Name,
				Value: ser.Values[i].Y,
				Style: seriesStyle(ser, j, s.Theme),
			})
		}
		data[i] = barchart.BarData{Label: label, Values: values}
	}

	opts := []barchart.Option{barchart.WithDataSet(data)}
	if s.YAxis.Max != nil {
		opts = append(opts, barchart.WithMaxValue(*s.YAxis.Max))
	}

	m := barchart.New(s.Width, s.Height, opts...)
	m.Draw()
	return &m, nil
}

// resolveXFloat resolves a point's numeric X: the point's own X, else the
// shared Data.XAxisData at idx, else the index itself. Shared with buildScatter.
func resolveXFloat(s Spec, p DataPoint, idx int) float64 {
	if p.X != nil {
		if x, ok := pointFloat(p.X); ok {
			return x
		}
	}
	if idx < len(s.Data.XAxisData) {
		if x, ok := pointFloat(s.Data.XAxisData[idx]); ok {
			return x
		}
	}
	return float64(idx)
}

// buildLine constructs a *wavelinechart.Model from s.
//
// Each spec.Series becomes a named data set (see PlotDataSet /
// SetDataSetStyles). DataPoint.X is resolved via resolveXFloat: the point's
// own numeric X, else the shared Data.XAxisData at that index, else the
// point's index — so callers may omit X entirely and rely on index-as-X.
// YAxis.Min / YAxis.Max pin the Y axis via WithYRange when both are set;
// otherwise the chart auto-scales.
//
// wavelinechart.Model embeds linechart.Model, which exposes the
// XLabelFormatter / YLabelFormatter fields publicly; when XAxis.Format /
// YAxis.Format carry a formatting directive, they are wired in before
// DrawAll so terminal axis labels match the same Format used elsewhere
// (e.g. ToECharts).
func buildLine(s Spec) (*wavelinechart.Model, error) {
	var opts []wavelinechart.Option
	if s.YAxis.Min != nil && s.YAxis.Max != nil {
		opts = append(opts, wavelinechart.WithYRange(*s.YAxis.Min, *s.YAxis.Max))
	}
	m := wavelinechart.New(s.Width, s.Height, opts...)

	if xf := s.XAxis.Format.labelFormatter(); xf != nil {
		m.XLabelFormatter = xf
	}
	if yf := s.YAxis.Format.labelFormatter(); yf != nil {
		m.YLabelFormatter = yf
	}

	for i, ser := range s.Data.Series {
		name := ser.Name
		m.SetDataSetStyles(name, runes.ArcLineStyle, seriesStyle(ser, i, s.Theme))
		for j, p := range ser.Values {
			m.PlotDataSet(name, canvas.Float64Point{X: resolveXFloat(s, p, j), Y: p.Y})
		}
	}
	m.DrawAll()
	return &m, nil
}

// buildScatter renders points onto a base linechart canvas. ntcharts has no
// scatter model; the spec surface owns range computation and point drawing.
// DataPoint.Size is accepted in the schema but ignored by this surface.
func buildScatter(s Spec) (any, error) {
	type styledPoint struct {
		x, y  float64
		style lipgloss.Style
	}
	var pts []styledPoint
	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for i, ser := range s.Data.Series {
		st := seriesStyle(ser, i, s.Theme)
		for j, p := range ser.Values {
			x := resolveXFloat(s, p, j)
			pts = append(pts, styledPoint{x: x, y: p.Y, style: st})
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
		}
	}
	if len(pts) == 0 {
		return nil, fmt.Errorf("spec: scatter requires at least one data point")
	}
	if s.YAxis.Min != nil {
		minY = *s.YAxis.Min
	}
	if s.YAxis.Max != nil {
		maxY = *s.YAxis.Max
	}
	if minX == maxX {
		minX, maxX = minX-1, maxX+1
	}
	if minY == maxY {
		minY, maxY = minY-1, maxY+1
	}
	var opts []linechart.Option
	if xf := s.XAxis.Format.labelFormatter(); xf != nil {
		opts = append(opts, linechart.WithXLabelFormatter(xf))
	}
	if yf := s.YAxis.Format.labelFormatter(); yf != nil {
		opts = append(opts, linechart.WithYLabelFormatter(yf))
	}
	m := linechart.New(s.Width, s.Height, minX, maxX, minY, maxY, opts...)
	m.DrawXYAxisAndLabel()
	for _, p := range pts {
		m.DrawRuneWithStyle(canvas.Float64Point{X: p.x, Y: p.y}, '•', p.style)
	}
	return &m, nil
}

// buildTimeSeries constructs a *timeserieslinechart.Model from s.
//
// Implementation notes and TODOs for contributors extending this:
//
//  1. Each spec.Series becomes a named data set in the timeserieslinechart
//     (see SetDataSetStyle / PushDataSet). The default data set name from
//     timeserieslinechart is reused for the first series when it has no name.
//  2. DataPoint.X accepts time.Time or RFC3339 strings or numeric ms; the
//     helper pointTime() converts all three.
//  3. YAxis.Min / YAxis.Max pin the Y axis via WithYRange. When both are nil
//     the chart auto-scales based on the pushed points.
//  4. YAxis.Format / XAxis.Format (when Kind == "time") thread custom
//     LabelFormatters through WithYLabelFormatter / WithXLabelFormatter at
//     construction time; when unset, the chart's own defaults (including
//     DateTimeLabelFormatter for X) are used.
//  5. Per-series Color is applied via SetDataSetStyle using a lipgloss
//     foreground. Background / gridlines come from Theme.
//
// This implementation covers the common case: one or more time-indexed
// series with a shared auto-ranged axis. Streamlined, stacked, and
// multi-axis variants are left for future extensions.
func buildTimeSeries(s Spec) (*timeserieslinechart.Model, error) {
	tMin, tMax, ok := timeBounds(s.Data.Series)
	if !ok {
		return nil, fmt.Errorf("spec: timeseries chart requires at least one DataPoint with a time X value")
	}

	opts := []timeserieslinechart.Option{
		timeserieslinechart.WithTimeRange(tMin, tMax),
	}
	if s.YAxis.Min != nil && s.YAxis.Max != nil {
		opts = append(opts, timeserieslinechart.WithYRange(*s.YAxis.Min, *s.YAxis.Max))
	}
	if yf := s.YAxis.Format.labelFormatter(); yf != nil {
		opts = append(opts, timeserieslinechart.WithYLabelFormatter(yf))
	}
	if xf := s.XAxis.Format.labelFormatter(); xf != nil && s.XAxis.Format.Kind == "time" {
		// timeserieslinechart's default DateTimeLabelFormatter (see
		// linechart/timeserieslinechart/timeserieslinechart.go) receives X
		// label values as seconds-since-epoch (it does
		// time.UnixMilli(int64(math.Round(v*1e3)))); mirror that conversion
		// exactly here, only swapping in the custom layout.
		layout := s.XAxis.Format.Layout
		if layout == "" {
			layout = "2006-01-02"
		}
		opts = append(opts, timeserieslinechart.WithXLabelFormatter(
			makeTimeAxisFormatter(layout)))
	}

	m := timeserieslinechart.New(s.Width, s.Height, opts...)

	for i, ser := range s.Data.Series {
		name := ser.Name
		if name == "" {
			name = timeserieslinechart.DefaultDataSetName
		}
		m.SetDataSetStyle(name, seriesStyle(ser, i, s.Theme))

		// Honour per-series Type: "bar" series on a time axis cannot be drawn
		// as real bars in the terminal (timeserieslinechart is a line chart),
		// so they are drawn as a line with a distinct line style as a visual
		// hint. ECharts renders them as true bars on a secondary Y axis.
		if ser.Type == "bar" {
			m.SetDataSetLineStyle(name, runes.ThinLineStyle)
		}

		for _, p := range ser.Values {
			t, ok := pointTime(p.X)
			if !ok {
				continue
			}
			m.PushDataSet(name, timeserieslinechart.TimePoint{Time: t, Value: p.Y})
		}
	}

	m.DrawBrailleAll()
	return &m, nil
}

// makeTimeAxisFormatter returns a linechart.LabelFormatter for the X axis of
// a timeserieslinechart that renders the given Go time layout instead of the
// package default DateTimeLabelFormatter's "MM/DD" / "'YY MM/DD" scheme.
//
// timeserieslinechart passes X label values as seconds-since-epoch (see
// DateTimeLabelFormatter in linechart/timeserieslinechart/timeserieslinechart.go,
// which computes time.UnixMilli(int64(math.Round(v * 1e3)))); this mirrors
// that value->time conversion exactly, only swapping in a custom layout.
func makeTimeAxisFormatter(layout string) linechart.LabelFormatter {
	return func(_ int, v float64) string {
		t := time.UnixMilli(int64(math.Round(v * 1e3))).UTC()
		return t.Format(layout)
	}
}

// deriveBarLabels falls back to the first series' DataPoint.X values
// (coerced to string) when the Spec did not specify explicit X-axis labels.
func deriveBarLabels(series []Series) []string {
	if len(series) == 0 {
		return nil
	}
	first := series[0].Values
	labels := make([]string, 0, len(first))
	for i, p := range first {
		if s, ok := pointString(p.X); ok {
			labels = append(labels, s)
			continue
		}
		labels = append(labels, fmt.Sprintf("%d", i+1))
	}
	return labels
}

// timeBounds scans every DataPoint.X in series and returns the minimum and
// maximum time.Time. The third return value is false if no series contained
// a time-valued X.
func timeBounds(series []Series) (time.Time, time.Time, bool) {
	var (
		min, max time.Time
		found    bool
	)
	for _, ser := range series {
		for _, p := range ser.Values {
			t, ok := pointTime(p.X)
			if !ok {
				continue
			}
			if !found {
				min, max = t, t
				found = true
				continue
			}
			if t.Before(min) {
				min = t
			}
			if t.After(max) {
				max = t
			}
		}
	}
	return min, max, found
}

// buildHeatmap renders Heat data onto a heatmap model. Theme.Gradient (hex
// stops) becomes an interpolated color scale; absent, the package default
// grayscale applies.
func buildHeatmap(s Spec) (any, error) {
	if s.Heat == nil || (len(s.Heat.Cells) == 0 && len(s.Heat.Matrix) == 0) {
		return nil, fmt.Errorf("spec: heatmap requires Heat data (cells or matrix)")
	}
	var opts []heatmap.Option
	cs, err := gradientScale(s.Theme.Gradient)
	if err != nil {
		return nil, err
	}
	if cs != nil {
		opts = append(opts, heatmap.WithColorScale(cs))
	}
	if s.Heat.MinValue != nil && s.Heat.MaxValue != nil {
		opts = append(opts, heatmap.WithValueRange(*s.Heat.MinValue, *s.Heat.MaxValue))
	} else {
		opts = append(opts, heatmap.WithAutoValueRange())
	}
	m := heatmap.New(s.Width, s.Height, opts...)
	if len(s.Heat.Matrix) > 0 {
		m.PushAllMatrixRow(s.Heat.Matrix)
	} else {
		for _, c := range s.Heat.Cells {
			m.Push(heatmap.NewHeatPoint(c.X, c.Y, c.Z))
		}
	}
	m.Draw()
	return &m, nil
}

// seriesStyle returns the lipgloss.Style for a Series, preferring the
// series' own Color, then the Theme palette (by index), then the default.
func seriesStyle(ser Series, index int, theme Theme) lipgloss.Style {
	colour := ser.Color
	if colour == "" && index < len(theme.Palette) {
		colour = theme.Palette[index]
	}
	if colour == "" {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colour))
}
