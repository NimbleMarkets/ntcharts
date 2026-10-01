// ntcharts - Copyright (c) 2026 Neomantra Corp.

// Logscale shows a timeseries linechart on a logarithmic Y axis.
//
// The data grows by more than four orders of magnitude, which a linear axis
// squashes into the last few columns. Press `l` to switch the same chart
// between a linear and a log Y axis. On the log axis the labels fall on
// powers of ten, equal ratios are equal distances, and zooming the Y axis
// (`+` and `-`) multiplies the range instead of adding to it.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	booba "github.com/NimbleMarkets/go-booba"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	tslc "github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	zone "github.com/lrstanley/bubblezone/v2"
)

// revenue is a monthly series that grows from 12 to 52,000.
var revenue = []float64{12, 30, 95, 210, 410, 900, 1800, 3900, 9200, 18500, 31000, 52000}

const (
	linearMin, linearMax = 0.0, 60000.0
	logMin, logMax       = 10.0, 100000.0

	// zoom steps, in the units of the active scale: data units on a linear
	// axis, decades on a log axis
	linearZoom = 4000.0
	logZoom    = 0.25
)

var (
	frameStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")) // purple
	lineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")) // green
	axisStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")) // yellow
	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")) // cyan
)

type model struct {
	chart tslc.Model
	zM    *zone.Manager
}

// dollars formats a value as a short dollar amount: $90, $1.5k, $100k.
func dollars(_ int, v float64) string {
	if v < 1000 {
		return "$" + strconv.FormatFloat(v, 'f', -1, 64)
	}
	k := strconv.FormatFloat(v/1000, 'f', 1, 64)
	return "$" + strings.TrimSuffix(k, ".0") + "k"
}

func newModel() model {
	zM := zone.New()
	chart := tslc.New(60, 16,
		tslc.WithYRange(linearMin, linearMax),
		tslc.WithAxesStyles(axisStyle, labelStyle),
		tslc.WithStyle(lineStyle),
		tslc.WithYLabelFormatter(dollars),
		tslc.WithUpdateHandler(tslc.DateUpdateHandler(30)),
		tslc.WithZoneManager(zM),
	)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, v := range revenue {
		chart.Push(tslc.TimePoint{Time: start.AddDate(0, i, 0), Value: v})
	}
	chart.Focus()
	m := model{chart: chart, zM: zM}
	m.chart.DrawBrailleAll()
	return m
}

// toggle switches the Y axis between linear and log. The expected and
// displayed ranges are set in data units on either scale.
func (m *model) toggle() {
	if m.chart.YScale() == linechart.ScaleLog {
		m.chart.SetYScale(linechart.ScaleLinear)
		m.chart.SetYRange(linearMin, linearMax)
		m.chart.SetViewYRange(linearMin, linearMax)
	} else {
		m.chart.SetYRange(logMin, logMax)
		m.chart.SetViewYRange(logMin, logMax)
		m.chart.SetYScale(linechart.ScaleLog)
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "l":
			m.toggle()
		case "+", "=":
			m.chart.ZoomIn(0, m.zoomStep())
		case "-", "_":
			m.chart.ZoomOut(0, m.zoomStep())
		}
	}
	// pgup/pgdown, arrow keys and the mouse zoom and move the X axis
	m.chart, _ = m.chart.Update(msg)
	m.chart.DrawBrailleAll()
	return m, nil
}

func (m model) zoomStep() float64 {
	if m.chart.YScale() == linechart.ScaleLog {
		return logZoom
	}
	return linearZoom
}

func (m model) View() tea.View {
	s := fmt.Sprintf("Y axis: %s   (l: toggle, +/-: zoom Y, pgup/pgdn: X, q: quit)\n",
		m.chart.YScale())
	s += frameStyle.Render(m.chart.View())
	v := tea.NewView(m.zM.Scan(s))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func main() {
	// booba.Run is a tea.Program substitute that dispatches to native Bubble Tea
	// or the WASM/ghostty-web bridge depending on build target.
	// See https://github.com/NimbleMarkets/go-booba-example.
	if err := booba.Run(newModel()); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
