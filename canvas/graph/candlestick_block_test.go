// ntcharts - Copyright (c) 2026 Neomantra Corp.

package graph

import (
	"math"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"

	"charm.land/lipgloss/v2"
)

// Block style: body interior █, boundary cells render the line style's
// POST-MERGE rune at each junction (█ where the line style upgrades to a
// full heavy line, ▄/▀ where it keeps a half-tip), wick cells light line
// runes (also post-merge where a tip adjoins the body).
//
// NOTE: this deviates from the original phase-7 plan's expected runes,
// which asserted the PRE-merge half-block glyphs (▄/▷ at every boundary
// regardless of adjacency). That plan was wrong — it didn't account for
// the line style's junction-merge upgrade (graph.go DrawCandlestickBottomToTop,
// ~lines 845-887), which is exactly what produced the half-cell slit bug
// this test now guards against.
func TestDrawCandlestickBlockRunes(t *testing.T) {
	s := lipgloss.NewStyle()
	c := canvas.New(3, 6)
	// l=0.0 → wick end row 0-from-bottom; body bl=1.0..bh=4.0; h=5.0
	DrawCandlestickBlockBottomToTop(&c, canvas.Point{X: 1, Y: 5}, 0.0, 1.0, 4.0, 5.0, s)

	get := func(row int) rune { return c.Cell(canvas.Point{X: 1, Y: 5 - row}).Rune }
	if get(0) != runes.LineVertical { // frac(l)=0 < 0.5, adjoins body → merged ╷→│
		t.Fatalf("bottom wick row: got %q want │\n%s", get(0), c.View())
	}
	if get(1) != runes.FullBlock { // body bottom frac 0 < 0.5 → merged ╻→┃ → █
		t.Fatalf("body bottom row: got %q want █\n%s", get(1), c.View())
	}
	for _, row := range []int{2, 3} {
		if get(row) != runes.FullBlock {
			t.Fatalf("body interior row %d: got %q want █\n%s", row, get(row), c.View())
		}
	}
	if get(4) != runes.LowerBlockFour { // body top frac 0 < 0.5, not merged → ▄ (line ╻)
		t.Fatalf("body top row: got %q want ▄\n%s", get(4), c.View())
	}
	if get(5) != runes.LineDown { // top wick end frac 0 < 0.5, not merged → ╷
		t.Fatalf("top wick row: got %q want ╷\n%s", get(5), c.View())
	}
}

// Regression: within a multi-cell body span, a half-block boundary cell
// must never sit directly below or above a full block in a way that
// leaves a half-cell slit — no cell directly below a █ is ▄, and no cell
// directly above a █ is ▀. Exercised at two fractional boundary pairs:
// one that triggers the junction-merge upgrade (1.2/3.7, both boundaries
// merge to █) and one that keeps both half-tips because neither
// boundary's frac crosses the merge threshold (1.7/3.2).
func TestDrawCandlestickBlockNoJunctionSlit(t *testing.T) {
	s := lipgloss.NewStyle()
	check := func(t *testing.T, bl, bh float64) {
		t.Helper()
		c := canvas.New(3, 8)
		p := canvas.Point{X: 1, Y: 7}
		DrawCandlestickBlockBottomToTop(&c, p, bl, bl, bh, bh, s)
		blf, bhf := int(math.Floor(bl)), int(math.Floor(bh))
		get := func(row int) rune { return c.Cell(canvas.Point{X: 1, Y: p.Y - row}).Rune }
		for row := blf; row <= bhf; row++ {
			if get(row) != runes.FullBlock {
				continue
			}
			if row-1 >= blf && get(row-1) == runes.LowerBlockFour {
				t.Fatalf("bl=%v bh=%v: row %d is ▄ directly below █ at row %d (slit)\n%s", bl, bh, row-1, row, c.View())
			}
			if row+1 <= bhf && get(row+1) == runes.UpperHalfBlock {
				t.Fatalf("bl=%v bh=%v: row %d is ▀ directly above █ at row %d (slit)\n%s", bl, bh, row+1, row, c.View())
			}
		}
	}
	t.Run("merged_boundaries", func(t *testing.T) { check(t, 1.2, 3.7) })
	t.Run("half_tip_boundaries", func(t *testing.T) { check(t, 1.7, 3.2) })
}

// Same inputs, cell occupancy must match the line primitive exactly — the
// block style changes glyphs, never geometry.
func TestDrawCandlestickBlockOccupancyMatchesLine(t *testing.T) {
	s := lipgloss.NewStyle()
	a := canvas.New(3, 6)
	b := canvas.New(3, 6)
	DrawCandlestickBottomToTop(&a, canvas.Point{X: 1, Y: 5}, 0.4, 1.6, 3.2, 4.8, s)
	DrawCandlestickBlockBottomToTop(&b, canvas.Point{X: 1, Y: 5}, 0.4, 1.6, 3.2, 4.8, s)
	for row := 0; row <= 5; row++ {
		la := a.Cell(canvas.Point{X: 1, Y: 5 - row}).Rune != runes.Null
		lb := b.Cell(canvas.Point{X: 1, Y: 5 - row}).Rune != runes.Null
		if la != lb {
			t.Fatalf("occupancy mismatch at row %d: line=%v block=%v\nline:\n%s\nblock:\n%s",
				row, la, lb, a.View(), b.View())
		}
	}
}

// A body confined to one cell renders as a single full block.
func TestDrawCandlestickBlockSingleCellBody(t *testing.T) {
	s := lipgloss.NewStyle()
	c := canvas.New(3, 4)
	DrawCandlestickBlockBottomToTop(&c, canvas.Point{X: 1, Y: 3}, 1.0, 1.2, 1.8, 2.0, s)
	if r := c.Cell(canvas.Point{X: 1, Y: 2}).Rune; r != runes.FullBlock {
		t.Fatalf("single-cell body: got %q want █\n%s", r, c.View())
	}
}
