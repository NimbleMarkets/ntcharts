# Picture Grid

This example demonstrates `picture.NewGridImage` for coordinate-aligned Kitty graphics. It composes a small game board into one bitmap, then renders that bitmap through `picture.Model` with `picture.Config{Fit: picture.FitFill}` so each logical grid cell maps exactly to a fixed terminal-cell region.

Run it with:

```sh
go run ./examples/picture/grid
```

Controls:

- Arrow keys, WASD, or HJKL move the player.
- `g` toggles Glyph and Kitty mode.
- `q` quits.

The important pattern is in `renderBoard`: build a `GridImage`, draw sprites into `grid.CellRect(...)` or with `grid.DrawCell(...)`, call `grid.TerminalSize()`, and pass `grid.Image()` to `picture.Model.SetImage`.
