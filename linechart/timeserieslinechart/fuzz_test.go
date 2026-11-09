// ntcharts - Copyright (c) 2026 Neomantra Corp.

package timeserieslinechart

import (
	"encoding/binary"
	"math"
	"testing"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
)

func FuzzTimeSeriesLineChart(f *testing.F) {
	f.Add(uint8(24), uint8(10), uint8(2), true, []byte{
		0, 0, 0, 0, 0, 0, 0xf0, 0x3f,
		0, 0, 0, 0, 0, 0, 0x00, 0x40,
		0, 0, 0, 0, 0, 0, 0x08, 0x40,
		0, 0, 0, 0, 0, 0, 0x10, 0x40,
	})
	f.Add(uint8(8), uint8(4), uint8(1), false, []byte{})

	style := lipgloss.NewStyle()

	f.Fuzz(func(t *testing.T, wRaw, hRaw, cwRaw uint8, block bool, raw []byte) {
		w := int(wRaw%36) + 1
		h := int(hRaw%20) + 1
		m := New(w, h)

		baseTime := time.Unix(1_700_000_000, 0).UTC()
		n := len(raw) / 8
		if n > 32 {
			n = 32
		}
		for i := 0; i < n; i++ {
			v := math.Float64frombits(binary.LittleEndian.Uint64(raw[i*8 : (i+1)*8]))
			ts := baseTime.Add(time.Duration(i) * time.Minute)
			m.Push(TimePoint{Time: ts, Value: v})
			m.PushDataSet("open", TimePoint{Time: ts, Value: v})
			m.PushDataSet("high", TimePoint{Time: ts, Value: v + 1})
			m.PushDataSet("low", TimePoint{Time: ts, Value: v - 1})
			m.PushDataSet("close", TimePoint{Time: ts, Value: v + 0.5})
		}

		_ = m.FitYToViewWithOpts(FitYOpts{IncludeZero: block})
		m.DrawAll()
		if !utf8.ValidString(m.View()) {
			t.Fatalf("timeserieslinechart DrawAll View produced invalid UTF-8")
		}

		m.DrawBrailleAll()
		if !utf8.ValidString(m.View()) {
			t.Fatalf("timeserieslinechart DrawBrailleAll View produced invalid UTF-8")
		}

		m.DrawCandleWithOpts("open", "high", "low", "close", style, style, DrawCandleOpts{
			Width:          int(cwRaw % 5),
			Block:          block,
			SnapTimeBounds: true,
		})
		if !utf8.ValidString(m.View()) {
			t.Fatalf("timeserieslinechart DrawCandleWithOpts View produced invalid UTF-8")
		}

		if n > 1 {
			_ = m.TrimBefore(baseTime.Add(time.Duration(n/2) * time.Minute))
		}
	})
}
