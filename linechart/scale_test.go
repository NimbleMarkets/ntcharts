// ntcharts - Copyright (c) 2026 Neomantra Corp.

package linechart

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"

	tea "charm.land/bubbletea/v2"
)

// trimView returns the chart's view with trailing spaces removed per line.
func trimView(m *Model) string {
	lines := strings.Split(m.View(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

func wantView(t *testing.T, m *Model, want string) {
	t.Helper()
	if got := trimView(m); got != strings.Trim(want, "\n") {
		t.Fatalf("view:\n%s\n--- want ---\n%s", got, strings.Trim(want, "\n"))
	}
}

// yTickRows returns label -> rows above the X axis for every Y label drawn.
func yTickRows(m *Model) map[string]int {
	rows := map[string]int{}
	for i, line := range strings.Split(m.View(), "\n") {
		if i > m.Origin().Y {
			break
		}
		if l := strings.TrimSpace(string([]rune(line)[:m.Origin().X])); l != "" {
			rows[l] = m.Origin().Y - i
		}
	}
	return rows
}

func xLabels(m *Model) []string {
	lines := strings.Split(m.View(), "\n")
	return strings.Fields(lines[len(lines)-1])
}

func TestScaleZeroValueIsLinear(t *testing.T) {
	var s Scale
	if s != ScaleLinear || s.String() != "linear" || ScaleLog.String() != "log" {
		t.Fatalf("zero Scale = %v", s)
	}
	m := New(20, 10, 0, 10, 0, 10)
	if m.XScale() != ScaleLinear || m.YScale() != ScaleLinear {
		t.Fatalf("default scales %v, %v", m.XScale(), m.YScale())
	}
	m = New(20, 10, 1, 10, 1, 10, WithXScale(ScaleLog), WithYScale(ScaleLog))
	if m.XScale() != ScaleLog || m.YScale() != ScaleLog {
		t.Fatalf("option scales %v, %v", m.XScale(), m.YScale())
	}
	m.SetXScale(ScaleLinear)
	m.SetYScale(ScaleLinear)
	if m.XScale() != ScaleLinear || m.YScale() != ScaleLinear {
		t.Fatalf("set scales %v, %v", m.XScale(), m.YScale())
	}
}

func TestScaleValid(t *testing.T) {
	for _, v := range []float64{-1, 0, 1, math.Inf(1), math.Inf(-1)} {
		if !ScaleLinear.Valid(v) {
			t.Errorf("ScaleLinear.Valid(%v) = false", v)
		}
	}
	for v, want := range map[float64]bool{-1: false, 0: false, 1e-300: true, 5: true, math.Inf(1): false} {
		if got := ScaleLog.Valid(v); got != want {
			t.Errorf("ScaleLog.Valid(%v) = %v", v, got)
		}
	}
	if ScaleLog.Valid(math.NaN()) {
		t.Error("ScaleLog.Valid(NaN) = true")
	}
}

// Decades are evenly spaced on a log axis, and powers of ten map exactly:
// math.Log10(1000) alone is 2.9999999999999996.
func TestLogScalePointsDecadesEvenly(t *testing.T) {
	m := New(34, 14, 1, 1000, 1, 1e6, WithXScale(ScaleLog), WithYScale(ScaleLog))
	if m.GraphWidth() != 26 || m.GraphHeight() != 12 {
		t.Fatalf("graph %dx%d", m.GraphWidth(), m.GraphHeight())
	}
	for k := 0; k <= 6; k++ {
		f := m.ScaleFloat64PointForLine(canvas.Float64Point{X: 1, Y: math.Pow10(k)})
		if f.X != 0 || f.Y != float64(2*k) {
			t.Fatalf("1e%d scales to %v, want Y %d", k, f, 2*k)
		}
	}
	if f := m.ScaleFloat64PointForLine(canvas.Float64Point{X: 1000, Y: 1}); f.X != 26 {
		t.Fatalf("X 1000 scales to %v, want 26", f.X)
	}
	// A coordinate with no place on a log axis scales to 0, never NaN.
	for _, v := range []float64{0, -3, math.NaN()} {
		f := m.ScaleFloat64Point(canvas.Float64Point{X: v, Y: v})
		if f.X != 0 || f.Y != 0 {
			t.Fatalf("%v scales to %v", v, f)
		}
	}
}

// TestLogYTicksFallOnDecades draws six decades at graph heights that do and
// do not divide evenly into them. Every label must be a power of ten on the
// row where that value falls, at least yStep rows from its neighbours.
func TestLogYTicksFallOnDecades(t *testing.T) {
	for _, tc := range []struct {
		graphHeight int
		want        []string // bottom to top
	}{
		{12, []string{"1", "10", "100", "1000", "10000", "100000", "1000000"}},
		{13, []string{"1", "10", "100", "1000", "10000", "100000", "1000000"}},
		{17, []string{"1", "10", "100", "1000", "10000", "100000", "1000000"}},
		{11, []string{"1", "100", "10000", "1000000"}},
		{8, []string{"1", "100", "10000", "1000000"}},
		{7, []string{"1", "100", "10000", "1000000"}},
		{5, []string{"1", "1000", "1000000"}},
		{3, []string{"100", "1000000"}},
		{2, []string{"1", "1000000"}},
	} {
		t.Run(fmt.Sprint(tc.graphHeight), func(t *testing.T) {
			m := New(30, tc.graphHeight+2, 0, 10, 1, 1e6, WithYScale(ScaleLog))
			if m.GraphHeight() != tc.graphHeight {
				t.Fatalf("graph height %d", m.GraphHeight())
			}
			m.DrawXYAxisAndLabel()
			rows := yTickRows(&m)
			if len(rows) != len(tc.want) {
				t.Fatalf("labels %v, want %v\n%s", rows, tc.want, m.View())
			}
			prev := -m.YStep()
			for _, label := range tc.want {
				row, ok := rows[label]
				var v float64
				fmt.Sscan(label, &v)
				want := int(math.Round(math.Log10(v) / 6 * float64(tc.graphHeight)))
				if !ok || row != want {
					t.Fatalf("label %s on row %d (drawn %v), want row %d\n%s", label, row, ok, want, m.View())
				}
				if row-prev < m.YStep() {
					t.Fatalf("label %s is %d rows above its neighbour\n%s", label, row-prev, m.View())
				}
				prev = row
			}
			if tc.graphHeight == 2 {
				return // one decade tick at most: evenly spaced fallback
			}
			if m.Origin().X != len(tc.want[len(tc.want)-1]) {
				t.Fatalf("Y margin %d does not fit the widest drawn label", m.Origin().X)
			}
		})
	}
}

// When decades are thinned, the labelled ones are counted from the top of
// the axis (then the bottom) when that end is a power of ten, so the ends
// of the range are labelled as they are on a linear axis.
func TestLogThinnedTicksLabelTheEnds(t *testing.T) {
	for _, tc := range []struct {
		name     string
		min, max float64
		height   int
		want     []string // bottom to top
	}{
		// 4 decades on 7 rows: 10, 1000, 100000 rather than 100, 10000.
		{"both ends", 10, 1e5, 9, []string{"10", "1000", "100000"}},
		// 5 decades on 7 rows: only one end can be kept, and the maximum wins.
		{"maximum first", 1, 1e5, 9, []string{"10", "1000", "100000"}},
		// The maximum is not a power of ten: count from the minimum.
		{"minimum", 10, 3e5, 9, []string{"10", "1000", "100000"}},
		// Neither end is: the most labels (100 and 10000 would be only two),
		{"neither", 3, 3e5, 9, []string{"10", "1000", "100000"}},
		// and on a tie, exponents that are multiples of the stride.
		{"neither, tie", 3, 3e4, 9, []string{"100", "10000"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := New(30, tc.height, 0, 10, tc.min, tc.max, WithYScale(ScaleLog),
				WithYLabelFormatter(func(_ int, v float64) string { return fmt.Sprint(v) }))
			m.DrawXYAxisAndLabel()
			rows := yTickRows(&m)
			var got []string
			for l := range rows {
				got = append(got, l)
			}
			slices.SortFunc(got, func(a, b string) int { return rows[a] - rows[b] })
			if !slices.Equal(got, tc.want) {
				t.Fatalf("labels %v, want %v\n%s", got, tc.want, m.View())
			}
		})
	}

	// X axis: the maximum's label is the best-effort final one, and the
	// thinned decades still count from it.
	m := New(23, 6, 10, 1e5, 0, 10, WithXScale(ScaleLog), WithXYSteps(8, 2))
	m.DrawXYAxisAndLabel()
	if got, want := xLabels(&m), []string{"10", "1000", "100000"}; !slices.Equal(got, want) {
		t.Fatalf("x labels %v, want %v\n%s", got, want, m.View())
	}
}

// The case a linear tick walk gets wrong: six decades on seven rows. Every
// second row is not a decade, so the labels move to the rows that are. Each
// step of the staircase is a line drawn at one power of ten.
func TestLogYTicksGoldenUnevenHeight(t *testing.T) {
	m := New(30, 9, 0, 7, 1, 1e6, WithYScale(ScaleLog))
	m.DrawXYAxisAndLabel()
	for k := 0; k <= 6; k++ {
		v := math.Pow10(k)
		m.DrawLine(canvas.Float64Point{X: float64(k), Y: v}, canvas.Float64Point{X: float64(k + 1), Y: v}, runes.ThinLineStyle)
	}
	wantView(t, &m, `
1000000│                  ╶──╴
       │               ╶──╴
  10000│            ╶──╴
       │        ╶───╴
       │
    100│     ╶──╴
       │  ╶──╴
      1└──────────────────────
       0 1   2 3   4   5 6  7
`)
}

func TestLogYMinorTicks(t *testing.T) {
	for _, tc := range []struct {
		name     string
		min, max float64
		height   int
		want     []string // bottom to top
	}{
		// Two decade ticks leave room for 2x and 5x.
		{"one decade", 1, 10, 10, []string{"1", "2", "5", "10"}},
		// ... but not on a plot too short to space them.
		{"one decade, short", 1, 10, 6, []string{"1", "10"}},
		// One decade tick inside the range.
		{"around a decade", 20, 500, 12, []string{"20", "50", "100", "200", "500"}},
		// Three decade ticks are enough: no minor ticks.
		{"two decades", 1, 100, 14, []string{"1", "10", "100"}},
		{"below one", 0.01, 0.1, 10, []string{"0.01", "0.02", "0.05", "0.1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := New(30, tc.height, 0, 10, tc.min, tc.max, WithYScale(ScaleLog),
				WithYLabelFormatter(func(_ int, v float64) string { return fmt.Sprint(v) }))
			m.DrawXYAxisAndLabel()
			rows := yTickRows(&m)
			var got []string
			for l := range rows {
				got = append(got, l)
			}
			slices.SortFunc(got, func(a, b string) int { return rows[a] - rows[b] })
			if !slices.Equal(got, tc.want) {
				t.Fatalf("labels %v, want %v\n%s", got, tc.want, m.View())
			}
			for _, l := range got {
				var v float64
				fmt.Sscan(l, &v)
				frac := (math.Log10(v) - math.Log10(tc.min)) / (math.Log10(tc.max) - math.Log10(tc.min))
				if want := int(math.Round(frac * float64(m.GraphHeight()))); rows[l] != want {
					t.Fatalf("label %s on row %d, want %d\n%s", l, rows[l], want, m.View())
				}
			}
		})
	}
}

// A range with fewer than two round ticks falls back to evenly spaced
// labels, like a linear axis, with values read off the log scale.
func TestLogYTicksNarrowRangeFallback(t *testing.T) {
	m := New(30, 10, 0, 10, 30, 40, WithYScale(ScaleLog),
		WithYLabelFormatter(func(_ int, v float64) string { return fmt.Sprintf("%.2f", v) }))
	m.DrawXYAxisAndLabel()
	rows := yTickRows(&m)
	if len(rows) != 5 || rows["30.00"] != 0 || rows["40.00"] != 8 {
		t.Fatalf("labels %v\n%s", rows, m.View())
	}
	mid := fmt.Sprintf("%.2f", math.Sqrt(30*40)) // halfway up a log axis
	if rows[mid] != 4 {
		t.Fatalf("no %s label at mid height: %v\n%s", mid, rows, m.View())
	}
}

func TestLogXTicks(t *testing.T) {
	m := New(44, 8, 1, 1e6, 0, 10, WithXScale(ScaleLog))
	m.DrawXYAxisAndLabel()
	wantView(t, &m, `
10│
  │
 7│
  │
 3│
  │
 0└─────────────────────────────────────────
  1      10     100    1000  10000  100000
`)

	// Room to right-align the final label once the others are drawn.
	m = New(90, 6, 1, 1e6, 0, 10, WithXScale(ScaleLog))
	m.DrawXYAxisAndLabel()
	if got, want := xLabels(&m), []string{"1", "10", "100", "1000", "10000", "100000", "1000000"}; !slices.Equal(got, want) {
		t.Fatalf("x labels %v, want %v\n%s", got, want, m.View())
	}
	if !strings.HasSuffix(m.View(), "1000000") {
		t.Fatalf("final label not right-aligned:\n%s", m.View())
	}

	// Too narrow for every decade: every second one, never a 2x or 5x.
	m = New(20, 6, 1, 1e6, 0, 10, WithXScale(ScaleLog))
	m.DrawXYAxisAndLabel()
	if got, want := xLabels(&m), []string{"1", "100", "10000"}; !slices.Equal(got, want) {
		t.Fatalf("x labels %v, want %v\n%s", got, want, m.View())
	}

	// One decade: 2x and 5x fill in, and the final label keeps its rules.
	m = New(30, 6, 1, 10, 0, 10, WithXScale(ScaleLog))
	m.DrawXYAxisAndLabel()
	if got, want := xLabels(&m), []string{"1", "2", "5", "10"}; !slices.Equal(got, want) {
		t.Fatalf("x labels %v, want %v\n%s", got, want, m.View())
	}

	// Narrow range: evenly spaced columns, under the linear axis's rules.
	m = New(30, 6, 30, 40, 0, 10, WithXScale(ScaleLog))
	m.DrawXYAxisAndLabel()
	linear := New(30, 6, 30, 40, 0, 10)
	linear.DrawXYAxisAndLabel()
	if got := xLabels(&m); got[0] != "30" || len(got) != len(xLabels(&linear)) {
		t.Fatalf("x labels %v\n%s", got, m.View())
	}

	// A step of 0 still hides the axis.
	m = New(30, 6, 1, 1000, 1, 1000, WithXScale(ScaleLog), WithYScale(ScaleLog), WithXYSteps(0, 0))
	m.DrawXYAxisAndLabel()
	if strings.TrimSpace(m.View()) != "" || m.GraphWidth() != 30 || m.GraphHeight() != 6 {
		t.Fatalf("axes drawn with zero steps:\n%s", m.View())
	}
}

// The label formatter gets the tick's value in data units.
func TestLogTickFormatterSeesDataUnits(t *testing.T) {
	var seen []float64
	m := New(30, 14, 0, 10, 1, 1e6, WithYScale(ScaleLog),
		WithYLabelFormatter(func(_ int, v float64) string {
			seen = append(seen, v)
			return "x"
		}))
	seen = nil
	m.DrawXYAxisAndLabel()
	if want := []float64{1, 10, 100, 1000, 1e4, 1e5, 1e6}; !slices.Equal(seen, want) {
		t.Fatalf("formatter saw %v, want %v", seen, want)
	}
}

func TestLogRangeSanitized(t *testing.T) {
	for _, tc := range []struct {
		min, max, wantMin, wantMax float64
	}{
		{2, 500, 2, 500},
		{0, 500, 50, 500},
		{-5, 500, 50, 500},
		{0, 0, 1, 10},
		{-8, -2, 1, 10},
		{5, -2, 5, 50},
		{math.NaN(), 500, 50, 500},
	} {
		check := func(how string, m *Model) {
			t.Helper()
			if m.MinY() != tc.wantMin || m.MaxY() != tc.wantMax || m.ViewMinY() != tc.wantMin || m.ViewMaxY() != tc.wantMax {
				t.Errorf("%s Y %v..%v: got %v..%v (view %v..%v), want %v..%v", how, tc.min, tc.max,
					m.MinY(), m.MaxY(), m.ViewMinY(), m.ViewMaxY(), tc.wantMin, tc.wantMax)
			}
			if m.MinX() != tc.wantMin || m.MaxX() != tc.wantMax || m.ViewMinX() != tc.wantMin || m.ViewMaxX() != tc.wantMax {
				t.Errorf("%s X %v..%v: got %v..%v (view %v..%v), want %v..%v", how, tc.min, tc.max,
					m.MinX(), m.MaxX(), m.ViewMinX(), m.ViewMaxX(), tc.wantMin, tc.wantMax)
			}
		}
		// when the scale is set ...
		m := New(30, 10, tc.min, tc.max, tc.min, tc.max, WithXScale(ScaleLog), WithYScale(ScaleLog))
		check("scale set", &m)
		// ... and when the range is.
		m = New(30, 10, 3, 4, 3, 4, WithXScale(ScaleLog), WithYScale(ScaleLog))
		m.SetXYRange(tc.min, tc.max, tc.min, tc.max)
		m.SetViewXYRange(tc.min, tc.max, tc.min, tc.max)
		check("range set", &m)
	}

	// Linear ranges are left alone.
	m := New(30, 10, -5, 0, -8, -2)
	m.SetYRange(-9, 0)
	if m.MinX() != -5 || m.MaxX() != 0 || m.MinY() != -9 || m.MaxY() != 0 {
		t.Fatalf("linear range changed: %v..%v, %v..%v", m.MinX(), m.MaxX(), m.MinY(), m.MaxY())
	}
}

func TestLogAutoRange(t *testing.T) {
	m := New(30, 10, 10, 100, 10, 100, WithXScale(ScaleLog), WithYScale(ScaleLog), WithAutoXYRange())
	// A point with no place on a log axis is ignored whole.
	for _, f := range []canvas.Float64Point{{X: 5000, Y: 0}, {X: -1, Y: 5000}, {X: math.NaN(), Y: 1}} {
		if m.AutoAdjustRange(f) {
			t.Fatalf("auto-range adjusted for %v", f)
		}
	}
	if m.MinX() != 10 || m.MaxX() != 100 || m.MinY() != 10 || m.MaxY() != 100 {
		t.Fatalf("range moved: %v..%v, %v..%v", m.MinX(), m.MaxX(), m.MinY(), m.MaxY())
	}
	if !m.AutoAdjustRange(canvas.Float64Point{X: 0.5, Y: 2e4}) {
		t.Fatal("auto-range did not adjust")
	}
	if m.MinX() != 0.5 || m.ViewMinX() != 0.5 || m.MaxY() != 2e4 || m.ViewMaxY() != 2e4 || m.MaxX() != 100 || m.MinY() != 10 {
		t.Fatalf("range %v..%v, %v..%v", m.MinX(), m.MaxX(), m.MinY(), m.MaxY())
	}
}

func approx(a, b float64) bool { return math.Abs(a-b) <= 1e-9*math.Abs(b) }

// Zoom and pan step in scale space: on a log axis an increment is a number
// of decades, so a zoom multiplies and a pan shifts by a constant ratio.
func TestLogZoomAndPan(t *testing.T) {
	m := New(30, 10, 1, 1e6, 1, 1e6, WithXScale(ScaleLog), WithYScale(ScaleLog))
	view := func(minX, maxX, minY, maxY float64) {
		t.Helper()
		if !approx(m.ViewMinX(), minX) || !approx(m.ViewMaxX(), maxX) || !approx(m.ViewMinY(), minY) || !approx(m.ViewMaxY(), maxY) {
			t.Fatalf("view X %v..%v Y %v..%v, want X %v..%v Y %v..%v",
				m.ViewMinX(), m.ViewMaxX(), m.ViewMinY(), m.ViewMaxY(), minX, maxX, minY, maxY)
		}
	}
	m.ZoomIn(1, 2)
	view(10, 1e5, 100, 1e4)
	if m.ViewMinX() != 10 || m.ViewMaxY() != 1e4 {
		t.Fatalf("zoom to a decade is not exact: %v, %v", m.ViewMinX(), m.ViewMaxY())
	}
	m.MoveRight(0.5)
	view(10*math.Sqrt(10), 1e5*math.Sqrt(10), 100, 1e4)
	m.MoveRight(3) // clamps at the expected maximum, keeping the 4-decade window
	view(100, 1e6, 100, 1e4)
	m.MoveLeft(1)
	view(10, 1e5, 100, 1e4)
	m.MoveLeft(9)
	view(1, 1e4, 100, 1e4)
	m.MoveUp(1)
	view(1, 1e4, 1000, 1e5)
	m.MoveUp(5)
	view(1, 1e4, 1e4, 1e6)
	m.MoveDown(1.5)
	view(1, 1e4, 1e4/math.Pow(10, 1.5), 1e6/math.Pow(10, 1.5))
	m.MoveDown(7)
	view(1, 1e4, 1, 100)
	m.ZoomOut(1, 1) // limited by the expected range
	view(1, 1e5, 1, 1000)
	m.ZoomIn(3, 3) // bounds would cross: ignored
	view(1, 1e5, 1, 1000)

	// The update handlers drive the same methods.
	m = New(30, 10, 1, 1e6, 1, 1e6, WithXScale(ScaleLog), WithYScale(ScaleLog))
	m.Focus()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	view(10, 1e5, 10, 1e5)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	view(100, 1e6, 10, 1e5)
}

func TestLinearZoomAndPanUnchanged(t *testing.T) {
	m := New(30, 10, -10, 10, 0, 100)
	m.ZoomIn(2, 10)
	m.MoveRight(1.5)
	m.MoveUp(100)
	m.MoveLeft(0.25)
	m.MoveDown(3)
	if m.ViewMinX() != -6.75 || m.ViewMaxX() != 9.25 || m.ViewMinY() != 17 || m.ViewMaxY() != 97 {
		t.Fatalf("view X %v..%v Y %v..%v", m.ViewMinX(), m.ViewMaxX(), m.ViewMinY(), m.ViewMaxY())
	}
}

// drawEverything exercises every drawing path with the given points.
func drawEverything(m *Model, a, b canvas.Float64Point) {
	m.DrawXYAxisAndLabel()
	m.DrawRune(a, 'a')
	m.DrawRune(b, 'b')
	m.DrawRuneLine(a, b, '+')
	m.DrawLine(a, b, runes.ArcLineStyle)
	m.DrawBrailleLine(a, b)
	m.DrawRuneCircle(a, 2, 'o')
	m.DrawBrailleCircle(b, 2)
}

// A point with a non-positive coordinate on a log axis is not drawn, and
// neither is a segment ending on one. Nothing lands on the plot's edge.
func TestLogSkipsNonPositivePoints(t *testing.T) {
	newChart := func() Model {
		m := New(30, 12, 1, 1000, 1, 1000, WithXScale(ScaleLog), WithYScale(ScaleLog), WithAutoXYRange())
		m.DrawXYAxisAndLabel()
		return m
	}
	blank := newChart()
	good := canvas.Float64Point{X: 10, Y: 100}
	for _, bad := range []canvas.Float64Point{
		{X: 10, Y: 0}, {X: 10, Y: -5}, {X: 0, Y: 100}, {X: -2, Y: -2},
		{X: 10, Y: math.NaN()}, {X: math.Inf(1), Y: 100},
	} {
		m := newChart()
		m.DrawRune(bad, 'x')
		m.DrawRuneLine(good, bad, '+')
		m.DrawRuneLine(bad, good, '+')
		m.DrawLine(good, bad, runes.ArcLineStyle)
		m.DrawLine(bad, good, runes.ArcLineStyle)
		m.DrawBrailleLine(good, bad)
		m.DrawBrailleLine(bad, good)
		if m.View() != blank.View() {
			t.Fatalf("%v was drawn:\n%s", bad, m.View())
		}
		if m.MinX() != 1 || m.MaxX() != 1000 || m.MinY() != 1 || m.MaxY() != 1000 {
			t.Fatalf("%v moved the range", bad)
		}
	}

	// Circles are traced in data units; the part at or below zero is left out.
	m := newChart()
	m.DrawRuneCircle(canvas.Float64Point{X: 3, Y: 3}, 5, 'o')
	m.DrawBrailleCircle(canvas.Float64Point{X: 3, Y: 3}, 5)
	if m.View() == blank.View() {
		t.Fatal("circle not drawn at all")
	}
	for i, line := range strings.Split(m.View(), "\n") {
		if strings.ContainsRune(line, 'o') && i > m.Origin().Y {
			t.Fatalf("circle drawn below the axis:\n%s", m.View())
		}
	}
}

func TestLogDrawingPathsGolden(t *testing.T) {
	m := New(30, 12, 1, 1000, 1, 1000, WithXScale(ScaleLog), WithYScale(ScaleLog))
	m.DrawXYAxisAndLabel()
	// y = x is a straight diagonal on log-log axes. (DrawRune keeps points
	// off the axes, so 1 sits one row above the X axis, as on a linear chart.)
	for _, v := range []float64{1, 3, 10, 30, 100, 300, 1000} {
		m.DrawRune(canvas.Float64Point{X: v, Y: v}, '*')
	}
	wantView(t, &m, `
1000│                        *
    │
    │                    *
 100│                *
    │
    │            *
    │        *
  10│
    │    *
    │*
   1└─────────────────────────
    1       10       100  1000
`)

	m = New(30, 12, 1, 1000, 1, 1000, WithXScale(ScaleLog), WithYScale(ScaleLog))
	m.DrawXYAxisAndLabel()
	m.DrawRuneLine(canvas.Float64Point{X: 1, Y: 1000}, canvas.Float64Point{X: 1000, Y: 1}, '+')
	m.DrawBrailleLine(canvas.Float64Point{X: 1, Y: 1}, canvas.Float64Point{X: 1000, Y: 1000})
	wantView(t, &m, `
1000│++                    ⢀⠤⠊
    │  +++               ⡠⠔⠁
    │     ++          ⢀⠤⠊
 100│       +++     ⡠⠔⠁
    │          ++⢀⠤⠊
    │          ⡠⠒⠁++
    │       ⢀⠔⠊     +++
  10│     ⡠⠒⠁          +++
    │  ⢀⠔⠊                ++
    │⡠⠒⠁                    ++
   1└─────────────────────────
    1       10       100  1000
`)
}

// Setting ScaleLinear explicitly is the same chart as setting no scale.
func TestExplicitLinearScaleIsIdentical(t *testing.T) {
	a := canvas.Float64Point{X: -3, Y: 2}
	b := canvas.Float64Point{X: 14, Y: 61}
	plain := New(40, 16, -5, 10, 0, 50, WithAutoXYRange())
	drawEverything(&plain, a, b)

	opt := New(40, 16, -5, 10, 0, 50, WithAutoXYRange(), WithXScale(ScaleLinear), WithYScale(ScaleLinear))
	drawEverything(&opt, a, b)

	set := New(40, 16, 1, 10, 1, 50, WithAutoXYRange(), WithXScale(ScaleLog), WithYScale(ScaleLog))
	set.SetXScale(ScaleLinear)
	set.SetYScale(ScaleLinear)
	set.SetXYRange(-5, 10, 0, 50)
	set.SetViewXYRange(-5, 10, 0, 50)
	drawEverything(&set, a, b)

	if opt.View() != plain.View() || set.View() != plain.View() {
		t.Fatalf("explicit linear differs:\n%s\n--- plain ---\n%s", opt.View(), plain.View())
	}
	for _, m := range []*Model{&plain, &opt, &set} {
		m.ZoomIn(1, 5)
		m.MoveLeft(2)
	}
	if opt.ViewMinX() != plain.ViewMinX() || opt.ViewMaxY() != plain.ViewMaxY() {
		t.Fatal("explicit linear zooms differently")
	}
}
