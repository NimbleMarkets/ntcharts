package main

import (
	"fmt"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/wavelinechart"
)

func main() {
	wlc := wavelinechart.New(30, 12, wavelinechart.WithYRange(-3, 3))

	// Plot alternating wave pattern
	wlc.Plot(canvas.Float64Point{X: 1, Y: 2})
	wlc.Plot(canvas.Float64Point{X: 3, Y: -2})
	wlc.Plot(canvas.Float64Point{X: 5, Y: 2})
	wlc.Plot(canvas.Float64Point{X: 7, Y: -2})
	wlc.Plot(canvas.Float64Point{X: 9, Y: 2})

	wlc.Draw()
	fmt.Print(wlc.View())
}
