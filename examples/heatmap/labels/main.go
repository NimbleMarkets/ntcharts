// ntcharts - Copyright (c) 2026 Neomantra Corp.

// Labels shows a heatmap drawn as a grid of filled cells with row and column
// names: visits by day of the week and time of day.
//
// Press `l` to switch the labels off, in which case the cells fill the whole
// chart, and `-` / `+` to resize the chart and see labels drop out when the
// cells get too small to name.
package main

import (
	"fmt"
	"image/color"
	"math"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	booba "github.com/NimbleMarkets/go-booba"
	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
)

var (
	days  = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	hours = []string{"00", "03", "06", "09", "12", "15", "18", "21"}
)

const (
	minW, maxW, stepW = 16, 96, 8
	minH, maxH, stepH = 5, 25, 2
)

// visits returns the value of the cell for day d and time-of-day bucket h:
// a workday peak around midday and an evening peak at the weekend.
func visits(d, h int) float64 {
	weekend := d >= 5
	peak := 4.0 // bucket of the busiest time
	if weekend {
		peak = 6
	}
	scale := 100.0
	if weekend {
		scale = 70
	}
	// a little day-to-day and hour-to-hour variation, so rows differ
	wobble := 0.82 + 0.18*math.Sin(float64(d*5+h*2))
	return scale * wobble * math.Exp(-math.Pow(float64(h)-peak, 2)/4.5)
}

// colorScale is a ramp from deep blue through green to yellow.
func colorScale() []color.Color {
	stops := [][3]float64{{12, 20, 90}, {20, 130, 140}, {120, 200, 90}, {250, 235, 90}}
	var scale []color.Color
	for i := 0; i < 24; i++ {
		t := float64(i) / 23 * float64(len(stops)-1)
		lo := min(int(t), len(stops)-2)
		f := t - float64(lo)
		var c [3]int
		for k := range c {
			c[k] = int(stops[lo][k] + (stops[lo+1][k]-stops[lo][k])*f)
		}
		scale = append(scale, lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2])))
	}
	return scale
}

type model struct {
	w, h   int
	labels bool
}

// chart builds the heatmap for the current size and label setting.
//
// Cells are an index grid: column x and row y are whole numbers and each is
// drawn as a block (WithCellSize). The model counts rows up from the bottom,
// so to read Mon at the top the rows, and their labels, are mirrored.
func (m model) chart() heatmap.Model {
	opts := []heatmap.Option{
		heatmap.WithColorScale(colorScale()),
		heatmap.WithValueRange(0, 100),
		heatmap.WithCellSize(1, 1),
	}
	hm := heatmap.New(m.w, m.h, opts...)
	if m.labels {
		rowLabels := make([]string, len(days))
		for i, d := range days {
			rowLabels[len(days)-1-i] = d
		}
		hm.SetXLabels(hours)
		hm.SetYLabels(rowLabels)
	} else {
		// no labels: reserve no margin or label rows, and fix the grid range
		hm.SetXStep(0)
		hm.SetYStep(0)
		hm.SetXYRange(-0.5, float64(len(hours))-0.5, -0.5, float64(len(days))-0.5)
		hm.SetViewXYRange(-0.5, float64(len(hours))-0.5, -0.5, float64(len(days))-0.5)
	}
	for d := range days {
		for h := range hours {
			hm.Push(heatmap.NewHeatPointInt(h, len(days)-1-d, visits(d, h)))
		}
	}
	hm.Draw()
	return hm
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "l":
			m.labels = !m.labels
		case "+", "=":
			m.w, m.h = min(m.w+stepW, maxW), min(m.h+stepH, maxH)
		case "-", "_":
			m.w, m.h = max(m.w-stepW, minW), max(m.h-stepH, minH)
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	state := "on"
	if !m.labels {
		state = "off"
	}
	hm := m.chart()
	s := fmt.Sprintf("Visits by day and time   labels: %s   size: %dx%d\n", state, m.w, m.h)
	s += "(l: toggle labels, +/-: resize, q: quit)\n\n"
	s += hm.View()
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func main() {
	// booba.Run is a tea.Program substitute that dispatches to native Bubble Tea
	// or the WASM/ghostty-web bridge depending on build target.
	// See https://github.com/NimbleMarkets/go-booba-example.
	if err := booba.Run(model{w: 48, h: 12, labels: true}); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
