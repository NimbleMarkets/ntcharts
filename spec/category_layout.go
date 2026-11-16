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

// drawTickLabels draws sparse category labels: ticks[i] under category i, an
// empty entry drawing nothing. Each label is centred on its category where
// it can be and may use the columns up to halfway to the next labelled
// categories, so a label of an ordered axis labelled every few categories is
// drawn whole.
func drawTickLabels(m *linechart.Model, ticks []string, centers []int) {
	w := m.GraphWidth()
	start := m.Origin().X + 1
	var at []int
	for i, t := range ticks {
		if i < len(centers) && t != "" {
			at = append(at, i)
		}
	}
	end := -1 // last column drawn, relative to start
	for k, i := range at {
		c := centers[i]
		lo, hi := 0, w-1
		if k > 0 {
			lo = (centers[at[k-1]]+c)/2 + 1
		}
		if k+1 < len(at) {
			hi = (c+centers[at[k+1]]+1)/2 - 1
		}
		lo = max(lo, end+2)
		if hi < lo {
			continue
		}
		text := ansi.Truncate(ticks[i], hi-lo+1, "")
		tw := ansi.StringWidth(text)
		if tw == 0 {
			continue
		}
		x := min(max(c-tw/2, lo), hi-tw+1)
		m.Canvas.SetStringWithStyle(canvas.Point{X: start + x, Y: m.Origin().Y + 1}, text, m.LabelStyle)
		end = x + tw - 1
	}
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
