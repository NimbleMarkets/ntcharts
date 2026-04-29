package main

import (
	"fmt"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

func main() {
	lc := linechart.New(40, 15, 0, 10, 0, 10)
	lc.DrawXYAxisAndLabel()

	// Draw braille line segments between known points (wave pattern)
	points := []canvas.Float64Point{
		{X: 0, Y: 5},
		{X: 1, Y: 7},
		{X: 2, Y: 9},
		{X: 3, Y: 10},
		{X: 4, Y: 9},
		{X: 5, Y: 7},
		{X: 6, Y: 5},
		{X: 7, Y: 3},
		{X: 8, Y: 1},
		{X: 9, Y: 0},
		{X: 10, Y: 1},
	}

	for i := 0; i < len(points)-1; i++ {
		lc.DrawBrailleLine(points[i], points[i+1])
	}

	fmt.Print(lc.View())
}
