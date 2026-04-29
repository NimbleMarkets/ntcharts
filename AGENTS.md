# AGENTS.md

This file provides guidance to AI coding agents working with this repository.

## Overview

`ntcharts` is a Go terminal charting library for the Bubble Tea framework. Module path: `github.com/NimbleMarkets/ntcharts/v2` (Go 1.24+, Bubble Tea v2, Lip Gloss v2, BubbleZone for mouse support).

## Architecture

### Package Hierarchy

```
canvas/          <- Foundation: 2D grid abstraction with Bubble Tea Model
  ├── buffer/    <- Underlying cell storage
  ├── graph/     <- Drawing primitives (lines, shapes, braille)
  └── runes/     <- Glyph definitions (LineStyle, braille patterns)

linechart/       <- Cartesian plotting, builds on canvas
  ├── streamlinechart/     <- Continuous right-to-left streaming
  ├── timeserieslinechart/ <- Time-based X axis
  └── wavelinechart/       <- Wave pattern connections

barchart/        <- Horizontal/vertical bar charts
sparkline/       <- Compact data visualization
heatmap/         <- Color-mapped 2D value display

cmd/ntcharts-ohlc/  <- CLI for OHLC CSV data visualization
examples/           <- Gallery demonstrating all components
```

### Key Patterns

**Bubble Tea Integration**: Each chart exposes a `Model` with `Init()`, `Update()`, `View()`. The `UpdateHandler` field allows swapping zoom/pan/mouse behaviors (e.g., `DefaultUpdateHandler`, `DateNoZoomUpdateHandler`, `SecondUpdateHandler`).

**Coordinate Systems**: Canvas uses top-left origin while data is Cartesian (bottom-left). Use `canvas.CanvasPoint()`, `canvas.CanvasYCoordinate()` for conversion.

**Mouse Support**: Requires BubbleZone. Set `zoneManager` on the chart and wrap root `View()` with `zoneManager.Scan()`. When multiple charts are displayed, coordinate focus and `Update` routing manually (see `examples/linechart/timeseries/main.go`).

**Styling**: Lip Gloss styles throughout. Colors use numeric strings for Lip Gloss v2 compatibility. Set via `WithStyle`, `WithDataSetStyle`, `AxisStyle`, etc.

**Drawing Modes**: Charts support line drawing (`Draw()`) and braille mode (`DrawBraille()`, `DrawBrailleAll()`) for higher resolution. Control glyphs via `runes.LineStyle`.

**Ranges vs View Ranges**: Models distinguish total data ranges (`SetYRange`, `SetTimeRange`) from current viewport (`SetViewYRange`, `SetViewTimeRange`). Always set base ranges before view ranges.

**Functional Options**: Constructors use `New(w, h, opts...)` with options like `WithYRange()`, `WithAxesStyles()`, `WithUpdateHandler()`. Structs are exported for direct manipulation when needed.

**Concurrency**: Models are single-threaded; add synchronization if needed for concurrent access.

## Commands

This project uses the [Task](https://taskfile.dev) runner. Install via `brew install go-task/tap/go-task` or `go install github.com/go-task/task/v3/cmd/task@latest`.

| Command | Purpose |
|---------|---------|
| `task` or `task build` | Build everything (CLI + examples) |
| `task build-cmds` | Build CLI only |
| `task build-examples` | Build all examples |
| `task build-ex-barchart` | Build specific example (see Taskfile.yml) |
| `task clean` | Remove built binaries |
| `task go-tidy` | Format modules |
| `task go-update` | Update dependencies |
| `task validate-tapes` | Validate VHS demo tapes |
| `task build-gifs` | Render GIFs from tapes (requires vhs + imagemagick) |

Binaries output to `./bin/`. Go native tooling also works (`go build ./...`, `go test ./...`).

## Testing

Unit tests exist for core packages. No lint tasks defined; use `gofmt`/`goimports`.

```bash
go test ./...             # Run all tests
go test ./canvas/...      # Test specific package
go test ./linechart/...   # Test linechart package
go test -v ./... -run TestName  # Run specific test
```

Test files: `canvas/canvas_test.go`, `canvas/buffer/buffer_test.go`, `barchart/barchart_test.go`, `sparkline/sparkline_test.go`, `linechart/linechart_test.go`.

## Gotchas

- **Module Path**: Import paths must use `/v2` suffix: `github.com/NimbleMarkets/ntcharts/v2/...`
- **Graph Dimensions**: `linechart` subtracts axis padding from width/height; account for axis labels in TUI layouts
- **Candlestick Sync**: Candlestick drawing requires aligned datasets (same number of points, synchronized times). Missing candles indicate data misalignment.
- **Focus Management**: Forgetting `Focus()` or `zoneManager.Scan()` leads to unresponsive mouse interactions
- **CI**: Only ensures `task build` succeeds on Go 1.23+; add tests for new subsystems

## cmd/ntcharts-ohlc

CLI that reads OHLC CSV (date, open, high, low, close, adj close, volume). Flags: `--open`, `--close`, `--all`, `--braille`, `--candle`, `--vol`. Uses `timeserieslinechart` + `sparkline` for volume overlay. Good reference for data ingestion and multi-chart layout.

## Examples

- Build via `task build`, run from `./bin/` (e.g., `./bin/ntcharts-quickstart`)
- `examples/quickstart/README.md` is the best reference for Bubble Tea + BubbleZone wiring
- Subfolders contain `main.go`, `demo.gif`, and `demo.tape` (VHS). Keep GIFs in sync with `task validate-tapes` + `task build-gifs`

## Workflow

1. **Build**: Run `task build` to ensure CLI and examples compile
2. **Test**: Run `go test ./path/...` for packages you modify
3. **Patterns**: Follow Lip Gloss/Bubble Tea idioms from existing examples
4. **Imports**: Always use `/v2` import paths
5. **Docs**: Update `README.md`, `examples/README.md` when adding charts or CLI flags
