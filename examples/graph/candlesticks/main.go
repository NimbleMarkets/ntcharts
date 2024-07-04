// ntcharts - Copyright (c) 2024 Neomantra Corp.

package main

import (
	"fmt"
	"os"

	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/NimbleMarkets/ntcharts/canvas/graph"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var candleStyle1 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")) // blue

var candleStyle2 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("5")) // pink

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

type model struct {
	c1     canvas.Model
	c2     canvas.Model
	cursor canvas.Point
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
	}

	// draw candlesticks with bodies and wicks
	m.c1.Clear()
	graph.DrawXYAxis(&m.c1, m.cursor, axisStyle)
	graph.DrawCandlestickBottomToTop(&m.c1, m.cursor.Add(canvas.Point{X: 1, Y: -1}), .3, 1.3, 2., 6.3, candleStyle1)
	graph.DrawCandlestickBottomToTop(&m.c1, m.cursor.Add(canvas.Point{X: 2, Y: -1}), .3, 1.6, 4, 6.3, candleStyle1)
	graph.DrawCandlestickBottomToTop(&m.c1, m.cursor.Add(canvas.Point{X: 3, Y: -1}), .6, 1.3, 4, 6.3, candleStyle1)
	graph.DrawCandlestickBottomToTop(&m.c1, m.cursor.Add(canvas.Point{X: 4, Y: -1}), .6, 1.6, 4, 6.3, candleStyle1)

	graph.DrawCandlestickBottomToTop(&m.c1, m.cursor.Add(canvas.Point{X: 6, Y: -1}), 1.6, 1.6, 4, 4, candleStyle1)
	graph.DrawCandlestickBottomToTop(&m.c1, m.cursor.Add(canvas.Point{X: 7, Y: -1}), 1.6, 2.6, 4, 4, candleStyle1)
	graph.DrawCandlestickBottomToTop(&m.c1, m.cursor.Add(canvas.Point{X: 8, Y: -1}), 1.6, 2.3, 4, 5, candleStyle1)
	graph.DrawCandlestickBottomToTop(&m.c1, m.cursor.Add(canvas.Point{X: 9, Y: -1}), 1.3, 4, 4.3, 5, candleStyle1)
	graph.DrawCandlestickBottomToTop(&m.c1, m.cursor.Add(canvas.Point{X: 10, Y: -1}), 1.6, 4, 4.6, 5, candleStyle1)

	// draw candlesticks with only 1 wick direction
	m.c2.Clear()
	graph.DrawXYAxis(&m.c2, m.cursor, axisStyle)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 1, Y: -1}), .3, .3, .3, .3, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 2, Y: -1}), .3, .3, .3, .6, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 3, Y: -1}), .3, .3, .6, .6, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 4, Y: -1}), .3, .6, .6, .6, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 5, Y: -1}), .6, .6, .6, .6, candleStyle2)

	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 7, Y: -1}), .3, .3, .3, .3, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 8, Y: -1}), .3, .3, .3, 2.3, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 9, Y: -1}), .3, .3, 2.3, 2.3, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 10, Y: -1}), .3, 2.3, 2.3, 2.3, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 11, Y: -1}), 2.3, 2.3, 2.3, 2.3, candleStyle2)

	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 13, Y: -1}), .3, .3, .3, .3, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 14, Y: -1}), .3, .3, .3, 2.6, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 15, Y: -1}), .3, .3, 2.6, 2.6, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 16, Y: -1}), .3, 2.6, 2.6, 2.6, candleStyle2)
	graph.DrawCandlestickBottomToTop(&m.c2, m.cursor.Add(canvas.Point{X: 17, Y: -1}), 2.6, 2.6, 2.6, 2.6, candleStyle2)
	return m, nil
}

func (m model) View() string {
	s := "arrow keys to move origin around, `q/ctrl+c` to quit\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(m.c1.View()),
		defaultStyle.Render(m.c2.View()),
	) + "\n"
	return s
}

func main() {
	w := 20
	h := 11
	c1 := canvas.New(w, h)
	c2 := canvas.New(w, h)

	// canvas shows different color style and usage of DrawCandlestickBottomToTop() function
	m := model{c1, c2, canvas.Point{0, h - 1}}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
