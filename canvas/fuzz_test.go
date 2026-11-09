// ntcharts - Copyright (c) 2026 Neomantra Corp.

package canvas

import (
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
)

func FuzzCanvasOperations(f *testing.F) {
	f.Add(uint8(10), uint8(6), uint8(12), uint8(8), "hello\nworld", []byte{1, 2, 3, 4, 5, 6})
	f.Add(uint8(0), uint8(0), uint8(4), uint8(4), "test", []byte{0, 1, 2, 3})
	f.Add(uint8(5), uint8(5), uint8(0), uint8(0), "", []byte{7, 8, 9})

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))

	f.Fuzz(func(t *testing.T, wRaw, hRaw, w2Raw, h2Raw uint8, text string, ops []byte) {
		if !utf8.ValidString(text) {
			return
		}
		w := int(wRaw%32) - 2
		h := int(hRaw%32) - 2
		m := New(w, h, WithViewWidth(int(w2Raw%20)), WithViewHeight(int(h2Raw%20)))
		if m.Width() < 0 || m.Height() < 0 {
			t.Fatalf("negative canvas dimensions: %dx%d", m.Width(), m.Height())
		}

		for i := 0; i < len(ops); i++ {
			op := ops[i]
			x := int(int8(op)) % 20
			y := int(int8(op>>2)) % 20
			switch op % 8 {
			case 0:
				m.SetCell(Point{X: x, Y: y}, NewCellWithStyle('x', style))
			case 1:
				m.SetStringWithStyle(Point{X: x, Y: y}, text, style)
			case 2:
				m.SetCursor(Point{X: x, Y: y})
			case 3:
				m.ShiftUp()
			case 4:
				m.ShiftDown()
			case 5:
				m.ShiftLeft()
			case 6:
				m.ShiftRight()
			case 7:
				nw := int(w2Raw%24) - 2
				nh := int(h2Raw%24) - 2
				m.Resize(nw, nh)
			}
		}

		_ = m.Cell(Point{X: 0, Y: 0})
		_ = m.Cell(Point{X: -1, Y: -1})
		out := m.View()
		if !utf8.ValidString(out) {
			t.Fatalf("canvas View produced invalid UTF-8")
		}
	})
}
