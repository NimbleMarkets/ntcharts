// ntcharts - Copyright (c) 2024 Neomantra Corp.

package heatmap

import (
	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/NimbleMarkets/ntcharts/linechart"

	"github.com/charmbracelet/lipgloss"
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
func WithColorScale(cs []lipgloss.Color) Option {
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
