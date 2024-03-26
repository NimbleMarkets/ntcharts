// bubbletea-charts - Copyright (c) 2024 Neomantra Corp.

package barchart

import (
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// Option is used to set options when initializing a barchart. Example:
//
//	bc := New(width, height, WithMaxValue(someValue))
type Option func(*Model)

// WithZoneManager sets the bubblezone Manager used
// when processing bubbletea Msg mouse events in Update().
func WithZoneManager(zm *zone.Manager) Option {
	return func(m *Model) {
		m.SetZoneManager(zm)
	}
}

// WithStyles sets the axis line and label string styles.
func WithStyles(as lipgloss.Style, ls lipgloss.Style) Option {
	return func(m *Model) {
		m.AxisStyle = as
		m.LabelStyle = ls
	}
}

// WithNoAxis disables drawing axis and labels on to the barchart.
func WithNoAxis() Option {
	return func(m *Model) {
		m.SetShowAxis(false)
	}
}

// WithHorizontalBars displays bars horizontally from right to left,
// with each bar displayed from top to bottom.
func WithHorizontalBars() Option {
	return func(m *Model) {
		m.SetHorizontal(true)
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

// WithNoAutoBarWidth disables automatically setting the bar widths
// to fill the canvas when drawing.
func WithNoAutoBarWidth() Option {
	return func(m *Model) {
		m.AutoBarWidth = false
	}
}

// WithBarWidth sets the bar widths when when drawing.
func WithBarWidth(w int) Option {
	return func(m *Model) {
		m.SetBarWidth(w)
	}
}

// WithBarGap sets the empty spaces between bars when drawing.
func WithBarGap(g int) Option {
	return func(m *Model) {
		m.SetBarGap(g)
	}
}

// WithDataSet adds all bar data values in
// []BarData to barchart data set.
func WithDataSet(d []BarData) Option {
	return func(m *Model) {
		m.PushAll(d)
	}
}
