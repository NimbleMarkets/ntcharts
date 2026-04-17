// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"fmt"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"

	"charm.land/lipgloss/v2"
)

// Build converts a Spec into the appropriate ntcharts terminal model.
//
// The concrete return type depends on Spec.Type:
//
//   - ChartTypeBar         -> *barchart.Model
//   - ChartTypeTimeSeries  -> *timeserieslinechart.Model
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
	case ChartTypeLine,
		ChartTypeStreamline,
		ChartTypeSparkline,
		ChartTypeHeatmap,
		ChartTypeOHLC,
		ChartTypeScatter,
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
//	spec.Data.XAxisLabels    -> per-bar Label
//	spec.Options.YAxisMax    -> WithMaxValue (if set; otherwise auto-max)
//	spec.Series[i].Color     -> BarValue.Style foreground
//	spec.Theme.Palette       -> fallback colour for series without Color
//
// Each X-axis label becomes a single bar whose stacked segments are the
// Y value of every series at the matching index. When XAxisLabels is empty,
// labels are derived from the first series' DataPoint.X values.
func buildBar(s Spec) (*barchart.Model, error) {
	labels := s.Data.XAxisLabels
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
	if s.Options.YAxisMax != nil {
		opts = append(opts, barchart.WithMaxValue(*s.Options.YAxisMax))
	}

	m := barchart.New(s.Width, s.Height, opts...)
	m.Draw()
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
//  3. YAxisMin / YAxisMax pin the Y axis via WithYRange. When both are nil
//     the chart auto-scales based on the pushed points.
//  4. Options.TimeFormat is reserved for future use: threading a custom
//     XLabelFormatter through requires using linechart.WithXLabelFormatter
//     at construction time; currently the default DateTimeLabelFormatter is
//     used.
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
	if s.Options.YAxisMin != nil && s.Options.YAxisMax != nil {
		opts = append(opts, timeserieslinechart.WithYRange(*s.Options.YAxisMin, *s.Options.YAxisMax))
	}

	m := timeserieslinechart.New(s.Width, s.Height, opts...)

	for i, ser := range s.Data.Series {
		name := ser.Name
		if name == "" {
			name = timeserieslinechart.DefaultDataSetName
		}
		m.SetDataSetStyle(name, seriesStyle(ser, i, s.Theme))

		for _, p := range ser.Values {
			t, ok := pointTime(p.X)
			if !ok {
				continue
			}
			m.PushDataSet(name, timeserieslinechart.TimePoint{Time: t, Value: p.Y})
		}
	}

	m.DrawBraille()
	return &m, nil
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
