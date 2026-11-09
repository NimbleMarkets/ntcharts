// ntcharts - Copyright (c) 2026 Neomantra Corp.

package sparkline

import (
	"encoding/binary"
	"math"
	"testing"
	"unicode/utf8"
)

func FuzzSparkline(f *testing.F) {
	f.Add(uint8(16), uint8(6), uint8(12), uint8(4), []byte{
		0, 0, 0, 0, 0, 0, 0xf0, 0x3f,
		0, 0, 0, 0, 0, 0, 0x00, 0x40,
		0, 0, 0, 0, 0, 0, 0x10, 0x40,
	})
	f.Add(uint8(0), uint8(0), uint8(4), uint8(2), []byte{})

	f.Fuzz(func(t *testing.T, wRaw, hRaw, w2Raw, h2Raw uint8, raw []byte) {
		w := int(wRaw%36) - 2
		h := int(hRaw%20) - 2
		m := New(w, h)

		n := len(raw) / 8
		if n > 64 {
			n = 64
		}
		vals := make([]float64, n)
		for i := 0; i < n; i++ {
			vals[i] = math.Float64frombits(binary.LittleEndian.Uint64(raw[i*8 : (i+1)*8]))
		}
		m.PushAll(vals)

		m.Draw()
		if !utf8.ValidString(m.View()) {
			t.Fatalf("sparkline Draw View produced invalid UTF-8")
		}
		m.DrawBraille()
		if !utf8.ValidString(m.View()) {
			t.Fatalf("sparkline DrawBraille View produced invalid UTF-8")
		}
		m.DrawQuadrants()
		if !utf8.ValidString(m.View()) {
			t.Fatalf("sparkline DrawQuadrants View produced invalid UTF-8")
		}

		m.Resize(int(w2Raw%24)-2, int(h2Raw%12)-2)
		m.Draw()
		m.DrawQuadrants()
		if !utf8.ValidString(m.View()) {
			t.Fatalf("sparkline resized View produced invalid UTF-8")
		}
	})
}
