# ntcharts/spec

Package `spec` defines a **neutral, surface-agnostic chart specification** for
ntcharts. A single `Spec` value describes a chart (type, axes, data, options,
theme) and can be rendered to multiple surfaces from the same source of
truth:

- **Terminal** — via `spec.Build(s)` → an ntcharts model (`*barchart.Model`,
  `*wavelinechart.Model`, `*linechart.Model`, `*timeserieslinechart.Model`,
  `*heatmap.Model`, …)
- **Web** — via `s.ToECharts()` → a go-echarts/v2 chart (`*charts.Bar`,
  `*charts.Line`, …)

Specs are JSON-marshalable so they can be persisted, transported over the
wire, or authored in configuration files.

## Status

| Chart type            | `Build()` (terminal)                                              | `ToECharts()` (web) |
| ---------------------- | ------------------------------------------------------------------ | :------------------: |
| `ChartTypeBar`         | full — stacked + horizontal; grouped (side-by-side) bars are **not** supported by the terminal surface | full |
| `ChartTypeTimeSeries`  | full                                                                | full |
| `ChartTypeLine`        | full                                                                | scaffold |
| `ChartTypeScatter`     | full                                                                | scaffold |
| `ChartTypeHeatmap`     | full                                                                | scaffold |
| `ChartTypeStreamline`  | scaffold                                                            | scaffold |
| `ChartTypeSparkline`   | scaffold                                                            | scaffold |
| `ChartTypeOHLC`        | scaffold                                                            | scaffold |
| `ChartTypeCanvas`      | scaffold                                                            | scaffold |

Scaffolded chart types return a clear `not yet implemented` error today.

`ChartTypeBar` is stacked-only in the terminal because `barchart.Model` has no
grouped/side-by-side rendering mode: a `Spec` with more than one `Series` must
set `Options.Stacked = true`, or `Build` returns an error instead of silently
stacking series a caller may have expected to be grouped side by side.
`Options.Orientation = OrientationHorizontal` renders horizontal bars via
`barchart.WithHorizontalBars()` in the terminal. **`ToECharts()` does not yet
honour `Options.Stacked` or `Options.Orientation` for bar charts** — every
web bar chart renders as ungrouped, vertical bars regardless of these
options; web stacking/orientation support is planned but not implemented.

### Surface fidelity matrix

Not every `Spec` option is honoured identically (or at all) by both surfaces.
This table reflects the current, real behaviour of `Build()` (terminal) and
`ToECharts()` (web) — consult it before assuming an option "just works" on
both:

| Spec option | Terminal (`Build`) | Web (`ToECharts`) |
| --- | --- | --- |
| `Options.Stacked` (bar) | honoured — required (else error) when a bar `Spec` has more than one `Series` | **ignored** — web stacking planned, not implemented |
| `Options.Orientation` (bar) | honoured via `barchart.WithHorizontalBars()` | **ignored** — always renders vertical bars |
| `YAxis.Min` / `YAxis.Max`, one-sided (line, scatter, timeseries) | honoured — see "One-sided Y-axis pins" below | honoured — ECharts auto-scales the unset bound natively |
| `YAxis.Min` (bar) | **ignored** — `barchart.Model` has no Y-minimum option, only `WithMaxValue` | honoured (set directly on the ECharts Y axis) |
| `XAxis.Format` / `YAxis.Format` (line, scatter) | honoured via `XLabelFormatter` / `YLabelFormatter` | **ignored** — not wired into `ToECharts()` |
| `XAxis.Format` (timeseries) | honoured only when `Kind == "time"`; other kinds are intentionally ignored (the X axis is time-valued) | `Layout` is translated and applied unconditionally as the ECharts time-axis label template (`Kind` is not checked) |
| `YAxis.Format` (timeseries) | honoured via `YLabelFormatter` | **ignored** — not wired into `ToECharts()` |
| `Theme.Gradient` (heatmap) | honoured — interpolated colour scale | **ignored** — heatmap is `ToECharts()`-scaffold only |
| `DataPoint.Size` | **ignored** everywhere — accepted by the schema, drawn as fixed-size markers on every surface | **ignored** everywhere |
| `XAxis.Title` / `YAxis.Title` | **ignored** — no current ntcharts terminal model surfaces an axis title | **ignored** — not wired into `ToECharts()` |

#### One-sided Y-axis pins

`YAxis.Min` and `YAxis.Max` can be set independently. The rule, enforced the
same way by `buildLine`, `buildTimeSeries`, and `buildScatter`: **a lone
`Min` or `Max` pins that bound; the other, unset bound is derived from the
series' actual Y data.** Setting both pins both bounds explicitly, and
setting neither leaves the chart fully auto-scaled. If the resulting
(effective) minimum exceeds the (effective) maximum — e.g. pinning `Min`
above the data's actual maximum — `Build` returns an error
(`spec: y_axis min %v exceeds max %v`) instead of silently constructing an
inverted range. `ToECharts()` achieves the equivalent one-sided behaviour
natively: an unset `opts.YAxis.Min`/`Max` (an `interface{}`, `omitempty`) is
left out of the JSON entirely, so ECharts auto-scales that side itself;
`ToECharts()` does not currently validate an inverted pin.

`ChartTypeBar` has no such rule: `barchart.Model` has no Y-minimum concept at
all (see the fidelity matrix above), so `YAxis.Min` is always ignored for bar
charts on the terminal surface.

## Layout

```
spec/
├── spec.go              Spec, XAxis, YAxis, Format, Data, Series, DataPoint,
│                         HeatData, OHLCPoint, Options, Theme, Validate
├── helpers.go            internal X-value coercion (pointTime / pointFloat / pointString)
├── format.go              FormatValue + Format.labelFormatter (number/percent/currency/si/time)
├── gradient.go            Theme.Gradient hex-stop interpolation for heatmap colour scales
├── build.go               Build(s) -> ntcharts terminal model
├── echarts.go             Spec.ToECharts() -> go-echarts/v2 chart
├── example_test.go        runnable examples (bar, line, scatter, timeseries, heatmap)
└── README.md              this file
```

## Schema v1 overview

- **`XAxis` / `YAxis`** describe each axis: `Title`, `Type` (`XAxisCategory`,
  `XAxisTime`, or `XAxisValue`; inferred from chart type when empty),
  `Labels` (category ticks), and `Format` (label formatting). `YAxis` adds
  `Min`/`Max` to pin the range: a lone `Min` or `Max` pins that bound while
  the other is data-derived, both pin an explicit range, and neither leaves
  the chart fully auto-scaled — see "One-sided Y-axis pins" above for the
  full rule and its edge cases.
- **`Format`** describes how axis labels (and `FormatValue`) render a
  `float64`. `Kind` selects the family:
  - `""` / `"number"` — plain numeric formatting, `Precision` decimals.
  - `"percent"` — multiplies the value by 100 and appends `"%"`.
  - `"currency"` — prefixes `Currency` (default `"$"`).
  - `"si"` — abbreviates magnitude with a k/M/G/T suffix.
  - `"time"` — treats the value as **milliseconds since the Unix epoch** and
    renders it with the Go time layout in `Layout` (default `"2006-01-02"`).
    Note: chart-internal axis values for timeseries charts are
    **seconds**-since-epoch, not milliseconds; the timeseries axis
    formatters convert between the two internally, so `FormatValue` callers
    always pass milliseconds.
  - `line` and `scatter` charts with a `Kind: "time"` X-axis `Format` expect
    `DataPoint.X` (or the resolved numeric X) to be **milliseconds since the
    Unix epoch** — the same convention `FormatValue` uses for `"time"`.
- **`Data.Series`** is the ordered list of named series; `Data.XAxisData` is
  an optional shared X value list so per-point `DataPoint.X` can be omitted.
- **`DataPoint.Size`** is accepted in the schema (a scatter point weight) but
  ignored by every current surface — terminal and ECharts both draw
  fixed-size markers today.
- **`Series.OHLC`** ([]OHLCPoint) is reserved for `ChartTypeOHLC`; the schema
  and `Validate()` already require it for that chart type, but rendering
  support (`Build`/`ToECharts`) lands in a later phase.
- **`Heat`** (`*HeatData`) holds heatmap data as either a sparse `Cells`
  list (`{X, Y, Z}` triples) or a dense row-major `Matrix`; `MinValue`/
  `MaxValue` pin the colour-scale domain (auto-ranged when nil).
- **`Theme.Gradient`** is an ordered list of `"#rrggbb"` hex stops
  interpolated into a colour scale for heatmap rendering; empty falls back to
  the package's default grayscale scale.

## Quick start

```go
import (
    "github.com/NimbleMarkets/ntcharts/v2/barchart"
    "github.com/NimbleMarkets/ntcharts/v2/spec"
)

s := spec.Spec{
    Type:   spec.ChartTypeBar,
    Title:  "Quarterly Revenue",
    Width:  60,
    Height: 20,
    XAxis: spec.XAxis{
        Type:   spec.XAxisCategory,
        Labels: []string{"Q1", "Q2", "Q3", "Q4"},
    },
    Data: spec.Data{
        Series: []spec.Series{{
            Name:   "Revenue",
            Color:  "#22aadd",
            Values: []spec.DataPoint{{Y: 120}, {Y: 180}, {Y: 95}, {Y: 210}},
        }},
    },
    Options: spec.Options{ShowLegend: true, ShowGrid: true},
}

// Terminal
term, err := spec.Build(s)
if err != nil { /* ... */ }
bc := term.(*barchart.Model)
_ = bc.View()

// Web
web, err := s.ToECharts()
if err != nil { /* ... */ }
// web is *charts.Bar — call web.Render(w) to produce HTML
```

Multi-series bar charts must set `Options.Stacked = true` (see Status above);
without it, `Build` returns an error rather than silently stacking series.

For line, scatter, and heatmap examples see `ExampleBuild_line`,
`ExampleBuild_scatter`, and `ExampleBuild_heatmap` in
[`example_test.go`](./example_test.go). For a time-series example see
`ExampleBuild_timeSeries`.

Time-series specs can also mix line and bar series by setting
`spec.Series.Type` per series. In terminal rendering, bar series are drawn as
line-chart data sets with a distinct line style because the terminal
time-series model is line-based. In ECharts rendering, `Type: "bar"` series
are overlaid as real bars on a secondary Y axis, while other series remain
lines. See `examples/combo-timeseries-bar` for a complete example.

## Running the tests

From the repository root:

```sh
# Build just this package
go build ./spec/

# Vet + run the runnable examples (ExampleBuild_bar, ExampleBuild_line,
# ExampleBuild_scatter, ExampleBuild_timeSeries, ExampleBuild_heatmap)
go vet ./spec/
go test ./spec/

# Verbose, with example output
go test -v ./spec/

# Run a single example
go test -v -run ExampleBuild_bar ./spec/
```

The examples use `// Output:` doc-test assertions and therefore verify both
the concrete `Build(s)` model type and, where applicable, the `ToECharts()`
chart type. `ExampleBuild_heatmap` cannot assert `View()` output directly
(heatmap cells are always ANSI-styled), so it asserts the concrete type and a
non-empty rendered view instead.

## Dependencies

This package adds a new module dependency to ntcharts:

- `github.com/go-echarts/go-echarts/v2` — used by `echarts.go`

`go get` has already added it to `go.mod` / `go.sum`.

`go.mod`/`go.sum` (both the module root and `examples/`) are deliberately
left **uncommitted** pending a module-wide dependency decision (e.g. whether
`go-echarts/v2` becomes a required or optional/build-tagged dependency of the
root module). Do not commit those four files as part of `spec/` work; they
carry unrelated, in-progress changes.

## Design notes

- **`Spec.Validate()`** catches the obvious mistakes (missing `Type`, zero
  dimensions, empty `Series`, invalid `Options.Orientation` or `Format.Kind`,
  missing `Heat`/`OHLC` data for the chart types that require it). Both
  `Build` and `ToECharts` call it first.
- **Time values** in `DataPoint.X` accept `time.Time`, RFC 3339 strings, or
  numeric milliseconds since epoch — see `pointTime` in `helpers.go`.
- **Colours** prefer `Series.Color`, fall back to `Theme.Palette[i]`, then the
  surface default.
- **Terminal sizes are tiny for a browser**: `ToECharts()` scales sub-200
  `Width`/`Height` by 8×/16× when emitting an `Initialization` hint, so a
  60×20 terminal spec renders as roughly 480×320 px on the web.

## Extending

To add a new chart type:

1. Add the constant to `ChartType` in `spec.go` and `ChartType.IsKnown()`.
2. Add a `build<Kind>` branch in `Build()` (`build.go`).
3. Add a `toECharts<Kind>` branch in `ToECharts()` (`echarts.go`).
4. Add a runnable `Example` in `example_test.go`.
5. Update the status table above.
