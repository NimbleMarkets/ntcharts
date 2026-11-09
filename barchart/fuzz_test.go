// ntcharts - Copyright (c) 2026 Neomantra Corp.

package barchart

import (
	"encoding/binary"
	"math"
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

func FuzzBarChart(f *testing.F) {
	f.Add(uint8(20), uint8(10), uint8(2), uint8(1), true, false, "A", []byte{
		0, 0, 0, 0, 0, 0, 0xf0, 0x3f,
		0, 0, 0, 0, 0, 0, 0x00, 0xc0,
		0, 0, 0, 0, 0, 0, 0x10, 0x40,
	})
	f.Add(uint8(12), uint8(8), uint8(0), uint8(0), false, true, "bar", []byte{})

	style := lipgloss.NewStyle()

	f.Fuzz(func(t *testing.T, wRaw, hRaw, bwRaw, gapRaw uint8, showAxis, horiz bool, label string, raw []byte) {
		if !utf8.ValidString(label) {
			return
		}
		w := int(wRaw%36) - 2
		h := int(hRaw%24) - 2
		opts := []Option{WithBarGap(int(gapRaw%6) - 1)}
		if !showAxis {
			opts = append(opts, WithNoAxis())
		}
		if horiz {
			opts = append(opts, WithHorizontalBars())
		}
		m := New(w, h, opts...)
		if bwRaw%2 == 1 {
			m.AutoBarWidth = false
			m.SetBarWidth(int(bwRaw % 6))
		}

		n := len(raw) / 8
		if n > 24 {
			n = 24
		}
		for i := 0; i < n; i += 2 {
			v1 := math.Float64frombits(binary.LittleEndian.Uint64(raw[i*8 : (i+1)*8]))
			vals := []BarValue{{Name: "s1", Value: v1, Style: style}}
			if i+1 < n {
				v2 := math.Float64frombits(binary.LittleEndian.Uint64(raw[(i+1)*8 : (i+2)*8]))
				vals = append(vals, BarValue{Name: "s2", Value: v2, Style: style})
			}
			m.Push(BarData{Label: label, Values: vals})
		}

		m.Draw()
		_ = m.BarDataFromPoint(canvas.Point{X: m.Width() / 2, Y: m.Height() / 2})
		_, _, _ = m.SegmentAt(canvas.Point{X: m.Width() / 2, Y: m.Height() / 2})
		_ = m.Data()

		if !utf8.ValidString(m.View()) {
			t.Fatalf("barchart View produced invalid UTF-8")
		}
	})
}
