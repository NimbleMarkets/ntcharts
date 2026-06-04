# Kitty Graphics for Grid-Aligned UIs

`picture.Model` can render any `image.Image` through Kitty graphics. For photos
and charts, the default `FitContain` is usually right. For games, boards, tile
maps, and other coordinate-aligned UIs, use `FitFill`.

## Grid-Aligned UIs

When source pixels must line up with terminal cells:

1. Keep game or board state in logical coordinates.
2. Choose how many terminal cells each logical cell occupies.
3. Compose the full scene into one bitmap.
4. Render that bitmap with `picture.Config{Fit: picture.FitFill}`.
5. Set the model size to the terminal-cell dimensions of the composed image.

Example:

```go
pic := picture.NewWithConfig(picture.Config{
	KittyID: 7,
	Fit:     picture.FitFill,
})

cellW, cellH := pic.CellPixelSize()
grid := picture.NewGridImage(picture.GridImageConfig{
	LogicalCols:         boardWidth,
	LogicalRows:         boardHeight,
	TerminalColsPerCell: 2,
	TerminalRowsPerCell: 1,
	CellPixelWidth:      cellW,
	CellPixelHeight:     cellH,
	Background:          backgroundImage,
})

grid.DrawCell(playerX, playerY, playerSprite)
grid.DrawCell(foodX, foodY, foodSprite)

cols, rows := grid.TerminalSize()
pic.SetSize(cols, rows)
cmd := pic.SetImage(grid.Image())
```

See `examples/picture/grid` for a complete Bubble Tea example with movement,
wall collision, cell-size updates, and Glyph/Kitty toggling.

If the terminal reports its actual cell pixel size, pass those values into
`GridImageConfig` and forward the same terminal messages to `pic.Update`.

## Why FitFill

`FitContain` preserves the source image aspect ratio and letterboxes when the
source ratio differs from the terminal-cell rectangle. That is correct for
photos and many charts, but wrong for coordinate-aligned UIs: the image can
look shifted relative to collision, mouse hit testing, or logical board cells.

`FitFill` stretches the source image to exactly the configured terminal-cell
rectangle. For a grid compositor, that is the desired behavior: each logical
cell maps to the terminal cells chosen by `TerminalColsPerCell` and
`TerminalRowsPerCell`.

## Scaling

`GridImage` defaults to `draw.NearestNeighbor` scaling for backgrounds and
sprites. That keeps pixel-art edges crisp and is usually the right choice for
games, boards, and tile maps. If you are scaling photos or anti-aliased art,
set `GridImageConfig.Scaler` to another `golang.org/x/image/draw.Scaler`, such
as `draw.CatmullRom`.

## Compose, Then Render

For moving sprites, prefer composing one image per frame:

```go
grid := picture.NewGridImage(cfg)
grid.DrawOverlay(gridLines)
grid.DrawCell(x, y, sprite)
pic.SetImage(grid.Image())
```

Avoid trying to place a Kitty background and then draw normal terminal text or
glyph sprites over it. Cursor-position overlays are fragile across terminals
and renderers, and replacing Kitty placeholder cells with text creates holes in
the image placement.

The stable pattern is:

- Build a single bitmap containing the background and all sprites.
- Feed that image to `picture.Model`.
- Let `picture.Model` own the Kitty placeholder grid.

## Placeholder Details

Most applications should not build placeholders manually. `picture.Model` emits
the correct placeholder grid for a Kitty image.

For debugging, Kitty virtual placement uses:

- `github.com/charmbracelet/x/ansi/kitty.Placeholder`: Unicode placeholder cell `U+10EEEE`
- `github.com/charmbracelet/x/ansi/kitty.Diacritic(row)`: combining mark for the row
- `github.com/charmbracelet/x/ansi/kitty.Diacritic(col)`: combining mark for the column
- foreground RGB color: encodes the Kitty image ID

If you need exact grid alignment, prefer `GridImage` plus `FitFill` over manual
placeholder construction.

## Running inside Tmux

When running your TUI inside [`tmux`](https://github.com/tmux/tmux/wiki) Terminal Multiplexer, raw Kitty graphics protocol escape sequences are typically blocked or stripped by `tmux`.   To resolve this, [`picture`](./examples/README.md#picture) supports wrapping these sequences inside tmux's DCS passthrough sequences.

### Tmux Configuration Requirement

For passthrough to work, you must configure `tmux` to allow it. Add the following line to your `~/.tmux.conf` (requires tmux 3.3+):

```tmux
set -g allow-passthrough on
```

Remember to reload your tmux configuration or restart your tmux session after making this change.

### Tmux Passthrough & Capability Overrides

By default, tmux passthrough wrapping is automatically enabled if the `TMUX` environment variable is present at program startup. 

For additional developer and user ergonomics, `ntcharts` supports overriding this behavior at runtime via environment variables without requiring any code changes:

- **`NTCHARTS_TMUX_PASSTHROUGH`**: Manually controls the tmux DCS passthrough wrapping mechanism.
  - `true`, `1`, `on`, `yes`: Manually forces tmux passthrough wrapping **on**.
  - `false`, `0`, `off`, `no`: Manually forces tmux passthrough wrapping **off**.
  - *Default*: Automatically enabled if `TMUX` is set.
- **`NTCHARTS_KITTY`**: Manually controls the Kitty graphics capability detection result, which is useful when terminal queries are slow or blocked.
  - `supported`, `true`, `1`, `on`, `yes`: Forces Kitty capability to `Supported`, bypassing terminal query probing entirely.
  - `unsupported`, `false`, `0`, `off`, `no`: Forces Kitty capability to `Unsupported`.

These variables can be prefixed when running your TUI binary:

```bash
# Force Kitty capability off (e.g., to fallback to Glyph mode)
NTCHARTS_KITTY=unsupported ./your-tui-app

# Force tmux passthrough on even if TMUX variable isn't visible
NTCHARTS_TMUX_PASSTHROUGH=true ./your-tui-app
```

### Programmatic Control

If you need to manually toggle or override tmux passthrough in your code:

- Call `picture.SetTmuxPassthrough(true)` to enable wrapping.
- Call `picture.ForceKittyCapability(picture.KittyCapabilitySupported)` to manually force Kitty graphics support.


