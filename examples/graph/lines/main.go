package main

import (
	"fmt"
	"os"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/graph"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var arcLineStyle1 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")) // blue

var arcLineStyle2 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("5")) // pink

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

type model struct {
	c1      canvas.Model
	c2      canvas.Model
	c3      canvas.Model
	coords1 []int
	coords2 []int

	cursor canvas.Point
	zM     *zone.Manager
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			m.cursor.Y--
			if m.cursor.Y < 0 {
				m.cursor.Y = 0
			}
		case "down":
			m.cursor.Y++
			if m.cursor.Y > m.c1.Height()-1 {
				m.cursor.Y = m.c1.Height() - 1
			}
		case "right":
			m.cursor.X++
			if m.cursor.X > m.c1.Width()-1 {
				m.cursor.X = m.c1.Width() - 1
			}
		case "left":
			m.cursor.X--
			if m.cursor.X < 0 {
				m.cursor.X = 0
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.MouseMsg:
		// move all cursors at the same time on any mouse action
		if m.zM.Get(m.c1.GetZoneID()).InBounds(msg) {
			m.cursor.X, m.cursor.Y = m.zM.Get(m.c1.GetZoneID()).Pos(msg)
		} else if m.zM.Get(m.c2.GetZoneID()).InBounds(msg) {
			m.cursor.X, m.cursor.Y = m.zM.Get(m.c2.GetZoneID()).Pos(msg)
		} else if m.zM.Get(m.c3.GetZoneID()).InBounds(msg) {
			m.cursor.X, m.cursor.Y = m.zM.Get(m.c3.GetZoneID()).Pos(msg)
		}
	}
	startX := 1 // start drawing sequence at X = 1 for demo, usually start at Y axis

	// draw line sequence data set 1 with thin lines
	m.c1.Clear()
	graph.DrawXYAxis(&m.c1, m.cursor, axisStyle)
	graph.DrawLineSequence(&m.c1, true, startX, m.coords1, runes.ThinLineStyle, arcLineStyle1)

	// draw line sequence data set 2 with arc lines
	m.c2.Clear()
	graph.DrawXYAxisDown(&m.c2, m.cursor, axisStyle)
	graph.DrawLineSequence(&m.c2, true, startX, m.coords2, runes.ArcLineStyle, arcLineStyle2)

	// draw line sequence data set 2 on top of data set 1
	m.c3.Clear()
	graph.DrawXYAxisAll(&m.c3, m.cursor, axisStyle)
	graph.DrawLineSequence(&m.c3, true, startX, m.coords1, runes.ThinLineStyle, arcLineStyle1)
	graph.DrawLineSequence(&m.c3, true, startX, m.coords2, runes.ArcLineStyle, arcLineStyle2)

	return m, nil
}

func (m model) View() string {
	s := "arrow keys or mouse to move origin around, `q/ctrl+c` to quit\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(m.c1.View()),
		defaultStyle.Render(m.c2.View()),
		defaultStyle.Render(m.c3.View()),
	) + "\n"
	return m.zM.Scan(s) // required if canvas has set bubblezone manager
}

func main() {
	w := 20
	h := 11
	yAxis := 0
	xAxis := 10
	z := zone.New() // bubblezone used to enable mouse functionality
	c1 := canvas.New(w, h, canvas.WithZoneManager(z))
	c2 := canvas.New(w, h, canvas.WithZoneManager(z))
	c3 := canvas.New(w, h, canvas.WithZoneManager(z))

	// canvas 1 draws arc lines for coordinates set 1
	// canvas 2 draws arc lines for coordinates set 2
	// canvas 3 draws arc lines for coordinates set 2 on top of coordinates set 1
	// CanvasYCoordinates is used to convert (X,Y) coordinates where (0,0) is on the bottom left
	// to the coordinates system used by canvas by passing in where the X axis would appear in the canvas

	graphYCoords1 := []int{7, 3, 0, 6, 5, 9, 1, 2, 4, 8, 8, 9, 4, 0, 0, 5} // Cartesian coordinates with (0,0) as bottom left
	canvasYCoords1 := canvas.CanvasYCoordinates(xAxis, graphYCoords1)      // Canvas coordinates with (0,0) as top left

	graphYCoords2 := []int{9, 3, 1, 8, 9, 7, 2, 4, 5, 4, 0, 4, 0, 5, 7, 8, 5} // Cartesian coordinates with (0,0) as bottom left
	canvasYCoords2 := canvas.CanvasYCoordinates(xAxis, graphYCoords2)         // Canvas coordinates with (0,0) as top left

	m := model{c1, c2, c3, canvasYCoords1, canvasYCoords2, canvas.Point{yAxis, xAxis}, z}
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
