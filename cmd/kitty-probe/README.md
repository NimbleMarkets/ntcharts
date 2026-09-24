# Kitty transport probe

Run outside Bubble Tea in the terminal being tested. The tool opens `/dev/tty`,
uses raw byte input with a 100 ms read timeout, restores terminal settings on
completion/interrupt, and saves a table plus optional JSON. Redirecting stdout
is safe: probe traffic still goes to the controlling terminal.

From the repository root (the command is in the separate `cmd` Go module):

```sh
go build -o /tmp/kitty-probe ./cmd/kitty-probe
/tmp/kitty-probe -count=10 -label='terminal version / local' -output=batch.json
/tmp/kitty-probe -count=10 -batch=false -output=sequential.json
/tmp/kitty-probe -action=T -count=3 -output=transmit.json
/tmp/kitty-probe -q=1 -output=q1.json
/tmp/kitty-probe -q=2 -output=q2.json
```

The six valid probes are PNG-direct, RGBA-direct, zlib RGBA-direct, RGBA tempfile,
RGBA shared memory, and zlib RGBA shared memory. Each uses a real 1×1 red pixel.
Two negative controls create and remove a tempfile/shm object **before** sending
its name. Success on a missing-resource control invalidates a terminal's claim
that it has read local resources.

`-action=q -q=0` is the default. An `OK` response acknowledges a query, not visual
correctness. Errors, local setup failures and silence are distinct in the report.
The tool intentionally does not implement an Auto selection policy.

For a visual check:

```sh
/tmp/kitty-probe -action=T -batch=false -visual -hold=3s -output=visual.json
```

Each valid transport should show red/green above blue/white. This mode uses a
16×16 RGBA pattern, scaled to 8×4 cells. Missing-resource controls should show
nothing. Record observations alongside JSON: the tool cannot detect corrupt
rendering by reading acknowledgements. Transmit placements are deleted by their
own image IDs after each collection window, including on interruption.

## tmux and SSH

Repeat the commands **inside an attached tmux client**, with
`set -g allow-passthrough on`. The tool wraps each APC separately if `TMUX` is set;
`-tmux=true/false` and `NTCHARTS_TMUX_PASSTHROUGH` override that choice. An isolated
server avoids changing existing sessions:

```sh
printf 'set -g allow-passthrough on\n' > /tmp/kitty-probe-tmux.conf
tmux -L kitty-probe -f /tmp/kitty-probe-tmux.conf new-session
# Run the matrix commands in this new session, then exit its shell.
```

For SSH, copy/build the probe on the remote host and run it with an allocated
remote tty (`ssh -t host /path/to/kitty-probe -timeout=1s ...`). Use `-output` on
the remote host and retrieve the JSON afterward. This actually tests separate
filesystems; running a subprocess locally does not simulate SSH transport.

The environment gate recognizes kitty/Ghostty/WezTerm/iTerm2 signals and
`NTCHARTS_KITTY`. **tmux alone is not a positive signal**. `-force` explicitly
bypasses the gate, including an unsupported env
override. SSH sessions without a propagated terminal signal require that
explicit override. Do not mistake a skipped run for a transport timeout.

Environment hints can be stale when one terminal launches another: this was
observed with `GHOSTTY_RESOURCES_DIR` inherited by Terminal.app. For the clean
non-Kitty gate control, remove stale `KITTY_WINDOW_ID`, `KITTY_INSTALLATION_DIR`,
`GHOSTTY_RESOURCES_DIR`, `WEZTERM_EXECUTABLE`, `WEZTERM_PANE`, and
`NTCHARTS_KITTY`, while retaining the terminal's actual `TERM`/`TERM_PROGRAM`.
Inspect `gate` and `bytes_sent` in the report. The gate-skipped path never opens
or writes the tty (the human-readable table still goes to stdout).

## Timing and cleanup

The default collection window is 500 ms; `-timeout` accepts 100 ms through one
minute. The complete window is observed even after every reply arrives, so
suppression/duplicate responses can be measured. A terminal read can overrun the
window by up to approximately 100 ms. `-visual` adds `-hold` to that window.
Each invocation defaults to one repetition; `-count` permits up to 1000.

Batch payloads are fully built before sending them in one write. IDs are distinct
within an invocation and randomized across invocations. Latency starts just
before the write and ends when the application reads the reply; it includes
terminal/pty/network/scheduling delays, and is **not** terminal decode time.
Fragmented/coalesced responses and delayed responses to previous IDs are retained.
When evaluating a deadline, compare `latency_ms` with that deadline, rather than
assuming every recorded response arrived on time. A timeout never implies support.

Each file/shm resource is inspected at the end of its collection window, before
local cleanup. `unlinked by terminal` means the name was gone at that point;
`retained` means this tool had to remove it. This is not an unlink-latency
measurement. Cleanup runs on error, timeout and interrupt as well as success.
`cleanup_error` and the report-level `error` must be checked before trusting a run.
A forced kill or machine crash cannot run deferred cleanup.

macOS shm uses a small cgo wrapper for `shm_open`/`shm_unlink`, then `ftruncate`,
`mmap`, a byte copy, and `munmap`. Linux uses `/dev/shm` with the same mapping path.
Names are random, exclusive, mode 0600, and 25 bytes long. macOS builds
with `CGO_ENABLED=0` compile and report shm as locally unsupported. Other platforms
compile with the shm stub. Windows terminal probing and tempfile transport are
unsupported.

`bytes_sent` counts all bytes successfully handed to the tty by this tool,
including tmux wrappers, transmit deletes and visual labels. It does not claim
that the terminal consumed those bytes. Reports contain only selected terminal
metadata; SSH addresses and tmux socket paths are not recorded.

## Validation

```sh
go test -race ./cmd/kitty-probe
go vet ./cmd/kitty-probe
CGO_ENABLED=0 go test ./cmd/kitty-probe
GOOS=windows CGO_ENABLED=0 go build ./cmd/kitty-probe
```

Tests cover payload decoding, chunk-independent reply fragmentation, environment
gating, passthrough framing, resource cleanup on error/timeout/cancellation, and
real shared-memory readback/unlinking on supported builds. Shared-memory lifecycle
tests skip if the runtime lacks the capability.

To summarize a directory of JSON reports (Python 3, no third-party dependencies):

```sh
python3 cmd/kitty-probe/summarize.py /path/to/reports
```

Include reports, terminal versions, and visual observations when reporting a
compatibility issue. Reports measure the specified versions and host setup;
they are not compatibility guarantees.
