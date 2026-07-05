# ntcharts/spec

Package `spec` defines a **neutral, surface-agnostic chart specification** for
ntcharts. A single `Spec` value describes a chart (type, data, options, theme)
and can be rendered to multiple surfaces from the same source of truth:

- **Terminal** — via `spec.Build(s)` → an ntcharts model (`*barchart.Model`,
  `*timeserieslinechart.Model`, …)
- **Web** — via `s.ToECharts()` → a go-echarts/v2 chart (`*charts.Bar`,
  `*charts.Line`, …)

Specs are JSON-marshalable so they can be persisted, transported over the
wire, or authored in configuration files.

## Status

| Chart type                 | `Build()` (terminal) | `ToECharts()` (web) |
| -------------------------- | :------------------: | :-----------------: |
| `ChartTypeBar`             | full                 | full                |
| `ChartTypeTimeSeries`      | full                 | full                |
| `ChartTypeLine`            | scaffold             | scaffold            |
| `ChartTypeStreamline`      | scaffold             | scaffold            |
| `ChartTypeSparkline`       | scaffold             | scaffold            |
| `ChartTypeHeatmap`         | scaffold             | scaffold            |
| `ChartTypeOHLC`            | scaffold             | scaffold            |
| `ChartTypeScatter`         | scaffold             | scaffold            |
| `ChartTypeCanvas`          | scaffold             | scaffold            |

Scaffolded chart types return a clear `not yet implemented` error today.

## Layout

```
spec/
├── spec.go          Spec, Data, Series, DataPoint, Options, Theme, Validate
├── helpers.go       internal X-value coercion (pointTime / pointFloat / pointString)
├── build.go         Build(s) -> ntcharts terminal model
├── echarts.go       Spec.ToECharts() -> go-echarts/v2 chart
├── example_test.go  runnable examples (bar + timeseries)
└── README.md        this file
```

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
    Data: spec.Data{
        XAxisType:   spec.XAxisCategory,
        XAxisLabels: []string{"Q1", "Q2", "Q3", "Q4"},
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

For a time-series example see `ExampleBuild_timeSeries` in
[`example_test.go`](./example_test.go).

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

# Vet + run the runnable examples (ExampleBuild_bar, ExampleBuild_timeSeries)
go vet ./spec/
go test ./spec/

# Verbose, with example output
go test -v ./spec/

# Run a single example
go test -v -run ExampleBuild_bar ./spec/
```

The examples use `// Output:` doc-test assertions and therefore verify that
`Build(s)` returns a `*barchart.Model` / `*timeserieslinechart.Model` and
that `ToECharts()` returns `*charts.Bar` / `*charts.Line`.

## Dependencies

This package adds a new module dependency to ntcharts:

- `github.com/go-echarts/go-echarts/v2` — used by `echarts.go`

`go get` has already added it to `go.mod` / `go.sum`.

## Design notes

- **`Spec.Validate()`** catches the obvious mistakes (missing `Type`, zero
  dimensions, empty `Series`). Both `Build` and `ToECharts` call it first.
- **Time values** in `DataPoint.X` accept `time.Time`, RFC 3339 strings, or
  numeric milliseconds since epoch — see `pointTime` in `spec.go`.
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
