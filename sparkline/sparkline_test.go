// ntcharts - Copyright (c) 2024 Neomantra Corp.

package sparkline

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
)

func TestNew(t *testing.T) {
	w := 30
	h := 15
	max := 100.0
	scale := float64(h) / max

	sl := New(w, h, WithMaxValue(max))

	if sl.Width() != w {
		t.Errorf("Width not initialized:%d", sl.Width())
	}
	if sl.Height() != h {
		t.Errorf("Height not initialized:%d", sl.Height())
	}
	if sl.MaxValue() != max {
		t.Errorf("MaxValue not initialized:%f", sl.MaxValue())
	}
	if sl.Scale() != scale {
		t.Errorf("Scale not initialized:%f", sl.Scale())
	}
}

func TestAutoMaxValue(t *testing.T) {
	w := 30
	h := 15
	max := 81.4

	sl := New(w, h)

	sl.Push(max / 2)
	scale := float64(h) / (max / 2)
	if sl.Scale() != scale {
		t.Errorf("Scale not correct with AutoMaxValue true after greater value than max:%f", sl.Scale())
	}

	max = 99.2
	scale = float64(h) / max
	sl.Push(max)
	if sl.Scale() != scale {
		t.Errorf("Scale not correct with AutoMaxValue true after greater value than max:%f", sl.Scale())
	}

	sl.Push(max / 2)
	scale = float64(h) / (max / 2)
	if sl.Scale() == scale {
		t.Errorf("Scale changed after lesser value than max:%f", sl.Scale())
	}

	sl.AutoMaxValue = false
	max = 104.7
	scale = float64(h) / max
	sl.Push(max)
	if sl.Scale() == scale {
		t.Errorf("Scale changed with AutoMaxValue false after greater value than max:%f", sl.Scale())
	}
}

func TestNoAutoMaxValue(t *testing.T) {
	w := 30
	h := 15
	max := 100.0
	newMax := 150.0
	scale := float64(h) / max

	sl := New(w, h, WithMaxValue(max), WithNoAutoMaxValue())

	sl.Push(max / 2)
	if sl.Scale() != scale {
		t.Errorf("Scale not correct:%f", sl.Scale())
	}

	sl.Push(newMax)
	if sl.Scale() != scale {
		t.Errorf("Scale changed with AutoMaxValue false after greater value than max:%f", sl.Scale())
	}

	sl.AutoMaxValue = true
	scale = float64(h) / newMax
	sl.Push(newMax)
	if sl.Scale() != scale {
		t.Errorf("Scale failed to changed with AutoMaxValue true after greater value than max:%f", sl.Scale())
	}

}

// halves returns data values for a sparkline of height h with max value h,
// so each value is measured in rows and halves[i]/2 rows tall.
func halves(hs ...int) []float64 {
	d := make([]float64, len(hs))
	for i, n := range hs {
		d[i] = float64(n) / 2
	}
	return d
}

func lines(s string) []string {
	ls := strings.Split(s, "\n")
	for i := range ls {
		ls[i] = strings.TrimRight(ls[i], " ")
	}
	return ls
}

func assertView(t *testing.T, got string, want []string) {
	t.Helper()
	g := lines(got)
	if len(g) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(g), len(want), got)
	}
	for i := range want {
		if g[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, g[i], want[i])
		}
	}
}

// Example from issue #8: 14 values in 7 columns.
func TestDrawQuadrants(t *testing.T) {
	sl := New(7, 4, WithMaxValue(4))
	sl.PushAll(halves(8, 7, 7, 5, 7, 5, 3, 7, 7, 8, 1, 5, 3, 3))
	sl.DrawQuadrants()
	assertView(t, sl.View(), []string{
		"▙▖▖▗▟",
		"█▙▙▐█▗",
		"███▟█▐▄",
		"█████▟█",
	})
}

func TestDrawQuadrantsOddCount(t *testing.T) {
	sl := New(2, 1, WithMaxValue(1))
	sl.PushAll(halves(2, 1, 2)) // first column only has its right half
	sl.DrawQuadrants()
	assertView(t, sl.View(), []string{"▐▟"})
}

func TestDrawQuadrantsRoundsToHalfRow(t *testing.T) {
	sl := New(1, 1, WithMaxValue(1))
	sl.PushAll([]float64{0.2, 0.3})
	sl.DrawQuadrants()
	assertView(t, sl.View(), []string{"▗"})
}

func TestDrawQuadrantsColumnsOnly(t *testing.T) {
	sl := New(2, 1, WithMaxValue(1), WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("1"))))
	sl.PushAll(halves(2, 1))
	sl.DrawQuadrantsColumnsOnly()
	if c := sl.Canvas.Cell(canvas.Point{X: 0, Y: 0}); c.Rune != runes.Null || c.Style.GetForeground() == sl.Style.GetForeground() {
		t.Errorf("empty cell styled or filled: %+v", c)
	}
	if c := sl.Canvas.Cell(canvas.Point{X: 1, Y: 0}); c.Rune != runes.QuadUpperLeftLowerAll {
		t.Errorf("cell 1 = %q, want %q", c.Rune, runes.QuadUpperLeftLowerAll)
	}
}

// Keeping twice the width of data must not change the existing draw modes.
func TestDrawShowsLatestWidthValues(t *testing.T) {
	full := New(3, 2, WithMaxValue(2))
	full.PushAll([]float64{2, 2, 2, 0.5, 1, 1.5})
	recent := New(3, 2, WithMaxValue(2))
	recent.PushAll([]float64{0.5, 1, 1.5})

	full.Draw()
	recent.Draw()
	if full.View() != recent.View() {
		t.Errorf("Draw:\n%s\nwant:\n%s", full.View(), recent.View())
	}
	full.DrawBraille()
	recent.DrawBraille()
	if full.View() != recent.View() {
		t.Errorf("DrawBraille:\n%s\nwant:\n%s", full.View(), recent.View())
	}
}

func TestResizeKeepsTwiceWidthValues(t *testing.T) {
	sl := New(1, 1, WithMaxValue(1))
	sl.Resize(2, 1)
	sl.PushAll(halves(1, 2, 1, 2))
	sl.DrawQuadrants()
	assertView(t, sl.View(), []string{"▟▟"})
}
