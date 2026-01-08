# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

This project uses the [Task](https://taskfile.dev) runner. Install via `brew install go-task/tap/go-task` or `go install github.com/go-task/task/v3/cmd/task@latest`.

| Command | Purpose |
|---------|---------|
| `task` or `task build` | Build everything (CLI + examples) |
| `task build-cmds` | Build CLI only |
| `task build-examples` | Build all examples |
| `go test ./...` | Run all tests |
| `go test ./canvas/...` | Run tests for specific package |
| `task clean` | Remove built binaries |

Binaries are output to `./bin/`.

## Architecture

**Module**: `github.com/NimbleMarkets/ntcharts/v2` (Go 1.24+, Bubble Tea v2, Lip Gloss v2)

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
```

### Key Patterns

**Bubble Tea Integration**: Each chart exposes a `Model` with `Init()`, `Update()`, `View()`. The `UpdateHandler` field allows swapping zoom/pan/mouse behaviors (e.g., `DefaultUpdateHandler`, `DateNoZoomUpdateHandler`).

**Coordinate Systems**: Canvas uses top-left origin while data is Cartesian (bottom-left). Use `canvas.CanvasPoint()`, `canvas.CanvasYCoordinate()` for conversion.

**Mouse Support**: Requires BubbleZone. Set `zoneManager` on the chart and wrap root `View()` with `zoneManager.Scan()`. Without this, mouse interactions won't work.

**Styling**: Lip Gloss styles throughout. Colors use numeric strings for Lip Gloss v2 compatibility. Set via `WithStyle`, `WithDataSetStyle`, `AxisStyle`, etc.

**Drawing Modes**: Charts support line drawing (`Draw()`) and braille mode (`DrawBraille()`, `DrawBrailleAll()`) for higher resolution. Control glyphs via `runes.LineStyle`.

**Ranges vs View Ranges**: Models distinguish total data ranges (`SetYRange`, `SetTimeRange`) from current viewport (`SetViewYRange`, `SetViewTimeRange`). Always set base ranges before view ranges.

### Functional Options

Constructors use the options pattern: `New(w, h, opts...)` with options like `WithYRange()`, `WithAxesStyles()`, `WithUpdateHandler()`.

## Testing

Unit tests exist for core packages: `canvas/canvas_test.go`, `barchart/barchart_test.go`, `sparkline/sparkline_test.go`, `linechart/linechart_test.go`, `canvas/buffer/buffer_test.go`.

```bash
go test ./canvas/...      # Test canvas package
go test ./linechart/...   # Test linechart package
go test -v ./... -run TestSpecificName  # Run specific test
```

## Important Notes

- Import paths must use `/v2` suffix: `github.com/NimbleMarkets/ntcharts/v2/...`
- CI runs `task build` on Go 1.23+; ensure all examples compile
- Chart dimensions subtract axis padding from width/height; account for labels in TUI layouts
- Candlestick drawing requires aligned datasets (same number of points, synchronized times)
- Models are single-threaded; add synchronization if needed for concurrent access
