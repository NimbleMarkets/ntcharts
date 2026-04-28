package chartpicture

import (
	"image/color"

	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/go-analyze/charts"
)

// Option configures a Model at construction time. All Options are applied in
// order; later Options override earlier ones.
type Option func(*Model)

// NewWithOptions is sugar for constructing a Model and applying functional
// Options to it. Equivalent to calling New() and the matching setter for each
// Option in order.
func NewWithOptions(opts ...Option) Model {
	m := New()
	for _, o := range opts {
		o(&m)
	}
	if m.pendingPicID != 0 || m.pendingPicBg != nil {
		m.pic = picture.NewWithConfig(picture.Config{
			KittyID:    m.pendingPicID,
			Background: m.pendingPicBg,
		})
	}
	m.pendingPicID = 0
	m.pendingPicBg = nil
	return m
}

// WithKittyID sets the Kitty graphics protocol image ID forwarded to the
// embedded picture.Model.
func WithKittyID(id int) Option {
	return func(m *Model) { m.pendingPicID = id }
}

// WithBackground sets the background color used when compositing.
func WithBackground(c color.Color) Option {
	return func(m *Model) { m.pendingPicBg = c }
}

// WithCellSize sets the pixel-per-cell multipliers used when sizing the chart
// painter from the cell-based SetSize.
func WithCellSize(w, h int) Option {
	return func(m *Model) {
		if w > 0 {
			m.cellW = w
		}
		if h > 0 {
			m.cellH = h
		}
	}
}

// WithTheme sets the go-analyze/charts theme name used by every render.
func WithTheme(name string) Option {
	return func(m *Model) { m.theme = name }
}

// WithLineChartOption sets the initial chart to a line chart rendered from
// opt. Equivalent to calling SetLineChartOption after construction.
func WithLineChartOption(opt charts.LineChartOption) Option {
	return func(m *Model) { m.recipe = newLineRecipe(opt); m.seq++ }
}

// WithBarChartOption sets the initial chart to a bar chart from opt.
func WithBarChartOption(opt charts.BarChartOption) Option {
	return func(m *Model) { m.recipe = newBarRecipe(opt); m.seq++ }
}

// WithEChartsJSON sets the initial chart from an ECharts option JSON document.
func WithEChartsJSON(jsonStr string) Option {
	return func(m *Model) { m.recipe = newEChartsJSONRecipe(jsonStr); m.seq++ }
}

// WithPainterFunc sets the initial chart from a painter callback.
func WithPainterFunc(f func(*charts.Painter) error) Option {
	return func(m *Model) { m.recipe = newPainterFuncRecipe(f); m.seq++ }
}
