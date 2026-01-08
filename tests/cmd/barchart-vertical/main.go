package main

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/barchart"
)

func main() {
	// Default is vertical bars
	bc := barchart.New(30, 15)

	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Background(lipgloss.Color("9"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Background(lipgloss.Color("2"))
	blueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Background(lipgloss.Color("4"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Background(lipgloss.Color("3"))

	data := []barchart.BarData{
		{Label: "A", Values: []barchart.BarValue{{Name: "val", Value: 80, Style: redStyle}}},
		{Label: "B", Values: []barchart.BarValue{{Name: "val", Value: 60, Style: greenStyle}}},
		{Label: "C", Values: []barchart.BarValue{{Name: "val", Value: 40, Style: blueStyle}}},
		{Label: "D", Values: []barchart.BarValue{{Name: "val", Value: 20, Style: yellowStyle}}},
	}

	bc.PushAll(data)
	bc.Draw()

	fmt.Print(bc.View())
}
