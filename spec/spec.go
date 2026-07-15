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

// X-axis type constants used by XAxis.Type.
const (
	// XAxisCategory indicates a discrete, ordered set of labels
	// (e.g. bar chart categories).
	XAxisCategory = "category"
	// XAxisTime indicates time-valued X coordinates (time.Time or ms-since-epoch).
	XAxisTime = "time"
	// XAxisValue indicates numeric (float64) X coordinates.
	XAxisValue = "value"
)

// Orientation constants for Options.Orientation.
const (
	OrientationVertical   = "vertical"
	OrientationHorizontal = "horizontal"
)

// Format describes how numeric or time values render as labels.
//
// Kind is one of:
//   - "" or "number": plain numeric formatting
//   - "percent": v*100 followed by a "%" suffix
//   - "currency": Currency symbol prefix
//   - "si": k/M/G/T suffix (SI-style magnitude abbreviation)
//   - "time": ms-since-epoch rendered via Layout
type Format struct {
	// Kind selects the formatting family. See the Format doc comment.
	Kind string `json:"kind,omitempty"`
	// Precision is the number of decimals. nil means the kind's default.
	Precision *int `json:"precision,omitempty"`
	// Currency is the symbol used for kind "currency". Defaults to "$".
	Currency string `json:"currency,omitempty"`
	// Layout is the Go time layout string used for kind "time".
	Layout string `json:"layout,omitempty"`
}

// IsZero reports whether f is the zero value (no formatting directive set).
func (f Format) IsZero() bool {
	return f.Kind == "" && f.Precision == nil && f.Currency == "" && f.Layout == ""
}

// XAxis describes the X axis of a chart.
type XAxis struct {
	// Title is the optional axis title.
	Title string `json:"title,omitempty"`
	// Type declares the kind of X axis: XAxisCategory, XAxisTime, or
	// XAxisValue. If empty, a sensible default is inferred from the chart type.
	Type string `json:"type,omitempty"`
	// Labels is the optional list of category labels, used when
	// Type == XAxisCategory.
	Labels []string `json:"labels,omitempty"`
	// Format describes how axis labels render.
	Format Format `json:"format,omitempty"`
}

// YAxis describes the Y axis of a chart.
type YAxis struct {
	// Title is the optional axis title.
	Title string `json:"title,omitempty"`
	// Min pins the Y axis minimum. If nil, auto-scale.
	Min *float64 `json:"min,omitempty"`
	// Max pins the Y axis maximum. If nil, auto-scale.
	Max *float64 `json:"max,omitempty"`
	// Format describes how axis labels render.
	Format Format `json:"format,omitempty"`
}

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

	// XAxis describes the X axis (title, type, labels, format).
	XAxis XAxis `json:"x_axis,omitempty"`
	// YAxis describes the Y axis (title, min/max, format).
	YAxis YAxis `json:"y_axis,omitempty"`

	// Data holds the series data for the chart.
	Data Data `json:"data"`
	// Heat holds heatmap cell/matrix data. Required (non-nil, non-empty) for
	// ChartTypeHeatmap.
	Heat *HeatData `json:"heat,omitempty"`
	// Options holds common, surface-agnostic rendering options.
	Options Options `json:"options,omitempty"`
	// Theme holds surface-agnostic colour theming.
	Theme Theme `json:"theme,omitempty"`
}

// Data describes the series of a chart.
type Data struct {
	// Series is the ordered collection of data series. At least one series is
	// required for most chart types.
	Series []Series `json:"series"`
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
	// OHLC is the ordered list of open/high/low/close points in the series,
	// used by ChartTypeOHLC. Rendering support lands in a later phase.
	OHLC []OHLCPoint `json:"ohlc,omitempty"`
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
	// Size is an optional scatter point weight. Surfaces that do not support
	// variable point sizing may ignore it.
	Size *float64 `json:"size,omitempty"`
}

// OHLCPoint is a single open/high/low/close datum, used by ChartTypeOHLC.
type OHLCPoint struct {
	// T is the point's time: time.Time, RFC3339 string, or ms-since-epoch.
	T any `json:"t"`
	// O is the opening value.
	O float64 `json:"o"`
	// H is the high value.
	H float64 `json:"h"`
	// L is the low value.
	L float64 `json:"l"`
	// C is the closing value.
	C float64 `json:"c"`
}

// HeatCell is a single (x, y, z) datum in a HeatData.Cells list.
type HeatCell struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// HeatData holds heatmap cell/matrix data, used by ChartTypeHeatmap.
type HeatData struct {
	// Cells is a sparse list of (x, y, z) cells.
	Cells []HeatCell `json:"cells,omitempty"`
	// Matrix is a row-major dense alternative to Cells.
	Matrix [][]float64 `json:"matrix,omitempty"`
	// MinValue pins the colour-scale domain minimum. If nil, auto.
	MinValue *float64 `json:"min_value,omitempty"`
	// MaxValue pins the colour-scale domain maximum. If nil, auto.
	MaxValue *float64 `json:"max_value,omitempty"`
}

// Options are common, surface-agnostic rendering options.
//
// Surfaces are free to ignore options they do not support.
type Options struct {
	// ShowLegend toggles the legend.
	ShowLegend bool `json:"show_legend,omitempty"`
	// ShowGrid toggles the background grid.
	ShowGrid bool `json:"show_grid,omitempty"`
	// Orientation controls bar chart orientation: "" or OrientationVertical
	// for vertical bars, OrientationHorizontal for horizontal bars.
	Orientation string `json:"orientation,omitempty"`
	// Stacked stacks bar chart series instead of grouping them.
	Stacked bool `json:"stacked,omitempty"`
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
	// Gradient is an ordered list of hex stops ("#rrggbb") used for
	// sequential colormaps (heatmap).
	Gradient []string `json:"gradient,omitempty"`
}

// validFormatKinds enumerates the recognized Format.Kind values.
var validFormatKinds = map[string]bool{
	"": true, "number": true, "percent": true, "currency": true, "si": true, "time": true,
}

// validateFormat reports an error if f.Kind is not one of the recognized
// Format kinds. name identifies the axis in the error message (e.g. "x_axis").
func validateFormat(name string, f Format) error {
	if !validFormatKinds[f.Kind] {
		return fmt.Errorf("spec: %s format kind %q is not one of number|percent|currency|si|time", name, f.Kind)
	}
	return nil
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

	switch s.Type {
	case ChartTypeHeatmap:
		if s.Heat == nil || (len(s.Heat.Cells) == 0 && len(s.Heat.Matrix) == 0) {
			return fmt.Errorf("spec: heatmap requires Heat data (cells or matrix)")
		}
	default:
		if len(s.Data.Series) == 0 {
			return fmt.Errorf("spec: at least one Series is required")
		}
	}

	for i, ser := range s.Data.Series {
		if ser.Name == "" {
			return fmt.Errorf("spec: series[%d] Name is required", i)
		}
	}

	if s.Type == ChartTypeOHLC {
		hasOHLC := false
		for _, ser := range s.Data.Series {
			if len(ser.OHLC) > 0 {
				hasOHLC = true
				break
			}
		}
		if !hasOHLC {
			return fmt.Errorf("spec: ohlc requires Series.OHLC points")
		}
	}

	switch s.Options.Orientation {
	case "", OrientationVertical, OrientationHorizontal:
	default:
		return fmt.Errorf("spec: orientation %q must be vertical or horizontal", s.Options.Orientation)
	}

	if err := validateFormat("x_axis", s.XAxis.Format); err != nil {
		return err
	}
	if err := validateFormat("y_axis", s.YAxis.Format); err != nil {
		return err
	}

	return nil
}

// MarshalJSON implements json.Marshaler so that Spec values round-trip
// cleanly even when DataPoint.X carries a time.Time.
func (s Spec) MarshalJSON() ([]byte, error) {
	type alias Spec
	return json.Marshal(alias(s))
}
