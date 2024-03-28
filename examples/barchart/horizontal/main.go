// ntcharts - Copyright (c) 2024 Neomantra Corp.

package main

import (
	"fmt"
	"os"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/NimbleMarkets/ntcharts/canvas/runes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var selectedBarData barchart.BarData

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

var labelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("63")) // purple

var blockStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("9")) // red

var blockStyle2 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("2")) // green

var blockStyle3 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6")) // cyan

var blockStyle4 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

type model struct {
	b1 barchart.Model
	b2 barchart.Model
	b3 barchart.Model
	lv []barchart.BarData
	zM *zone.Manager
}

func title(m *barchart.Model) string {
	return fmt.Sprintf("Max:%.1f, AutoMax:%t\nBarGap:%d, ShowAxis:%t\n", m.MaxValue(), m.AutoMaxValue, m.BarGap(), m.ShowAxis())
}

func legend(bd barchart.BarData) (r string) {
	r = "Legend\n"
	for _, bv := range bd.Values {
		r += "\n" + bv.Style.Render(fmt.Sprintf("%c %s", runes.FullBlock, bv.Name))
	}
	return
}

func totals(lv []barchart.BarData) (r string) {
	r = "Totals\n"
	for _, bd := range lv {
		var sum float64
		for _, bv := range bd.Values {
			sum += bv.Value
		}
		r += "\n" + fmt.Sprintf("%s %.01f", bd.Label, sum)
	}
	return
}
func selectedData() (r string) {
	r = "Selected\n"
	if len(selectedBarData.Values) == 0 {
		return
	}
	r += selectedBarData.Label
	for _, bv := range selectedBarData.Values {
		r += " " + bv.Style.Render(fmt.Sprintf("%.01f", bv.Value))
	}
	return
}

func (m *model) setBarData(b *barchart.Model, msg tea.MouseMsg) {
	x, y := m.zM.Get(b.ZoneID()).Pos(msg)
	selectedBarData = b.BarDataFromPoint(canvas.Point{x, y})
}

func (m model) Init() tea.Cmd {
	m.b1.Draw()
	m.b2.Draw()
	m.b3.Draw()
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			switch {
			case m.zM.Get(m.b1.ZoneID()).InBounds(msg):
				m.setBarData(&m.b1, msg)
			case m.zM.Get(m.b2.ZoneID()).InBounds(msg):
				m.setBarData(&m.b2, msg)
			case m.zM.Get(m.b3.ZoneID()).InBounds(msg):
				m.setBarData(&m.b3, msg)
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "Same data values are pushed to all horizontal bar charts, `q/ctrl+c` to quit\n"
	s += "Click bar segment to select and display values\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(title(&m.b1)+m.b1.View()),
		defaultStyle.Render(title(&m.b2)+m.b2.View()),
		defaultStyle.Render(title(&m.b3)+m.b3.View()),
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.JoinHorizontal(lipgloss.Top,
				defaultStyle.Render(totals(m.lv)),
				defaultStyle.Render(legend(m.lv[0])),
			),
			defaultStyle.Render(selectedData()),
		),
	)
	return m.zM.Scan(s) // call zone Manager.Scan() at root model
}

func main() {
	width := 23
	height := 12

	v1 := barchart.BarData{
		Label: "A",
		Values: []barchart.BarValue{
			{Name: "Name1", Value: 21.2, Style: blockStyle},
			{Name: "Name2", Value: 10.1, Style: blockStyle2},
			{Name: "Name3", Value: 6.5, Style: blockStyle3},
			{Name: "Name4", Value: 7.7, Style: blockStyle4},
		},
	}
	v2 := barchart.BarData{
		Label: "B",
		Values: []barchart.BarValue{
			{Name: "Name1", Value: 15.1, Style: blockStyle},
			{Name: "Name2", Value: 15.1, Style: blockStyle2},
			{Name: "Name3", Value: 3.3, Style: blockStyle3},
			{Name: "Name4", Value: 7.7, Style: blockStyle4},
		},
	}
	v3 := barchart.BarData{
		Label: "C",
		Values: []barchart.BarValue{
			{Name: "Name1", Value: 13.6, Style: blockStyle},
			{Name: "Name2", Value: 14.1, Style: blockStyle2},
			{Name: "Name3", Value: 4.4, Style: blockStyle3},
			{Name: "Name4", Value: 4.4, Style: blockStyle4},
		},
	}
	v4 := barchart.BarData{
		Label: "D",
		Values: []barchart.BarValue{
			{Name: "Name1", Value: 13.1, Style: blockStyle},
			{Name: "Name2", Value: 11.1, Style: blockStyle2},
			{Name: "Name3", Value: 10.9, Style: blockStyle3},
			{Name: "Name4", Value: 9.8, Style: blockStyle4},
		},
	}
	values := []barchart.BarData{v1, v2, v3, v4}

	// create new bubblezone Manager to enable mouse support to zoom in and out of chart
	zoneManager := zone.New()

	// all barcharts contain the same values
	// different options are displayed on the screen and below
	// and first barchart has axis and label styles
	m := model{
		barchart.New(width, height,
			barchart.WithZoneManager(zoneManager),
			barchart.WithDataSet(values),
			barchart.WithStyles(axisStyle, labelStyle),
			barchart.WithHorizontalBars()),
		barchart.New(width, height,
			barchart.WithZoneManager(zoneManager),
			barchart.WithDataSet(values),
			barchart.WithMaxValue(100.0),
			barchart.WithNoAutoMaxValue(),
			barchart.WithNoAxis(),
			barchart.WithHorizontalBars()),
		barchart.New(width, height,
			barchart.WithZoneManager(zoneManager),
			barchart.WithDataSet(values),
			barchart.WithMaxValue(30.0),
			barchart.WithNoAutoMaxValue(),
			barchart.WithBarGap(0),
			barchart.WithHorizontalBars()),
		values,
		zoneManager,
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
