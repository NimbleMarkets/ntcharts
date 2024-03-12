package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"
	"github.com/NimbleMarkets/bubbletea-charts/linechart/timeserieslinechart"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

const dataSet2 = "dataSet2"

var timePoint1 timeserieslinechart.TimePoint
var timePoint2 timeserieslinechart.TimePoint
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
	tslc1 timeserieslinechart.Model
	tslc2 timeserieslinechart.Model
	zM    *zone.Manager
}

func (m model) Init() tea.Cmd {
	m.tslc1.DrawXYAxisAndLabel()
	m.tslc2.DrawXYAxisAndLabel()
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	addPoint := false
	forwardMsg := false
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.tslc1.ClearAllData()
			m.tslc1.Clear()
			m.tslc1.DrawXYAxisAndLabel()

			m.tslc2.ClearAllData()
			m.tslc2.Clear()
			m.tslc2.DrawXYAxisAndLabel()
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
			if m.zM.Get(m.tslc1.GetZoneID()).InBounds(msg) { // switch to canvas 1 if clicked on it
				m.tslc2.Blur()
				m.tslc1.Focus()
			} else if m.zM.Get(m.tslc2.GetZoneID()).InBounds(msg) { // switch to canvas 2 if clicked on it
				m.tslc1.Blur()
				m.tslc2.Focus()
			} else {
				m.tslc1.Blur()
				m.tslc2.Blur()
			}
		}
		forwardMsg = true
	}
	if addPoint {
		// generate random numbers within the given Y value range
		rangeNumbers := m.tslc2.MaxY() - m.tslc2.MinY()
		randf1 = rand.Float64()*rangeNumbers + m.tslc2.MinY()
		randf2 = rand.Float64()*rangeNumbers + m.tslc2.MinY()

		now := time.Now()
		timePoint1 = timeserieslinechart.TimePoint{Time: now, Value: randf1}
		timePoint2 = timeserieslinechart.TimePoint{Time: now, Value: randf2}

		// timeserieslinechart 1 pushes random value randf1 to default data set
		m.tslc1.Push(timePoint1)
		m.tslc1.Draw()

		// timeserieslinechart 1 pushes random value randf1 to default data set
		// and random value randf2 to second dataset
		m.tslc2.Push(timePoint1)
		m.tslc2.PushDataSet(dataSet2, timePoint2)
		m.tslc2.DrawAll()
	}
	// timeserieslinechart handles mouse events
	if forwardMsg {
		if m.tslc1.Focused() {
			m.tslc1, _ = m.tslc1.Update(msg)
			m.tslc1.DrawAll()
		} else if m.tslc2.Focused() {
			m.tslc2, _ = m.tslc2.Update(msg)
			m.tslc2.DrawAll()
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "any key to push randomized data value,`r` to clear data, `q/ctrl+c` to quit\n"
	s += "mouse wheel scroll to zoom in and out along X axis\n"
	s += "mouse click+drag or arrow keys to move view along X axis while zoomed in\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(fmt.Sprintf("ts:%s, f1:(%.02f)\n", timePoint1.Time.UTC().Format("15:04:05"), randf1)+m.tslc1.View()),
		defaultStyle.Render(fmt.Sprintf("ts:%s, f1:(%.02f) f2:(%.02f)\n", timePoint1.Time.UTC().Format("15:04:05"), randf1, randf2)+m.tslc2.View()),
	) + "\n"
	return m.zM.Scan(s) // call zone Manager.Scan() at root model
}

func main() {
	width := 36
	height := 11
	minYValue := 0.0
	maxYValue := 100.0

	// create new bubblezone Manager to enable mouse support to zoom in and out of chart
	zoneManager := zone.New()

	// timeserieslinechart 1 created with New() and setting options afterwards
	tslc1 := timeserieslinechart.New(width, height)
	tslc1.AxisStyle = axisStyle
	tslc1.LabelStyle = labelStyle
	tslc1.XLabelFormatter = timeserieslinechart.HourTimeLabelFormatter()
	tslc1.UpdateHandler = timeserieslinechart.SecondUpdateHandler(1)
	tslc1.SetYRange(minYValue, maxYValue)                 // set expected Y values (values can be less or greater than what is displayed)
	tslc1.SetViewYRange(minYValue, maxYValue)             // setting display Y values will fail unless set expected Y values first
	tslc1.SetStyles(runes.ThinLineStyle, graphLineStyle1) // graphLineStyle1 replaces linechart rune style
	tslc1.SetZoneManager(zoneManager)

	// timeserieslinechart 2 created with New() using options
	// and setting second data set style
	tslc2 := timeserieslinechart.New(width, height,
		timeserieslinechart.WithYRange(minYValue, maxYValue),
		timeserieslinechart.WithAxesStyles(axisStyle, labelStyle),
		timeserieslinechart.WithStyles(runes.ThinLineStyle, graphLineStyle1), // default data set
		timeserieslinechart.WithDataSetStyles(dataSet2, runes.ArcLineStyle, graphLineStyle2),
	)
	tslc2.XLabelFormatter = timeserieslinechart.HourTimeLabelFormatter()
	tslc2.UpdateHandler = timeserieslinechart.SecondUpdateHandler(1)
	tslc2.SetZoneManager(zoneManager)

	m := model{tslc1, tslc2, zoneManager}
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
