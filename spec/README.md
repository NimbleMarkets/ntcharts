# ntcharts/spec

Package `spec` defines a **neutral, surface-agnostic chart specification** for
ntcharts. A single `Spec` value describes a chart (type, axes, data, options,
theme) and can be rendered to multiple surfaces from the same source of
truth:

- **Terminal** — via `spec.Build(s)` → an ntcharts model (`*barchart.Model`,
  `*wavelinechart.Model`, `*linechart.Model`, `*timeserieslinechart.Model`,
  `*heatmap.Model`, …)
- **Web** — via `echarts.ToECharts(s)` → a go-echarts/v2 chart (`*charts.Bar`,
  `*charts.Line`, …). This surface lives in its own module,
  `github.com/NimbleMarkets/ntcharts/spec/echarts/v2` (directory
  [`spec/echarts`](./echarts)), so the go-echarts dependency is only pulled
  in by programs that render to the web; terminal consumers of
  `github.com/NimbleMarkets/ntcharts/v2/spec` never see it. It is released
  alongside the root module with the `spec/echarts/v2.X.Y` tag.

Specs are JSON-marshalable so they can be persisted, transported over the
wire, or authored in configuration files. Zero-valued option structs are
omitted; pointer fields preserve explicit zero pins and precision. A series'
`values` field is always emitted (`null` for a nil slice, `[]` for an empty
non-nil slice), including OHLC series whose points live in `ohlc`.

## Status

| Chart type            | `Build()` (terminal)                                              | `echarts.ToECharts()` (web) |
| ---------------------- | ------------------------------------------------------------------ | :------------------: |
| `ChartTypeBar`         | full — stacked + horizontal; grouped (side-by-side) bars are **not** supported by the terminal surface | full |
| `ChartTypeTimeSeries`  | full                                                                | full |
| `ChartTypeLine`        | full                                                                | scaffold |
| `ChartTypeScatter`     | full                                                                | scaffold |
| `ChartTypeHeatmap`     | full                                                                | scaffold |
| `ChartTypeStreamline`  | scaffold                                                            | scaffold |
| `ChartTypeSparkline`   | full                                                                | scaffold |
| `ChartTypeOHLC`        | full — candlesticks on `*timeserieslinechart.Model`                 | scaffold |
| `ChartTypeCanvas`      | scaffold                                                            | scaffold |

Scaffolded chart types return a clear `not yet implemented` error today.

`ChartTypeBar` is stacked-only in the terminal because `barchart.Model` has no
grouped/side-by-side rendering mode: a `Spec` with more than one `Series` must
set `Options.Stacked = true`, or `Build` returns an error instead of silently
stacking series a caller may have expected to be grouped side by side.
`Options.Orientation = OrientationHorizontal` renders horizontal bars via
`barchart.WithHorizontalBars()` in the terminal. **`ToECharts()` does not yet
honour `Options.Stacked` or `Options.Orientation` for bar charts** — every
web bar chart renders as grouped, vertical bars regardless of these
options; web stacking/orientation support is planned but not implemented.

`ChartTypeOHLC` is built on `timeserieslinechart.Model` the same way
`ChartTypeTimeSeries` is: `buildOHLC` pushes `open`/`high`/`low`/`close`
data sets keyed by each `OHLCPoint`'s parsed time and draws candles with
`(*timeserieslinechart.Model).DrawCandleWithOpts`, so candles render at
**time-scaled X positions across the full graph width** — the same axis
machinery (including labeled axes via `XAxis.Format` / `YAxis.Format`) as
the timeseries surface. Candles are coloured by `Theme.Palette[0]` (up,
close >= open) / `Theme.Palette[1]` (down), defaulting to `#26a69a` /
`#ef5350` when the palette does not set those slots. `YAxis.Min` / `YAxis.Max`
follow the same one-sided pin rule as the line/scatter/timeseries surfaces
(see "One-sided Y-axis pins" below). Candle body width is chosen
automatically from chart density (`autoCandleWidth`): roughly an equal share
of the graph width per candle, clamped so time-scaled neighbours on
irregular calendars never overlap, forced odd, and capped at 7 columns so
sparse charts still read as individual candles. `Options.CandleStyle` selects
the rendering style: `""` or `CandleStyleLine` renders box-drawing line runes
(the default), `CandleStyleBlock` renders solid block bodies; unknown values
are a `Build` error. Edge candles (those whose true time position would run
their body into the axis or past the chart edge) are shifted inward (inset)
up to half a body width so their whole body renders; nothing is clipped.
There is no dropping of points for density —
a `Spec` with more OHLC points than comfortably fit overdraws densely, exactly
like any other dense timeseries, rather than discarding older candles.
OHLC requires exactly one series; additional series return a `Build` error.
`Series.Color` is ignored for OHLC because colours encode up/down direction.

Line, scatter, timeseries, and OHLC builders return an error when the requested
dimensions leave no drawable columns or rows after allocating axes and labels.
Singleton time ranges (including timestamps within the same millisecond) are
widened by twelve hours on each side so their points remain visible.

### Surface fidelity matrix

Not every `Spec` option is honoured identically (or at all) by both surfaces.
This table reflects the current, real behaviour of `Build()` (terminal) and
`ToECharts()` (web) — consult it before assuming an option "just works" on
both:

| Spec option | Terminal (`Build`) | Web (`ToECharts`) |
| --- | --- | --- |
| `Options.Stacked` (bar) | honoured — required (else error) when a bar `Spec` has more than one `Series` | **ignored** — web stacking planned, not implemented |
| `Options.Orientation` (bar) | honoured via `barchart.WithHorizontalBars()` | **ignored** — always renders vertical bars |
| `XAxis.Scale` / `YAxis.Scale` = `"log"` | honoured — Y for line, scatter, timeseries, ohlc; X for line, scatter. A `Build` error for X on timeseries/ohlc (time axis), for either axis on bar, heatmap, sparkline, and for any value or pin `<= 0` — see "Logarithmic axes" below | **ignored** — `ToECharts()` does not read `scale` yet and draws a linear axis |
| `YAxis.Min` / `YAxis.Max`, one-sided (line, scatter, timeseries, ohlc) | honoured — see "One-sided Y-axis pins" below | honoured — ECharts auto-scales the unset bound natively |
| `Theme.Palette[0]` / `[1]` (ohlc) | honoured — slot 0 = up candles, slot 1 = down candles; defaults `#26a69a` / `#ef5350` when unset | N/A — `ToECharts()` is scaffold-only |
| `Options.CandleStyle` (ohlc) | honoured — `""` or `CandleStyleLine` renders box-drawing line runes, `CandleStyleBlock` renders solid block bodies; unknown values error | N/A — `ToECharts()` is scaffold-only |
| `Series.OHLC` count vs. chart width (ohlc) | candles render at time-scaled X positions across the full graph width with an automatically chosen, density-based body width (odd, capped at 7, never overlapping); dense series overdraw rather than dropping points | N/A — `ToECharts()` is scaffold-only |
| `YAxis.Min` / `YAxis.Max` (bar) | forwarded to `barchart.WithMinValue()` / `WithMaxValue()`; positive minima clamp to zero; a lone pin leaves the other bound auto-scaled by the model; explicit `Min > Max` is a `Build` error | honoured (set directly on the ECharts Y axis) |
| `YAxis.Min` (sparkline) | **ignored** — `sparkline.Model` has no Y-minimum concept, only `WithMaxValue` | N/A — `ToECharts()` is scaffold-only |
| `YAxis.Max` (sparkline) | honoured via `sparkline.WithMaxValue()` when set; otherwise auto-scales | N/A — `ToECharts()` is scaffold-only |
| `Data.Series` (sparkline) | **requires exactly one** — multiple series error (like grouped bars) | N/A — `ToECharts()` is scaffold-only |
| `XAxis.Format` / `YAxis.Format` (line, scatter) | honoured via `XLabelFormatter` / `YLabelFormatter` | **ignored** — not wired into `ToECharts()` |
| `XAxis.Format` (timeseries) | honoured only when `Kind == "time"`; other kinds are intentionally ignored (the X axis is time-valued) | `Layout` is translated and applied unconditionally as the ECharts time-axis label template (`Kind` is not checked) |
| `YAxis.Format` (timeseries) | honoured via `YLabelFormatter` | **ignored** — not wired into `ToECharts()` |
| `Theme.Gradient` (heatmap) | honoured — interpolated colour scale | **ignored** — heatmap is `ToECharts()`-scaffold only |
| `Heat.MinValue` / `Heat.MaxValue` | honoured independently; an unset bound is derived from the selected cells/matrix; inverted effective ranges error | N/A — heatmap is scaffold-only |
| `Heat.Matrix` / `Heat.Cells` | matrix rows are Y indices and columns are X indices; a nonempty matrix takes precedence over cells | N/A — heatmap is scaffold-only |
| `Data.XAxisData` (bar, line, scatter, timeseries) | resolves omitted point X values; explicit point X takes precedence | honoured for bar and timeseries; line/scatter are scaffold-only |
| `Data.Series` / `Series.Color` (ohlc) | exactly one series required; `Series.Color` ignored in favour of up/down palette slots | N/A — OHLC is scaffold-only |
| `Theme.Palette` (bar, line, scatter, timeseries, sparkline) | fallback for series without `Series.Color` | honoured for bar and timeseries; other types are scaffold-only |
| `XAxis.Format` / `YAxis.Format` (bar, heatmap, sparkline) | **ignored** | **ignored** for bar; other types are scaffold-only |
| `XAxis.Labels` | honoured for bar (categories) and heatmap (column names); **ignored** by other terminal builders | honoured for bar; **ignored** for timeseries |
| `YAxis.Min` / `YAxis.Max` (heatmap) | **ignored** — use `Heat.MinValue` / `Heat.MaxValue` for the colour domain | N/A — heatmap is scaffold-only |
| `DataPoint.Size` | **ignored** everywhere — accepted by the schema, drawn as fixed-size markers on every surface | **ignored** everywhere |
| `XAxis.Title` / `YAxis.Title` | **ignored** — no current ntcharts terminal model surfaces an axis title | **ignored** — not wired into `ToECharts()` |
| `YAxis.Labels` (heatmap rows) | honoured — drawn left of the plot, first label on the top row; other chart types ignore it | **ignored** — not wired into `ToECharts()`; reserved for future grid-chart row labels |
| `Title` / `Subtitle` | **ignored** — the terminal models draw no title; the host TUI is expected to place one (e.g. flint-tui's status line) | honoured — ECharts title/subtitle |
| `Options.ShowLegend` / `Options.ShowGrid` | **ignored** — no terminal model draws a legend or background grid | honoured |
| `Theme.Background` / `Theme.Foreground` | **ignored** — the terminal inherits the host's colours; only `Theme.Palette` (series colours) and `Theme.Gradient` (heatmap) are read | `Background` honoured as the canvas colour; `Foreground` ignored |
| `XAxis.Type` | validated (`""`, `category`, `time`, `value`) but not consulted — each terminal builder infers the axis kind from the chart type (`timeseries`/`ohlc` are time-valued, `bar`/`heatmap` categorical, `line`/`scatter` numeric) | honoured for bar (category vs value); timeseries always `time` |
| `Series.Type` | validated (empty or a known chart type), otherwise **ignored**; timeseries series all use braille lines | `"bar"` inside a `timeseries` spec overlays real bars on a secondary Y axis |

Fields belonging to another chart family are ignored: `Heat` outside heatmaps,
`Series.OHLC` outside OHLC, `Series.Values` for OHLC, and `Data.Series` for
heatmaps. `Options.Orientation` / `Stacked` apply only to bars,
`Options.CandleStyle` only to OHLC, and `Theme.Gradient` only to heatmaps.
Sparklines ignore X coordinates and shared X data; OHLC uses each point's `T`.

#### One-sided Y-axis pins

`YAxis.Min` and `YAxis.Max` can be set independently. The rule, enforced the
same way by `buildLine`, `buildTimeSeries`, `buildScatter`, and `buildOHLC`:
**a lone `Min` or `Max` pins that bound; the other, unset bound is derived
from the series' actual Y data** (for `buildOHLC`, the observed high/low
across all `OHLCPoint`s). Setting both pins both bounds explicitly, and
setting neither leaves the chart fully auto-scaled. If the resulting
(effective) minimum exceeds the (effective) maximum — e.g. pinning `Min`
above the data's actual maximum — `Build` returns an error
(`spec: y_axis min %v exceeds max %v`) instead of silently constructing an
inverted range. `ToECharts()` achieves the equivalent one-sided behaviour
natively: an unset `opts.YAxis.Min`/`Max` (an `interface{}`, `omitempty`) is
left out of the JSON entirely, so ECharts auto-scales that side itself;
`ToECharts()` does not currently validate an inverted pin.

For these four terminal chart types, equal explicit `Min` and `Max` return
an error because they leave no drawable Y range. If a lone pin equals the
data-derived opposite bound, only the unpinned side expands by one unit.
Unpinned flat scatter/OHLC ranges expand by one unit on each side.

On a log Y axis (`YAxis.Scale = "log"`) the same rule holds, evaluated on
the real values: `Min` / `Max` are given in data units, a lone pin fixes that
bound while the other comes from the data, equal pins and an inverted range
are the same errors, and a pin `<= 0` is an error of its own. Only the
unpinned bound differs — it widens to a whole decade instead of sitting on
the data's extreme. See "Logarithmic axes" below.

`ChartTypeBar` pins each bound directly on the model (`barchart.WithMinValue`
/ `WithMaxValue`), so a lone pin leaves the other bound to the model's own
auto-scaling, which already tracks negative values below zero. Setting both
with `Min > Max` is the same `Build` error as above. A positive bar `Min`
clamps to zero because the native bar model always retains a zero baseline.

#### Logarithmic axes

`XAxis.Scale` and `YAxis.Scale` are `""` / `"linear"` (`ScaleLinear`, the
default) or `"log"` (`ScaleLog`, base 10). `Validate` rejects anything else.
Linear specs build exactly as they did before the field existed.

| Chart type | `y_axis.scale: "log"` | `x_axis.scale: "log"` |
| --- | --- | --- |
| line, scatter | honoured | honoured |
| timeseries, ohlc | honoured | `Build` error — the X axis is time |
| bar, heatmap, sparkline | `Build` error | `Build` error |

Bar, heatmap, and sparkline charts have no way to draw a log axis, so `Build`
refuses rather than drawing a linear chart under a spec that asked for log.

- **Values must be positive.** Any value `<= 0` (or NaN / infinite) on a log
  axis is a `Build` error naming it, e.g. `spec: y_axis scale "log" requires
  positive finite values; series "a" point 3 has y = 0`. That covers every
  point's Y, each OHLC open/high/low/close, a pinned `YAxis.Min` / `Max`, and
  on a log X axis every resolved X — including the index fallback, whose
  first point is X = 0. Nothing is clamped, skipped, or substituted.
- **Range.** An unpinned bound widens to a whole decade
  (`floor(log10(min))` … `ceil(log10(max))`), so the axis ends on powers of
  ten; a pinned bound is used exactly. When that leaves no range (all values
  on one power of ten), each unpinned bound moves out one more decade. The X
  axis has no pins and always uses whole decades.
- **Labels.** `Format` applies to the value as on a linear axis, so
  currency, SI, percent, and number formats work unchanged. With no
  `Format`, labels are three significant digits with trailing zeros dropped
  and a k/M/G/T suffix from 1000 up (`1`, `2.5`, `0.001`, `1k`, `3.16M`),
  falling back to exponent notation below 0.0001 and from 1e15; the charts'
  own default, a whole number, would label every tick below 1 as `0`.
- **Ticks** sit on powers of ten, on the row or column where each one
  falls, whatever the chart's size. Labels keep the models' usual minimum
  spacing (two rows; two columns for line and scatter), so when decades are
  closer than that every second (third, ...) one is labelled. When a range
  holds fewer than three powers of ten, 2× and 5× are added where they fit,
  and a range too narrow for two round values falls back to evenly spaced
  labels. A pinned bound that is not itself a round value is not labelled.
- **How it is drawn.** `Build` sets the chart model's own scale
  (`linechart.ScaleLog`), so the returned model's ranges (`ViewMinY()`,
  `MaxX()`, …) are in data units and its zoom and pan work in decades.
  `ChartTypeLine` draws columns without a point along the bottom of the
  range, since a log axis has no zero for the wave to rest on.

See `ExampleBuild_logScale` in [`example_test.go`](./example_test.go).

## Layout

```
spec/
├── spec.go              Spec, XAxis, YAxis, Format, Data, Series, DataPoint,
│                         HeatData, OHLCPoint, Options, Theme, Validate
├── helpers.go            X-value coercion (exported PointX / PointTime; pointFloat / pointString)
├── format.go              FormatValue + Format.labelFormatter (number/percent/currency/si/time)
├── scale.go               log axis scale: supported chart types, value errors, decade ranges, default labels
├── gradient.go            Theme.Gradient hex-stop interpolation for heatmap colour scales
├── build.go               Build(s) -> ntcharts terminal model (includes buildOHLC, on timeserieslinechart.Model;
│                         exported DeriveBarLabels shared with the web surface)
├── example_test.go        runnable examples (bar, line, log scale, scatter, timeseries, heatmap, ohlc)
├── README.md              this file
└── echarts/               separate module github.com/NimbleMarkets/ntcharts/spec/echarts/v2
    ├── go.mod
    ├── echarts.go         echarts.ToECharts(s) -> go-echarts/v2 chart
    └── example_test.go    ExampleToECharts, ExampleToECharts_timeSeries
```

## Schema v1 overview

- **`XAxis` / `YAxis`** describe each axis: `Title`, `Labels`, `Format`
  (label formatting), and `Scale` (`""` / `"linear"`, or `"log"` for a
  base-10 logarithmic axis — see "Logarithmic axes" above for which chart
  types honour it). `XAxis.Type` is `XAxisCategory`, `XAxisTime`, or
  `XAxisValue` (inferred from chart type when empty). `YAxis` adds
  `Min`/`Max` to pin the range: a lone `Min` or `Max` pins that bound while
  the other is data-derived, both pin an explicit range, and neither leaves
  the chart fully auto-scaled — see "One-sided Y-axis pins" above for the
  full rule and its edge cases. `XAxis.Labels` and `YAxis.Labels` name the
  columns and rows of a heatmap; see the heatmap notes below.
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
- **`Series.OHLC`** ([]OHLCPoint) is required by `Validate()` for
  `ChartTypeOHLC`. `Build()` renders it as time-scaled candlesticks on a
  `*timeserieslinechart.Model` (see `buildOHLC` in `build.go` and the
  `ChartTypeOHLC` notes above); `ToECharts()` support is not yet implemented
  (scaffold error).
- **`Heat`** (`*HeatData`) holds heatmap data as either a sparse `Cells`
  list (`{X, Y, Z}` triples) or a dense row-major `Matrix`; `MinValue`/
  `MaxValue` independently pin the colour-scale domain (data-derived when nil).
  A nonempty matrix takes precedence over cells; `Matrix[y][x]` corresponds
  to a cell at `(x, y)`. Cell indices are whole, non-negative numbers
  (`Build` rejects a negative one); each cell is drawn as a filled block, and
  the blocks tile the plot. Row `Y=0` is the **first row, drawn at the top**,
  and `y_axis.labels[0]` names it, as in flint's own heatmaps; `x_axis.labels[i]`
  names column `X=i`, counting from the left. Row labels sit right-aligned in a
  margin left of the plot, centred on their rows; column labels sit under the
  plot, each at the left edge of its column. A label that would land on a row
  already labelled, or run into the previous column label, is left out. The
  plot grows to cover the labels, so a label with no cells is an empty row or
  column. With no labels, no margin or label row is reserved and the cells
  fill the whole chart.
- **`Theme.Gradient`** is an ordered list of `"#rrggbb"` hex stops
  interpolated into a colour scale for heatmap rendering; empty falls back to
  the package's default grayscale scale.

## Quick start

```go
import (
    "github.com/NimbleMarkets/ntcharts/v2/barchart"
    "github.com/NimbleMarkets/ntcharts/v2/spec"

    // Web surface only — a separate module, so terminal programs can skip it:
    //   go get github.com/NimbleMarkets/ntcharts/spec/echarts/v2
    "github.com/NimbleMarkets/ntcharts/spec/echarts/v2"
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
web, err := echarts.ToECharts(s)
if err != nil { /* ... */ }
// web is *charts.Bar — call web.Render(w) to produce HTML
```

Multi-series bar charts must set `Options.Stacked = true` (see Status above);
without it, `Build` returns an error rather than silently stacking series.

For line, scatter, and heatmap examples see `ExampleBuild_line`,
`ExampleBuild_scatter`, and `ExampleBuild_heatmap` in
[`example_test.go`](./example_test.go). For a time-series example see
`ExampleBuild_timeSeries`. For candlesticks see `ExampleBuild_ohlc`. For a
logarithmic axis see `ExampleBuild_logScale`.

Time-series specs can also mix line and bar series by setting
`spec.Series.Type` per series. In terminal rendering, bar series are drawn as
braille lines just like the other series. In ECharts rendering, `Type: "bar"` series
are overlaid as real bars on a secondary Y axis, while other series remain
lines. See `examples/combo-timeseries-bar` for a complete example.

## Running the tests

From the repository root:

```sh
# Build just this package
go build ./spec/

# Vet + run the runnable examples (ExampleBuild_bar, ExampleBuild_line,
# ExampleBuild_logScale, ExampleBuild_scatter, ExampleBuild_timeSeries,
# ExampleBuild_heatmap, ExampleBuild_ohlc)
go vet ./spec/
go test ./spec/

# The web surface is its own module
(cd spec/echarts && go vet ./... && go test ./...)

# Both together
task test-spec

# Verbose, with example output
go test -v ./spec/

# Run a single example
go test -v -run ExampleBuild_bar ./spec/
```

The examples use `// Output:` doc-test assertions and therefore verify the
concrete `Build(s)` model type (and, in `spec/echarts`, the `ToECharts()`
chart type). `ExampleBuild_heatmap` and `ExampleBuild_ohlc` show the concrete
type and a non-empty view as compact usage examples. Rendering tests inspect
heatmap cell colours and candle glyphs directly so ANSI styling does not hide
failures.

## Dependencies

The `spec` package itself adds no dependencies to the root module beyond the
ntcharts chart packages it builds. The web surface's dependency,
`github.com/go-echarts/go-echarts/v2`, belongs to the nested
`spec/echarts` module only; `task check-release` fails if it ever leaks into
the root module's dependency graph.

## Design notes

- **`Spec.Validate()`** catches the obvious mistakes (missing `Type`, zero
  dimensions, empty `Series`, invalid `Options.Orientation`, `Format.Kind`, or
  axis `Scale`, missing `Heat`/`OHLC` data for the chart types that require
  it). Both
  `Build` and `ToECharts` call it first.
- **Time values** in `DataPoint.X` accept `time.Time`, RFC 3339 strings, or
  numeric milliseconds since epoch — see `PointTime` in `helpers.go`, which
  is exported so every surface parses X values identically.
  Timeseries lines connect points in supplied order; producers should send
  chronological data. OHLC positions are determined by timestamp; candles
  sharing a column overdraw in supplied order.
- **Colours** prefer `Series.Color`, fall back to `Theme.Palette[i]`, then the
  surface default.
- **Terminal sizes are tiny for a browser**: `ToECharts()` scales sub-200
  `Width`/`Height` by 8×/16× when emitting an `Initialization` hint, so a
  60×20 terminal spec renders as roughly 480×320 px on the web.

## Extending

To add a new chart type:

1. Add the constant to `ChartType` in `spec.go` and `ChartType.IsKnown()`.
2. Add a `build<Kind>` branch in `Build()` (`build.go`).
3. Add a `toECharts<Kind>` branch in `ToECharts()` (`echarts/echarts.go`).
4. Add a runnable `Example` in `example_test.go` (and `echarts/example_test.go`).
5. Update the status table above.
