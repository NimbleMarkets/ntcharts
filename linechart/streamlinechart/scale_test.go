// ntcharts - Copyright (c) 2026 Neomantra Corp.

package streamlinechart

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

func trimView(view string) string {
	lines := strings.Split(view, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

var stream = []float64{1, 1, 10, 10, 100, 100, 0, 0, 1000, 1000, 1e4, 1e4, 30, 30, -5, 300, 300, 3, 3}

// On a log Y axis each decade takes the same height, and values that are
// not above zero break the line instead of dropping to the bottom.
func TestLogYAxis(t *testing.T) {
	m := New(30, 9, WithYScale(linechart.ScaleLog), WithYRange(1, 1e4), WithStream(stream))
	m.Draw()
	want := strings.Trim(`
     │               │ │
 1000│             ──╯ │
     │                 │  ──╮
  100│         ╭─      │    │
     │         │       │    │
     │         │       ╰─   │
   10│       ╭─╯            │
     │       │              ╰─
    1│     ──╯
`, "\n")
	if got := trimView(m.View()); got != want {
		t.Fatalf("view:\n%s\n--- want ---\n%s", got, want)
	}
	if m.YScale() != linechart.ScaleLog || m.ViewMinY() != 1 || m.ViewMaxY() != 1e4 {
		t.Fatalf("scale %v, Y range %v..%v", m.YScale(), m.ViewMinY(), m.ViewMaxY())
	}
}

func TestLogYAxisAutoRange(t *testing.T) {
	m := New(30, 9, WithYScale(linechart.ScaleLog))
	for _, v := range []float64{5, 0, -3, 5000} {
		m.Push(v)
	}
	// the default 0..1 range becomes 0.1..1, and only positive values widen it
	if m.MinY() != 0.1 || m.MaxY() != 5000 {
		t.Fatalf("auto range %v..%v", m.MinY(), m.MaxY())
	}
	m.Draw() // must not panic on the values it cannot place
}

// Data is kept in data units, so the scale can change after pushing.
func TestScaleCanChangeAfterPushing(t *testing.T) {
	// a fixed range: auto-ranging would follow the non-positive values
	// while linear, and a log axis could not keep that range
	push := func(opts ...Option) Model {
		m := New(30, 9, append(opts, WithYRange(1, 1e4))...)
		m.AutoMinY, m.AutoMaxY = false, false
		for _, v := range stream {
			m.Push(v)
		}
		return m
	}
	linear := push()
	linear.Draw()

	explicit := push(WithYScale(linechart.ScaleLinear))
	explicit.Draw()
	if explicit.View() != linear.View() {
		t.Fatalf("explicit linear scale differs:\n%s\n--- want ---\n%s", explicit.View(), linear.View())
	}

	log := push(WithYScale(linechart.ScaleLog))
	log.Draw()

	m := push()
	m.SetYScale(linechart.ScaleLog)
	m.Draw()
	if m.View() != log.View() || log.View() == linear.View() {
		t.Fatalf("log after pushing differs:\n%s\n--- want ---\n%s", m.View(), log.View())
	}
	m.SetYScale(linechart.ScaleLinear)
	m.Draw()
	if m.View() != linear.View() {
		t.Fatalf("back to linear differs:\n%s\n--- want ---\n%s", m.View(), linear.View())
	}
}
