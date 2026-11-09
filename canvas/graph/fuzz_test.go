// ntcharts - Copyright (c) 2026 Neomantra Corp.

package graph

import (
	"math"
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
)

func FuzzGraphPrimitives(f *testing.F) {
	f.Add(int16(0), int16(0), int16(10), int16(8), uint8(5), 1.5, 2.5, 4.2, 6.8, true)
	f.Add(int16(-10), int16(20), int16(30), int16(-5), uint8(12), 0.0, 0.0, 0.0, 0.0, false)
	f.Add(int16(5), int16(5), int16(5), int16(5), uint8(0), math.NaN(), math.Inf(1), math.Inf(-1), -3.0, true)

	style := lipgloss.NewStyle()

	f.Fuzz(func(t *testing.T, x1, y1, x2, y2 int16, rRaw uint8, v1, v2, v3, v4 float64, arc bool) {
		limit := 256
		p1 := canvas.Point{X: int(x1), Y: int(y1)}
		p2 := canvas.Point{X: int(x2), Y: int(y2)}
		radius := int(rRaw)

		lp := GetLinePointsWithLimit(p1, p2, limit)
		if len(lp) > limit {
			t.Fatalf("GetLinePointsWithLimit exceeded limit: %d > %d", len(lp), limit)
		}
		cp := GetCirclePointsWithLimit(p1, radius, limit)
		if len(cp) > limit {
			t.Fatalf("GetCirclePointsWithLimit exceeded limit: %d > %d", len(cp), limit)
		}
		fcp := GetFullCirclePointsWithLimit(p1, radius, limit)
		if len(fcp) > limit {
			t.Fatalf("GetFullCirclePointsWithLimit exceeded limit: %d > %d", len(fcp), limit)
		}

		m := canvas.New(20, 12)
		ls := runes.ThinLineStyle
		if arc {
			ls = runes.ArcLineStyle
		}
		Origin := canvas.Point{X: 2, Y: 10}
		DrawXYAxisAll(&m, Origin, style)
		if len(lp) >= 2 {
			DrawLinePoints(&m, lp, ls, style)
		}

		// Bound finite lengths so fuzzing exercises fractional/overlap drawing without huge loops.
		clampLen := func(v float64) float64 {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return v
			}
			return math.Mod(v, 24)
		}
		c1, c2, c3, c4 := clampLen(v1), clampLen(v2), clampLen(v3), clampLen(v4)

		DrawColumnBottomToTop(&m, canvas.Point{X: 3, Y: 10}, c1, style)
		DrawColumnBottomToTop(&m, canvas.Point{X: 3, Y: 10}, c2, style)
		DrawColumnTopToBottom(&m, canvas.Point{X: 4, Y: 2}, c1, style)
		DrawColumnTopToBottom(&m, canvas.Point{X: 4, Y: 2}, c2, style)

		DrawRowLeftToRight(&m, canvas.Point{X: 2, Y: 4}, c1, style)
		DrawRowLeftToRight(&m, canvas.Point{X: 2, Y: 4}, c2, style)
		DrawRowRightToLeft(&m, canvas.Point{X: 15, Y: 5}, c1, style)
		DrawRowRightToLeft(&m, canvas.Point{X: 15, Y: 5}, c2, style)

		DrawCandlestickBottomToTop(&m, canvas.Point{X: 6, Y: 10}, c1, c2, c3, c4, style)
		DrawCandlestickBottomToTopWide(&m, canvas.Point{X: 9, Y: 10}, int(rRaw%5), c1, c2, c3, c4, style)
		DrawCandlestickBlockBottomToTop(&m, canvas.Point{X: 13, Y: 10}, c1, c2, c3, c4, style)

		bg := NewBrailleGrid(10, 6, -10, 10, -10, 10)
		gp1 := bg.GridPoint(canvas.Float64Point{X: v1, Y: v2})
		gp2 := bg.GridPoint(canvas.Float64Point{X: v3, Y: v4})
		bg.Set(gp1)
		bg.Set(gp2)
		DrawBraillePatterns(&m, canvas.Point{X: 0, Y: 0}, bg.BraillePatterns(), style)

		if !utf8.ValidString(m.View()) {
			t.Fatalf("graph canvas View produced invalid UTF-8")
		}
	})
}
