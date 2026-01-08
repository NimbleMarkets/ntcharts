package main

import (
	"fmt"
	"math"

	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
)

func main() {
	hm := heatmap.New(20, 20, heatmap.WithValueRange(0, 1))
	hm.SetXYRange(-1, 1, -1, 1)

	// Create radial gradient: distance from center determines value
	for x := float64(-1); x < 1.0; x += 2.0 / float64(hm.GraphWidth()) {
		for y := float64(-1); y < 1.0; y += 2.0 / float64(hm.GraphHeight()) {
			// Distance from center (0,0), inverted so center is max
			dist := math.Sqrt(x*x + y*y)
			val := 1.0 - math.Min(dist, 1.0) // 1 at center, 0 at edges
			hm.Push(heatmap.NewHeatPoint(x, y, val))
		}
	}

	hm.Draw()
	fmt.Print(hm.View())
}
