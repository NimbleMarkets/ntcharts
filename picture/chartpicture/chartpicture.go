package chartpicture

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/go-analyze/charts"
)

const (
	DefaultCellWidthPx  = 10
	DefaultCellHeightPx = 20
	minPainterWidthPx   = 200
	minPainterHeightPx  = 120
)

// Config configures a Model at construction.
type Config struct {
	KittyID      int
	Background   color.Color
	CellWidthPx  int
	CellHeightPx int
	Theme        string
}

// Model wraps picture.Model with go-analyze/charts as the image source.
// Forward every tea.Msg to its Update; it routes chart-render completion
// internally and delegates everything else to the embedded picture.Model.
type Model struct {
	pic          picture.Model
	cellW, cellH int
	cols, rows   int
	theme        string
	recipe       renderRecipe
	seq          uint64
	err          error

	// Construction-time pic config buffered until NewWithOptions finalizes.
	pendingPicID int
	pendingPicBg color.Color
}

// New returns a Model with default Config.
func New() Model { return NewWithConfig(Config{}) }

// NewWithConfig returns a Model with the supplied Config. Zero/nil fields are
// filled with defaults.
func NewWithConfig(cfg Config) Model {
	if cfg.CellWidthPx <= 0 {
		cfg.CellWidthPx = DefaultCellWidthPx
	}
	if cfg.CellHeightPx <= 0 {
		cfg.CellHeightPx = DefaultCellHeightPx
	}
	return Model{
		pic: picture.NewWithConfig(picture.Config{
			KittyID:    cfg.KittyID,
			Background: cfg.Background,
		}),
		cellW: cfg.CellWidthPx,
		cellH: cfg.CellHeightPx,
		theme: cfg.Theme,
	}
}

// pixelSize returns the Painter dimensions in pixels, floored at the
// minimums required by the underlying chart library.
func (m *Model) pixelSize() (int, int) {
	w := m.cellW * m.cols
	h := m.cellH * m.rows
	if w < minPainterWidthPx {
		w = minPainterWidthPx
	}
	if h < minPainterHeightPx {
		h = minPainterHeightPx
	}
	return w, h
}

// Err returns the last render error, or nil if the most recent render succeeded.
func (m *Model) Err() error { return m.err }

// Mode forwards to the embedded picture.Model.
func (m *Model) Mode() picture.PictureMode { return m.pic.Mode() }

// Toggle forwards to the embedded picture.Model.
func (m *Model) Toggle() tea.Cmd { return m.pic.Toggle() }

// String returns the rendered image content as a plain string.
func (m *Model) String() string { return m.View().Content }

// SetLineChartOption replaces the current chart with a line chart rendered
// from opt. Returns a tea.Cmd to dispatch the render, or nil if the Model has
// no size yet (call SetSize first).
func (m *Model) SetLineChartOption(opt charts.LineChartOption) tea.Cmd {
	return m.setRecipe(newLineRecipe(opt))
}

// SetBarChartOption replaces the current chart with a bar chart from opt.
func (m *Model) SetBarChartOption(opt charts.BarChartOption) tea.Cmd {
	return m.setRecipe(newBarRecipe(opt))
}

// SetEChartsJSON replaces the current chart with the result of rendering an
// ECharts option JSON document. The JSON's own width/height/theme are
// overridden by the Model's size and current theme.
func (m *Model) SetEChartsJSON(jsonStr string) tea.Cmd {
	return m.setRecipe(newEChartsJSONRecipe(jsonStr))
}

// SetPainterFunc replaces the current chart with whatever the callback paints
// onto a freshly-allocated *charts.Painter. Use this for chart types that do
// not have a typed setter (pie, scatter, candlestick, …) or for composing
// multiple draws on one canvas.
func (m *Model) SetPainterFunc(f func(*charts.Painter) error) tea.Cmd {
	return m.setRecipe(newPainterFuncRecipe(f))
}

func (m *Model) setRecipe(r renderRecipe) tea.Cmd {
	m.recipe = r
	m.seq++
	m.err = nil
	return m.renderCmd()
}

// renderCmd schedules the current recipe to render at the current size and
// theme, returning a tea.Cmd that produces a chartRenderedMsg. Returns nil if
// no recipe is set or the Model has no size yet.
func (m *Model) renderCmd() tea.Cmd {
	if m.recipe == nil || m.cols <= 0 || m.rows <= 0 {
		return nil
	}
	recipe, theme, seq := m.recipe, m.theme, m.seq
	w, h := m.pixelSize()
	return func() tea.Msg {
		buf, err := recipe(w, h, theme)
		if err != nil {
			return chartRenderedMsg{seq: seq, err: err}
		}
		img, _, derr := decodePNGToImage(buf)
		if derr != nil {
			return chartRenderedMsg{seq: seq, err: derr}
		}
		return chartRenderedMsg{seq: seq, img: img}
	}
}

// SetSize updates the rendering dimensions in terminal cells. Forwards to the
// embedded picture.Model and re-runs the current chart recipe (if any) at the
// new size. Returns a Cmd that may batch a Kitty-encode Cmd from the picture
// layer with the chart re-render Cmd.
func (m *Model) SetSize(cols, rows int) tea.Cmd {
	if cols == m.cols && rows == m.rows {
		return nil
	}
	m.cols = cols
	m.rows = rows
	m.seq++
	picCmd := m.pic.SetSize(cols, rows)
	renderCmd := m.renderCmd()
	if picCmd == nil {
		return renderCmd
	}
	if renderCmd == nil {
		return picCmd
	}
	return tea.Batch(picCmd, renderCmd)
}

// SetTheme stores the named go-analyze/charts theme and re-runs the current
// recipe. The empty string means "use the upstream default theme".
func (m *Model) SetTheme(name string) tea.Cmd {
	if name == m.theme {
		return nil
	}
	m.theme = name
	m.seq++
	return m.renderCmd()
}

// Update routes chart-render completion messages and delegates everything
// else to the embedded picture.Model.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if rendered, ok := msg.(chartRenderedMsg); ok {
		if rendered.seq != m.seq {
			return nil // stale
		}
		if rendered.err != nil {
			m.err = rendered.err
			return m.pic.SetImage(nil)
		}
		m.err = nil
		return m.pic.SetImage(rendered.img)
	}
	return m.pic.Update(msg)
}

// View returns the rendered chart, or an error placeholder if the most recent
// render failed. The prior image is cleared when an error is set.
func (m *Model) View() tea.View {
	if m.err != nil {
		return tea.NewView("Chart error:\n" + m.err.Error())
	}
	return m.pic.View()
}
