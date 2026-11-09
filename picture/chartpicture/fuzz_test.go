// ntcharts - Copyright (c) 2026 Neomantra Corp.

package chartpicture

import (
	"encoding/binary"
	"math"
	"testing"
	"unicode/utf8"

	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

func FuzzAdapterOptions(f *testing.F) {
	f.Add(uint8(20), uint8(10), true, false, "cat", "ser", []byte{
		0, 0, 0, 0, 0, 0, 0xf0, 0x3f,
		0, 0, 0, 0, 0, 0, 0x00, 0xc0,
		0, 0, 0, 0, 0, 0, 0x08, 0x40,
	})
	f.Add(uint8(10), uint8(6), false, true, "x", "y", []byte{})

	f.Fuzz(func(t *testing.T, wRaw, hRaw uint8, showAxis, horiz bool, label, name string, raw []byte) {
		if !utf8.ValidString(label) || !utf8.ValidString(name) {
			return
		}
		w := int(wRaw%32) + 4
		h := int(hRaw%20) + 4

		var opts []barchart.Option
		if !showAxis {
			opts = append(opts, barchart.WithNoAxis())
		}
		if horiz {
			opts = append(opts, barchart.WithHorizontalBars())
		}
		bc := barchart.New(w, h, opts...)
		n := len(raw) / 8
		if n > 16 {
			n = 16
		}
		vals := make([]float64, 0, n)
		for i := 0; i < n; i++ {
			v := math.Float64frombits(binary.LittleEndian.Uint64(raw[i*8 : (i+1)*8]))
			if math.IsNaN(v) || math.IsInf(v, 0) {
				v = 0
			}
			vals = append(vals, v)
			bc.Push(barchart.BarData{
				Label:  label,
				Values: []barchart.BarValue{{Name: name, Value: v}},
			})
		}

		barOpt := BarChartOptionFromNT(&bc)
		if barOpt.Horizontal != horiz {
			t.Fatalf("BarChartOptionFromNT Horizontal = %v, want %v", barOpt.Horizontal, horiz)
		}

		lc := linechart.New(w, h, 0, 10, -10, 10)
		lc.SetY2Axis(linechart.Axis{Min: 0, Max: 100})
		lineOpt := LineChartOptionFromNTAxes(
			&lc,
			[][]float64{vals, vals},
			[]string{name, name + "2"},
			[]linechart.YAxisSide{linechart.YAxisLeft, linechart.YAxisRight},
		)
		if len(lineOpt.YAxis) != 2 {
			t.Fatalf("expected 2 Y axes when Y2Axis is configured, got %d", len(lineOpt.YAxis))
		}
	})
}
