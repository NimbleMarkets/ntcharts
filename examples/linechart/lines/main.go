package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"
	"github.com/NimbleMarkets/bubbletea-charts/linechart"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var randomFloat64Point1 canvas.Float64Point
var randomFloat64Point2 canvas.Float64Point

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var replacedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("5")) // pink

var lineStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")) // blue

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

var labelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6")) // cyan

type model struct {
	lc1 linechart.Model
	lc2 linechart.Model
	lc3 linechart.Model
}

func (m model) Init() tea.Cmd {
	m.lc1.DrawXYAxisAndLabel()
	m.lc2.DrawXYAxisAndLabel()
	m.lc3.DrawXYAxisAndLabel()
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
			m.lc3.Clear()
			m.lc3.DrawXYAxisAndLabel()
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
	xRand2 := rand.Float64()*dx + m.lc1.MinX()
	yRand2 := rand.Float64()*dy + m.lc1.MinY()
	randomFloat64Point1 = canvas.Float64Point{X: xRand1, Y: yRand1}
	randomFloat64Point2 = canvas.Float64Point{X: xRand2, Y: yRand2}

	// linecharts drawing methods with style replaces initialized graph style

	// linechart1 draws line with all runes as 'X' with default lipgloss style
	m.lc1.DrawRuneLineWithStyle(randomFloat64Point1, randomFloat64Point2, 'X', lineStyle)

	// linechart2 draws line using ArcLineStyle
	m.lc2.DrawLineWithStyle(randomFloat64Point1, randomFloat64Point2, runes.ArcLineStyle, lineStyle)

	// linechart3 draws braille line
	m.lc3.DrawBrailleLineWithStyle(randomFloat64Point1, randomFloat64Point2, lineStyle)
	return m, nil
}

func (m model) View() string {
	t := fmt.Sprintf("Drawing lines between the two points: ((%0.1f, %0.1f), (%0.1f, %0.1f))",
		randomFloat64Point1.X, randomFloat64Point1.Y,
		randomFloat64Point2.X, randomFloat64Point2.Y)
	s := "any key to draw randomized line, `r` to reset, `q/ctrl+c` to quit\n"
	s += defaultStyle.Render(t) + "\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render("DrawRuneLine('X')\n"+m.lc1.View()),
		defaultStyle.Render("DrawLine()\n"+m.lc2.View()),
		defaultStyle.Render("DrawBrailleLine()\n"+m.lc3.View()),
	) + "\n"
	return s
}

func main() {
	width := 33
	height := 12
	minXValue := -50.0
	maxXValue := 100.0
	minYValue := -50.0
	maxYValue := 100.0

	lc1 := linechart.New(
		width, height,
		minXValue, maxXValue,
		minYValue, maxYValue,
		linechart.WithXYSteps(1, 1),
		linechart.WithStyles(axisStyle, labelStyle, replacedStyle))
	lc2 := linechart.New(
		width, height,
		minXValue, maxXValue,
		minYValue, maxYValue,
		linechart.WithXYSteps(1, 1),
		linechart.WithStyles(axisStyle, labelStyle, replacedStyle))
	lc3 := linechart.New(
		width, height,
		minXValue, maxXValue,
		minYValue, maxYValue,
		linechart.WithXYSteps(1, 1),
		linechart.WithStyles(axisStyle, labelStyle, replacedStyle))
	m := model{lc1, lc2, lc3}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
