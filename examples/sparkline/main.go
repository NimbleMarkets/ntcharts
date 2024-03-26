// bubbletea-charts - Copyright (c) 2024 Neomantra Corp.

package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/NimbleMarkets/bubbletea-charts/sparkline"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var randomFloat64 float64

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var titleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

var blockStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("63")) // purple

var blockStyle2 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("9")). // red
	Background(lipgloss.Color("2"))  // green

var blockStyle3 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6")). // cyan
	Background(lipgloss.Color("3"))  // yellow

var blockStyle4 = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

type model struct {
	s1  sparkline.Model
	s2  sparkline.Model
	s3  sparkline.Model
	s4  sparkline.Model
	s5  sparkline.Model
	max float64
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	randomFloat64 = rand.Float64() * m.max

	// add same random value to all sparkline
	m.s1.Push(randomFloat64)
	m.s2.Push(randomFloat64)
	m.s3.Push(randomFloat64)
	m.s4.Push(randomFloat64)
	m.s5.Push(randomFloat64)

	// call different Draw functions with different Style combinations
	m.s1.Draw()
	m.s2.DrawColumnsOnly()
	m.s3.Draw()
	m.s4.Draw()
	m.s5.DrawBraille()
	return m, nil
}

func (m model) View() string {
	s := "press any button to push the same random value to all sparklines, `q/ctrl+c` to quit\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		defaultStyle.Render("Draw() w/o background\n"+m.s1.View()),
		defaultStyle.Render(titleStyle.Render("style w/ background")+"\nDrawColumnsOnly()\n"+m.s2.View()+"\nDraw()\n"+m.s3.View()),
		lipgloss.JoinVertical(lipgloss.Left,
			defaultStyle.Render(fmt.Sprintf("Max: %.0f, Random: %.2f", m.max, randomFloat64)),
			defaultStyle.Render("Draw() w/ background\n"+m.s4.View()+"\nDrawBraille()\n"+m.s5.View()),
		),
	) + "\n"
	return s
}

func main() {
	width := 25
	height := 12
	max := 100.0
	// all sparklines contain the same values,
	// but will be scaled when displayed based on sparkline height
	// sparkline1 calls Draw with no background style
	// sparkline2 calls DrawColumnsOnly with background style
	// sparkline3 calls Draw with background style (same style as sparkline2)
	// sparkline4 calls Draw with background style
	// sparkline5 calls DrawBraille with no background style

	m := model{
		sparkline.New(width, height, sparkline.WithMaxValue(max), sparkline.WithStyle(blockStyle)),
		sparkline.New(width, (height/2)-1, sparkline.WithMaxValue(max), sparkline.WithStyle(blockStyle2)),
		sparkline.New(width, (height/2)-1, sparkline.WithMaxValue(max), sparkline.WithStyle(blockStyle2)),
		sparkline.New(width, height/4, sparkline.WithMaxValue(max), sparkline.WithStyle(blockStyle3)),
		sparkline.New(width, height/4, sparkline.WithMaxValue(max), sparkline.WithStyle(blockStyle4)),
		max}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
