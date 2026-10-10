// ntcharts - Copyright (c) 2026 Neomantra Corp.

package streamlinechart

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden from the current output")

// golden compares got with testdata/<name>.golden, or rewrites the file
// when the tests run with -update.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test -run %s -update)", err, t.Name())
	}
	if got != string(want) {
		t.Fatalf("%s changed:\n%s\n--- want ---\n%s", path, got, want)
	}
}

var streamA = []float64{1, 3, 2, 6, 5, 8, 7, 9, 4, 6, 2, 5}

// TestGoldenOneAxis pins the renders of charts without a right Y axis.
func TestGoldenOneAxis(t *testing.T) {
	cases := map[string]func() Model{
		"draw": func() Model {
			m := New(30, 10, WithYRange(0, 10), WithStream(streamA))
			m.Draw()
			return m
		},
		"drawall": func() Model {
			m := New(30, 10, WithYRange(0, 10), WithStream(streamA),
				WithDataSetStream("b", []float64{9, 8, 6, 7, 4, 3, 2}))
			m.DrawAll()
			return m
		},
		"log": func() Model {
			m := New(30, 9, WithYScale(linechart.ScaleLog), WithYRange(1, 1e4), WithStream(stream))
			m.Draw()
			return m
		},
		"xy_steps": func() Model {
			m := New(30, 10, WithXYSteps(2, 2), WithYRange(0, 10), WithStream(streamA))
			m.Draw()
			return m
		},
		"view": func() Model {
			m := New(30, 10, WithYRange(0, 10), WithStream(streamA))
			m.SetViewYRange(2, 8)
			m.Draw()
			return m
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			m := build()
			golden(t, "oneaxis_"+name, m.View())
		})
	}
}
