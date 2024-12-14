package main

import (
	"fmt"
	"math"

	"github.com/NimbleMarkets/ntcharts/heatmap"
)

func main() {
	hm := heatmap.New(20, 20, heatmap.WithValueRange(0, 1))
	hm.SetXYRange(-1, 1, -1, 1)
	for x := float64(-1); x < 1.0; x += 1.0 / float64(hm.GraphWidth()) {
		for y := float64(-1); y < 1.0; y += 1.0 / float64(hm.GraphHeight()) {
			val := math.Sin(math.Sqrt(x*x + y*y))
			hm.Push(heatmap.NewHeatPoint(x, y, val))
		}
	}
	hm.Draw()
	fmt.Println(hm.View())
}
