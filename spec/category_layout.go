// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/charmbracelet/x/ansi"
)

// categoryCenters partitions the full plot into ordered slots, including empty
// categories. Dense charts may share centers but do not shift later categories.
func categoryCenters(width, count int) []int {
	centers := make([]int, count)
	for i := range centers {
		left, right := i*width/count, (i+1)*width/count
		centers[i] = min(width-1, (left+max(left, right-1))/2)
	}
	return centers
}

func drawCategoryLabels(m *linechart.Model, labels []string, centers []int) {
	w := m.GraphWidth()
	start := m.Origin().X + 1
	end := start - 1
	for i, label := range labels {
		left, right := i*w/len(labels), (i+1)*w/len(labels)
		text := ansi.Truncate(label, max(0, right-left), "")
		x := start + centers[i] - ansi.StringWidth(text)/2
		if text != "" && x > end && x >= start && x+ansi.StringWidth(text) <= start+w {
			m.Canvas.SetStringWithStyle(canvas.Point{X: x, Y: m.Origin().Y + 1}, text, m.LabelStyle)
			end = x + ansi.StringWidth(text)
		}
	}
}
