# CHANGELOG

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
