# CHANGELOG

## Unreleased

 * fix(buffer): `Float64PointScaleBuffer.Offset` now returns its offset instead of its scale, avoiding redundant rescaling of time-series data.

## v2.6.0 (2026-10-01)

 * fix(spec): Numeric-X line charts now join each series' points with braille lines instead of drawing baseline spikes. Linear/log scales, series colours, and Y pins are preserved; singleton points draw as dots. **Behavior/API change from v2.5.0:** `Build` for `ChartTypeLine` returns `*linechart.Model` instead of `*wavelinechart.Model`. Points are connected in input order, with no connection between series.
 * feat(linechart): Add logarithmic axes. `linechart.Scale` is `ScaleLinear` (the default) or `ScaleLog` (base 10), set with `WithXScale` / `WithYScale` or `SetXScale` / `SetYScale` and read with `XScale` / `YScale`; `wavelinechart` exposes both options, `timeserieslinechart` and `streamlinechart` expose `WithYScale`. Ranges, data points, and label formatter values stay in data units. A log axis labels powers of ten on the rows and columns where they fall (adding 2× and 5× when a range holds fewer than three, and falling back to even spacing in a range too narrow for two round values), with the axis steps as the minimum label spacing. Points at or below zero are not drawn and are ignored by auto-ranging, non-positive range bounds are replaced (minimum = maximum/10, or 1..10), and zoom and pan steps are measured in decades. On a log Y axis a `wavelinechart` rests on the bottom of the view range. Linear axes are unchanged.
 * feat(heatmap): Heatmaps can draw filled cells and name their rows and columns. `WithCellSize` draws each point as a block that tiles the plot instead of a single character, and `WithXLabels` / `WithYLabels` (or `SetXLabels` / `SetYLabels`) draw column names under the plot and row names, centred on their rows, in a margin to its left. Existing heatmaps are unchanged unless they opt in.
 * feat(examples): Add `heatmap/labels`, a day-by-time heatmap with filled cells and row and column names, with keys to toggle the labels and resize the chart. It is in the WASM gallery as "Heatmap (labels)".
 * feat(spec): Heatmaps are now drawn as filled blocks with their `x_axis.labels` and `y_axis.labels`; previously each cell was one isolated character and the labels were ignored. **Behavior change:** row 0 (and `y_axis.labels[0]`) is now the top row, as in flint heatmaps, where it used to be the bottom; heatmaps with no labels fill the whole chart; a negative cell index is a `Build` error.
 * feat(spec): Add a logarithmic axis scale. `XAxis.Scale` / `YAxis.Scale` accept `"linear"` (`spec.ScaleLinear`, the default when empty) or `"log"` (`spec.ScaleLog`, base 10); `Validate` rejects other values. `Build` honours a log Y axis for line, scatter, timeseries, and OHLC charts and a log X axis for line and scatter, using the chart models' own log scale. Unpinned bounds widen to whole decades, `YAxis.Min` / `Max` stay in data units with the same one-sided pin rule, and `Format` applies as on a linear axis (with a short k/M default when unset). It is a `Build` error to ask for a log axis on bar, heatmap, or sparkline charts or on a time X axis, or to put a value or pin `<= 0` on one. `echarts.ToECharts` ignores `scale` for now. See "Logarithmic axes" in `spec/README.md`.
 * feat(examples): Add `linechart/logscale`, a time series on a log Y axis with a key to toggle linear/log and zoom the Y axis. It is in the WASM gallery as "Log Scale".

## v2.5.0 (2026-09-30)

 * fix(wasm-build): build TinyGo demos with `-gc=boehm`, fixing the slow "GPU Shaders (TinyGo)" gallery demo.
 * feat(spec): Add `spec` package for surface-agnostic chart specifications: a JSON-marshalable `spec.Spec` (type, axes, format directives, series, heat data, OHLC points, options, theme) that `spec.Build` renders to a bar, line, timeseries, scatter, heatmap, sparkline, or candlestick (OHLC) model. `Validate` catches structural mistakes up front. See `spec/README.md` for the schema and the per-surface fidelity matrix.
   * The web surface lives in its own nested module, `github.com/NimbleMarkets/ntcharts/spec/echarts/v2` (`echarts.ToECharts(s)`), published with the `spec/echarts/v2.X.Y` tag, so the go-echarts dependency stays out of the core library. `spec.PointX`, `spec.PointTime`, and `spec.DeriveBarLabels` are exported for surfaces to share.
   * Bar charts forward `y_axis.min` to `barchart.WithMinValue` (negative bars extend the floor automatically); `Validate` also rejects an unknown `x_axis.type` or `series.type`.
   * Terminal builders preserve one-sided Y pins, widen singleton time ranges, and reject insufficient plot space and equal explicit Y bounds. Shared `Data.XAxisData` supplies omitted X values for bar, line, scatter, and timeseries charts; `DeriveBarLabels` accepts `spec.Data`. Heatmap colour-domain pins work independently, and dense matrices map `Matrix[y][x]` to the same cells as sparse data. OHLC requires exactly one series. ECharts bar charts honour the theme palette.
 * feat(timeserieslinechart): `DrawCandleWithOpts` draws OHLC candles with multi-column bodies, edge insets (so candles at the axis or chart edge shift inward instead of losing part of their body), and a choice of line-rune or solid block bodies through `DrawCandleOpts.Block`. New `graph.DrawCandlestickBottomToTopWide` and `graph.DrawCandlestickBlockBottomToTop` primitives and the `runes.UpperHalfBlock` constant back them. Candles outside the time viewport remain clipped. **Behavior change:** `DrawCandle` now includes candles exactly at the viewport's maximum timestamp.
 * fix(linechart): the final X tick uses the true axis maximum. When its label does not fit left-anchored, it is right-aligned if the entire span and its left neighbour are free. Labels that cannot fit, collide, or repeat the preceding label remain hidden. **Behavior change:** final labels may change value or visibility; existing demo captures may differ.
 * fix(picture): query the terminal for Kitty shared-memory (`t=s`) support at startup, and send direct frames unless it answers `OK`. Frames were silently dropped on remote hosts and in terminals that cannot read the objects.
 * feat(shaders): time each stage of a frame. The header gains a line dividing R into `setup / submit / map / copy`, and `-report` adds per-stage summaries (mean, median, 95th percentile over the last 600 frames), startup times, heap size, and the compiler. Existing report fields are unchanged. The browser build publishes the same report in `globalThis.ntchartsShadersReport`.
 * feat(shaders): add `-medium shm|direct`, and read flags from the page's query string in the browser (`?preset=julia&medium=direct`).

## v2.4.0 (2026-09-28)

 * **Import migration:** the optional chartpicture module is now `github.com/NimbleMarkets/ntcharts/picture/chartpicture/v2`. Update imports from `github.com/NimbleMarkets/ntcharts/v2/picture/chartpicture` and run `go get github.com/NimbleMarkets/ntcharts/picture/chartpicture/v2@v2.4.0`. The old path was a valid package in the root module through v2.2.0; splitting it into a nested module in v2.3.0 broke remote resolution. Core library import paths are unchanged.
 * chore(release): the root, chartpicture, examples, and shaders modules now share the same release version, with directory-prefixed tags for nested modules. Local development uses workspaces; published modules have no local replacements. Tidy preserves released sibling checksums, and releases generate new hashes in dependency order before tagging. `task check-release` verifies standalone builds with readonly module files, vendoring, checksums, dependency isolation, and demo installation using temporary Git tags before anything is published. CI fetches full history and tags so a new unreleased changelog section can still resolve older pinned modules.
 * feat(wasm-build): each demo page includes a native `go run ...@latest` command using its owning module's path.
 * fix(chartpicture): `BarChartOptionFromNT` now forwards the bar chart's minimum to the image value axis and stacks multi-value bars, so the go-analyze image matches the glyph chart, including negative segments below the baseline. **Behavior change:** multi-value bars were previously drawn grouped side by side in the image.
 * feat(examples): the shaders and picture demos request the Kitty shared-memory medium (`picture.KittyMediumSharedMemory`), so in the gallery with booba 0.7.0 frames are handed to the terminal as raw RGBA buffers instead of PNG escape streams; unsupported terminals fall back to direct PNG automatically. The shaders stats line now names the transport actually used (`shm`, `png`, or `rgba`).
 * fix(shaders): the browser build passes `tea.WithoutSignalHandler()`, since TinyGo 0.42.0's `os/signal` loop spins without yielding and hung the TinyGo gallery demo. Signals are meaningless in a browser, so the Go build is unaffected.
 * chore(deps): update to [go-booba v0.7.0](https://github.com/NimbleMarkets/go-booba) (GPU terminal renderers, browser Kitty shared-memory images, faster input echo), bubbletea v2.0.10, and the matching bubbletea WASM fork. **ntcharts now requires Go 1.26.8+.**
 * feat(wasm-build): demos can set `toolchain: tinygo` in `web/demos.yaml` to be compiled with TinyGo (`-opt=2 -no-debug`) and served with TinyGo's `wasm_exec.js`. The gallery gains "GPU Shaders (TinyGo)", the same source at about a third of the wasm size. Building the site now needs TinyGo 0.42.0+, or `-skip-tinygo`.
 * feat(shaders): the shaders example joins the live WASM gallery, running its WGSL compute shaders on the browser's WebGPU. Browsers without WebGPU see an explanatory message instead of a blank terminal.
 * chore(shaders): starting with v2.4.0, run the shader gallery without a clone using `go run github.com/NimbleMarkets/ntcharts/examples/shaders/v2@latest`. It is published with the `examples/shaders/v2.X.Y` tag and remains isolated from ordinary examples and core library dependencies.
 * feat(picture): add opt-in raw RGBA and shared-memory Kitty transport, with PNG/direct fallback when shared-memory creation is unavailable. Native shared memory requires a local compatible terminal; direct transmission supports SSH.
 * feat(picture): expose requested/actual transport metadata and signed Kitty placement depth. Preserve per-chunk tmux framing for PNG and RGBA, and single wrapping for shared-memory references.

## v2.3.0 (2026-09-26)

 * **Bidirectional bar charts:** positive and negative values stack around a floating zero axis, with improved fractional rendering and hit testing (#13).
 * **Kitty graphics improvements:** automatic tmux passthrough, faster frame encoding, and a fix for animation flicker at reduced resolution.
 * **New demos and diagnostics:** a WebGPU shader example, `kitty-probe`, and a `kitty-animation` diagnostic.
 * **Leaner dependencies:** separate modules and opt-in image decoder registration via `picture/decoders`. **Breaking change:** `picture` no longer registers image decoders automatically.
 * **Go 1.26+ required**, alongside updated dependencies and a unified `task release` workflow.

<details>
<summary>Full release notes</summary>

 * fix(picture): fix animation flicker when `KittyResolutionFactor` is below 1. Geometry bookkeeping now uses the same scaled cell-pixel dimensions as encoding, so unchanged animation frames no longer emit a Kitty image delete.
 * chore(deps): update Go dependencies (bubbles v2.2.1, bubbletea v2.0.9, lipgloss v2.0.6, go-analyze/charts v0.6.1, and others) and GitHub Actions. **ntcharts now requires Go 1.26+.**
 * chore(build): scope the bubbletea WASM fork to a dedicated `wasm.work` workspace used only by the WASM showcase build. Normal builds and tests now use upstream `charm.land/bubbletea/v2`, matching what library consumers resolve.
 * feat(barchart): support negative values (#13). Within a bar, positive segments stack up (or right) from zero and negative segments stack down (or left), and the axis moves to wherever zero falls. Added `SetMin`, `MinValue`, and `WithMinValue`; `AutoMaxValue` now tracks the minimum as well. **Behavior change:** negative values were previously drawn as zero. New `graph.DrawColumnTopToBottom` and `graph.DrawRowRightToLeft` draw the downward and leftward bars, using inverse block elements in reverse video for fractional ends.
 * fix(barchart): handle bidirectional rendering (#13) and hit-test edge cases. Fractional negative stacks preserve their total visible length and existing boundary colors; when two segments share a fractional cell with the background, the segment occupying more of that cell supplies its color. Charts with only one drawable cell show the larger side instead of disappearing (positive wins ties). Hit testing respects rounded bar endpoints and clipping, excludes labels and out-of-canvas points, and safely handles cleared charts. Added regression coverage for horizontal and vertical charts, tiny layouts, and shared stack boundaries.
 * feat(picture): add automatic tmux passthrough wrapping for Kitty graphics, with `picture.SetTmuxPassthrough` for programmatic control. Each 4 KiB Kitty chunk is wrapped in its own DCS so large images are not discarded by tmux's input buffer limit. Added `NTCHARTS_TMUX_PASSTHROUGH` and `NTCHARTS_KITTY` environment variables for easy runtime overrides. When passthrough is enabled, tmux counts as a positive signal for the Kitty capability probe, since the outer terminal's environment is hidden.
 * perf(picture): encode Kitty images with `png.BestSpeed`. Frames encode faster at the cost of somewhat larger payloads.
 * feat(kitty): add `kitty-probe` test program, with transport diagnostics and JSON reports
 * feat(examples): add `kitty-animation` picture transport lifecycle diagnostic
 * feat(examples): add `shaders` example, running WebGPU WGSL shaders via `wgpu` and compositing them with a `picture.Model`. It has its own Go module; run it with `task shaders`. The `m` key (or `-mosaic`) shows four shaders at once in a 2×2 mosaic composed into a single picture frame. The `s` key (or `-source`) shows the shader's WGSL source with syntax highlighting beside the image.
 * feat(ci): Restructure `go.mod` into subdirectories, to shield users from unnecessary dependencies.
 * chore(release): add `task release VERSION=v2.X.Y`, which bumps the nested modules, dates the changelog, commits, and tags the root and `picture/chartpicture/v0.X.Y` together.
 * feat(picture): Add `picture/decoders` package for automatic registration of `Image` decoders like PNG.  Previously, `picture` would pull these in automatically, even if one didn't need it. **BREAKING**

</details>

## v2.2.0 (2026-05-28)

  * feat(picture): detect Kitty graphics support and gate Toggle on capability.  In other words, don't blast the terminal with characters when it doesn't support Kitty graphics. `QueryKittySupport` gates the probe based on environment variables; `ForceKittyCapability` can override detection.
  * feat(picture): add `GridImage` for composing logical-cell boards and sprites into a single image for grid-aligned Kitty UIs. The config uses explicit logical-grid and terminal-cells-per-cell fields, defaults to nearest-neighbor scaling for crisp board/tile sprites, and includes a full-board `DrawOverlay` helper for grid lines and effects.
  * feat(picture): add `FitAnchor` (`AnchorCenter`, `AnchorTop`, `AnchorBottom`, `AnchorLeft`, `AnchorRight`) for `FitCover` cropping. `Config.Anchor` and `Model.Anchor()` /
  `Model.SetAnchor()` let callers preserve a specific edge when cover-scaling overflows; the zero value remains `AnchorCenter`, matching previous center-crop behavior. `chartpicture`,
  `heatpicture`, and `pictureurl` forward the same anchor config/accessors for API parity.
  * feat(picture): add `KittyResolutionFactor` to scale encoded Kitty image resolution, saving rendering bandwidth and CPU.
  * feat(yomamma): add Mother's Day example app.
  * feat(wasm): add cooperative yielding (`yieldToJS`) during heavy rendering, and enhance web demo UI and builder.
  * fix(examples): enable AltScreen in full-screen views to restore terminal state on exit.
  * fix(kitty): improve rendering robustness under Kitty graphics (geometry-based staleness, deferred commits, capability timeouts).
  * Various bug fixes and performance improvements.

## v2.1.0 (2026-05-01)

  * **BREAKING (visual only):** `picture.Model` in Kitty mode now defaults to `FitContain` (preserve aspect ratio, letterbox) instead of stretching to fill the cell rectangle. Glyph and Kitty paths now produce identical aspect ratio. Restore previous Kitty behavior with `picture.Config{Fit: picture.FitFill}`.
  * feat(picture): add `FitMode` (`FitContain`, `FitFill`, `FitCover`), `Config.Fit`, `Model.Fit()`, `Model.SetFit()`. Both render paths flow through a shared `prepareSource` helper so fit semantics are applied identically in Glyph and Kitty modes. `chartpicture`, `heatpicture`, and `pictureurl` add the same `Config.Fit` field and `Fit()` / `SetFit()` forwarders for API parity.
  * fix(picture): Glyph mode uses ansimage's `ScaleModeResize` (rather than `ScaleModeFit`) so terminals reporting non-1:2 cell pixel ratios (line-spacing, retina cells) no longer cause Glyph to letterbox while Kitty fills. `prepareSource` already applies the chosen `FitMode`, so the half-block grid just renders the prepared bitmap at exactly `(cols, rows*2)`.
  * feat(examples/picture): `f` key cycles fit modes (Contain → Fill → Cover) on both panes; footer shows the current fit.
  * Replace `github.com/eliukblau/pixterm => github.com/NimbleMarkets/pixterm` until bugfixes are upstreamed
  * fix(kitty): geometry-change renders now delete the previous placement before re-transmitting, fixing stuck-at-old-geometry behavior in Ghostty (and other terminals where TransmitAndPut at new c/r doesn't relocate an already-on-screen virtual placement)
  * Add [web-based demos](https://nimblemarkets.github.io/ntcharts) using [`go-booba`](https://github.com/NimbleMarkets/go-booba)
  * feat(picture): add `Config.CellPixelWidth` / `CellPixelHeight`,
      `Model.SetCellPixelSize` / `CellPixelSize`, and `Model.Init` /                                                                         
      `RequestCellSize` 
  * fix(canvas,picture): clamp negative dims to 0 in `canvas.Model.Resize`,                                                                
      `picture.Model.SetSize`, and `chartpicture.Model.SetSize`
  * feat: add [`picture/heatpicture`](examples/README.md#heat-picture) — continuous-field heatmap rendered through `picture.Model`. Sampler-driven (`func(x, y float64) float64`); Kitty mode samples at full terminal-pixel resolution for smooth gradients, Glyph mode samples at half-block resolution for fast fallback. Includes a render throttle (one in-flight at a time, with dirty-flush follow-up) and a runtime `SamplingFactor` knob to trade quality for animation smoothness on large terminals. New `examples/heatpicture/perlin` demo.

## v2.0.3 (2026-04-28)
 
[`chartpicture`](examples/README.md#chart-picture) is experimental. It renders charts as images; it can also bridge Apache ECharts.  It is not very useful unless your terminal supports the Kitty Graphics Protocol.  We are exploring how to improve it, but wanted to share with the community.

 * feat: add `picture/chartpicture` package — renders [github.com/go-analyze/charts](https://github.com/go-analyze/charts) chart images via the embedded `picture.Model`
   * `chartpicture.Model` produces chart frames asynchronously through `tea.Cmd`
   * Recipes for common sources: `WithLineRecipe`, `WithBarRecipe`, `WithEChartsJSONRecipe`, `WithPainterFuncRecipe`
   * Configurable via `WithChartSource`, `WithRecipe`, `WithKittyID`, `WithBackground`, `WithTheme`
   * Bridges `barchart.Model` and `linechart.Model` directly via the new `barchart.Model.Data()` accessor
 * fix: propagate streamlinechart height fix to wave/timeserieslinechart (#7)
 * Correctness and security fixes

## v2.0.2 (2026-04-27)

 * feat: add [`picture` and `pictureurl`](examples/README.md#picture)
 * feat: add controls to prevent intentional or accidental memory bombs (#12 #17)
   * add `DefaultMaxPoints` global and `config.MaxInterpolationPoints`
   * add `GetLinePointsWithLimit` and `GetCirclePointsWithLimit`
 * chore: update Golang dependencies

## v2.0.1 (2026-04-13)

 * feat: Support millisecond resolution in timeseries line chart (#15)
 * fix: linechart label glitch (#16)
 * ci: Update GitHub Action versions and add test phase

## v2.0.0 (2026-02-28)

 * Upgrade to BubbleTea v2.0.0 -- official release! :tada:

## v2.0.0-beta.6 (2026-01-10)

 * Fix incorrect height in streamlinechart (#7)

This was teased out by adding better test scaffolding
and then asking an LLM for help.

## v2.0.0-beta.5 (2026-01-08)

 * Upgrade to BubbleTea v2.  This lives in the `v2` branches.
 * Thanks to **@kpumuk** for this contribution.

NOTE: The `v2` designation is for BubbleTea API compatibility.  The `ntcharts` API is still subject to change, as indicated by `beta`.

## v0.4.0 (2026-01-08)

 * Bug fixes and added [GitHub Actions](https://github.com/NimbleMarkets/ntcharts/actions) to test builds

## v0.3.1 (2024-12-17)

 * Sanitize AoC example

## v0.3.0 (2024-12-14)

Initial Heatmap support is here! :tada:   It is still missing axis labels and better UX.  We are still exploring the API.  Please provide feedback on GitHub.

 * ADD: Initial [heatmap support](./examples/README.md#heatmap) (#2)
 * FIX: `canvas.SetRune` did not honor Canvas' default style.
 * ADD: `canvas.SetRuneWithStyle` and `canvas.GetCellStyle`

## v0.2.0 (2024-11-15)

 * Add [candlestick/OHLC support](./examples/README.md#candlesticks) with (#3)
 * Added `ntcharts-ohlc` example
 * Thanks to @tonyling for this work.

## v0.1.2 (2024-03-28)

 * Fix pkgsite documentation and badges.

## v0.1.0 (2024-03-28)

 * Welcome to the world `ntcharts`! :tada:
