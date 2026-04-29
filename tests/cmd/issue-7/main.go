package main

import (
	"fmt"

	"github.com/NimbleMarkets/ntcharts/v2/linechart/streamlinechart"
)

func main() {
	slc := streamlinechart.New(30, 10)

	// Issue #7 reproduction data
	values := []float64{4, 6, 8, 10, 8, 6, 4, 2, 0, 2, 4}
	for _, v := range values {
		// Use PushPoint to emulate time progression if needed, or just Push
		// The issue used Push(v)
		slc.Push(v)
	}

	slc.Draw()
	fmt.Print(slc.View())
}
