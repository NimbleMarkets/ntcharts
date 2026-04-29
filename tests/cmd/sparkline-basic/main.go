package main

import (
	"fmt"

	"github.com/NimbleMarkets/ntcharts/v2/sparkline"
)

func main() {
	sl := sparkline.New(20, 5)

	// Push values that form a mountain pattern
	values := []float64{1, 3, 5, 7, 9, 7, 5, 3, 1}
	sl.PushAll(values)
	sl.Draw()

	fmt.Print(sl.View())
}
