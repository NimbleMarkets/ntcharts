// ntcharts - Copyright (c) 2026 Neomantra Corp.

// Throughput keeps one minute of transfer speeds and fits Y to that window.
package main

import (
	"fmt"
	"math"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	booba "github.com/NimbleMarkets/go-booba"
	tslc "github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

type model struct {
	chart tslc.Model
	step  int
}

func newModel() model {
	now := time.Now().Truncate(time.Second)
	m := model{chart: tslc.New(76, 18,
		tslc.WithTimeRange(now.Add(-time.Minute), now),
		tslc.WithXLabelFormatter(tslc.HourTimeLabelFormatter()),
		tslc.WithYLabelFormatter(func(_ int, v float64) string { return fmt.Sprintf("%.0f", v) }),
	)}
	m.chart.SetDataSetStyle("download", lipgloss.NewStyle().Foreground(lipgloss.Color("10")))
	m.chart.SetDataSetStyle("upload", lipgloss.NewStyle().Foreground(lipgloss.Color("12")))
	// Fit explicitly after each batch so arriving points do not alter Y zoom.
	m.chart.AutoMinY, m.chart.AutoMaxY = false, false
	for i := 60; i >= 0; i-- {
		m.push(now.Add(-time.Duration(i) * time.Second))
	}
	m.refresh(now)
	return m
}

func (m *model) push(t time.Time) {
	download := 80 + 15*math.Sin(float64(m.step)*0.5)
	// The seeded spike leaves the window about ten seconds after startup.
	if m.step%90 == 10 {
		download = 3500
	}
	m.chart.PushDataSet("download", tslc.TimePoint{Time: t, Value: download})
	m.chart.PushDataSet("upload", tslc.TimePoint{Time: t, Value: 35 + 8*math.Cos(float64(m.step)*0.3)})
	m.step++
}

func (m *model) refresh(now time.Time) {
	start := now.Add(-time.Minute)
	// Keep one sample preceding the viewport for interpolation at its edge.
	m.chart.TrimBefore(start.Add(-time.Second))
	m.chart.SetViewTimeRange(start, now)
	m.chart.FitYToViewWithOpts(tslc.FitYOpts{
		DataSets: []string{"download", "upload"}, IncludeZero: true,
	})
	m.chart.DrawBrailleDataSets([]string{"download", "upload"})
}

func (m model) Init() tea.Cmd { return tick() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.chart.Resize(max(20, msg.Width-2), max(8, msg.Height-4))
		m.chart.DrawBrailleDataSets([]string{"download", "upload"})
	case tickMsg:
		now := time.Time(msg).Truncate(time.Second)
		m.push(now)
		m.refresh(now)
		return m, tick()
	}
	return m, nil
}

func (m model) View() tea.View {
	v := tea.NewView(fmt.Sprintf("Transfer speed (MB/s), last 60 seconds | download: green, upload: blue\nY: %.0f..%.0f | spikes expire from the window | q: quit\n%s",
		m.chart.ViewMinY(), m.chart.ViewMaxY(), m.chart.View()))
	v.AltScreen = true
	return v
}

func main() {
	if err := booba.Run(newModel()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
