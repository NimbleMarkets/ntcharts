// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec_test

import (
	"fmt"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

// ExampleBuild_bar shows how to describe a bar chart once and render it
// to a terminal (via Build) and the web (via ToECharts) from the same Spec.
func ExampleBuild_bar() {
	s := spec.Spec{
		Type:   spec.ChartTypeBar,
		Title:  "Quarterly Revenue",
		Width:  60,
		Height: 20,
		XAxis: spec.XAxis{
			Type:   spec.XAxisCategory,
			Labels: []string{"Q1", "Q2", "Q3", "Q4"},
		},
		Data: spec.Data{
			Series: []spec.Series{{
				Name:  "Revenue",
				Color: "#22aadd",
				Values: []spec.DataPoint{
					{Y: 120}, {Y: 180}, {Y: 95}, {Y: 210},
				},
			}},
		},
		Options: spec.Options{ShowLegend: true, ShowGrid: true},
	}

	term, err := spec.Build(s)
	if err != nil {
		fmt.Println("build error:", err)
		return
	}
	_ = term.(*barchart.Model) // *barchart.Model, ready to View()

	web, err := s.ToECharts()
	if err != nil {
		fmt.Println("echarts error:", err)
		return
	}
	fmt.Printf("terminal: %T, web: %T\n", term, web)
	// Output: terminal: *barchart.Model, web: *charts.Bar
}

// ExampleBuild_timeSeries shows describing a time-indexed line chart.
func ExampleBuild_timeSeries() {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := make([]spec.DataPoint, 7)
	for i := range points {
		points[i] = spec.DataPoint{
			X: start.Add(time.Duration(i) * 24 * time.Hour),
			Y: float64(10 + i*3),
		}
	}

	s := spec.Spec{
		Type:   spec.ChartTypeTimeSeries,
		Title:  "Daily Active Users",
		Width:  80,
		Height: 24,
		XAxis:  spec.XAxis{Type: spec.XAxisTime},
		Data: spec.Data{
			Series: []spec.Series{{
				Name:   "DAU",
				Values: points,
			}},
		},
		Options: spec.Options{ShowLegend: true},
	}

	term, err := spec.Build(s)
	if err != nil {
		fmt.Println("build error:", err)
		return
	}
	_ = term.(*timeserieslinechart.Model)

	web, err := s.ToECharts()
	if err != nil {
		fmt.Println("echarts error:", err)
		return
	}
	fmt.Printf("terminal: %T, web: %T\n", term, web)
	// Output: terminal: *timeserieslinechart.Model, web: *charts.Line
}
