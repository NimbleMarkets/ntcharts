// ntcharts - Copyright (c) 2026 Neomantra Corp.

package timeserieslinechart

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

func pushCandles(m *Model) {
	base := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	data := []struct{ o, h, l, c float64 }{
		{100, 108, 97, 105}, {105, 112, 103, 110}, {110, 111, 98, 99},
	}
	for i, d := range data {
		ts := base.AddDate(0, 0, i)
		m.PushDataSet("open", TimePoint{Time: ts, Value: d.o})
		m.PushDataSet("high", TimePoint{Time: ts, Value: d.h})
		m.PushDataSet("low", TimePoint{Time: ts, Value: d.l})
		m.PushDataSet("close", TimePoint{Time: ts, Value: d.c})
	}
}

func TestDrawCandleClipsTimesOutsideViewport(t *testing.T) {
	base := time.Unix(0, 0)
	for _, opts := range []DrawCandleOpts{{Width: 1}, {Width: 5}, {Width: 5, Block: true}} {
		m := New(30, 10, WithTimeRange(base, base.Add(20*time.Second)), WithYRange(0, 10))
		for _, seconds := range []int{0, 20} {
			for _, name := range []string{"o", "h", "l", "c"} {
				m.PushDataSet(name, TimePoint{Time: base.Add(time.Duration(seconds) * time.Second), Value: 5})
			}
		}
		m.SetViewTimeRange(base.Add(5*time.Second), base.Add(10*time.Second))
		m.DrawCandleWithOpts("o", "h", "l", "c", lipgloss.NewStyle(), lipgloss.NewStyle(), opts)
		for y := 0; y < m.Origin().Y; y++ {
			for x := m.Origin().X + 1; x < m.Width(); x++ {
				if r := m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune; r != 0 {
					t.Fatalf("offscreen candle at %d,%d: %q (opts %+v)", x, y, r, opts)
				}
			}
		}
		// The inclusive viewport endpoint must still draw after clipping.
		for _, name := range []string{"o", "h", "l", "c"} {
			m.PushDataSet(name, TimePoint{Time: base.Add(10 * time.Second), Value: 5})
		}
		m.DrawCandleWithOpts("o", "h", "l", "c", lipgloss.NewStyle(), lipgloss.NewStyle(), opts)
		found := false
		for y := 0; y < m.Origin().Y; y++ {
			if m.Canvas.Cell(canvas.Point{X: m.Width() - 1, Y: y}).Rune != 0 {
				found = true
			}
		}
		if !found {
			t.Fatalf("endpoint candle lost (opts %+v)", opts)
		}
	}
}

func TestDrawCandleClipsValuesToProtectXAxis(t *testing.T) {
	for _, opts := range []DrawCandleOpts{{Width: 1}, {Width: 5}, {Width: 5, Block: true}} {
		m := newCandleModel(t)
		m.SetViewYRange(100, 108)
		m.Clear()
		m.DrawXYAxisAndLabel()
		var want []rune
		for y := m.Origin().Y; y < m.Height(); y++ {
			for x := 0; x < m.Width(); x++ {
				want = append(want, m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune)
			}
		}
		m.DrawCandleWithOpts("open", "high", "low", "close", lipgloss.NewStyle(), lipgloss.NewStyle(), opts)
		i := 0
		for y := m.Origin().Y; y < m.Height(); y++ {
			for x := 0; x < m.Width(); x++ {
				if got := m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune; got != want[i] {
					t.Fatalf("candle overwrote x-axis/labels at %d,%d: %q vs %q (opts %+v)", x, y, got, want[i], opts)
				}
				i++
			}
		}
	}
}

func newCandleModel(t *testing.T) Model {
	t.Helper()
	m := New(40, 12,
		WithTimeRange(
			time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC)),
		WithYRange(95, 115))
	pushCandles(&m)
	return m
}

// DrawCandle output must be byte-identical to DrawCandleWithOpts at width 1.
func TestDrawCandleDelegatesAtWidthOne(t *testing.T) {
	s := lipgloss.NewStyle()
	a := newCandleModel(t)
	a.DrawCandle("open", "high", "low", "close", s, s)
	b := newCandleModel(t)
	b.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: 1})
	if a.View() != b.View() {
		t.Fatal("DrawCandle and DrawCandleWithOpts(Width: 1) differ")
	}
}

// Boundary: with a wide body and the earliest candle at graph column 0,
// side columns must not overdraw the y-axis (Critical 1), and the newest
// candle's center column must be clamped into the drawable area rather
// than clipped past the last canvas column (Critical 2).
func TestDrawCandleWithOptsPreservesAxisAndClampsNewest(t *testing.T) {
	s := lipgloss.NewStyle()
	m := newCandleModel(t)
	m.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: 7})
	view := m.View()
	lines := strings.Split(view, "\n")

	axisCol := m.Origin().X
	axisFound := false
	for _, line := range lines {
		runes := []rune(line)
		if axisCol < len(runes) && runes[axisCol] == '│' {
			axisFound = true
			break
		}
	}
	if !axisFound {
		t.Fatalf("y-axis column %d missing its rune — overdrawn by wide candle at graph column 0:\n%s", axisCol, view)
	}

	lastCol := m.Canvas.Width() - 1
	newestFound := false
	for _, line := range lines {
		runes := []rune(line)
		start := lastCol - 3
		if start < 0 {
			start = 0
		}
		end := lastCol + 1
		if end > len(runes) {
			end = len(runes)
		}
		for _, r := range runes[start:end] {
			switch r {
			case '┃', '╽', '╿', '│':
				newestFound = true
			}
		}
	}
	if !newestFound {
		t.Fatalf("newest candle not found within the last canvas columns (<=%d):\n%s", lastCol, view)
	}
}

// Width 3 must produce strictly more candle-body columns than width 1.
func TestDrawCandleWithOptsWidensBodies(t *testing.T) {
	s := lipgloss.NewStyle()
	countBodyCols := func(view string) int {
		cols := 0
		for _, line := range strings.Split(view, "\n") {
			cols += strings.Count(line, "┃")
		}
		return cols
	}
	a := newCandleModel(t)
	a.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: 1})
	b := newCandleModel(t)
	b.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: 3})
	if countBodyCols(b.View()) <= countBodyCols(a.View()) {
		t.Fatalf("width 3 (%d heavy runes) not wider than width 1 (%d)",
			countBodyCols(b.View()), countBodyCols(a.View()))
	}
}

// Edge insets: with wide candles spanning the full time range, the first
// candle's body must be fully present immediately right of the axis and
// the last candle's body must end exactly at the final canvas column —
// no half-clipped edge candles.
func TestDrawCandleInsetsEdgeBodies(t *testing.T) {
	s := lipgloss.NewStyle()
	m := newCandleModel(t)
	const w = 5
	m.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: w})
	view := m.View()

	minCol := m.Origin().X + 1 // axis column + 1
	lastCol := m.Canvas.Width() - 1
	rows := strings.Split(view, "\n")
	fullLeft, fullRight := false, false
	for _, row := range rows {
		r := []rune(row)
		if len(r) <= lastCol {
			continue
		}
		leftRun, rightRun := 0, 0
		for x := minCol; x < minCol+w && x < len(r); x++ {
			if strings.ContainsRune("┃╽╿█▄▀", r[x]) {
				leftRun++
			}
		}
		for x := lastCol; x > lastCol-w && x >= 0; x-- {
			if strings.ContainsRune("┃╽╿█▄▀", r[x]) {
				rightRun++
			}
		}
		if leftRun == w {
			fullLeft = true
		}
		if rightRun == w {
			fullRight = true
		}
	}
	if !fullLeft {
		t.Fatalf("first candle body not fully inset right of the axis:\n%s", view)
	}
	if !fullRight {
		t.Fatalf("last candle body not fully inset against the right edge:\n%s", view)
	}
}

// Edge insets, even width: the same inset guarantee as
// TestDrawCandleInsetsEdgeBodies must hold when the body width is even (no
// exact center column), where the left/right column split in the inset
// math (half, right := cw-1)/2, cw-1-half) is uneven.
func TestDrawCandleInsetsEdgeBodiesEvenWidth(t *testing.T) {
	s := lipgloss.NewStyle()
	m := newCandleModel(t)
	const w = 4
	m.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: w})
	view := m.View()

	minCol := m.Origin().X + 1 // axis column + 1
	rows := strings.Split(view, "\n")
	fullLeft := false
	for _, row := range rows {
		r := []rune(row)
		leftRun := 0
		for x := minCol; x < minCol+w && x < len(r); x++ {
			if strings.ContainsRune("┃╽╿█▄▀", r[x]) {
				leftRun++
			}
		}
		if leftRun == w {
			fullLeft = true
		}
	}
	if !fullLeft {
		t.Fatalf("first candle's %d-column body not fully inset right of the axis:\n%s", w, view)
	}
}

// Block style renders solid blocks; line style must not.
func TestDrawCandleBlockStyle(t *testing.T) {
	s := lipgloss.NewStyle()
	a := newCandleModel(t)
	a.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: 3})
	b := newCandleModel(t)
	b.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: 3, Block: true})
	if strings.ContainsRune(a.View(), '█') {
		t.Fatal("line style must not contain full blocks")
	}
	if !strings.ContainsRune(b.View(), '█') {
		t.Fatalf("block style missing full blocks:\n%s", b.View())
	}
}
