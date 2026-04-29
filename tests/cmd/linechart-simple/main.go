package main

import (
	"fmt"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

func main() {
	lc := linechart.New(40, 15, 0, 10, 0, 10)
	lc.DrawXYAxisAndLabel()

	// Draw connected line segments between known points
	points := []canvas.Float64Point{
		{X: 0, Y: 0},
		{X: 2, Y: 4},
		{X: 4, Y: 8},
		{X: 6, Y: 6},
		{X: 8, Y: 2},
		{X: 10, Y: 5},
	}

	for i := 0; i < len(points)-1; i++ {
		lc.DrawLine(points[i], points[i+1], runes.ArcLineStyle)
	}

	fmt.Print(lc.View())
}
