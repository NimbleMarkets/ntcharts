// bubbletea-charts - Copyright (c) 2024 Neomantra Corp.

package main

import (
	"fmt"
	"os"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/graph"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var blockStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6")) // cyan

var blockStyle2 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("9")) // red

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("5")) // pink

type model struct {
	c1 canvas.Model
	c2 canvas.Model
	c3 canvas.Model

	cursor canvas.Point
}

func (m model) Init() tea.Cmd {
	columnLens1 := []float64{1, 1.891, 2, 2.694, 1.561, 2.109, 0.889, 6.836, 0, 12, 6.708, 8.943, 1.066, 2.55, 8.242, 3.513, 9.768, 9.989, 4.372}
	columnLens2 := []float64{2.907, 1.382, 7.086, 5.401, 8.561, 4.848, 2.398, 4.302, 8.822, 1.149, 0, 6.733, 2.805, 5, 7.138, 10.385, 2, 3.707, 9.529, 4.272}

	// draw data set 1
	m.c1.Clear()
	graph.DrawXYAxisLeft(&m.c1, m.cursor, axisStyle) // demo X axis extending left
	graph.DrawColumns(&m.c1, m.cursor.Add(canvas.Point{1, -1}), columnLens1, blockStyle)

	// draw data set 2
	m.c2.Clear()
	graph.DrawXYAxisLeft(&m.c2, m.cursor, axisStyle)
	graph.DrawColumns(&m.c2, m.cursor.Add(canvas.Point{1, -1}), columnLens2, blockStyle2)

	// draw data set 2 on top of data set 1
	m.c3.Clear()
	graph.DrawXYAxisLeft(&m.c3, m.cursor, axisStyle)
	graph.DrawColumns(&m.c3, m.cursor.Add(canvas.Point{1, -1}), columnLens1, blockStyle)
	graph.DrawColumns(&m.c3, m.cursor.Add(canvas.Point{1, -1}), columnLens2, blockStyle2)
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	m.c1, _ = m.c1.Update(msg)
	m.c2, _ = m.c2.Update(msg)
	m.c3, _ = m.c3.Update(msg)
	return m, nil
}

func (m model) View() string {
	s := "arrow keys to move canvas cursor around, `q/ctrl+c` to quit\n"
	s += "columns of same height will replace existing columns,\n"
	s += "even if previous column top block was taller\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render("Data Set 1\n"+m.c1.View()),
		defaultStyle.Render("Data Set 2\n"+m.c2.View()),
		defaultStyle.Render("Set 2 over Set 1\n"+m.c3.View()),
	) + "\n"
	return s
}

func main() {
	cWidth := 100
	cHeight := 100
	vWidth := 20
	vHeight := 11
	yAxis := 0
	xAxis := 10
	c1 := canvas.New(cWidth, cHeight, canvas.WithViewWidth(vWidth), canvas.WithViewHeight(vHeight), canvas.WithFocus())
	c3 := canvas.New(cWidth, cHeight, canvas.WithViewWidth(vWidth), canvas.WithViewHeight(vHeight), canvas.WithFocus())
	c2 := canvas.New(cWidth, cHeight, canvas.WithViewWidth(vWidth), canvas.WithViewHeight(vHeight), canvas.WithFocus())

	// canvas 1 shows columns set 1
	// canvas 2 draws columns set 2
	// canvas 3 draws columns set 2 on top of columns set 1
	// columns will replace existing ones if the total rune heights are the same

	m := model{c1, c2, c3, canvas.Point{yAxis, xAxis}}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
