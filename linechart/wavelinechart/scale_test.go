// ntcharts - Copyright (c) 2026 Neomantra Corp.

package wavelinechart

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
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

// On a log Y axis the wave rests on the bottom of the view range (there is
// no zero), and points that are not above zero leave their column there.
func TestLogYAxis(t *testing.T) {
	m := New(34, 11, WithYScale(linechart.ScaleLog), WithXRange(0, 6), WithYRange(0.01, 1000))
	for i, v := range []float64{0.02, 0.5, 0, 30, -4, 1000} {
		m.Plot(canvas.Float64Point{X: float64(i + 1), Y: v})
	}
	m.Draw()
	wantView(t, m.View(), `
1000│                            ╭
    │                            │
    │                            │
    │                  ╭╮        │
  10│                  ││        │
    │                  ││        │
    │         ╭╮       ││        │
   0│         ││       ││        │
    │    ╭╮   ││       ││        │
    └────┴┴───┴┴───────┴┴────────┴
    0   1   2     3   4   5     6
`)
	if m.YScale() != linechart.ScaleLog || m.XScale() != linechart.ScaleLinear {
		t.Fatalf("scales %v, %v", m.XScale(), m.YScale())
	}
	if m.ViewMinY() != 0.01 || m.ViewMaxY() != 1000 {
		t.Fatalf("Y range %v..%v is not in data units", m.ViewMinY(), m.ViewMaxY())
	}
}

func TestLogXYAxesAutoRange(t *testing.T) {
	m := New(34, 11, WithXScale(linechart.ScaleLog), WithYScale(linechart.ScaleLog))
	// the default 0..1 ranges cannot start at zero on a log axis
	if m.MinX() != 0.1 || m.MaxX() != 1 || m.MinY() != 0.1 || m.MaxY() != 1 {
		t.Fatalf("default ranges %v..%v, %v..%v", m.MinX(), m.MaxX(), m.MinY(), m.MaxY())
	}
	for _, p := range []canvas.Float64Point{
		{X: 1, Y: 1}, {X: 10, Y: 100}, {X: 100, Y: 10}, {X: 1000, Y: 5000},
		{X: 0, Y: 9e9}, {X: 9e9, Y: -1}, // no place on the axes: ignored
	} {
		m.Plot(p)
	}
	m.Draw()
	if m.MinX() != 0.1 || m.MaxX() != 1000 || m.MinY() != 0.1 || m.MaxY() != 5000 {
		t.Fatalf("auto ranges %v..%v, %v..%v", m.MinX(), m.MaxX(), m.MinY(), m.MaxY())
	}
	wantView(t, m.View(), `
    │                            ╭
1000│                            │
    │                            │
 100│              ╭╮            │
    │              ││            │
  10│              ││     ╭╮     │
    │              ││     ││     │
   1│      ╭╮      ││     ││     │
    │      ││      ││     ││     │
   0└──────┴┴──────┴┴─────┴┴─────┴
    0      1       10     100 1000
`)
}

// Data is kept in data units, so the scale can change after plotting.
func TestScaleCanChangeAfterPlotting(t *testing.T) {
	plot := func(opts ...Option) Model {
		m := New(34, 11, append(opts, WithXRange(0, 6), WithYRange(0.01, 1000))...)
		// a fixed range: auto-ranging would follow the non-positive values
		// while linear, and a log axis could not keep that range
		m.AutoMinY, m.AutoMaxY = false, false
		for i, v := range []float64{0.02, 0.5, 0, 30, -4, 1000} {
			m.Plot(canvas.Float64Point{X: float64(i + 1), Y: v})
		}
		return m
	}
	linear := plot()
	linear.Draw()

	explicit := plot(WithXScale(linechart.ScaleLinear), WithYScale(linechart.ScaleLinear))
	explicit.Draw()
	if explicit.View() != linear.View() {
		t.Fatalf("explicit linear scale differs:\n%s\n--- want ---\n%s", explicit.View(), linear.View())
	}

	log := plot(WithYScale(linechart.ScaleLog))
	log.Draw()

	m := plot()
	m.SetYScale(linechart.ScaleLog)
	m.Draw()
	if m.View() != log.View() || log.View() == linear.View() {
		t.Fatalf("log after plotting differs:\n%s\n--- want ---\n%s", m.View(), log.View())
	}
	m.SetYScale(linechart.ScaleLinear)
	m.Draw()
	if m.View() != linear.View() {
		t.Fatalf("back to linear differs:\n%s\n--- want ---\n%s", m.View(), linear.View())
	}
}
