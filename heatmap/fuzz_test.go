// ntcharts - Copyright (c) 2026 Neomantra Corp.

package heatmap

import (
	"encoding/binary"
	"math"
	"testing"
	"unicode/utf8"
)

func FuzzHeatmap(f *testing.F) {
	f.Add(uint8(20), uint8(10), true, "colA", "row1", []byte{
		0, 0, 0, 0, 0, 0, 0x00, 0x00,
		0, 0, 0, 0, 0, 0, 0x00, 0x00,
		0, 0, 0, 0, 0, 0, 0xf0, 0x3f,
	})
	f.Add(uint8(8), uint8(4), false, "", "", []byte{})

	f.Fuzz(func(t *testing.T, wRaw, hRaw uint8, cellGrid bool, xLbl, yLbl string, raw []byte) {
		if !utf8.ValidString(xLbl) || !utf8.ValidString(yLbl) {
			return
		}
		w := int(wRaw%32) + 1
		h := int(hRaw%20) + 1
		opts := []Option{WithAutoValueRange()}
		if cellGrid {
			opts = append(opts,
				WithCellSize(1, 1),
				WithXLabels([]string{xLbl, "B"}),
				WithYLabels([]string{yLbl, "2"}),
			)
		}
		m := New(w, h, opts...)

		n := len(raw) / 24
		if n > 24 {
			n = 24
		}
		for i := 0; i < n; i++ {
			x := math.Float64frombits(binary.LittleEndian.Uint64(raw[i*24 : i*24+8]))
			y := math.Float64frombits(binary.LittleEndian.Uint64(raw[i*24+8 : i*24+16]))
			v := math.Float64frombits(binary.LittleEndian.Uint64(raw[i*24+16 : i*24+24]))
			if cellGrid {
				x = float64(i % 4)
				y = float64((i / 4) % 4)
			}
			m.Push(NewHeatPoint(x, y, v))
		}

		m.Draw()
		if cellGrid && n > 0 {
			_ = m.CellPoints(NewHeatPoint(0, 0, 1))
		}
		if !utf8.ValidString(m.View()) {
			t.Fatalf("heatmap View produced invalid UTF-8")
		}
	})
}
