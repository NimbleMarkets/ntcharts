# CHANGELOG

## v2.1.1 (2026-05-02)

 * feat(picture): detect Kitty graphics support and gate Toggle on capability.  In other words, don't blast the terminal with characters when it doesn't support Kitty graphics. `QueryKittySupport` gates the probe based on environment variables; `ForceKittyCapability` can override detection.

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
