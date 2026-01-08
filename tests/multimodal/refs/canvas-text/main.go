package main

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

func main() {
	c := canvas.New(20, 5)

	// Use SetLinesWithStyle for multiple lines
	cyanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	c.SetLinesWithStyle([]string{
		"HELLO WORLD",
		"ntcharts test",
		"",
		"Line 4",
		"12345",
	}, cyanStyle)

	fmt.Print(c.View())
}
