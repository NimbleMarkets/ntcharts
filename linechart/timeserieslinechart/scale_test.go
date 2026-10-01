// ntcharts - Copyright (c) 2026 Neomantra Corp.

package timeserieslinechart

import (
	"strings"
	"testing"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	"charm.land/lipgloss/v2"
)

func trimView(view string) string {
	lines := strings.Split(view, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

func wantView(t *testing.T, got, want string) {
	t.Helper()
	if got, want = trimView(got), strings.Trim(want, "\n"); got != want {
		t.Fatalf("view:\n%s\n--- want ---\n%s", got, want)
	}
}

var day0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// logSeries is ten days growing from 3 to a million, with a zero on day 3
// and a negative value on day 7.
func logSeries(opts ...Option) Model {
	opts = append(opts, WithTimeRange(day0, day0.Add(9*24*time.Hour)), WithYRange(1, 1e6))
	m := New(44, 11, opts...)
	// a fixed range: auto-ranging would follow the non-positive values
	// while linear, and a log axis could not keep that range
	m.AutoMinY, m.AutoMaxY = false, false
	for i, v := range []float64{3, 20, 150, 0, 900, 4000, 30000, -1, 200000, 1e6} {
		m.Push(TimePoint{Time: day0.Add(time.Duration(i) * 24 * time.Hour), Value: v})
	}
	return m
}

// Six decades on nine rows: the labelled rows are powers of ten, and the
// line breaks around the two values that have no place on a log axis.
func TestLogYAxisBraille(t *testing.T) {
	m := logSeries(WithYScale(linechart.ScaleLog))
	m.DrawBraille()
	wantView(t, m.View(), `
1000000│                                ⢀⡠⠔⠊
       │                               ⠈⠁
       │                      ⡠⠔
  10000│                  ⢀⡠⠒⠉
       │                ⠔⠊⠁
       │       ⣀⠄
    100│    ⡠⠔⠊
       │⢀⡠⠔⠉
       │⠁
      1└────────────────────────────────────
       '26 01/01   01/04   01/06   01/08
`)
	if m.YScale() != linechart.ScaleLog || m.ViewMinY() != 1 || m.ViewMaxY() != 1e6 {
		t.Fatalf("scale %v, Y range %v..%v", m.YScale(), m.ViewMinY(), m.ViewMaxY())
	}
}

func TestLogYAxisLineRunes(t *testing.T) {
	m := logSeries(WithYScale(linechart.ScaleLog))
	m.Draw()
	wantView(t, m.View(), `
1000000│                                  ╭─
       │                               ───╯
       │                       ╭
  10000│                     ╭─╯
       │                  ╭──╯
       │               ───╯
    100│      ╭─
       │  ╭───╯
       ├──╯
      1└────────────────────────────────────
       '26 01/01   01/04   01/06   01/08
`)
}

// Candles are placed by the log of their values; the third has a low of
// zero and is left out.
func TestLogYAxisCandles(t *testing.T) {
	m := New(44, 11, WithYScale(linechart.ScaleLog), WithTimeRange(day0, day0.Add(5*24*time.Hour)), WithYRange(1, 1e4))
	for i, c := range [][4]float64{{2, 30, 1.5, 20}, {20, 400, 10, 300}, {300, 500, 0, 40}, {40, 9000, 30, 5000}, {5000, 8000, 800, 1000}} {
		ts := day0.Add(time.Duration(i+1) * 20 * time.Hour)
		m.PushDataSet("open", TimePoint{Time: ts, Value: c[0]})
		m.PushDataSet("high", TimePoint{Time: ts, Value: c[1]})
		m.PushDataSet("low", TimePoint{Time: ts, Value: c[2]})
		m.PushDataSet("close", TimePoint{Time: ts, Value: c[3]})
	}
	m.DrawCandleWithOpts("open", "high", "low", "close", lipgloss.NewStyle(), lipgloss.NewStyle(), DrawCandleOpts{Width: 3})
	wantView(t, m.View(), `
10000│                        ╻╽╻   ╻╽╻
     │                        ┃┃┃   ┃┃┃
 1000│                        ┃┃┃   ╹╹╹
     │           ┃┃┃          ┃┃┃
  100│           ┃┃┃          ┃┃┃
     │      ╷    ┃┃┃          ╹╿╹
     │     ┃┃┃   ╹╿╹
   10│     ┃┃┃
     │     ╹╿╹
    1└──────────────────────────────────────
     '26 01/01   01/02   01/03   01/04 01/06
`)
}

func TestLogYAxisAutoRange(t *testing.T) {
	m := New(44, 11, WithYScale(linechart.ScaleLog))
	for i, v := range []float64{5, 0, -3, 5000} {
		m.Push(TimePoint{Time: day0.Add(time.Duration(i) * time.Hour), Value: v})
	}
	// the default 0..1 range becomes 0.1..1, and only positive values widen it
	if m.MinY() != 0.1 || m.MaxY() != 5000 {
		t.Fatalf("auto range %v..%v", m.MinY(), m.MaxY())
	}
	m.DrawBraille() // must not panic on the values it cannot place
	m.Draw()
}

// Data is kept in data units, so the scale can change after pushing.
func TestScaleCanChangeAfterPushing(t *testing.T) {
	for name, draw := range map[string]func(*Model){
		"braille": (*Model).DrawBraille,
		"runes":   (*Model).Draw,
	} {
		linear := logSeries()
		draw(&linear)

		explicit := logSeries(WithYScale(linechart.ScaleLinear))
		draw(&explicit)
		if explicit.View() != linear.View() {
			t.Fatalf("%s: explicit linear scale differs:\n%s\n--- want ---\n%s", name, explicit.View(), linear.View())
		}

		log := logSeries(WithYScale(linechart.ScaleLog))
		draw(&log)

		m := logSeries()
		m.SetYScale(linechart.ScaleLog)
		draw(&m)
		if m.View() != log.View() || log.View() == linear.View() {
			t.Fatalf("%s: log after pushing differs:\n%s\n--- want ---\n%s", name, m.View(), log.View())
		}
		m.SetYScale(linechart.ScaleLinear)
		draw(&m)
		if m.View() != linear.View() {
			t.Fatalf("%s: back to linear differs:\n%s\n--- want ---\n%s", name, m.View(), linear.View())
		}
	}
}
