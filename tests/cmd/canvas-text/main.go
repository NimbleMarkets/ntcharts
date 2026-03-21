package main

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

func main() {
	c := canvas.New(20, 5)

	// Line 1: Cyan colored text
	cyanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	c.SetStringWithStyle(canvas.Point{X: 0, Y: 0}, "HELLO WORLD", cyanStyle)

	// Line 2: Default color (no style)
	c.SetString(canvas.Point{X: 0, Y: 1}, "ntcharts test")

	// Line 3: Empty (skip)

	// Line 4: Default color
	c.SetString(canvas.Point{X: 0, Y: 3}, "Line 4")

	// Line 5: Numbers in default color
	c.SetString(canvas.Point{X: 0, Y: 4}, "12345")

	fmt.Print(c.View())
}
