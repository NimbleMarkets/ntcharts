// ntcharts - Copyright (c) 2026 Neomantra Corp.

package wavelinechart

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
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

var (
	wavePts  = []canvas.Float64Point{{X: 1, Y: 2}, {X: 3, Y: -2}, {X: 5, Y: 4}, {X: 7, Y: -4}, {X: 9, Y: 1}}
	wavePtsB = []canvas.Float64Point{{X: 2, Y: -1}, {X: 6, Y: 3}, {X: 10, Y: 0}}
)

// TestGoldenOneAxis pins the renders of charts without a right Y axis.
func TestGoldenOneAxis(t *testing.T) {
	cases := map[string]func() Model{
		"draw": func() Model {
			m := New(30, 12, WithXRange(0, 10), WithYRange(-5, 5), WithPoints(wavePts))
			m.Draw()
			return m
		},
		"drawall": func() Model {
			m := New(30, 12, WithXRange(0, 10), WithYRange(-5, 5), WithPoints(wavePts),
				WithDataSetPoints("b", wavePtsB))
			m.DrawAll()
			return m
		},
		"log": func() Model {
			m := New(34, 11, WithYScale(linechart.ScaleLog), WithXRange(0, 6), WithYRange(0.01, 1000))
			for i, v := range []float64{0.02, 0.5, 0, 30, -4, 1000} {
				m.Plot(canvas.Float64Point{X: float64(i + 1), Y: v})
			}
			m.Draw()
			return m
		},
		"view": func() Model {
			m := New(30, 12, WithXRange(0, 10), WithYRange(-5, 5), WithPoints(wavePts))
			m.SetViewXRange(2, 8)
			m.Draw()
			return m
		},
		"no_x_axis": func() Model {
			m := New(30, 12, WithXYSteps(0, 2), WithXRange(0, 10), WithYRange(-5, 5), WithPoints(wavePts))
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
