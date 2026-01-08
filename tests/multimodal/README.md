# Multimodal Visual Tests

This directory contains visual tests for ntcharts that use multimodal LLM capabilities to verify chart rendering correctness.

## How It Works

1. Each test has a Go program that renders a chart with known data
2. VHS captures the terminal output as a GIF
3. A multimodal LLM examines the GIF against expected criteria
4. Issues are reported (and optionally fixed)

## Directory Structure

```
tests/multimodal/
├── tests.json          # Test definitions for tooling
├── Taskfile.yml        # Task runner for test operations
├── refs/               # Reference test definitions
│   └── {test-id}/
│       ├── prompt.md   # Expected visual criteria for LLM
│       ├── main.go     # Test program
│       └── demo.tape   # VHS recording script
└── results/            # Generated test outputs
    └── {test-id}-out.gif
```

## Commands

Run from the repository root using `-t` to specify the Taskfile:

```bash
task -t tests/multimodal/Taskfile.yml validate          # Validate all test structures and VHS files
task -t tests/multimodal/Taskfile.yml validate-test TEST=canvas-text  # Validate single test
task -t tests/multimodal/Taskfile.yml check-compiles TEST=linechart-simple  # Check single test compiles
task -t tests/multimodal/Taskfile.yml check-compiles-all  # Check all tests compile
task -t tests/multimodal/Taskfile.yml run               # Run all tests and generate GIFs
task -t tests/multimodal/Taskfile.yml run-test TEST=canvas-text       # Run single test
task -t tests/multimodal/Taskfile.yml list-tests        # List available tests
task -t tests/multimodal/Taskfile.yml clean             # Remove generated results
```

Or run from within `tests/multimodal/`:

```bash
cd tests/multimodal
task validate
task check-compiles TEST=canvas-text
```

## Test List

| ID | Name | Package | Description |
|----|------|---------|-------------|
| `canvas-text` | Canvas Text Rendering | canvas | Verifies basic canvas text placement and styling |
| `linechart-simple` | Simple Line Chart | linechart | Line chart with known data points, verifies axes and line drawing |
| `linechart-braille` | Braille Mode Line Chart | linechart | Line chart rendered in braille mode for higher resolution |
| `barchart-horizontal` | Horizontal Bar Chart | barchart | Bar chart with horizontal bars and labels |
| `barchart-vertical` | Vertical Bar Chart | barchart | Bar chart with vertical columns and labels |
| `sparkline-basic` | Basic Sparkline | sparkline | Simple sparkline visualization with known values |
| `timeseries-dates` | Time Series with Dates | linechart/timeserieslinechart | Time series chart with date-based X axis |
| `heatmap-gradient` | Heatmap Gradient | heatmap | Heatmap showing color gradient from low to high values |
| `streamline-wave` | Streamline Wave Pattern | linechart/streamlinechart | Streaming line chart with wave pattern data |
| `wavelinechart-sine` | Waveline Sine Pattern | linechart/wavelinechart | Waveline chart connecting points in wave pattern |

## Adding a New Test

1. Add entry to `tests.json`
2. Create directory `refs/{test-id}/`
3. Create `prompt.md` with visual verification criteria
4. Create `main.go` test program (must exit cleanly, render to stdout)
5. Create `demo.tape` VHS script
6. Run `task validate-test TEST={test-id}` to verify structure
7. Update this README's test list

## Execution Modes

**Validation Mode**: `task validate` checks that all test files exist and VHS tapes are valid.

**Execution Mode**: `task run` builds and runs each test via VHS, outputs GIFs to `results/`.

**Review Mode**: A multimodal LLM examines each GIF against its `prompt.md` criteria and reports pass/fail with details.

**Fix Mode**: On failure, the LLM can attempt to fix the library code based on the visual discrepancy.
