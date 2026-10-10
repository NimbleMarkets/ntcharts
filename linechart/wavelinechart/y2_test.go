// ntcharts - Copyright (c) 2026 Neomantra Corp.

package wavelinechart

import (
	"math"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

// newY2Model returns a chart with X 0..10, left Y 0..10, right Y -50..50
// and the data set "r" on the right axis, then opts.
func newY2Model(opts ...Option) Model {
	return New(30, 12, append([]Option{
		WithXRange(0, 10),
		WithYRange(0, 10),
		WithY2Axis(linechart.Axis{Min: -50, Max: 50}),
		WithDataSetYAxis("r", linechart.YAxisRight),
	}, opts...)...)
}

func TestRightBaselineIsRightZero(t *testing.T) {
	m := newY2Model()
	seq := m.getLineSequence(nil, m.baseline("r"))
	want := canvas.CanvasYCoordinate(m.Origin().Y,
		int(math.Round(m.ScaleFloat64Point(canvas.Float64Point{Y: 5}).Y))) // right 0 is left 5
	if len(seq) != m.GraphRight()-m.Origin().X {
		t.Fatalf("sequence length %d, want %d", len(seq), m.GraphRight()-m.Origin().X)
	}
	for i, y := range seq {
		if y != want {
			t.Fatalf("column %d rests on row %d, want %d", i, y, want)
		}
	}
}

func TestRightPlotAutoRangesRightAxisOnly(t *testing.T) {
	m := newY2Model(WithY2Axis(linechart.Axis{Min: -50, Max: 50, AutoMax: true}))
	m.PlotDataSet("r", canvas.Float64Point{X: 5, Y: 80})
	if a, _ := m.Y2Axis(); a.Max != 80 || m.MaxY() != 10 {
		t.Fatalf("right max %v, left max %v; want 80, 10", a.Max, m.MaxY())
	}
}

func TestClearDataSetKeepsAssignment(t *testing.T) {
	m := newY2Model()
	m.PlotDataSet("r", canvas.Float64Point{X: 5, Y: 10})
	m.ClearDataSet("r")
	if !m.y2Sets["r"] {
		t.Fatal("ClearDataSet dropped the axis assignment")
	}
}

func TestLogRightAxisSkipsNonPositive(t *testing.T) {
	m := newY2Model(WithY2Axis(linechart.Axis{Min: 1, Max: 1000, Scale: linechart.ScaleLog}))
	for i, v := range []float64{10, 0, 100} {
		m.PlotDataSet("r", canvas.Float64Point{X: float64(i + 1), Y: v})
	}
	if n := len(m.graphPoints("r", m.dSets["r"])); n != 2 {
		t.Fatalf("%d drawable points, want 2", n)
	}
	m.DrawAll()
}

func TestTwoAxisLinesStayLeftOfRightAxis(t *testing.T) {
	m := newY2Model(WithPoints(wavePts))
	for _, p := range []canvas.Float64Point{{X: 0, Y: -50}, {X: 5, Y: 50}, {X: 10, Y: 40}} {
		m.PlotDataSet("r", p)
	}
	m.DrawAll()
	x := m.GraphRight()
	for y := 0; y < m.Origin().Y; y++ {
		if r := m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune; r != runes.LineVertical {
			t.Fatalf("right axis cell (%d,%d) = %q", x, y, r)
		}
	}
}

func TestGoldenTwoAxes(t *testing.T) {
	m := newY2Model(WithPoints([]canvas.Float64Point{{X: 1, Y: 2}, {X: 3, Y: 8}, {X: 5, Y: 4}, {X: 7, Y: 9}, {X: 9, Y: 1}}))
	for _, p := range []canvas.Float64Point{{X: 2, Y: -40}, {X: 4, Y: 30}, {X: 6, Y: -10}, {X: 8, Y: 45}} {
		m.PlotDataSet("r", p)
	}
	m.DrawAll()
	golden(t, "twoaxes", m.View())
}
