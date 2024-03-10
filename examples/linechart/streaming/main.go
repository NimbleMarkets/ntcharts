package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"
	"github.com/NimbleMarkets/bubbletea-charts/linechart/streamlinechart"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
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
	slc1 streamlinechart.Model
	slc2 streamlinechart.Model
	zM   *zone.Manager
}

func (m model) Init() tea.Cmd {
	m.slc1.DrawXYAxisAndLabel()
	m.slc2.DrawXYAxisAndLabel()
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	addPoint := false
	forwardMsg := false
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.slc1.ClearAllData()
			m.slc1.Clear()
			m.slc1.DrawXYAxisAndLabel()

			m.slc2.ClearAllData()
			m.slc2.Clear()
			m.slc2.DrawXYAxisAndLabel()
			return m, nil
		case "up", "down", "left", "right":
			forwardMsg = true
		case "q", "ctrl+c":
			return m, tea.Quit
		default:
			addPoint = true
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			if m.zM.Get(m.slc1.GetZoneID()).InBounds(msg) { // switch to canvas 1 if clicked on it
				m.slc2.Blur()
				m.slc1.Focus()
			} else if m.zM.Get(m.slc2.GetZoneID()).InBounds(msg) { // switch to canvas 2 if clicked on it
				m.slc1.Blur()
				m.slc2.Focus()
			} else {
				m.slc1.Blur()
				m.slc2.Blur()
			}
		}
		forwardMsg = true
	}
	if addPoint {
		// generate random numbers within the given Y value range
		rangeNumbers := m.slc2.MaxY() - m.slc2.MinY()
		randf1 = rand.Float64()*rangeNumbers + m.slc2.MinY()
		randf2 = rand.Float64()*rangeNumbers + m.slc2.MinY()

		// streamlinechart 1 pushes random value randf1 to default data set
		m.slc1.Push(randf1)
		m.slc1.Draw()

		// streamlinechart 2 pushes randf1 twice to default dataset
		// and randf2 once to second data set
		m.slc2.Push(randf1)
		m.slc2.Push(randf1)
		m.slc2.PushDataSet(dataSet2, randf2)
		m.slc2.DrawAll()
	}
	// streamlinechart handles mouse events
	if forwardMsg {
		if m.slc1.Focused() {
			m.slc1, _ = m.slc1.Update(msg)
			m.slc1.DrawAll()
		} else if m.slc2.Focused() {
			m.slc2, _ = m.slc2.Update(msg)
			m.slc2.DrawAll()
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "any key to push randomized data value,`r` to clear data, `q/ctrl+c` to quit\n"
	s += "mouse wheel scroll to zoom in and out along Y axis\n"
	s += "mouse click+drag or arrow keys to move view along Y axis while zoomed in\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(fmt.Sprintf("DataSet1(%.02f)\n", randf1)+m.slc1.View()),
		defaultStyle.Render(fmt.Sprintf("DataSet1(%.02f)x2, DataSet2(%.02f)\n", randf1, randf2)+m.slc2.View()),
	) + "\n"
	return m.zM.Scan(s) // call zone Manager.Scan() at root model
}

func main() {
	width := 36
	height := 11
	minYValue := -50.0
	maxYValue := 100.0

	// create new bubblezone Manager to enable mouse support to zoom in and out of chart
	zoneManager := zone.New()

	// streamlinechart 1 created with New() and setting options afterwards
	slc1 := streamlinechart.New(width, height, minYValue, maxYValue)
	slc1.AxisStyle = axisStyle
	slc1.LabelStyle = labelStyle
	slc1.SetStyles(runes.ThinLineStyle, graphLineStyle1) // graphLineStyle1 replaces linechart rune style
	slc1.SetZoneManager(zoneManager)

	// streamlinechart 2 created with New() using options
	// and setting second data set style
	slc2 := streamlinechart.New(width, height, minYValue, maxYValue,
		streamlinechart.WithAxesStyles(axisStyle, labelStyle),
		streamlinechart.WithStyles(runes.ArcLineStyle, graphLineStyle1), // graphLineStyle1 replaces linechart rune style
		streamlinechart.WithDataSetStyles(dataSet2, runes.ArcLineStyle, graphLineStyle2),
	)
	slc2.SetZoneManager(zoneManager)

	m := model{slc1, slc2, zoneManager}
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
