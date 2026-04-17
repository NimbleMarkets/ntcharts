// ntcharts - Copyright (c) 2026 Neomantra Corp.

// Package spec defines a neutral, surface-agnostic chart specification.
//
// A Spec is a declarative description of a chart (type, data, options, theme)
// that can be rendered to multiple surfaces:
//
//   - ntcharts terminal models via Build()
//   - Apache ECharts (go-echarts/v2) web charts via Spec.ToECharts()
//
// The goal is a single source of truth: author once, render anywhere.
// Specs are JSON-marshalable so they can be persisted, transported over the
// wire, or authored in configuration files.
package spec

import (
	"encoding/json"
	"fmt"
)

// ChartType identifies the kind of chart described by a Spec.
// The constants below mirror the chart families exposed by ntcharts.
type ChartType string

// ChartType constants. These mirror the ntcharts chart families. The Streamline,
// Sparkline, Heatmap, OHLC, Scatter and Canvas constants are reserved for
// future rendering support; Build and ToECharts currently implement Bar and
// TimeSeries fully.
const (
	// ChartTypeBar renders a bar chart (barchart package).
	ChartTypeBar ChartType = "bar"
	// ChartTypeLine renders a generic (numeric-X) line chart (linechart package).
	ChartTypeLine ChartType = "line"
	// ChartTypeTimeSeries renders a time-indexed line chart
	// (linechart/timeserieslinechart package).
	ChartTypeTimeSeries ChartType = "timeseries"
	// ChartTypeStreamline renders a streamline chart
	// (linechart/streamlinechart package).
	ChartTypeStreamline ChartType = "streamline"
	// ChartTypeSparkline renders a sparkline (sparkline package).
	ChartTypeSparkline ChartType = "sparkline"
	// ChartTypeHeatmap renders a heatmap (heatmap package).
	ChartTypeHeatmap ChartType = "heatmap"
	// ChartTypeOHLC renders an open/high/low/close financial chart.
	ChartTypeOHLC ChartType = "ohlc"
	// ChartTypeScatter renders a scatter plot.
	ChartTypeScatter ChartType = "scatter"
	// ChartTypeCanvas renders a raw canvas chart (canvas package).
	ChartTypeCanvas ChartType = "canvas"
)

// IsKnown reports whether the ChartType is one of the recognized constants.
func (t ChartType) IsKnown() bool {
	switch t {
	case ChartTypeBar, ChartTypeLine, ChartTypeTimeSeries,
		ChartTypeStreamline, ChartTypeSparkline, ChartTypeHeatmap,
		ChartTypeOHLC, ChartTypeScatter, ChartTypeCanvas:
		return true
	}
	return false
}

// X-axis type constants used by Data.XAxisType.
const (
	// XAxisCategory indicates a discrete, ordered set of labels
	// (e.g. bar chart categories).
	XAxisCategory = "category"
	// XAxisTime indicates time-valued X coordinates (time.Time or ms-since-epoch).
	XAxisTime = "time"
	// XAxisValue indicates numeric (float64) X coordinates.
	XAxisValue = "value"
)

// Spec is the top-level neutral description of a chart.
//
// A Spec fully describes the chart. Surface-specific concerns (styles,
// interactivity, mouse/keyboard handling) are derived from Spec by the
// corresponding renderer.
type Spec struct {
	// Type selects the chart family. Required.
	Type ChartType `json:"type"`
	// Title is the optional main chart title.
	Title string `json:"title,omitempty"`
	// Subtitle is an optional secondary title.
	Subtitle string `json:"subtitle,omitempty"`
	// Width is the suggested width of the chart. For terminal surfaces this
	// is columns; for web surfaces this is a pixel hint.
	Width int `json:"width"`
	// Height is the suggested height. Columns/rows on terminal, pixels on web.
	Height int `json:"height"`

	// Data holds the series and axis data for the chart.
	Data Data `json:"data"`
	// Options holds common, surface-agnostic rendering options.
	Options Options `json:"options,omitempty"`
	// Theme holds surface-agnostic colour theming.
	Theme Theme `json:"theme,omitempty"`
}

// Data describes the axes and series of a chart.
type Data struct {
	// Series is the ordered collection of data series. At least one series is
	// required for most chart types.
	Series []Series `json:"series"`
	// XAxisType declares the kind of X axis: "category", "time", or "value".
	// If empty, a sensible default is inferred from the chart type.
	XAxisType string `json:"x_axis_type,omitempty"`
	// XAxisLabels is the optional list of category labels, used when
	// XAxisType == "category".
	XAxisLabels []string `json:"x_axis_labels,omitempty"`
	// XAxisData is an optional shared X-axis value list (time.Time, float64,
	// string, or int). If provided, per-series DataPoint.X may be omitted and
	// points are zipped against this slice.
	XAxisData []any `json:"x_axis_data,omitempty"`
}

// Series is a named collection of DataPoints.
type Series struct {
	// Name is the series name (shown in legends and tooltips). Required.
	Name string `json:"name"`
	// Type optionally overrides the parent Spec.Type for this series
	// (e.g. mixing "line" into a "bar" chart).
	Type string `json:"type,omitempty"`
	// Values is the ordered list of points in the series.
	Values []DataPoint `json:"values"`
	// Color is an optional per-series colour (hex, "#rrggbb", or a named
	// colour). If empty, the Theme palette or surface default is used.
	Color string `json:"color,omitempty"`
}

// DataPoint is a single datum in a Series.
//
// X may be a string (category), time.Time (time series), float64 or int
// (numeric value). When unmarshaled from JSON, time values are represented
// as RFC 3339 strings and resolved by helpers in this package when needed.
type DataPoint struct {
	// X is the independent-axis coordinate.
	X any `json:"x,omitempty"`
	// Y is the dependent-axis value.
	Y float64 `json:"y"`
}

// Options are common, surface-agnostic rendering options.
//
// Surfaces are free to ignore options they do not support.
type Options struct {
	// ShowLegend toggles the legend.
	ShowLegend bool `json:"show_legend"`
	// ShowGrid toggles the background grid.
	ShowGrid bool `json:"show_grid"`
	// YAxisMin pins the Y axis minimum. If nil, auto-scale.
	YAxisMin *float64 `json:"y_axis_min,omitempty"`
	// YAxisMax pins the Y axis maximum. If nil, auto-scale.
	YAxisMax *float64 `json:"y_axis_max,omitempty"`
	// TimeFormat is a Go time layout string applied to time-axis labels.
	// Ignored by non-time charts.
	TimeFormat string `json:"time_format,omitempty"`
}

// Theme describes surface-agnostic colours. Individual surfaces map these
// to their native style systems (lipgloss on terminal, CSS colours on web).
type Theme struct {
	// Background is a hex or named background colour for the chart area.
	Background string `json:"background,omitempty"`
	// Foreground is the default colour for axes, labels and text.
	Foreground string `json:"foreground,omitempty"`
	// Palette is an ordered list of colours used for series that do not
	// supply their own Color.
	Palette []string `json:"palette,omitempty"`
}

// Validate reports the first structural error in the Spec, if any.
//
// Validate is intentionally lightweight: it catches the obvious mistakes
// (missing type, zero dimensions, empty series) without prejudging what the
// target surface considers renderable.
func (s Spec) Validate() error {
	if s.Type == "" {
		return fmt.Errorf("spec: Type is required")
	}
	if !s.Type.IsKnown() {
		return fmt.Errorf("spec: unknown chart type %q", s.Type)
	}
	if s.Width <= 0 || s.Height <= 0 {
		return fmt.Errorf("spec: Width and Height must be positive (got %d x %d)", s.Width, s.Height)
	}
	if len(s.Data.Series) == 0 {
		return fmt.Errorf("spec: at least one Series is required")
	}
	for i, ser := range s.Data.Series {
		if ser.Name == "" {
			return fmt.Errorf("spec: series[%d] Name is required", i)
		}
	}
	return nil
}

// MarshalJSON implements json.Marshaler so that Spec values round-trip
// cleanly even when DataPoint.X carries a time.Time.
func (s Spec) MarshalJSON() ([]byte, error) {
	type alias Spec
	return json.Marshal(alias(s))
}
