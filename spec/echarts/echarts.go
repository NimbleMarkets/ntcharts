// ntcharts - Copyright (c) 2026 Neomantra Corp.

package echarts

import (
	"fmt"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

// ToECharts converts a Spec into a go-echarts/v2 chart renderer.
//
// The concrete return type depends on Spec.Type:
//
//   - spec.ChartTypeBar         -> *charts.Bar
//   - spec.ChartTypeOHLC        -> *charts.Kline (optional volume grid)
//   - spec.ChartTypeTimeSeries  -> *charts.Line (with "time" type X axis)
//
// Callers can render the result with its Render(io.Writer) method, embed it
// in an *echarts.Page, or type-assert it for finer control.
//
// ToECharts does not mutate s.
func ToECharts(s spec.Spec) (any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if spec.IsCategoricalCombo(s) {
		return toEChartsCombo(s)
	}
	for i, ser := range s.Data.Series {
		// Preserve the adapter's existing time-series bar overlay support.
		if ser.Type != "" && ser.Type != string(s.Type) && !(s.Type == spec.ChartTypeTimeSeries && (ser.Type == "bar" || ser.Type == "line")) {
			return nil, fmt.Errorf("echarts: series[%d] type %q is unsupported in %q charts", i, ser.Type, s.Type)
		}
	}
	if s.Type == spec.ChartTypeBar && s.Y2Axis != nil {
		return nil, fmt.Errorf("echarts: bar y2_axis requires a categorical combo")
	}
	switch s.Type {
	case spec.ChartTypeBar:
		return toEChartsBar(s)
	case spec.ChartTypeTimeSeries:
		return toEChartsTimeSeries(s)
	case spec.ChartTypeOHLC:
		return toEChartsOHLC(s)
	case spec.ChartTypeLine,
		spec.ChartTypeStreamline,
		spec.ChartTypeSparkline,
		spec.ChartTypeHeatmap,
		spec.ChartTypeScatter,
		spec.ChartTypeCanvas:
		return nil, fmt.Errorf("echarts: ToECharts for chart type %q is not yet implemented", s.Type)
	default:
		return nil, fmt.Errorf("echarts: unknown chart type %q", s.Type)
	}
}

// toEChartsBar produces a go-echarts Bar chart that mirrors the Spec.
func toEChartsBar(s spec.Spec) (*charts.Bar, error) {
	bar := charts.NewBar()

	bar.SetGlobalOptions(
		titleOpt(s),
		initOpt(s),
		legendOpt(s),
		gridOpt(s),
		charts.WithYAxisOpts(yAxisOpt(s.YAxis, "")),
		charts.WithColorsOpts(s.Theme.Palette),
		charts.WithXAxisOpts(opts.XAxis{
			Type: defaultStr(s.XAxis.Type, spec.XAxisCategory),
			Data: toAnySlice(s.XAxis.Labels),
		}),
	)

	labels := s.XAxis.Labels
	if len(labels) == 0 {
		labels = spec.DeriveBarLabels(s.Data)
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
//  2. Spec.XAxis.Format.Layout is translated by goLayoutToECharts from a Go
//     time layout (e.g. "2006-01") into an ECharts axis-label template
//     (e.g. "{yyyy}-{MM}") and surfaced via AxisLabel.Formatter. Layouts
//     that already look like ECharts templates (containing "{") pass
//     through verbatim; when nothing is translated, no formatter is set and
//     ECharts uses its own default time-axis labels.
//  3. Series Color is passed through via LineStyleOpts. Theme.Palette is
//     wired into the chart's Colors global option.
//  4. YAxis.Min / YAxis.Max pin the Y axis; YAxis.Title is its name.
//     Spec.Y2Axis adds a right-hand axis the same way, and series with
//     YAxis == spec.YAxisRight (lines or bars) are drawn against it. A bar
//     series without it shares the left axis.
func toEChartsTimeSeries(s spec.Spec) (*charts.Line, error) {
	line := charts.NewLine()

	// A series on the right-hand axis (Series.YAxis == spec.YAxisRight) is
	// drawn against the extended axis (index 1) built from Spec.Y2Axis.
	// Bar series are overlaid on the Line chart as real bars, on whichever
	// axis their YAxis names.
	hasBar := false
	for _, ser := range s.Data.Series {
		if ser.Type == "bar" {
			hasBar = true
			break
		}
	}

	line.SetGlobalOptions(
		titleOpt(s),
		initOpt(s),
		legendOpt(s),
		gridOpt(s),
		charts.WithYAxisOpts(yAxisOpt(s.YAxis, "")),
		charts.WithColorsOpts(s.Theme.Palette),
		charts.WithXAxisOpts(opts.XAxis{
			Type:      spec.XAxisTime,
			AxisLabel: xAxisTimeLabelOpt(s.XAxis.Format.Layout),
		}),
	)
	if s.Y2Axis != nil {
		line.ExtendYAxis(yAxisOpt(*s.Y2Axis, "right"))
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
		for i, p := range ser.Values {
			t, ok := spec.PointTime(spec.PointX(p, i, s.Data.XAxisData))
			if !ok {
				// Skip points that cannot be interpreted as time — a strict
				// renderer would error here; we prefer best-effort rendering.
				continue
			}
			items = append(items, opts.LineData{Value: []any{t.Format(time.RFC3339Nano), p.Y}})
		}
		seriesOpts := []charts.SeriesOpts{}
		if ser.YAxis == spec.YAxisRight {
			seriesOpts = append(seriesOpts, charts.WithLineChartOpts(opts.LineChart{YAxisIndex: 1}))
		}
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
			for i, p := range ser.Values {
				t, ok := spec.PointTime(spec.PointX(p, i, s.Data.XAxisData))
				if !ok {
					continue
				}
				items = append(items, opts.BarData{Value: []any{t.Format(time.RFC3339Nano), p.Y}})
			}
			seriesOpts := []charts.SeriesOpts{}
			if ser.YAxis == spec.YAxisRight {
				seriesOpts = append(seriesOpts, charts.WithBarChartOpts(opts.BarChart{YAxisIndex: 1}))
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
func titleOpt(s spec.Spec) charts.GlobalOpts {
	return charts.WithTitleOpts(opts.Title{
		Title:    s.Title,
		Subtitle: s.Subtitle,
	})
}

// initOpt configures the chart canvas (pixel dimensions, background).
func initOpt(s spec.Spec) charts.GlobalOpts {
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
func legendOpt(s spec.Spec) charts.GlobalOpts {
	return charts.WithLegendOpts(opts.Legend{Show: opts.Bool(s.Options.ShowLegend)})
}

// gridOpt toggles the background grid according to Options.ShowGrid.
func gridOpt(s spec.Spec) charts.GlobalOpts {
	return charts.WithGridOpts(opts.Grid{Show: opts.Bool(s.Options.ShowGrid)})
}

// yAxisOpt describes one Y axis: its title as the ECharts axis name, Min /
// Max pins when they are non-nil, and its position ("" for the default
// left, "right" for a second axis).
func yAxisOpt(a spec.YAxis, position string) opts.YAxis {
	y := opts.YAxis{Type: spec.XAxisValue, Name: a.Title, Position: position}
	if a.Min != nil {
		y.Min = *a.Min
	}
	if a.Max != nil {
		y.Max = *a.Max
	}
	return y
}

// goLayoutReplacer best-effort-translates the common Go time-layout tokens
// into their ECharts axis-label template equivalents.
var goLayoutReplacer = strings.NewReplacer(
	"2006", "{yyyy}", "01", "{MM}", "02", "{dd}",
	"15", "{HH}", "04", "{mm}", "05", "{ss}",
)

// goLayoutToECharts best-effort-translates a Go time layout into an ECharts
// axis-label template ({yyyy}-{MM} style). Layouts already containing "{"
// are assumed to be ECharts templates and pass through verbatim.
func goLayoutToECharts(layout string) string {
	if strings.Contains(layout, "{") {
		return layout
	}
	return goLayoutReplacer.Replace(layout)
}

// xAxisTimeLabelOpt builds the AxisLabel formatter for a time-type X axis
// from a Go time layout (or an already-ECharts-style template). When the
// layout is empty or nothing recognizable was translated, nil is returned so
// ECharts falls back to its own default time-axis labels instead of
// rendering the raw, untranslated layout string as literal text.
func xAxisTimeLabelOpt(layout string) *opts.AxisLabel {
	translated := goLayoutToECharts(layout)
	if translated == layout && !strings.Contains(translated, "{") {
		return nil
	}
	return &opts.AxisLabel{Show: opts.Bool(true), Formatter: types.FuncStr(translated)}
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
