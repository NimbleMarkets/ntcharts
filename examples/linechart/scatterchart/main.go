package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/linechart"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var randomFloat64Point canvas.Float64Point

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var graphStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")) // blue

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

var labelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6")) // cyan

type model struct {
	lc1 linechart.Model
	lc2 linechart.Model
}

func (m model) Init() tea.Cmd {
	m.lc1.DrawXYAxisAndLabel()
	m.lc2.DrawXYAxisAndLabel()
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.lc1.Clear()
			m.lc1.DrawXYAxisAndLabel()
			m.lc2.Clear()
			m.lc2.DrawXYAxisAndLabel()
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	// draw a random point within X,Y value ranges onto the graph
	dx := m.lc1.MaxX() - m.lc1.MinX()
	dy := m.lc1.MaxY() - m.lc1.MinY()
	xRand := rand.Float64()*dx + m.lc1.MinX()
	yRand := rand.Float64()*dy + m.lc1.MinY()
	randomFloat64Point = canvas.Float64Point{X: xRand, Y: yRand}

	// linechart1 draws point as 'X'
	m.lc1.DrawRune(randomFloat64Point, 'X')

	// linecharte2 draws point as braille rune
	// (a line between the two identical points is a single point)
	m.lc2.DrawBrailleLine(randomFloat64Point, randomFloat64Point)
	return m, nil
}

func (m model) View() string {
	s := "any key to draw randomized point, `r` to reset, `q/ctrl+c` to quit\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(fmt.Sprintf("DrawRune(%0.1f, %0.1f)\n", randomFloat64Point.X, randomFloat64Point.Y)+m.lc1.View()),
		defaultStyle.Render(fmt.Sprintf("DrawBrailleLine(%0.1f, %0.1f)\n", randomFloat64Point.X, randomFloat64Point.Y)+m.lc2.View()),
	) + "\n"
	return s
}

func main() {
	width := 40
	height := 12
	minXValue := -50.0
	maxXValue := 100.0
	minYValue := -50.0
	maxYValue := 100.0

	// linechart1 draws a 'X' on a randomized (X,Y) coordinate
	lc1 := linechart.New(
		width, height,
		minXValue, maxXValue,
		minYValue, maxYValue,
		linechart.WithXYSteps(1, 1),
		linechart.WithStyles(axisStyle, labelStyle, graphStyle))

	// linechart2 draws a braille rune on a randomized (X,Y) coordinate
	lc2 := linechart.New(
		width, height,
		minXValue, maxXValue,
		minYValue, maxYValue,
		linechart.WithXYSteps(1, 1),
		linechart.WithStyles(axisStyle, labelStyle, graphStyle))

	m := model{lc1, lc2}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
