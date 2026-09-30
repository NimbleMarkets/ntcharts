// ntcharts - Copyright (c) 2026 Neomantra Corp.

package echarts_test

import (
	"fmt"
	"time"

	"github.com/NimbleMarkets/ntcharts/spec/echarts/v2"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/go-echarts/go-echarts/v2/charts"
)

// ExampleToECharts shows rendering a bar chart Spec to the web. The same
// Spec renders to a terminal with spec.Build.
func ExampleToECharts() {
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

	web, err := echarts.ToECharts(s)
	if err != nil {
		fmt.Println("echarts error:", err)
		return
	}
	_ = web.(*charts.Bar) // call Render(w) to produce HTML
	fmt.Printf("web: %T\n", web)
	// Output: web: *charts.Bar
}

// ExampleToECharts_timeSeries shows a time-series Spec rendered as an
// ECharts line chart with a real "time" X axis.
func ExampleToECharts_timeSeries() {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := spec.Spec{
		Type:   spec.ChartTypeTimeSeries,
		Title:  "Daily Close",
		Width:  80,
		Height: 24,
		XAxis:  spec.XAxis{Type: spec.XAxisTime, Format: spec.Format{Kind: "time", Layout: "2006-01-02"}},
		Data: spec.Data{
			Series: []spec.Series{{
				Name: "Close",
				Values: []spec.DataPoint{
					{X: start, Y: 100},
					{X: start.Add(24 * time.Hour), Y: 102.5},
					{X: start.Add(48 * time.Hour), Y: 101.25},
				},
			}},
		},
		Options: spec.Options{ShowLegend: true},
	}

	web, err := echarts.ToECharts(s)
	if err != nil {
		fmt.Println("echarts error:", err)
		return
	}
	fmt.Printf("web: %T\n", web)
	// Output: web: *charts.Line
}
