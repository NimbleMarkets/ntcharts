// bubbletea-charts - Copyright (c) 2024 Neomantra Corp.

package main

import (
	"fmt"
	"os"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/graph"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var lineStyle1 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")) // blue

var lineStyle2 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("5")) // pink

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

type model struct {
	c1     canvas.Model
	c2     canvas.Model
	c3     canvas.Model
	center canvas.Point
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

	// braille pattern dots are 4 high and 2 wide
	brailleDotsHigh := 4
	brailleDotsWide := 2

	// canvas 1 draws 2x1 full braille pattern runes diagonal from the axes
	grid1W := brailleDotsWide * 2
	grid1H := brailleDotsHigh * 1
	topRight := canvas.Point{X: 1, Y: -1}
	topLeft := canvas.Point{X: -2, Y: -1}
	bottomRight := canvas.Point{X: 1, Y: 1}
	bottomLeft := canvas.Point{X: -2, Y: 1}
	grid1 := runes.NewPatternDotsGrid(grid1W, grid1H)
	for y := 0; y < grid1H; y++ {
		for x := 0; x < grid1W; x++ {
			grid1.Set(x, y)
		}
	}
	pattern1 := grid1.BraillePatterns()
	m.c1.Clear()
	graph.DrawXYAxisAll(&m.c1, m.cursor, axisStyle)
	graph.DrawBraillePatterns(&m.c1, m.cursor.Add(topRight), pattern1, lineStyle1)
	graph.DrawBraillePatterns(&m.c1, m.cursor.Add(topLeft), pattern1, lineStyle1)
	graph.DrawBraillePatterns(&m.c1, m.cursor.Add(bottomRight), pattern1, lineStyle1)
	graph.DrawBraillePatterns(&m.c1, m.cursor.Add(bottomLeft), pattern1, lineStyle1)

	// canvas 2 draws a 3x3 thin braille pattern rune box onto canvas at the center
	offset2 := canvas.Point{X: -1, Y: -1}
	grid2W := brailleDotsWide * 3
	grid2H := brailleDotsHigh * 3
	grid2 := runes.NewPatternDotsGrid(grid2W, grid2H)
	for y := 0; y < grid2H; y++ {
		for x := 0; x < grid2W; x++ {
			// only display outter dot layer of box
			if (y < 1) || (y > 10) || (x < 1) || (x > 4) {
				grid2.Set(x, y)
			}
		}
	}
	m.c2.Clear()
	graph.DrawXYAxisAll(&m.c2, m.cursor, axisStyle)
	graph.DrawBraillePatterns(&m.c2, m.center.Add(offset2), grid2.BraillePatterns(), lineStyle2)

	// canvas 3 draws braille pattern rune lines crossing each other
	// horizontal line pattern - rune from BraillePatternFromPatternDots
	hLine := runes.PatternDots{}
	hLine[1] = true
	hLine[4] = true
	hLineR := runes.BraillePatternFromPatternDots(hLine)
	// vertical line pattern - rune from offset
	vLineR := rune(runes.BrailleBlockOffset)
	vLineR |= 0x0008
	vLineR |= 0x0010
	vLineR |= 0x0020
	vLineR |= 0x0080
	m.c3.Clear()
	graph.DrawXYAxisAll(&m.c3, m.cursor, axisStyle)
	for i := 0; i < 10; i++ {
		offset := canvas.Point{X: i - 5, Y: -1}
		point := m.center.Add(offset)
		existingRune := m.c3.Cell(point).Rune                              // get existing rune on canvas
		combinedRune := runes.CombineBraillePatterns(existingRune, hLineR) // combine existing and new braille runes
		graph.DrawBrailleRune(&m.c3, point, combinedRune, lineStyle1)
	}
	for i := 0; i < 6; i++ {
		offset := canvas.Point{X: 2, Y: i - 3}
		point := m.center.Add(offset)
		existingRune := m.c3.Cell(point).Rune                              // get existing rune on canvas
		combinedRune := runes.CombineBraillePatterns(existingRune, vLineR) // combine existing and new braille runes
		graph.DrawBrailleRune(&m.c3, point, combinedRune, lineStyle2)
	}
	return m, nil
}

func (m model) View() string {
	s := "arrow keys to move origin around, `q/ctrl+c` to quit\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render(m.c1.View()),
		defaultStyle.Render(m.c2.View()),
		defaultStyle.Render(m.c3.View()),
	) + "\n"
	return s
}

func main() {
	w := 20
	h := 11
	center := canvas.Point{X: w / 2, Y: h / 2}

	c1 := canvas.New(w, h)
	c2 := canvas.New(w, h)
	c3 := canvas.New(w, h)

	// canvas 1 draws 2x1 full braille pattern runes diagonal from the axes
	// canvas 2 draws a 3x3 braille pattern runes box at the center of the canvas
	// canvas 3 draws a braille pattern runes lines overlapping on to the a canvas

	m := model{c1, c2, c3, center, center}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
