// ntcharts - Copyright (c) 2024 Neomantra Corp.

package sparkline

import (
	"github.com/NimbleMarkets/ntcharts/canvas"

	"github.com/charmbracelet/lipgloss"
)

// Option is used to set options when initializing a sparkline. Example:
//
//	sl := New(width, height, WithMaxValue(someValue), WithNoAuto())
type Option func(*Model)

// WithStyle sets the default column style.
func WithStyle(s lipgloss.Style) Option {
	return func(m *Model) {
		m.Style = s
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

// WithNoAutoMaxValue disables automatically setting the max value
// if new data greater than the current max is added.
func WithNoAutoMaxValue() Option {
	return func(m *Model) {
		m.AutoMaxValue = false
	}
}

// WithMaxValue sets the expected maximum data value
// to given float64.
func WithMaxValue(f float64) Option {
	return func(m *Model) {
		m.SetMax(f)
	}
}

// WithData adds all data values in []float64 to sparkline data buffer.
func WithData(d []float64) Option {
	return func(m *Model) {
		m.PushAll(d)
	}
}
