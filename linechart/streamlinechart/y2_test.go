// ntcharts - Copyright (c) 2026 Neomantra Corp.

package streamlinechart

import (
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

// newY2Model returns a chart with left Y 0..10, right Y 0..100 and the
// data set "r" on the right axis, then opts.
func newY2Model(opts ...Option) Model {
	return New(30, 10, append([]Option{
		WithYRange(0, 10),
		WithY2Axis(linechart.Axis{Min: 0, Max: 100}),
		WithDataSetYAxis("r", linechart.YAxisRight),
	}, opts...)...)
}

func TestBuffersEndAtRightAxis(t *testing.T) {
	m := newY2Model()
	want := m.GraphRight() - m.Origin().X
	for n, ds := range m.dSets {
		if ds.sBuf.Size() != want {
			t.Fatalf("data set %q buffer size %d, want %d", n, ds.sBuf.Size(), want)
		}
	}
}

func TestRightPushAutoRangesRightAxisOnly(t *testing.T) {
	m := newY2Model(WithY2Axis(linechart.Axis{Min: 0, Max: 100, AutoMax: true}))
	m.PushDataSet("r", 500)
	if a, _ := m.Y2Axis(); a.Max != 500 || m.MaxY() != 10 {
		t.Fatalf("right max %v, left max %v; want 500, 10", a.Max, m.MaxY())
	}
}

// stayLeft fails if anything but the right axis line is in its column.
func stayLeft(t *testing.T, m *Model) {
	t.Helper()
	x := m.GraphRight()
	for y := 0; y <= m.Origin().Y; y++ {
		if r := m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune; r != runes.LineVertical {
			t.Fatalf("right axis cell (%d,%d) = %q", x, y, r)
		}
	}
}

func TestTwoAxisLinesStayLeftOfRightAxis(t *testing.T) {
	m := newY2Model(WithStream(streamA))
	for _, v := range []float64{10, 90, 40, 100, 0, 70, 20, 80, 30, 60, 50, 100} {
		m.PushDataSet("r", v)
	}
	m.DrawAll()
	stayLeft(t, &m)
}

func TestLogRightAxisDraws(t *testing.T) {
	m := newY2Model(WithY2Axis(linechart.Axis{Min: 1, Max: 1000, Scale: linechart.ScaleLog}))
	for _, v := range []float64{10, 0, 100, 1000} {
		m.PushDataSet("r", v)
	}
	m.DrawAll()
	stayLeft(t, &m)
}

func TestGoldenTwoAxes(t *testing.T) {
	m := newY2Model(WithStream(streamA))
	for _, v := range []float64{95, 90, 80, 85, 60, 70, 40, 50, 20, 30, 5, 10} {
		m.PushDataSet("r", v)
	}
	m.DrawAll()
	golden(t, "twoaxes", m.View())
}
