# CHANGELOG

## v2.0.4 (unreleased)

  * Replace `github.com/eliukblau/pixterm => github.com/NimbleMarkets/pixterm` until bugfixes are upstreamed

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
