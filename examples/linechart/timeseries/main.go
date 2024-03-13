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
	tslc3 timeserieslinechart.Model
	tslc4 timeserieslinechart.Model
	zM    *zone.Manager
}

func (m model) Init() tea.Cmd {
	m.tslc1.DrawXYAxisAndLabel()
	m.tslc2.DrawXYAxisAndLabel()
	m.tslc3.DrawXYAxisAndLabel()
	m.tslc4.DrawXYAxisAndLabel()
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

			m.tslc3.ClearAllData()
			m.tslc3.Clear()
			m.tslc3.DrawXYAxisAndLabel()

			m.tslc4.ClearAllData()
			m.tslc4.Clear()
			m.tslc4.DrawXYAxisAndLabel()
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
			m.tslc1.Blur()
			m.tslc2.Blur()
			m.tslc3.Blur()
			m.tslc4.Blur()

			// switch to whichever canvas was clicked on
			switch {
			case m.zM.Get(m.tslc1.GetZoneID()).InBounds(msg):
				m.tslc1.Focus()
			case m.zM.Get(m.tslc2.GetZoneID()).InBounds(msg):
				m.tslc2.Focus()
			case m.zM.Get(m.tslc3.GetZoneID()).InBounds(msg):
				m.tslc3.Focus()
			case m.zM.Get(m.tslc4.GetZoneID()).InBounds(msg):
				m.tslc4.Focus()
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

		// timeserieslinechart 1 and 3 pushes random value randf1 to default data set
		m.tslc1.Push(timePoint1)
		m.tslc3.Push(timePoint1)

		m.tslc1.Draw()
		m.tslc3.DrawBraille()

		// timeserieslinechart 2 and 4 pushes random value randf1 to default data set
		// and random value randf2 to second dataset
		m.tslc2.Push(timePoint1)
		m.tslc4.Push(timePoint1)
		m.tslc2.PushDataSet(dataSet2, timePoint2)
		m.tslc4.PushDataSet(dataSet2, timePoint2)
		m.tslc2.DrawAll()
		m.tslc4.DrawBrailleAll()
	}
	// timeserieslinechart handles mouse events
	if forwardMsg {
		switch {
		case m.tslc1.Focused():
			m.tslc1, _ = m.tslc1.Update(msg)
			m.tslc1.Draw()
		case m.tslc2.Focused():
			m.tslc2, _ = m.tslc2.Update(msg)
			m.tslc2.DrawAll()
		case m.tslc3.Focused():
			m.tslc3, _ = m.tslc3.Update(msg)
			m.tslc3.DrawBraille()
		case m.tslc4.Focused():
			m.tslc4, _ = m.tslc4.Update(msg)
			m.tslc4.DrawBrailleAll()
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "any key to push randomized data value,`r` to clear data, `q/ctrl+c` to quit\n"
	s += "mouse wheel scroll to zoom in and out along X axis\n"
	s += "mouse click+drag or arrow keys to move view along X axis while zoomed in\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left,
			defaultStyle.Render(fmt.Sprintf("ts:%s, f1:(%.02f)\n", timePoint1.Time.UTC().Format("15:04:05"), randf1)+m.tslc1.View()),
			defaultStyle.Render(fmt.Sprintf("ts:%s, f1:(%.02f)\n", timePoint1.Time.UTC().Format("15:04:05"), randf1)+m.tslc3.View()),
		),
		lipgloss.JoinVertical(lipgloss.Left,
			defaultStyle.Render(fmt.Sprintf("ts:%s, f1:(%.02f) f2:(%.02f)\n", timePoint1.Time.UTC().Format("15:04:05"), randf1, randf2)+m.tslc2.View()),
			defaultStyle.Render(fmt.Sprintf("ts:%s, f1:(%.02f) f2:(%.02f)\n", timePoint1.Time.UTC().Format("15:04:05"), randf1, randf2)+m.tslc4.View()),
		),
	) + "\n"
	return m.zM.Scan(s) // call zone Manager.Scan() at root model
}

func main() {
	width := 36
	height := 8
	minYValue := 0.0
	maxYValue := 100.0

	// timeserieslinecharts creates line charts starting with time as time.Now()
	// There are two sets of charts, one show regular lines and one showing braille lines
	// Pressing keys will insert random Y value data into chart with time.Now() (when key was pressed)

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
		timeserieslinechart.WithUpdateHandler(timeserieslinechart.SecondUpdateHandler(1)),
		timeserieslinechart.WithXLabelFormatter(timeserieslinechart.HourTimeLabelFormatter()), // replace default Date with Hour formatter
	)
	tslc2.SetZoneManager(zoneManager)

	//timeserieslinechart 3 and 4 are copies of 1 and 2 respectively
	tslc3 := timeserieslinechart.New(width, height)
	tslc3.AxisStyle = axisStyle
	tslc3.LabelStyle = labelStyle
	tslc3.XLabelFormatter = timeserieslinechart.HourTimeLabelFormatter()
	tslc3.UpdateHandler = timeserieslinechart.SecondUpdateHandler(1)
	tslc3.SetYRange(minYValue, maxYValue)                 // set expected Y values (values can be less or greater than what is displayed)
	tslc3.SetViewYRange(minYValue, maxYValue)             // setting display Y values will fail unless set expected Y values first
	tslc3.SetStyles(runes.ThinLineStyle, graphLineStyle1) // graphLineStyle1 replaces linechart rune style
	tslc3.SetZoneManager(zoneManager)

	tslc4 := timeserieslinechart.New(width, height,
		timeserieslinechart.WithYRange(minYValue, maxYValue),
		timeserieslinechart.WithAxesStyles(axisStyle, labelStyle),
		timeserieslinechart.WithStyles(runes.ThinLineStyle, graphLineStyle1), // default data set
		timeserieslinechart.WithDataSetStyles(dataSet2, runes.ArcLineStyle, graphLineStyle2),
		timeserieslinechart.WithUpdateHandler(timeserieslinechart.SecondUpdateHandler(1)),
		timeserieslinechart.WithXLabelFormatter(timeserieslinechart.HourTimeLabelFormatter()), // replace default Date with Hour formatter
	)
	tslc4.SetZoneManager(zoneManager)

	m := model{tslc1, tslc2, tslc3, tslc4, zoneManager}
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
