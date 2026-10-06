# CHANGELOG

**BubbleTea `v2` NOTE:** See the [ntcharts/v2](https://github.com/NimbleMarkets/ntcharts/tree/v2) branch and its [CHANGELOG](https://github.com/NimbleMarkets/ntcharts/blob/v2/CHANGELOG.md).  `v2` is now the default development branch.

## v0.5.2 (2026-10-06)

Fixes backported from the `v2` branch:

 * Fix Y scaling in wavelinechart and timeserieslinechart so values line up with the Y-axis ticks when the X axis is hidden, as streamlinechart was fixed in v0.5.0 (#7)
 * Never label an axis value as `-0` with `DefaultLabelFormatter` (#7)
 * Fix missing or misplaced final axis labels when the label step doesn't divide the axis evenly (#16)
 * Label the final X-axis tick with the true axis maximum
 * `Float64PointScaleBuffer.Offset` now returns the offset instead of the scale

## v0.5.1 (2026-03-20)

 * Fix CI/CD of v1 tags

## v0.5.0 (2026-03-20)

 * Fix incorrect height in streamlinechart (#7)

This was teased out by adding better test scaffolding
and then asking an LLM for help.

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
