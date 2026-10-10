// ntcharts - Copyright (c) 2026 Neomantra Corp.

// Dualaxis shows two series on very different scales sharing one time
// axis: a load between 0 and 1 on the left Y axis and a temperature
// between 5 and 20 C on the right Y axis.
//
// Press + and - to zoom the Y axes: the right axis follows the left one.
// pgup/pgdn, the arrow keys and the mouse zoom and move the time axis.
package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	booba "github.com/NimbleMarkets/go-booba"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	tslc "github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	zone "github.com/lrstanley/bubblezone/v2"
)

const (
	loadSet = "load"
	tempSet = "temp"

	points           = 120 // one per minute
	loadMin, loadMax = 0.0, 1.0
	tempMin, tempMax = 5.0, 20.0
	zoomStep         = 0.05 // in left-axis units
)

var (
	frameStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")) // purple
	axisStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")) // yellow
	loadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")) // green
	tempStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")) // red
)

type model struct {
	chart tslc.Model
	zM    *zone.Manager
}

// walk returns n steps of a random walk from start, by at most step,
// clamped to lo..hi.
func walk(r *rand.Rand, n int, start, step, lo, hi float64) []float64 {
	v := make([]float64, n)
	x := start
	for i := range v {
		x = min(hi, max(lo, x+(r.Float64()*2-1)*step))
		v[i] = x
	}
	return v
}

func oneDecimal(_ int, v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }

func celsius(_ int, v float64) string { return strconv.FormatFloat(v, 'f', 0, 64) + "C" }

func newModel() model {
	zM := zone.New()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	chart := tslc.New(64, 16,
		tslc.WithTimeRange(start, start.Add(points*time.Minute)),
		tslc.WithYRange(loadMin, loadMax),
		tslc.WithYLabelFormatter(oneDecimal),
		tslc.WithXLabelFormatter(tslc.HourTimeLabelFormatter()),
		tslc.WithY2Axis(linechart.Axis{Min: tempMin, Max: tempMax, LabelFormatter: celsius, LabelStyle: tempStyle}),
		tslc.WithDataSetYAxis(tempSet, linechart.YAxisRight),
		tslc.WithAxesStyles(axisStyle, lipgloss.NewStyle()),
		tslc.WithDataSetStyle(loadSet, loadStyle),
		tslc.WithDataSetStyle(tempSet, tempStyle),
		tslc.WithUpdateHandler(tslc.SecondUpdateHandler(300)),
		tslc.WithZoneManager(zM),
	)
	r := rand.New(rand.NewPCG(1, 2))
	load := walk(r, points, 0.5, 0.08, loadMin, loadMax)
	temp := walk(r, points, 12, 0.6, tempMin, tempMax)
	for i := range points {
		t := start.Add(time.Duration(i) * time.Minute)
		chart.PushDataSet(loadSet, tslc.TimePoint{Time: t, Value: load[i]})
		chart.PushDataSet(tempSet, tslc.TimePoint{Time: t, Value: temp[i]})
	}
	chart.Focus()
	m := model{chart: chart, zM: zM}
	m.chart.DrawAll()
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "+", "=":
			m.chart.ZoomIn(0, zoomStep)
		case "-", "_":
			m.chart.ZoomOut(0, zoomStep)
		}
	}
	// pgup/pgdown, arrow keys and the mouse zoom and move the time axis;
	// Update also rescales the data sets to the new view
	m.chart, _ = m.chart.Update(msg)
	m.chart.DrawAll()
	return m, nil
}

func (m model) View() tea.View {
	s := "load (left)  temp (right)  +/-: zoom Y  pgup/pgdn: time  q: quit\n"
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
