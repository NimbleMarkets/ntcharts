// ntcharts - Copyright (c) 2026 Neomantra Corp.

package timeserieslinechart

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	"charm.land/lipgloss/v2"
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

// pushTS pushes vs[i] to the data set n at day0 + i hours.
func pushTS(m *Model, n string, vs []float64) {
	for i, v := range vs {
		m.PushDataSet(n, TimePoint{Time: day0.Add(time.Duration(i) * time.Hour), Value: v})
	}
}

// newTS returns a 40x12 chart over day0..day0+10h and Y 0..10, then opts.
func newTS(opts ...Option) Model {
	return New(40, 12, append([]Option{
		WithTimeRange(day0, day0.Add(10*time.Hour)),
		WithYRange(0, 10),
	}, opts...)...)
}

var (
	tsA = []float64{1, 3, 2, 6, 5, 8, 7, 9, 4, 6, 2}
	tsB = []float64{9, 8, 8, 6, 7, 4, 5, 3, 3, 2, 1}
)

// TestGoldenOneAxis pins the renders of charts without a right Y axis.
func TestGoldenOneAxis(t *testing.T) {
	s := lipgloss.NewStyle()
	cases := map[string]func(t *testing.T) Model{
		"draw": func(t *testing.T) Model {
			m := newTS()
			pushTS(&m, DefaultDataSetName, tsA)
			m.Draw()
			return m
		},
		"drawall": func(t *testing.T) Model {
			m := newTS()
			pushTS(&m, "a", tsA)
			pushTS(&m, "b", tsB)
			m.DrawAll()
			return m
		},
		"braille": func(t *testing.T) Model {
			m := newTS()
			pushTS(&m, "a", tsA)
			pushTS(&m, "b", tsB)
			m.DrawBrailleAll()
			return m
		},
		"candle": func(t *testing.T) Model {
			m := newCandleModel(t)
			m.DrawCandle("open", "high", "low", "close", s, s)
			return m
		},
		"candle_wide_block": func(t *testing.T) Model {
			m := newCandleModel(t)
			m.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: 3, Block: true})
			return m
		},
		"fit": func(t *testing.T) Model {
			m := newTS()
			pushTS(&m, "a", tsA)
			m.FitYToView()
			m.DrawAll()
			return m
		},
		"log": func(t *testing.T) Model {
			m := newTS(WithYScale(linechart.ScaleLog), WithYRange(1, 1000))
			pushTS(&m, "a", []float64{1, 5, 30, 0, 200, 900, 50, 10, 400, 700, 2})
			m.DrawAll()
			return m
		},
		"no_x_axis": func(t *testing.T) Model {
			m := newTS(WithXYSteps(0, 2))
			pushTS(&m, "a", tsA)
			pushTS(&m, "b", tsB)
			m.DrawAll()
			return m
		},
		"no_y_axis": func(t *testing.T) Model {
			m := newTS(WithXYSteps(4, 0))
			pushTS(&m, "a", tsA)
			m.DrawAll()
			return m
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			m := build(t)
			golden(t, "oneaxis_"+name, m.View())
		})
	}
}
