// ntcharts - Copyright (c) 2026 Neomantra Corp.

package graph

import (
	"image/color"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"

	"charm.land/lipgloss/v2"
)

var (
	red  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	blue = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
)

func expectCell(t *testing.T, m *canvas.Model, p canvas.Point, r rune, fg, bg color.Color, reverse bool) {
	t.Helper()
	c := m.Cell(p)
	if c.Rune != r {
		t.Errorf("cell %v rune = %q, want %q", p, c.Rune, r)
	}
	if got := c.Style.GetForeground(); got != fg {
		t.Errorf("cell %v fg = %v, want %v", p, got, fg)
	}
	got := c.Style.GetBackground()
	if _, unset := got.(lipgloss.NoColor); bg == nil && !unset {
		t.Errorf("cell %v bg = %v, want unset", p, got)
	} else if bg != nil && got != bg {
		t.Errorf("cell %v bg = %v, want %v", p, got, bg)
	}
	if got := c.Style.GetReverse(); got != reverse {
		t.Errorf("cell %v reverse = %v, want %v", p, got, reverse)
	}
}

func TestDrawColumnTopToBottom(t *testing.T) {
	m := canvas.New(1, 5)
	DrawColumnTopToBottom(&m, canvas.Point{X: 0, Y: 0}, 2.25, red)
	expectCell(t, &m, canvas.Point{X: 0, Y: 0}, runes.FullBlock, red.GetForeground(), nil, false)
	expectCell(t, &m, canvas.Point{X: 0, Y: 1}, runes.FullBlock, red.GetForeground(), nil, false)
	// far end: top quarter in bar color, drawn as the inverse block in reverse video
	expectCell(t, &m, canvas.Point{X: 0, Y: 2}, runes.LowerBlockSix, red.GetForeground(), nil, true)
	if c := m.Cell(canvas.Point{X: 0, Y: 3}); c.Rune != runes.Null && c.Rune != ' ' {
		t.Errorf("cell beyond column was drawn: %q", c.Rune)
	}
}

func TestDrawColumnTopToBottomZeroAndTiny(t *testing.T) {
	m := canvas.New(1, 3)
	DrawColumnTopToBottom(&m, canvas.Point{X: 0, Y: 0}, 0, red)
	DrawColumnTopToBottom(&m, canvas.Point{X: 0, Y: 0}, 0.01, red)
	if c := m.Cell(canvas.Point{X: 0, Y: 0}); runes.IsLowerBlockElement(c.Rune) {
		t.Errorf("zero-length column drew %q", c.Rune)
	}
	DrawColumnTopToBottom(&m, canvas.Point{X: 0, Y: 0}, 1, red)
	expectCell(t, &m, canvas.Point{X: 0, Y: 0}, runes.FullBlock, red.GetForeground(), nil, false)
}

func TestDrawColumnTopToBottomStackedBoundary(t *testing.T) {
	// Outer (longer) segment first, then the inner segment over it: the
	// boundary cell shows the inner color on top and the outer color below.
	m := canvas.New(1, 5)
	DrawColumnTopToBottom(&m, canvas.Point{X: 0, Y: 0}, 3, red)
	DrawColumnTopToBottom(&m, canvas.Point{X: 0, Y: 0}, 1.5, blue)
	expectCell(t, &m, canvas.Point{X: 0, Y: 0}, runes.FullBlock, blue.GetForeground(), nil, false)
	expectCell(t, &m, canvas.Point{X: 0, Y: 1}, runes.LowerBlockFour, red.GetForeground(), blue.GetForeground(), false)
	expectCell(t, &m, canvas.Point{X: 0, Y: 2}, runes.FullBlock, red.GetForeground(), nil, false)
}

func TestDrawRowRightToLeft(t *testing.T) {
	m := canvas.New(5, 1)
	DrawRowRightToLeft(&m, canvas.Point{X: 4, Y: 0}, 2.25, red)
	expectCell(t, &m, canvas.Point{X: 4, Y: 0}, runes.FullBlock, red.GetForeground(), nil, false)
	expectCell(t, &m, canvas.Point{X: 3, Y: 0}, runes.FullBlock, red.GetForeground(), nil, false)
	// far end: right quarter in bar color, drawn as the inverse block in reverse video
	expectCell(t, &m, canvas.Point{X: 2, Y: 0}, runes.LeftBlockSix, red.GetForeground(), nil, true)
	if c := m.Cell(canvas.Point{X: 1, Y: 0}); c.Rune != runes.Null && c.Rune != ' ' {
		t.Errorf("cell beyond row was drawn: %q", c.Rune)
	}
}

func TestDrawRowRightToLeftStackedBoundary(t *testing.T) {
	m := canvas.New(5, 1)
	DrawRowRightToLeft(&m, canvas.Point{X: 4, Y: 0}, 3, red)
	DrawRowRightToLeft(&m, canvas.Point{X: 4, Y: 0}, 1.5, blue)
	expectCell(t, &m, canvas.Point{X: 4, Y: 0}, runes.FullBlock, blue.GetForeground(), nil, false)
	expectCell(t, &m, canvas.Point{X: 3, Y: 0}, runes.LeftBlockFour, red.GetForeground(), blue.GetForeground(), false)
	expectCell(t, &m, canvas.Point{X: 2, Y: 0}, runes.FullBlock, red.GetForeground(), nil, false)
}
