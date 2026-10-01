// ntcharts - Copyright (c) 2024 Neomantra Corp.

package heatmap

import (
	"image/color"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	"charm.land/lipgloss/v2"
)

// Option is used to set options when initializing a heatmap. Example:
//
//	sl := New(width, height, WithMaxValue(someValue), WithNoAuto())
type Option func(*Model)

// WithLineChart sets the initial LineChart
func WithStyle(lm linechart.Model) Option {
	return func(m *Model) {
		m.Model = lm
	}
}

// WithKeyMap sets the canvas KeyMap used
// when processing keyboard event messages in Update().
func WithKeyMap(k canvas.KeyMap) Option {
	return func(m *Model) {
		m.Canvas.KeyMap = k
	}
}

// WithUpdateHandler sets the canvas UpdateHandler used
// when processing bubbletea Msg events in Update().
func WithUpdateHandler(h canvas.UpdateHandler) Option {
	return func(m *Model) {
		m.Canvas.UpdateHandler = h
	}
}

// WithColorScale uses the given Color array for the ColorScale.
func WithColorScale(cs []color.Color) Option {
	return func(m *Model) {
		m.ColorScale = cs
	}
}

// WithCellStyle sets the default cell style
func WithCellStyle(style lipgloss.Style) Option {
	return func(m *Model) {
		m.cellStyle = style
	}
}

// WithValueScale sets the minMax/maxValues that for the color mapping
func WithValueRange(minVal, maxVal float64) Option {
	return func(m *Model) {
		m.minValue = minVal
		m.maxValue = maxVal
	}
}

// WithAutoValueRange enables automatically setting the minimum and maximum
// values if new data values are beyond the current ranges.
func WithAutoValueRange() Option {
	return func(m *Model) {
		m.AutoMinValue = true
		m.AutoMaxValue = true
	}
}

// WithPoints adds all data values in []float64 to sparkline data buffer.
func WithPoints(d []HeatPoint) Option {
	return func(m *Model) {
		m.PushAll(d)
	}
}

// WithCellSize sets the size of one data cell in data units and draws each
// point as a filled block of that size, centred on the point, instead of a
// single canvas cell. For an index grid (points at whole X and Y) use 1, 1
// together with ranges of -0.5..n-0.5 (see WithXLabels) and the cells tile
// the graph. A size that is not positive restores single-cell points.
func WithCellSize(w, h float64) Option {
	return func(m *Model) {
		m.cellW, m.cellH = w, h
	}
}

// WithXLabels names the cell columns: labels[i] is drawn under the column at
// X = i. The labels define the grid: the X range becomes -0.5..len(labels)-0.5
// and is no longer auto-ranged. Draw draws them.
func WithXLabels(labels []string) Option {
	return func(m *Model) {
		m.SetXLabels(labels)
	}
}

// WithYLabels names the cell rows: labels[i] is drawn left of the row at
// Y = i, counting up from the bottom. The labels define the grid: the Y range
// becomes -0.5..len(labels)-0.5 and is no longer auto-ranged, and the margin
// left of the graph grows to fit the longest label. Draw draws them.
func WithYLabels(labels []string) Option {
	return func(m *Model) {
		m.SetYLabels(labels)
	}
}
