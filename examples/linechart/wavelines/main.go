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
	zone "github.com/lrstanley/bubblezone"
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
	wlc1 wavelinechart.Model
	wlc2 wavelinechart.Model
	zM   *zone.Manager
}

func (m model) Init() tea.Cmd {
	m.wlc1.Draw()
	m.wlc2.Draw()
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	addPoint := false
	forwardMsg := false
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// wavelinechart Clear() resets the canvas
			// and ClearData() resets internal data storage
			m.wlc1.ClearAllData()
			m.wlc1.Clear()
			m.wlc1.DrawXYAxisAndLabel()
			m.wlc1.Draw()

			m.wlc2.Clear()
			m.wlc2.ClearAllData()
			m.wlc2.SetDataSetStyle(dataSet2, runes.ArcLineStyle, graphLineStyle2)
			m.wlc2.DrawXYAxisAndLabel()
			m.wlc2.DrawAll()
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "down", "left", "right":
			forwardMsg = true
		default:
			addPoint = true
		}
	case tea.MouseMsg:
		if msg.Type == tea.MouseLeft {
			if m.zM.Get(m.wlc1.GetZoneID()).InBounds(msg) { // switch to canvas 1 if clicked on it
				m.wlc2.Blur()
				m.wlc1.Focus()
			} else if m.zM.Get(m.wlc2.GetZoneID()).InBounds(msg) { // switch to canvas 2 if clicked on it
				m.wlc1.Blur()
				m.wlc2.Focus()
			} else {
				m.wlc1.Blur()
				m.wlc2.Blur()
			}
		}
		forwardMsg = true
	}
	if addPoint {
		// generate a random points within the given X,Y value ranges
		dx := m.wlc1.MaxX() - m.wlc1.MinX()
		dy := m.wlc1.MaxY() - m.wlc1.MinY()
		xRand1 := rand.Float64()*dx + m.wlc1.MinX()
		yRand1 := rand.Float64()*dy + m.wlc1.MinY()
		xRand2 := rand.Float64()*dx + m.wlc1.MinX()
		yRand2 := rand.Float64()*dy + m.wlc1.MinY()
		randomFloat64Point1 = canvas.Float64Point{X: xRand1, Y: yRand1}
		randomFloat64Point2 = canvas.Float64Point{X: xRand2, Y: yRand2}

		//  wavelinechart 1 plots random point 1 to default data set
		m.wlc1.Plot(randomFloat64Point1)
		m.wlc1.Draw()

		// wavelinechart 2 plots random point 1 to default data set
		// and random point 2 to different data sets
		m.wlc2.Plot(randomFloat64Point1)
		m.wlc2.PlotDataSet(dataSet2, randomFloat64Point2)
		m.wlc2.DrawDataSets([]string{
			dataSet2,
			wavelinechart.DefaultDataSetName,
		}) // can also use DrawAll()
	}

	// wavelinechart handles mouse events
	if forwardMsg {
		if m.wlc1.Focused() {
			m.wlc1, _ = m.wlc1.Update(msg)
			m.wlc1.DrawAll()
		} else if m.wlc2.Focused() {
			m.wlc2, _ = m.wlc2.Update(msg)
			m.wlc2.DrawAll()
		}
	}

	return m, nil
}

func (m model) View() string {
	t1 := fmt.Sprintf("DataSet1(%0.1f, %0.1f)\n",
		randomFloat64Point1.X, randomFloat64Point1.Y)
	t2 := fmt.Sprintf("DataSet1(%0.1f, %0.1f), DataSet2(%0.1f, %0.1f)\n",
		randomFloat64Point1.X, randomFloat64Point1.Y,
		randomFloat64Point2.X, randomFloat64Point2.Y)
	s := "any key to add randomized point,`r` to clear data, `q/ctrl+c` to quit\n"
	s += "mouse wheel scroll to zoom in and out\n"
	s += "mouse click+drag or arrow keys to move view while zoomed in\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(t1+m.wlc1.View()),
		defaultStyle.Render(t2+m.wlc2.View()),
	) + "\n"
	return m.zM.Scan(s) // call zone Manager.Scan() at root model
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

	// create new bubblezone Manager to enable mouse support to zoom in and out of chart
	zoneManager := zone.New()

	// wavelinechart 1 created with New() and SetStyle()
	wlc1 := wavelinechart.New(
		linechart.NewWithStyle(
			width, height,
			minXValue, maxXValue,
			minYValue, maxYValue,
			xStep, yStep,
			axisStyle, labelStyle))
	wlc1.SetStyle(runes.ThinLineStyle, graphLineStyle1)
	wlc1.SetZoneManager(zoneManager)
	wlc1.Focus()

	// wavelinechart 2 created with NewWithStyle()
	// and setting second data set style
	wlc2 := wavelinechart.NewWithStyle(
		linechart.NewWithStyle(
			width, height,
			minXValue, maxXValue,
			minYValue, maxYValue,
			xStep, xStep,
			axisStyle, labelStyle),
		runes.ThinLineStyle,
		graphLineStyle1,
	)
	wlc2.SetDataSetStyle(dataSet2, runes.ArcLineStyle, graphLineStyle2)
	wlc2.SetZoneManager(zoneManager)

	m := model{wlc1, wlc2, zoneManager}
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
