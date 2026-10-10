// ntcharts - Copyright (c) 2026 Neomantra Corp.

package linechart

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
)

func TestYAxisReflectsLeftAxis(t *testing.T) {
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	m := New(30, 12, 0, 10, -5, 5, WithAutoYRange(), WithStyles(defaultStyle, label, defaultStyle))
	m.SetViewYRange(-2, 3)
	a := m.YAxis()
	if a.Min != -5 || a.Max != 5 || a.ViewMin != -2 || a.ViewMax != 3 ||
		a.Scale != ScaleLinear || !a.AutoMin || !a.AutoMax {
		t.Fatalf("YAxis() = %+v", a)
	}
	if got := a.LabelFormatter(0, 2); got != "2" {
		t.Fatalf("LabelFormatter(0, 2) = %q", got)
	}
	if fmt.Sprint(a.LabelStyle.GetForeground()) != fmt.Sprint(label.GetForeground()) {
		t.Fatal("LabelStyle is not the chart's LabelStyle")
	}
}

func TestSetYAxisRoundTrip(t *testing.T) {
	m := New(30, 12, 0, 10, 1, 1000, WithXYSteps(2, 2), WithYScale(ScaleLog))
	m.SetViewYRange(10, 100)
	m.DrawXYAxisAndLabel()
	before, width := m.View(), m.GraphWidth()
	m.SetYAxis(m.YAxis())
	if m.ViewMinY() != 10 || m.ViewMaxY() != 100 || m.YScale() != ScaleLog || m.GraphWidth() != width {
		t.Fatalf("view %v..%v scale %v width %d", m.ViewMinY(), m.ViewMaxY(), m.YScale(), m.GraphWidth())
	}
	m.Clear()
	m.DrawXYAxisAndLabel()
	if got := m.View(); got != before {
		t.Fatalf("render changed:\n%s\n--- want ---\n%s", got, before)
	}
}

func TestSetYAxis(t *testing.T) {
	m := New(30, 12, 0, 10, 0, 1)
	m.SetYAxis(Axis{Min: 0, Max: 100, ViewMin: 20, ViewMax: 40, AutoMax: true})
	if m.MinY() != 0 || m.MaxY() != 100 || m.ViewMinY() != 20 || m.ViewMaxY() != 40 || m.AutoMinY || !m.AutoMaxY {
		t.Fatalf("range %v..%v view %v..%v auto %v %v",
			m.MinY(), m.MaxY(), m.ViewMinY(), m.ViewMaxY(), m.AutoMinY, m.AutoMaxY)
	}
	if m.YLabelFormatter == nil || m.YLabelFormatter(0, -0.3) != "0" {
		t.Fatal("nil LabelFormatter should become DefaultLabelFormatter")
	}
	// ViewMin == ViewMax means the whole range
	m.SetYAxis(Axis{Min: 0, Max: 50})
	if m.ViewMinY() != 0 || m.ViewMaxY() != 50 {
		t.Fatalf("view %v..%v, want 0..50", m.ViewMinY(), m.ViewMaxY())
	}
}

func TestSetY2AxisSanitizes(t *testing.T) {
	for _, c := range []struct {
		name     string
		in       Axis
		min, max float64
	}{
		{"empty range", Axis{Min: 5, Max: 5}, 0, 1},
		{"reversed", Axis{Min: 9, Max: 1}, 0, 1},
		{"nan", Axis{Min: math.NaN(), Max: 1}, 0, 1},
		{"log bad min", Axis{Scale: ScaleLog, Min: -1, Max: 100}, 10, 100},
		{"log bad both", Axis{Scale: ScaleLog}, 1, 10},
		{"log reversed", Axis{Scale: ScaleLog, Min: 100, Max: 10}, 1, 10},
		{"view ignored", Axis{Min: 0, Max: 100, ViewMin: 10, ViewMax: 20}, 0, 100},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := New(30, 12, 0, 10, 0, 10)
			m.SetY2Axis(c.in)
			a, ok := m.Y2Axis()
			if !ok || a.Min != c.min || a.Max != c.max || a.ViewMin != c.min || a.ViewMax != c.max {
				t.Fatalf("Y2Axis() = %v..%v view %v..%v ok %v, want %v..%v",
					a.Min, a.Max, a.ViewMin, a.ViewMax, ok, c.min, c.max)
			}
		})
	}
}

func TestY2AxisDefaultsAndClear(t *testing.T) {
	m := New(30, 12, 0, 10, 0, 10)
	if _, ok := m.Y2Axis(); ok {
		t.Fatal("new chart has a right axis")
	}
	m.DrawXYAxisAndLabel()
	want, width := m.View(), m.GraphWidth()

	m.SetY2Axis(Axis{Max: 100})
	a, ok := m.Y2Axis()
	if !ok || a.LabelFormatter == nil || a.LabelFormatter(0, -0.2) != "0" {
		t.Fatal("nil LabelFormatter should become DefaultLabelFormatter")
	}
	m.ClearY2Axis()
	if _, ok := m.Y2Axis(); ok {
		t.Fatal("ClearY2Axis left a right axis")
	}
	m.Clear()
	m.DrawXYAxisAndLabel()
	if m.View() != want || m.GraphWidth() != width {
		t.Fatalf("after ClearY2Axis:\n%s\n--- want ---\n%s", m.View(), want)
	}
}

func TestWithY2Axis(t *testing.T) {
	m := New(30, 12, 0, 10, 0, 10, WithY2Axis(Axis{Min: 5, Max: 20}))
	if a, ok := m.Y2Axis(); !ok || a.Min != 5 || a.Max != 20 {
		t.Fatalf("WithY2Axis: %+v %v", a, ok)
	}
}

func TestY2ViewFollowsLeftView(t *testing.T) {
	for _, c := range []struct {
		name             string
		left, right      Axis
		wantMin, wantMax float64
	}{
		{"unzoomed", Axis{Min: 0, Max: 10, ViewMin: 0, ViewMax: 10}, Axis{Min: 0, Max: 100}, 0, 100},
		{"linear", Axis{Min: 0, Max: 10, ViewMin: 2, ViewMax: 6}, Axis{Min: 0, Max: 100}, 20, 60},
		{"log right", Axis{Min: 0, Max: 10, ViewMin: 5, ViewMax: 10}, Axis{Min: 1, Max: 1e4, Scale: ScaleLog}, 100, 1e4},
		{"log left", Axis{Min: 1, Max: 1000, ViewMin: 10, ViewMax: 100, Scale: ScaleLog}, Axis{Min: 0, Max: 300}, 100, 200},
		{"both log", Axis{Min: 1, Max: 100, ViewMin: 10, ViewMax: 100, Scale: ScaleLog}, Axis{Min: 10, Max: 1e5, Scale: ScaleLog}, 1000, 1e5},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := New(30, 12, 0, 10, 0, 1)
			m.SetYAxis(c.left)
			m.SetY2Axis(c.right)
			a, _ := m.Y2Axis()
			if !approx(a.ViewMin, c.wantMin) || !approx(a.ViewMax, c.wantMax) {
				t.Fatalf("right view %v..%v, want %v..%v", a.ViewMin, a.ViewMax, c.wantMin, c.wantMax)
			}
		})
	}
}

func TestY2ViewFollowsZoomAndMove(t *testing.T) {
	m := New(30, 12, 0, 10, 0, 10, WithY2Axis(Axis{Min: 0, Max: 100}))
	check := func(step string, lo, hi float64) {
		t.Helper()
		a, _ := m.Y2Axis()
		if !approx(a.ViewMin, lo) || !approx(a.ViewMax, hi) {
			t.Fatalf("%s: right view %v..%v, want %v..%v", step, a.ViewMin, a.ViewMax, lo, hi)
		}
	}
	m.ZoomIn(0, 2) // left 2..8
	check("ZoomIn", 20, 80)
	m.MoveUp(1) // left 3..9
	check("MoveUp", 30, 90)
	m.MoveDown(1)
	m.ZoomOut(0, 2) // left 0..10
	for range 50 {
		m.ZoomIn(0, 1)
		m.ZoomOut(0, 1)
	}
	if a, _ := m.Y2Axis(); a.ViewMin != 0 || a.ViewMax != 100 {
		t.Fatalf("right view drifted to %v..%v", a.ViewMin, a.ViewMax)
	}
}

func TestY2ToY(t *testing.T) {
	m := New(30, 12, 0, 10, 0, 10)
	if got := m.Y2ToY(42); got != 42 {
		t.Fatalf("without a right axis Y2ToY(42) = %v", got)
	}
	check := func(want map[float64]float64) {
		t.Helper()
		for v, w := range want {
			if got := m.Y2ToY(v); !approx(got, w) {
				t.Fatalf("Y2ToY(%v) = %v, want %v", v, got, w)
			}
		}
	}
	m.SetY2Axis(Axis{Min: 0, Max: 100})
	check(map[float64]float64{0: 0, 50: 5, 100: 10, 150: 15})
	m.SetViewYRange(2, 6) // right view 20..60
	check(map[float64]float64{20: 2, 40: 4, 60: 6})

	m = New(30, 12, 0, 10, 0, 30, WithY2Axis(Axis{Min: 1, Max: 1000, Scale: ScaleLog}))
	check(map[float64]float64{1: 0, 10: 10, 100: 20, 1000: 30})
	if got := m.Y2ToY(0); !math.IsNaN(got) {
		t.Fatalf("Y2ToY(0) on a log right axis = %v, want NaN", got)
	}

	m = New(30, 12, 0, 10, 1, 1000, WithYScale(ScaleLog), WithY2Axis(Axis{Min: 0, Max: 3}))
	check(map[float64]float64{0: 1, 1: 10, 2: 100, 3: 1000})
}

// rows returns the view's lines with trailing spaces removed.
func rows(m *Model) []string { return strings.Split(trimView(m), "\n") }

// rightOf returns each line of the view from column x on.
func rightOf(m *Model, x int) []string {
	var r []string
	for _, line := range strings.Split(m.View(), "\n") {
		r = append(r, string([]rune(line)[x:]))
	}
	return r
}

// y2TickRows returns label -> rows above the X axis for every right Y label.
func y2TickRows(m *Model) map[string]int {
	got := map[string]int{}
	for i, line := range strings.Split(m.View(), "\n") {
		if i > m.Origin().Y {
			break
		}
		if l := strings.TrimSpace(string([]rune(line)[m.GraphRight()+1:])); l != "" {
			got[l] = m.Origin().Y - i
		}
	}
	return got
}

func TestY2AxisLayout(t *testing.T) {
	m := New(20, 7, 0, 10, 0, 5, WithXYSteps(2, 2))
	if m.GraphWidth() != 18 || m.GraphRight() != 20 {
		t.Fatalf("precondition: width %d right %d", m.GraphWidth(), m.GraphRight())
	}
	m.SetY2Axis(Axis{Min: 0, Max: 100})
	if m.GraphWidth() != 14 { // 18 - 1 - len("100")
		t.Fatalf("GraphWidth() = %d, want 14", m.GraphWidth())
	}
	if m.Origin() != (canvas.Point{X: 1, Y: 5}) || m.GraphRight() != 16 {
		t.Fatalf("origin %v right %d", m.Origin(), m.GraphRight())
	}
	m.DrawXYAxisAndLabel()
	want := []string{
		"5│              │100",
		"4│              │80",
		" │              │",
		"2│              │40",
		" │              │",
		"0└──────────────┘0",
	}
	if got := rows(&m)[:6]; !slices.Equal(got, want) {
		t.Fatalf("view:\n%s\n--- want ---\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestY2AxisWithoutXAxis(t *testing.T) {
	m := New(20, 5, 0, 10, 0, 5, WithXYSteps(0, 2), WithY2Axis(Axis{Min: 0, Max: 100}))
	m.DrawXYAxisAndLabel()
	want := []string{
		"4│              │80",
		" │              │",
		"2│              │40",
		" │              │",
		"0│              │0",
	}
	if got := rows(&m); !slices.Equal(got, want) {
		t.Fatalf("view:\n%s\n--- want ---\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestY2LogAxisLabelsOwnRows(t *testing.T) {
	m := New(30, 9, 0, 10, 0, 7, WithXYSteps(2, 2), WithY2Axis(Axis{Min: 1, Max: 1000, Scale: ScaleLog}))
	m.DrawXYAxisAndLabel()
	want := map[string]int{"1": 0, "10": 2, "100": 5, "1000": 7}
	if got := y2TickRows(&m); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("right labels %v, want %v", got, want)
	}
}

func TestY2LabelStyle(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	m := New(20, 7, 0, 10, 0, 5, WithXYSteps(2, 2), WithY2Axis(Axis{Min: 0, Max: 100, LabelStyle: red}))
	m.DrawXYAxisAndLabel()
	right := m.Canvas.Cell(canvas.Point{X: 17, Y: 0}) // "1" of "100"
	left := m.Canvas.Cell(canvas.Point{X: 0, Y: 0})   // "5"
	if right.Rune != '1' || fmt.Sprint(right.Style.GetForeground()) != fmt.Sprint(red.GetForeground()) {
		t.Fatalf("right label %q not in the right LabelStyle", right.Rune)
	}
	if left.Rune != '5' || fmt.Sprint(left.Style.GetForeground()) == fmt.Sprint(red.GetForeground()) {
		t.Fatalf("left label %q took the right LabelStyle", left.Rune)
	}
}

func TestY2AxisHiddenWithoutYStep(t *testing.T) {
	m := New(20, 7, 0, 10, 0, 5, WithXYSteps(2, 0))
	m.DrawXYAxisAndLabel()
	want, width := m.View(), m.GraphWidth()
	m.SetY2Axis(Axis{Min: 0, Max: 100})
	m.Clear()
	m.DrawXYAxisAndLabel()
	if m.View() != want || m.GraphWidth() != width || m.GraphRight() != m.Width() {
		t.Fatalf("yStep 0 with a right axis:\n%s\n--- want ---\n%s", m.View(), want)
	}
}

func TestY2AxisNarrowCanvas(t *testing.T) {
	for w := 1; w <= 8; w++ {
		m := New(w, 7, 0, 10, 0, 5, WithXYSteps(2, 2), WithY2Axis(Axis{Min: 0, Max: 100}))
		if m.GraphWidth() < 0 {
			t.Fatalf("width %d: GraphWidth() = %d", w, m.GraphWidth())
		}
		m.DrawXYAxisAndLabel()
		m.DrawLine(canvas.Float64Point{X: 0, Y: 0}, canvas.Float64Point{X: 10, Y: 5}, runes.ArcLineStyle)
		m.DrawRune(canvas.Float64Point{X: 5, Y: 2}, 'x')
		_ = m.View()
	}
}

func TestDrawStaysLeftOfY2Axis(t *testing.T) {
	m := New(20, 7, 0, 10, 0, 5, WithXYSteps(2, 2), WithY2Axis(Axis{Min: 0, Max: 100}))
	m.DrawXYAxisAndLabel()
	want := rightOf(&m, m.GraphRight())
	m.DrawRune(canvas.Float64Point{X: 12, Y: 3}, 'x')
	m.DrawRuneLine(canvas.Float64Point{X: 5, Y: 1}, canvas.Float64Point{X: 20, Y: 4}, '*')
	m.DrawLine(canvas.Float64Point{X: 5, Y: 1}, canvas.Float64Point{X: 20, Y: 4}, runes.ArcLineStyle)
	m.DrawRuneCircle(canvas.Float64Point{X: 10, Y: 2}, 3, 'o')
	if got := rightOf(&m, m.GraphRight()); !slices.Equal(got, want) {
		t.Fatalf("right axis overdrawn:\n%s\n--- want ---\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestAutoAdjustY2Range(t *testing.T) {
	m := New(30, 12, 0, 10, 0, 1)
	if m.AutoAdjustY2Range(canvas.Float64Point{X: 5, Y: 50}) {
		t.Fatal("no right axis: want false")
	}
	m.SetY2Axis(Axis{Min: 0, Max: 10, AutoMax: true})
	width := m.GraphWidth()
	if !m.AutoAdjustY2Range(canvas.Float64Point{X: 5, Y: 500}) {
		t.Fatal("widened right axis: want true")
	}
	a, _ := m.Y2Axis()
	if a.Min != 0 || a.Max != 500 || a.ViewMax != 500 || m.MinY() != 0 || m.MaxY() != 1 {
		t.Fatalf("right %v..%v view max %v, left %v..%v", a.Min, a.Max, a.ViewMax, m.MinY(), m.MaxY())
	}
	if m.GraphWidth() >= width {
		t.Fatalf("layout not recomputed for wider labels: width %d, was %d", m.GraphWidth(), width)
	}
	if m.AutoAdjustY2Range(canvas.Float64Point{X: 5, Y: -1}) {
		t.Fatal("AutoMin is off: want false")
	}
	m.AutoMaxX = true
	if !m.AutoAdjustY2Range(canvas.Float64Point{X: 20, Y: 1}) || m.MaxX() != 20 {
		t.Fatalf("X not widened: max %v", m.MaxX())
	}
	m.SetY2Axis(Axis{Min: 1, Max: 10, Scale: ScaleLog, AutoMin: true})
	if m.AutoAdjustY2Range(canvas.Float64Point{X: 5, Y: 0}) {
		t.Fatal("0 has no place on a log right axis: want false")
	}
}

func TestY2AxisCopiesAreIndependent(t *testing.T) {
	m := New(30, 12, 0, 10, 0, 1, WithY2Axis(Axis{Min: 0, Max: 10, AutoMax: true}))
	c := m // as Bubble Tea copies a model
	c.AutoAdjustY2Range(canvas.Float64Point{X: 5, Y: 50})
	if a, _ := m.Y2Axis(); a.Max != 10 {
		t.Fatalf("auto-ranging a copy changed the original: max %v", a.Max)
	}
}

func TestAutoAdjustRangeReportsRightViewChange(t *testing.T) {
	m := New(30, 12, 0, 10, 0, 10, WithAutoYRange())
	m.SetViewYRange(0, 5)
	if m.AutoAdjustRange(canvas.Float64Point{X: 1, Y: 20}) {
		t.Fatal("one axis, zoomed in: want false as before")
	}
	m = New(30, 12, 0, 10, 0, 10, WithAutoYRange(), WithY2Axis(Axis{Min: 0, Max: 100}))
	m.SetViewYRange(0, 5)
	before, _ := m.Y2Axis()
	if !m.AutoAdjustRange(canvas.Float64Point{X: 1, Y: 20}) {
		t.Fatal("left range widened under a right axis: want true")
	}
	if after, _ := m.Y2Axis(); after.ViewMax == before.ViewMax {
		t.Fatalf("right view did not move: %v", after.ViewMax)
	}
}

func TestSetYRangeRecomputesRightAxisLayout(t *testing.T) {
	m := New(30, 10, 0, 10, 0, 1, WithXYSteps(2, 2), WithY2Axis(Axis{Min: 0, Max: 1000}))
	m.SetViewYRange(0, 0.01) // right view 0..10: narrow labels
	m.SetYRange(0, 0.01)     // left view is now the whole range: right view 0..1000 again
	m.DrawXYAxisAndLabel()
	if got := y2TickRows(&m); got["1000"] != m.GraphHeight() {
		t.Fatalf("right labels %v, want 1000 on row %d", got, m.GraphHeight())
	}
}
