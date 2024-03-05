package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"
	"github.com/NimbleMarkets/bubbletea-charts/linechart"
	"github.com/NimbleMarkets/bubbletea-charts/linechart/wavelinechart"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const dataSet2 = "dataSet2"

var randomFloat64Point1 canvas.Float64Point
var randomFloat64Point2 canvas.Float64Point

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var graphLineStyle1 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")) // blue

var graphLineStyle2 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("9")) // red

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

var labelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6")) // cyan

type model struct {
	alc1 wavelinechart.Model
	alc2 wavelinechart.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// wavelinechart Clear() resets the canvas
			// and ClearData() resets internal data storage
			m.alc1.ClearAllData()
			m.alc1.Clear()
			m.alc1.DrawXYAxisAndLabel()
			m.alc1.Draw()

			m.alc2.Clear()
			m.alc2.ClearAllData()
			m.alc2.SetDataSetStyle(dataSet2, runes.ArcLineStyle, graphLineStyle2)
			m.alc2.DrawXYAxisAndLabel()
			m.alc2.DrawAll()
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	// generate a random points within the given X,Y value ranges
	dx := m.alc1.MaxX() - m.alc1.MinX()
	dy := m.alc1.MaxY() - m.alc1.MinY()
	xRand1 := rand.Float64()*dx + m.alc1.MinX()
	yRand1 := rand.Float64()*dy + m.alc1.MinY()
	xRand2 := rand.Float64()*dx + m.alc1.MinX()
	yRand2 := rand.Float64()*dy + m.alc1.MinY()
	randomFloat64Point1 = canvas.Float64Point{X: xRand1, Y: yRand1}
	randomFloat64Point2 = canvas.Float64Point{X: xRand2, Y: yRand2}

	//  wavelinechart 1 plots random point 1 to default data set
	m.alc1.Plot(randomFloat64Point1)
	m.alc1.Draw()

	// wavelinechart 2 plots random point 1 to default data set
	// and random point 2 to different data sets
	m.alc2.Plot(randomFloat64Point1)
	m.alc2.PlotDataSet(dataSet2, randomFloat64Point2)
	m.alc2.DrawDataSets([]string{
		wavelinechart.DefaultDataSetName,
		dataSet2,
	}) // can also use DrawAll()
	return m, nil
}

func (m model) View() string {
	t1 := fmt.Sprintf("DataSet1(%0.1f, %0.1f)\n",
		randomFloat64Point1.X, randomFloat64Point1.Y)
	t2 := fmt.Sprintf("DataSet1(%0.1f, %0.1f), DataSet2(%0.1f, %0.1f)\n",
		randomFloat64Point1.X, randomFloat64Point1.Y,
		randomFloat64Point2.X, randomFloat64Point2.Y)
	s := "any key to add randomized point,`r` to clear data, `q/ctrl+c` to quit\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(t1+m.alc1.View()),
		defaultStyle.Render(t2+m.alc2.View()),
	) + "\n"
	return s
}

func main() {
	width := 42
	height := 11
	xStep := 1
	yStep := 1
	minXValue := -5.0
	maxXValue := 10.0
	minYValue := -5.0
	maxYValue := 10.0

	// wavelinechart 1 created with New() and SetStyle()
	alc1 := wavelinechart.New(
		linechart.NewWithStyle(
			width, height,
			minXValue, maxXValue,
			minYValue, maxYValue,
			xStep, yStep,
			axisStyle, labelStyle))
	alc1.SetStyle(runes.ThinLineStyle, graphLineStyle1)

	// wavelinechart 2 created with NewWithStyle()
	// and setting second data set style
	alc2 := wavelinechart.NewWithStyle(
		linechart.NewWithStyle(
			width, height,
			minXValue, maxXValue,
			minYValue, maxYValue,
			xStep, xStep,
			axisStyle, labelStyle),
		runes.ThinLineStyle,
		graphLineStyle1,
	)
	alc2.SetDataSetStyle(dataSet2, runes.ArcLineStyle, graphLineStyle2)

	m := model{alc1, alc2}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
