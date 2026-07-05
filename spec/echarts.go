// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"fmt"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

// ToECharts converts a Spec into a go-echarts/v2 chart renderer.
//
// The concrete return type depends on Spec.Type:
//
//   - ChartTypeBar         -> *charts.Bar
//   - ChartTypeTimeSeries  -> *charts.Line (with "time" type X axis)
//
// Callers can render the result with its Render(io.Writer) method, embed it
// in an *echarts.Page, or type-assert it for finer control.
//
// ToECharts does not mutate s.
func (s Spec) ToECharts() (any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	switch s.Type {
	case ChartTypeBar:
		return s.toEChartsBar()
	case ChartTypeTimeSeries:
		return s.toEChartsTimeSeries()
	case ChartTypeLine,
		ChartTypeStreamline,
		ChartTypeSparkline,
		ChartTypeHeatmap,
		ChartTypeOHLC,
		ChartTypeScatter,
		ChartTypeCanvas:
		return nil, fmt.Errorf("spec: ToECharts for chart type %q is not yet implemented", s.Type)
	default:
		return nil, fmt.Errorf("spec: unknown chart type %q", s.Type)
	}
}

// toEChartsBar produces a go-echarts Bar chart that mirrors the Spec.
func (s Spec) toEChartsBar() (*charts.Bar, error) {
	bar := charts.NewBar()

	bar.SetGlobalOptions(
		s.titleOpt(),
		s.initOpt(),
		s.legendOpt(),
		s.gridOpt(),
		s.yAxisOpt(),
		charts.WithXAxisOpts(opts.XAxis{
			Type: defaultStr(s.Data.XAxisType, XAxisCategory),
			Data: toAnySlice(s.Data.XAxisLabels),
		}),
	)

	labels := s.Data.XAxisLabels
	if len(labels) == 0 {
		labels = deriveBarLabels(s.Data.Series)
	}
	bar.SetXAxis(labels)

	for _, ser := range s.Data.Series {
		items := make([]opts.BarData, 0, len(ser.Values))
		for _, p := range ser.Values {
			item := opts.BarData{Value: p.Y}
			if ser.Color != "" {
				item.ItemStyle = &opts.ItemStyle{Color: ser.Color}
			}
			items = append(items, item)
		}
		bar.AddSeries(ser.Name, items)
	}

	return bar, nil
}

// toEChartsTimeSeries produces a go-echarts Line chart configured with a
// "time" type X axis, so the browser renders the series using real calendar
// time rather than evenly-spaced categorical ticks.
//
// Implementation notes for extenders:
//
//  1. Every DataPoint is emitted as an []any of [time.Time, float64] which
//     go-echarts serialises as a two-element tuple understood by the ECharts
//     frontend when xAxis.type == "time".
//  2. Spec.Options.TimeFormat is surfaced via AxisLabel.Formatter. ECharts
//     uses its own format tokens (e.g. "{yyyy}-{MM}-{dd}"); passing a Go
//     layout through unchanged is best-effort — callers wanting strict
//     control should set TimeFormat to an ECharts-compatible string.
//  3. Series Color is passed through via LineStyleOpts. Theme.Palette is
//     wired into the chart's Colors global option.
//  4. YAxisMin / YAxisMax pin the Y axis. Nil leaves auto-ranging in place.
func (s Spec) toEChartsTimeSeries() (*charts.Line, error) {
	line := charts.NewLine()

	// Combo support: if any series sets Type == "bar", overlay a Bar chart on
	// top of the Line chart and give the bar series its own Y axis (index 1),
	// so wildly different magnitudes (e.g. price vs volume) coexist cleanly.
	hasBar := false
	for _, ser := range s.Data.Series {
		if ser.Type == "bar" {
			hasBar = true
			break
		}
	}

	line.SetGlobalOptions(
		s.titleOpt(),
		s.initOpt(),
		s.legendOpt(),
		s.gridOpt(),
		s.yAxisOpt(),
		charts.WithColorsOpts(s.Theme.Palette),
		charts.WithXAxisOpts(opts.XAxis{
			Type:      XAxisTime,
			AxisLabel: &opts.AxisLabel{Show: opts.Bool(true), Formatter: types.FuncStr(s.Options.TimeFormat)},
		}),
	)
	if hasBar {
		line.ExtendYAxis(opts.YAxis{Type: XAxisValue})
	}

	// ECharts requires an X axis to be set on a Line chart before adding series;
	// for a time axis the values come from the series tuples, so an empty slice
	// is sufficient.
	line.SetXAxis([]any{})

	for _, ser := range s.Data.Series {
		if ser.Type == "bar" {
			continue // handled by the bar overlay below
		}
		items := make([]opts.LineData, 0, len(ser.Values))
		for _, p := range ser.Values {
			t, ok := pointTime(p.X)
			if !ok {
				// Skip points that cannot be interpreted as time — a strict
				// renderer would error here; we prefer best-effort rendering.
				continue
			}
			items = append(items, opts.LineData{Value: []any{t.Format(time.RFC3339Nano), p.Y}})
		}
		seriesOpts := []charts.SeriesOpts{}
		if ser.Color != "" {
			seriesOpts = append(seriesOpts,
				charts.WithLineStyleOpts(opts.LineStyle{Color: ser.Color}),
				charts.WithItemStyleOpts(opts.ItemStyle{Color: ser.Color}),
			)
		}
		line.AddSeries(ser.Name, items, seriesOpts...)
	}

	if hasBar {
		bar := charts.NewBar()
		bar.SetXAxis([]any{})
		for _, ser := range s.Data.Series {
			if ser.Type != "bar" {
				continue
			}
			items := make([]opts.BarData, 0, len(ser.Values))
			for _, p := range ser.Values {
				t, ok := pointTime(p.X)
				if !ok {
					continue
				}
				items = append(items, opts.BarData{Value: []any{t.Format(time.RFC3339Nano), p.Y}})
			}
			seriesOpts := []charts.SeriesOpts{
				charts.WithBarChartOpts(opts.BarChart{YAxisIndex: 1}),
			}
			if ser.Color != "" {
				seriesOpts = append(seriesOpts, charts.WithItemStyleOpts(opts.ItemStyle{Color: ser.Color}))
			}
			bar.AddSeries(ser.Name, items, seriesOpts...)
		}
		line.Overlap(bar)
	}

	return line, nil
}

// titleOpt builds the ECharts title option from the Spec.
func (s Spec) titleOpt() charts.GlobalOpts {
	return charts.WithTitleOpts(opts.Title{
		Title:    s.Title,
		Subtitle: s.Subtitle,
	})
}

// initOpt configures the chart canvas (pixel dimensions, background).
func (s Spec) initOpt() charts.GlobalOpts {
	w, h := s.Width, s.Height
	// Terminal-sized specs are tiny for a web browser; scale up so they
	// render usefully by default while keeping the aspect ratio.
	if w < 200 {
		w *= 8
	}
	if h < 200 {
		h *= 16
	}
	return charts.WithInitializationOpts(opts.Initialization{
		Width:           fmt.Sprintf("%dpx", w),
		Height:          fmt.Sprintf("%dpx", h),
		BackgroundColor: s.Theme.Background,
	})
}

// legendOpt toggles the legend according to Options.ShowLegend.
func (s Spec) legendOpt() charts.GlobalOpts {
	return charts.WithLegendOpts(opts.Legend{Show: opts.Bool(s.Options.ShowLegend)})
}

// gridOpt toggles the background grid according to Options.ShowGrid.
func (s Spec) gridOpt() charts.GlobalOpts {
	return charts.WithGridOpts(opts.Grid{Show: opts.Bool(s.Options.ShowGrid)})
}

// yAxisOpt pins the Y axis if YAxisMin / YAxisMax are non-nil.
func (s Spec) yAxisOpt() charts.GlobalOpts {
	y := opts.YAxis{Type: XAxisValue}
	if s.Options.YAxisMin != nil {
		y.Min = *s.Options.YAxisMin
	}
	if s.Options.YAxisMax != nil {
		y.Max = *s.Options.YAxisMax
	}
	return charts.WithYAxisOpts(y)
}

// defaultStr returns v unless it is empty, in which case fallback is returned.
func defaultStr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// toAnySlice widens a []string to an []any (ECharts' XAxis.Data type).
func toAnySlice(in []string) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}
