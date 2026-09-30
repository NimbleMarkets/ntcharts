// ntcharts - Copyright (c) 2026 Neomantra Corp.

package graph

import (
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"

	"charm.land/lipgloss/v2"
)

// Width 1 must be exactly the single-column primitive.
func TestDrawCandlestickWideWidthOneIdentical(t *testing.T) {
	s := lipgloss.NewStyle()
	a := canvas.New(5, 5)
	b := canvas.New(5, 5)
	DrawCandlestickBottomToTop(&a, canvas.Point{X: 2, Y: 4}, 0.0, 1.0, 3.0, 4.0, s)
	DrawCandlestickBottomToTopWide(&b, canvas.Point{X: 2, Y: 4}, 1, 0.0, 1.0, 3.0, 4.0, s)
	if a.View() != b.View() {
		t.Fatalf("width-1 wide candle differs from single-column primitive:\n%q\nvs\n%q", a.View(), b.View())
	}
}

// Width 3: side columns carry body rows only; wick rows stay empty there.
func TestDrawCandlestickWideThreeColumns(t *testing.T) {
	s := lipgloss.NewStyle()
	c := canvas.New(5, 5)
	// l=0 → row 4 (lower wick), body 1..3 → rows 3..1, h=4 → row 0 (upper wick)
	DrawCandlestickBottomToTopWide(&c, canvas.Point{X: 2, Y: 4}, 3, 0.0, 1.0, 3.0, 4.0, s)

	for _, x := range []int{1, 3} { // side columns
		for _, y := range []int{1, 2, 3} { // body rows
			if c.Cell(canvas.Point{X: x, Y: y}).Rune == runes.Null {
				t.Fatalf("side column %d missing body rune at row %d:\n%s", x, y, c.View())
			}
		}
		for _, y := range []int{0, 4} { // pure wick rows
			if c.Cell(canvas.Point{X: x, Y: y}).Rune != runes.Null {
				t.Fatalf("side column %d must not draw wick row %d:\n%s", x, y, c.View())
			}
		}
	}
	for y := 0; y <= 4; y++ { // center column: full candle
		if c.Cell(canvas.Point{X: 2, Y: y}).Rune == runes.Null {
			t.Fatalf("center column missing rune at row %d:\n%s", y, c.View())
		}
	}
}

// Even width renders as next-lower-odd plus one extra column on the right.
func TestDrawCandlestickWideEvenWidth(t *testing.T) {
	s := lipgloss.NewStyle()
	c := canvas.New(6, 5)
	DrawCandlestickBottomToTopWide(&c, canvas.Point{X: 2, Y: 4}, 4, 0.0, 1.0, 3.0, 4.0, s)
	// columns 1,2,3,4 covered; column 0 and 5 empty at body row 2
	for _, x := range []int{1, 2, 3, 4} {
		if c.Cell(canvas.Point{X: x, Y: 2}).Rune == runes.Null {
			t.Fatalf("column %d missing body rune for even width:\n%s", x, c.View())
		}
	}
	for _, x := range []int{0, 5} {
		if c.Cell(canvas.Point{X: x, Y: 2}).Rune != runes.Null {
			t.Fatalf("column %d must be empty for width 4 centered at 2:\n%s", x, c.View())
		}
	}
}
