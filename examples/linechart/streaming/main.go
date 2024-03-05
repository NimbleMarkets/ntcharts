package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"
	"github.com/NimbleMarkets/bubbletea-charts/linechart"
	"github.com/NimbleMarkets/bubbletea-charts/linechart/streamlinechart"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const dataSet2 = "dataSet2"

var randf1 float64
var randf2 float64

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var graphLineStyle1 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")) // blue

var graphLineStyle2 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("10")) // green

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

var labelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6")) // cyan

type model struct {
	alc1 streamlinechart.Model
	alc2 streamlinechart.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.alc1.ClearAllData()
			m.alc1.Clear()
			m.alc1.DrawXYAxisAndLabel()

			m.alc2.ClearAllData()
			m.alc2.Clear()
			m.alc2.DrawXYAxisAndLabel()
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	// generate random numbers within the given Y value range
	rangeNumbers := m.alc2.MaxY() - m.alc2.MinY()
	randf1 = rand.Float64()*rangeNumbers + m.alc2.MinY()
	randf2 = rand.Float64()*rangeNumbers + m.alc2.MinY()

	// streamlinechart 1 pushes random value randf1 to default data set
	m.alc1.Push(randf1)
	m.alc1.Draw()

	// streamlinechart 2 pushes randf1 twice to default dataset
	// and randf2 once to second data set
	m.alc2.Push(randf1)
	m.alc2.Push(randf1)
	m.alc2.PushDataSet(dataSet2, randf2)
	m.alc2.DrawAll()
	return m, nil
}

func (m model) View() string {
	s := "any key to push randomized data value,`r` to clear data, `q/ctrl+c` to quit\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(fmt.Sprintf("DataSet1(%.02f)\n", randf1)+m.alc1.View()),
		defaultStyle.Render(fmt.Sprintf("DataSet1(%.02f)x2, DataSet2(%.02f)\n", randf1, randf2)+m.alc2.View()),
	) + "\n"
	return s
}

func main() {
	width := 36
	height := 11
	minYValue := -50.0
	maxYValue := 100.0
	xStep := 0 // xStep = 0 means do not display X axis
	yStep := 1

	// streamlinechart 1 created with New() and SetStyle()
	alc1 := streamlinechart.New(
		linechart.NewWithStyle(
			width, height,
			0, 0, // x values not used
			minYValue, maxYValue,
			xStep, yStep,
			axisStyle, labelStyle),
	)
	alc1.SetStyle(runes.ThinLineStyle, graphLineStyle1)

	// wavelinechart 2 created with NewWithStyle()
	// and setting second data set style
	alc2 := streamlinechart.NewWithStyle(
		linechart.NewWithStyle(
			width, height,
			0, 0, // x values not used
			minYValue, maxYValue,
			xStep, yStep,
			axisStyle, labelStyle),
		runes.ArcLineStyle,
		graphLineStyle1,
	)
	alc2.SetDataSetStyle(dataSet2, runes.ArcLineStyle, graphLineStyle2)

	m := model{alc1, alc2}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
