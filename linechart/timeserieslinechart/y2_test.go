// ntcharts - Copyright (c) 2026 Neomantra Corp.

package timeserieslinechart

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	"charm.land/lipgloss/v2"
)

// newY2Model returns a chart with left Y 0..1, right Y 5..20 and the data
// set "r" on the right axis, then opts.
func newY2Model(opts ...Option) Model {
	return New(40, 12, append([]Option{
		WithTimeRange(day0, day0.Add(10*time.Hour)),
		WithYRange(0, 1),
		WithY2Axis(linechart.Axis{Min: 5, Max: 20}),
		WithDataSetYAxis("r", linechart.YAxisRight),
	}, opts...)...)
}

func TestSetDataSetYAxis(t *testing.T) {
	m := New(40, 12)
	m.SetDataSetYAxis("r", linechart.YAxisRight)
	if _, ok := m.dSets["r"]; !ok || !m.y2Sets["r"] {
		t.Fatal("SetDataSetYAxis did not create and assign the data set")
	}
	m.SetDataSetYAxis("r", linechart.YAxisLeft)
	if m.y2Sets["r"] {
		t.Fatal("data set still on the right axis")
	}
}

func TestRightPushAutoRangesRightAxisOnly(t *testing.T) {
	m := newY2Model(WithY2Axis(linechart.Axis{Min: 5, Max: 20, AutoMax: true}))
	m.PushDataSet("r", TimePoint{Time: day0.Add(time.Hour), Value: 30})
	if a, _ := m.Y2Axis(); a.Max != 30 || m.MaxY() != 1 {
		t.Fatalf("right max %v, left max %v; want 30, 1", a.Max, m.MaxY())
	}
}

func TestRightDataSetWithoutRightAxisIsLeft(t *testing.T) {
	vals := []float64{1, 3, 2, 6, 5}
	a := newTS(WithDataSetYAxis("s", linechart.YAxisRight))
	b := newTS()
	pushTS(&a, "s", vals)
	pushTS(&b, "s", vals)
	a.DrawAll()
	b.DrawAll()
	if a.View() != b.View() {
		t.Fatalf("right-assigned set without a right axis:\n%s\n--- want ---\n%s", a.View(), b.View())
	}
	if !a.FitYToView() || a.ViewMinY() != 1 || a.ViewMaxY() != 6 {
		t.Fatalf("FitYToView view %v..%v, want 1..6", a.ViewMinY(), a.ViewMaxY())
	}
}

func TestClearY2AxisKeepsAssignment(t *testing.T) {
	m := newY2Model()
	m.ClearY2Axis()
	if _, right := m.y2Scale("r"); right || !m.y2Sets["r"] {
		t.Fatal("after ClearY2Axis: want assignment kept, drawn on the left")
	}
	m.SetY2Axis(linechart.Axis{Min: 5, Max: 20})
	if _, right := m.y2Scale("r"); !right {
		t.Fatal("assignment lost after SetY2Axis")
	}
}

func TestRightDataSetFollowsLeftZoom(t *testing.T) {
	m := newY2Model()
	m.PushDataSet("r", TimePoint{Time: day0.Add(time.Hour), Value: 12.5}) // middle of 5..20
	p, _ := m.graphPoints("r", m.dSets["r"])
	if h := float64(m.GraphHeight()); math.Abs(p[0].Y-h/2) > 1e-9 {
		t.Fatalf("unzoomed Y %v, want %v", p[0].Y, h/2)
	}
	m.SetViewYRange(0, 0.5) // right view 5..12.5
	p, _ = m.graphPoints("r", m.dSets["r"])
	if h := float64(m.GraphHeight()); math.Abs(p[0].Y-h) > 1e-9 {
		t.Fatalf("zoomed Y %v, want %v", p[0].Y, h)
	}
}

func TestFitYToViewIgnoresRightDataSets(t *testing.T) {
	m := newY2Model()
	pushTS(&m, "l", []float64{0.2, 0.4, 0.3})
	pushTS(&m, "r", []float64{6, 19, 7})
	if !m.FitYToView() || m.ViewMinY() != 0.2 || m.ViewMaxY() != 0.4 {
		t.Fatalf("view %v..%v, want 0.2..0.4", m.ViewMinY(), m.ViewMaxY())
	}
	if m.FitYToViewWithOpts(FitYOpts{DataSets: []string{"r"}}) {
		t.Fatal("only a right data set selected: want false")
	}
}

func TestLogRightAxisSkipsNonPositive(t *testing.T) {
	m := newY2Model(WithY2Axis(linechart.Axis{Min: 1, Max: 1000, Scale: linechart.ScaleLog}))
	pushTS(&m, "r", []float64{10, 0, 100})
	if _, ok := m.graphPoints("r", m.dSets["r"]); !slices.Equal(ok, []bool{true, false, true}) {
		t.Fatalf("ok = %v", ok)
	}
	m.DrawAll()
	m.DrawBrailleAll()
}

func TestTwoAxisLinesStayLeftOfRightAxis(t *testing.T) {
	m := newY2Model()
	pushTS(&m, "l", []float64{0.1, 0.9, 0.5, 1, 0.2, 0.7, 0.4, 0.8, 0.3, 0.6, 1})
	pushTS(&m, "r", []float64{20, 5, 12, 19, 6, 15, 9, 18, 7, 11, 20})
	for name, draw := range map[string]func(){"lines": m.DrawAll, "braille": m.DrawBrailleAll} {
		draw()
		x := m.GraphRight()
		for y := 0; y < m.Origin().Y; y++ {
			if r := m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune; r != runes.LineVertical {
				t.Fatalf("%s: right axis cell (%d,%d) = %q", name, x, y, r)
			}
		}
	}
}

func TestGoldenTwoAxes(t *testing.T) {
	oneDecimal := func(_ int, v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }
	m := newY2Model(WithYLabelFormatter(oneDecimal))
	pushTS(&m, "l", []float64{0.1, 0.3, 0.2, 0.6, 0.5, 0.8, 0.7, 0.9, 0.4, 0.6, 0.2})
	pushTS(&m, "r", []float64{19, 17, 18, 14, 15, 11, 12, 8, 9, 6, 5})
	m.DrawAll()
	golden(t, "twoaxes_lines", m.View())
	m.DrawBrailleAll()
	golden(t, "twoaxes_braille", m.View())
}

func TestWideCandleStaysLeftOfRightAxis(t *testing.T) {
	// 13 columns: left labels "115" + axis + 5 plot columns + axis + "100"
	m := New(13, 12,
		WithTimeRange(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC)),
		WithYRange(95, 115),
		WithY2Axis(linechart.Axis{Min: 0, Max: 100}))
	pushCandles(&m)
	if m.GraphWidth() != 5 {
		t.Fatalf("precondition: GraphWidth() = %d, want 5", m.GraphWidth())
	}
	rightOf := func() []string {
		var r []string
		for _, line := range strings.Split(m.View(), "\n") {
			r = append(r, string([]rune(line)[m.GraphRight():]))
		}
		return r
	}
	m.DrawXYAxisAndLabel()
	want := rightOf()
	s := lipgloss.NewStyle()
	m.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: 9})
	if got := rightOf(); !slices.Equal(got, want) {
		t.Fatalf("right axis overdrawn:\n%s\n--- want ---\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
