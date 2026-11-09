// ntcharts - Copyright (c) 2026 Neomantra Corp.

package linechart

import (
	"math"
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
)

func FuzzLineChartScaleAndTicks(f *testing.F) {
	f.Add(uint8(24), uint8(10), 0.1, 1000.0, 1.0, 100.0, uint8(2), uint8(2), true, true)
	f.Add(uint8(12), uint8(6), -50.0, 50.0, -10.0, 10.0, uint8(0), uint8(1), false, false)
	f.Add(uint8(1), uint8(1), math.NaN(), math.Inf(1), 0.0, 0.0, uint8(3), uint8(3), true, false)

	style := lipgloss.NewStyle()

	f.Fuzz(func(t *testing.T, wRaw, hRaw uint8, minX, maxX, minY, maxY float64, xStepRaw, yStepRaw uint8, logX, logY bool) {
		w := int(wRaw % 40)
		h := int(hRaw % 24)
		xStep := int(xStepRaw % 6)
		yStep := int(yStepRaw % 6)

		opts := []Option{WithXYSteps(xStep, yStep)}
		if logX {
			opts = append(opts, WithXScale(ScaleLog))
		}
		if logY {
			opts = append(opts, WithYScale(ScaleLog))
		}

		m := New(w, h, minX, maxX, minY, maxY, opts...)
		m.SetY2Axis(Axis{
			Min:   minY,
			Max:   maxY,
			Scale: m.YScale(),
		})

		m.DrawXYAxisAndLabel()
		p1 := canvas.Float64Point{X: minX, Y: minY}
		p2 := canvas.Float64Point{X: maxX, Y: maxY}
		m.DrawBrailleLineWithStyle(p1, p2, style)
		m.DrawLineWithStyle(p1, p2, runes.ArcLineStyle, style)
		m.DrawRuneWithStyle(p1, 'x', style)
		_, _ = m.PointCell(p1)
		_ = m.Y2ToY(minY)
		_ = m.Y2ToY(maxY)

		out := m.View()
		if !utf8.ValidString(out) {
			t.Fatalf("linechart View produced invalid UTF-8")
		}
	})
}
