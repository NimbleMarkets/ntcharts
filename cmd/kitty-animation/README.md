# Kitty animation transport check

This command exercises the production `picture.Model` and Bubble Tea lifecycle
with an opaque animated NRGBA pattern at a fixed raster size. Run it in a real
Kitty-capable terminal, with stdout attached to that terminal:

```sh
go run ./cmd/kitty-animation -transport=png -frames=60 -output=png.json
```

The diagnostic uses PNG-direct transport. Width must be a multiple of 80,
height a multiple of 30; defaults are 1280×960. `-interval=16ms` controls the
pause between frames. Press q or Ctrl+C to stop. Use `-output` to save JSON;
**do not redirect stdout**, which Bubble Tea uses for terminal output. The
standalone `kitty-probe` instead writes through `/dev/tty` and can be redirected.

The command forwards probe replies, submits encoded frames through `pic.Update`,
and uses virtual placeholders. The reported encode time includes source
preparation and grid construction. Payload
bytes count APC data, not the placeholder grid. Neither metric measures terminal
display FPS or confirms visual correctness. The run includes a 16 ms frame pause
and startup work, so elapsed time is not a pure throughput benchmark.

Repeat the same command inside an attached tmux client with passthrough enabled,
or build and run it on an SSH host with an allocated tty, to compare preparation
timing. Save each run to a separate JSON file and record terminal versions
and visual observations alongside it.
