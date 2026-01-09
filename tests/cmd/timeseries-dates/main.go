package main

import (
	"fmt"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
)

func main() {
	tslc := timeserieslinechart.New(50, 12)

	// Set a fixed base date for reproducible output
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Push data points over 7 days
	values := []float64{20, 40, 35, 60, 80, 70, 90}
	for i, v := range values {
		date := baseDate.Add(time.Duration(i) * 24 * time.Hour)
		tslc.Push(timeserieslinechart.TimePoint{Time: date, Value: v})
	}

	// Explicitly set time range to avoid including time.Now() (2026) in the auto-scaled range
	// This ensures deterministic axis labels
	minTime := baseDate
	maxTime := baseDate.Add(8 * 24 * time.Hour)
	tslc.SetTimeRange(minTime, maxTime)

	tslc.DrawAll()

	fmt.Print(tslc.View())
}
