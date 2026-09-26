# CHANGELOG

## v2.3.0 (unreleased)

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
