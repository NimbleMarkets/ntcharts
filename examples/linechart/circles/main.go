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
var radiusFloat64 float64

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var lineStyle = lipgloss.NewStyle().
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

	// draw a line between two random points within X,Y value ranges
	dx := m.lc1.MaxX() - m.lc1.MinX()
	dy := m.lc1.MaxY() - m.lc1.MinY()
	xRand1 := rand.Float64()*dx + m.lc1.MinX()
	yRand1 := rand.Float64()*dy + m.lc1.MinY()
	yRand2 := (rand.Float64()*dy + m.lc1.MinY()) / 2
	randomFloat64Point = canvas.Float64Point{X: xRand1, Y: yRand1}
	radiusFloat64 = yRand2
	if radiusFloat64 < 0 {
		radiusFloat64 *= -1
	}

	// linechart1 draws circle with all runes as 'X'
	m.lc1.DrawRuneCircle(randomFloat64Point, radiusFloat64, 'X', lineStyle)

	// linechart2 draws braille circle
	m.lc2.DrawBrailleCircle(randomFloat64Point, radiusFloat64, lineStyle)
	return m, nil
}

func (m model) View() string {
	t := fmt.Sprintf("Drawing circles at center: ((%0.1f, %0.1f), radius: %0.1f",
		randomFloat64Point.X, randomFloat64Point.Y,
		radiusFloat64)
	s := "any key to draw randomized line, `r` to reset, `q/ctrl+c` to quit\n"
	s += defaultStyle.Render(t) + "\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render("DrawRuneCircle('X')\n"+m.lc1.View()),
		defaultStyle.Render("DrawBrailleCircle()\n"+m.lc2.View()),
	) + "\n"
	return s
}

func main() {
	width := 33
	height := 12
	minXValue := -50.0
	maxXValue := 100.0
	minYValue := -50.0
	maxYValue := 50.0

	lc1 := linechart.NewWithStyle(
		width, height,
		minXValue, maxXValue,
		minYValue, maxYValue,
		1, 1,
		axisStyle, labelStyle)
	lc2 := linechart.NewWithStyle(
		width, height,
		minXValue, maxXValue,
		minYValue, maxYValue,
		1, 1,
		axisStyle, labelStyle)

	m := model{lc1, lc2}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
