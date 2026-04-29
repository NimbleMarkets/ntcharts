# ntcharts Visual and Terminal Tests

This directory contains regression tests for ntcharts, including:
1. **Visual Tests**: Using multimodal LLM capabilities to verify chart rendering correctness (GIFs).
2. **Terminal Tests**: Verifying deterministic string output against reference files.

## Directory Structure

```
tests/
├── tests.json          # Test definitions
├── Taskfile.yml        # Task runner for test operations
├── README.md           # This file
├── cmd/                # Test program sources
│   └── {test-id}/
│       ├── prompt.md   # Visual verification criteria
│       ├── main.go     # Test program
│       └── demo.tape   # VHS recording script
├── terminal/
│   └── refs/           # Reference string outputs
│       └── {test-id}.txt
└── multimodal/
    └── refs/           # Reference visual outputs (pictures)
```

## Commands

Run from the `tests/` directory or root using `-t tests/Taskfile.yml`.

```bash
# Terminal Verification
task verify-terminal            # Run all tests and compare output to terminal/refs

# Validation & Execution
task validate                   # Validate test structures
task run                        # Generate GIFs for visual tests
task clean                      # Remove generated results
```

## Adding a New Test

1. Add entry to `tests.json`
2. Create directory `cmd/{test-id}/`
3. Create `prompt.md`, `main.go`, and `demo.tape`
4. Generate terminal reference: `go run cmd/{test-id}/main.go > terminal/refs/{test-id}.txt`
5. Run `task verify-terminal` to confirm
