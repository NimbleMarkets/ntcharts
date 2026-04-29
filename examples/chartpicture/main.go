// examples/chartpicture/main.go
//
// Single-pane demo of chartpicture.Model:
//   - Live-updating line chart, new data point every 500ms (60-point window).
//   - 't' cycles the go-analyze/charts theme.
//   - 'g' toggles Glyph / Kitty rendering.
//   - 'r' swaps between line-chart and bar-chart sample datasets.
//   - 'q' or ctrl+c quits.
package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	booba "github.com/NimbleMarkets/go-booba"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/NimbleMarkets/ntcharts/v2/picture/chartpicture"
	"github.com/go-analyze/charts"
)

const (
	kittyID    = 4244
	tickPeriod = 500 * time.Millisecond
	windowLen  = 60
)

var themes = []string{"light", "dark", "vivid-light", "vivid-dark"}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(tickPeriod, func(t time.Time) tea.Msg { return tickMsg(t) })
}

type model struct {
	chart         chartpicture.Model
	width, height int
	themeIdx      int
	mode          string // "line" or "bar"
	series        []float64
	initCmd       tea.Cmd
}

func initialModel() model {
	c := chartpicture.NewWithConfig(chartpicture.Config{
		KittyID: kittyID,
		Theme:   themes[0],
	})
	m := model{chart: c, mode: "line"}
	m.series = generateSeries(windowLen)
	cmd := m.chart.SetLineChartOption(lineOptFromSeries(m.series))
	m.initCmd = tea.Batch(m.chart.Init(), cmd, tickCmd())
	return m
}

func generateSeries(n int) []float64 {
	out := make([]float64, n)
	for i := range n {
		out[i] = 50 + 20*math.Sin(float64(i)/4) + rand.Float64()*5
	}
	return out
}

func lineOptFromSeries(values []float64) charts.LineChartOption {
	opt := charts.NewLineChartOptionWithData([][]float64{values})
	opt.Title = charts.TitleOption{Text: "live data"}
	return opt
}

func barOpt() charts.BarChartOption {
	opt := charts.NewBarChartOptionWithData([][]float64{{12, 24, 18, 30, 22}})
	opt.Title = charts.TitleOption{Text: "quarters"}
	opt.CategoryAxis = charts.CategoryAxisOption{Labels: []string{"Q1", "Q2", "Q3", "Q4", "Q5"}}
	return opt
}

func (m model) Init() tea.Cmd { return m.initCmd }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "g":
			if c := m.chart.Toggle(); c != nil {
				cmds = append(cmds, c)
			}
		case "t":
			m.themeIdx = (m.themeIdx + 1) % len(themes)
			if c := m.chart.SetTheme(themes[m.themeIdx]); c != nil {
				cmds = append(cmds, c)
			}
		case "r":
			if m.mode == "line" {
				m.mode = "bar"
				if c := m.chart.SetBarChartOption(barOpt()); c != nil {
					cmds = append(cmds, c)
				}
			} else {
				m.mode = "line"
				if c := m.chart.SetLineChartOption(lineOptFromSeries(m.series)); c != nil {
					cmds = append(cmds, c)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve a 1-row title and 1-row footer.
		if c := m.chart.SetSize(m.width-2, m.height-4); c != nil {
			cmds = append(cmds, c)
		}
	case tickMsg:
		// Append a new sample, drop the oldest.
		next := 50 + 20*math.Sin(float64(time.Now().UnixNano()/int64(tickPeriod))/4) + rand.Float64()*5
		m.series = append(m.series[1:], next)
		if m.mode == "line" {
			if c := m.chart.SetLineChartOption(lineOptFromSeries(m.series)); c != nil {
				cmds = append(cmds, c)
			}
		}
		cmds = append(cmds, tickCmd())
	}
	if c := m.chart.Update(msg); c != nil {
		cmds = append(cmds, c)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Width(m.width).
		Align(lipgloss.Center).
		Render("📈  ntcharts · chartpicture")

	pane := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(m.width).
		Height(m.height-3).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.chart.View().Content)

	mode := "Glyph"
	if m.chart.Mode() == picture.PictureKitty {
		mode = "Kitty"
	}
	footer := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color("242")).
		Render(fmt.Sprintf("theme: %s · render: %s · type: %s · t theme · g toggle · r switch · q quit",
			themes[m.themeIdx], mode, m.mode))

	parts := []string{title, pane, footer}
	if err := m.chart.Err(); err != nil {
		errBar := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(err.Error())
		parts = []string{title, pane, errBar, footer}
	}
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func main() {
	// booba.Run is a tea.Program substitute that dispatches to native Bubble Tea
	// or the WASM/ghostty-web bridge depending on build target.
	// See https://github.com/NimbleMarkets/go-booba-example.
	if err := booba.Run(initialModel()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
