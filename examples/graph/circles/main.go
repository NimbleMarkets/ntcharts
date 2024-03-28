// ntcharts - Copyright (c) 2024 Neomantra Corp.

package main

import (
	"fmt"
	"os"

	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/NimbleMarkets/ntcharts/canvas/graph"
	"github.com/NimbleMarkets/ntcharts/canvas/runes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var circleStyle1 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")) // blue

var circleStyle2 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("5")) // pink

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

type model struct {
	c1     canvas.Model
	c2     canvas.Model
	c3     canvas.Model
	cursor canvas.Point
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			m.cursor.Y--
			if m.cursor.Y < 0 {
				m.cursor.Y = 0
			}
		case "down":
			m.cursor.Y++
			if m.cursor.Y > m.c1.Height()-1 {
				m.cursor.Y = m.c1.Height() - 1
			}
		case "right":
			m.cursor.X++
			if m.cursor.X > m.c1.Width()-1 {
				m.cursor.X = m.c1.Width() - 1
			}
		case "left":
			m.cursor.X--
			if m.cursor.X < 0 {
				m.cursor.X = 0
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	// circlePoints := graph.GetCirclePoints(m.cursor, 3)
	// startX := 1 // start drawing sequence at X = 1 for demo, usually start at Y axis

	// draw circle around cursor
	m.c1.Clear()
	graph.DrawXYAxis(&m.c1, m.cursor, axisStyle)
	for _, p := range graph.GetCirclePoints(m.cursor, 3) {
		m.c1.SetCell(p, canvas.NewCellWithStyle(runes.FullBlock, circleStyle1))
	}

	// draw circle that is filled around the cursor
	m.c2.Clear()
	graph.DrawXYAxisDown(&m.c2, m.cursor, axisStyle)
	for _, p := range graph.GetFullCirclePoints(m.cursor, 3) {
		m.c2.SetCell(p, canvas.NewCellWithStyle(runes.FullBlock, circleStyle1))
	}

	// draw two circles left and right of the cursor
	// such that the overlapping areas are empty
	m.c3.Clear()
	graph.DrawXYAxisAll(&m.c3, m.cursor, axisStyle)
	leftCircle := graph.GetFullCirclePoints(m.cursor.Add(canvas.Point{X: -1, Y: -1}), 3)
	rightCircle := graph.GetFullCirclePoints(m.cursor.Add(canvas.Point{X: 1, Y: 1}), 3)
	leftSet := map[canvas.Point]struct{}{}
	rightSet := map[canvas.Point]struct{}{}
	for _, p := range leftCircle {
		leftSet[p] = struct{}{}
	}
	for _, p := range rightCircle {
		rightSet[p] = struct{}{}
	}
	// only draw runes if a point from one circle is not in the other
	for _, p := range leftCircle {
		if _, ok := rightSet[p]; !ok {
			m.c3.SetCell(p, canvas.NewCellWithStyle(runes.FullBlock, circleStyle1))
		}
	}
	for _, p := range rightCircle {
		if _, ok := leftSet[p]; !ok {
			m.c3.SetCell(p, canvas.NewCellWithStyle(runes.FullBlock, circleStyle2))
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "arrow keys to move origin around, `q/ctrl+c` to quit\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(m.c1.View()),
		defaultStyle.Render(m.c2.View()),
		defaultStyle.Render(m.c3.View()),
	) + "\n"
	return s
}

func main() {
	w := 20
	h := 11
	c1 := canvas.New(w, h)
	c2 := canvas.New(w, h)
	c3 := canvas.New(w, h)

	// canvas 1 draws a circle around the cursor that is not filled
	// canvas 2 draws a circle around the cursor that is filled
	// canvas 3 draws two filled circles around the cursor
	// with the intersection of the circles not filled

	m := model{c1, c2, c3, canvas.Point{w / 2, h / 2}}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
