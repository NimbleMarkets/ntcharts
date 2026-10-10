// ntcharts - Copyright (c) 2026 Neomantra Corp.

package linechart

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
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
	linearPath = []canvas.Float64Point{{X: -5, Y: -3}, {X: 0, Y: 4}, {X: 4, Y: 1}, {X: 10, Y: 7}}
	logPath    = []canvas.Float64Point{{X: 1, Y: 1}, {X: 10, Y: 50}, {X: 100, Y: 20}, {X: 1000, Y: 1000}}
)

// drawPath draws the axes, then line runes through pts.
func drawPath(m *Model, pts []canvas.Float64Point) {
	m.DrawXYAxisAndLabel()
	for i := 1; i < len(pts); i++ {
		m.DrawLine(pts[i-1], pts[i], runes.ArcLineStyle)
	}
}

// TestGoldenOneAxis pins the renders of charts without a right Y axis.
func TestGoldenOneAxis(t *testing.T) {
	cases := map[string]func() Model{
		"linear": func() Model {
			m := New(30, 12, -5, 10, -3, 7, WithXYSteps(2, 2))
			drawPath(&m, linearPath)
			m.DrawRune(canvas.Float64Point{X: 2, Y: 2}, 'x')
			return m
		},
		"no_x_axis": func() Model {
			m := New(30, 12, -5, 10, -3, 7, WithXYSteps(0, 2))
			drawPath(&m, linearPath)
			return m
		},
		"no_y_axis": func() Model {
			m := New(30, 12, -5, 10, -3, 7, WithXYSteps(2, 0))
			drawPath(&m, linearPath)
			return m
		},
		"odd_height": func() Model {
			m := New(30, 10, -5, 10, -3, 7, WithXYSteps(2, 3)) // graphHeight 8
			drawPath(&m, linearPath)
			return m
		},
		"log_y": func() Model {
			m := New(30, 12, 1, 1000, 1, 1000, WithXYSteps(2, 2), WithYScale(ScaleLog))
			drawPath(&m, logPath)
			return m
		},
		"log_x": func() Model {
			m := New(40, 12, 1, 1000, 1, 1000, WithXYSteps(2, 2), WithXScale(ScaleLog))
			drawPath(&m, logPath)
			return m
		},
		"zoom_in": func() Model {
			m := New(30, 12, -5, 10, -3, 7, WithXYSteps(2, 2))
			m.ZoomIn(1, 1)
			drawPath(&m, linearPath)
			return m
		},
		"move_up": func() Model {
			m := New(30, 12, -5, 10, -3, 7, WithXYSteps(2, 2))
			m.SetViewYRange(-3, 3)
			m.MoveUp(2)
			drawPath(&m, linearPath)
			return m
		},
		"auto_range": func() Model {
			m := New(30, 12, 0, 1, 0, 1, WithXYSteps(2, 2), WithAutoXYRange())
			for _, p := range linearPath {
				m.AutoAdjustRange(p)
			}
			m.UpdateGraphSizes()
			drawPath(&m, linearPath)
			return m
		},
		"braille": func() Model {
			m := New(30, 12, -5, 10, -3, 7, WithXYSteps(2, 2))
			m.DrawXYAxisAndLabel()
			for i := 1; i < len(linearPath); i++ {
				m.DrawBrailleLine(linearPath[i-1], linearPath[i])
			}
			return m
		},
		"circles": func() Model {
			m := New(30, 12, -10, 10, -10, 10, WithXYSteps(2, 2))
			m.DrawXYAxisAndLabel()
			m.DrawRuneCircle(canvas.Float64Point{X: 0, Y: 0}, 5, 'o')
			m.DrawBrailleCircle(canvas.Float64Point{X: 3, Y: 3}, 4)
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
