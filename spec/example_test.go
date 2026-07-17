// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/wavelinechart"
	"github.com/NimbleMarkets/ntcharts/v2/sparkline"
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

// ExampleBuild_line shows describing a numeric-X line chart and rendering it
// via Build to a *wavelinechart.Model.
func ExampleBuild_line() {
	s := spec.Spec{
		Type:   spec.ChartTypeLine,
		Title:  "Two Series",
		Width:  40,
		Height: 10,
		Data: spec.Data{
			Series: []spec.Series{
				{Name: "a", Values: []spec.DataPoint{
					{X: 0.0, Y: 1}, {X: 1.0, Y: 3}, {X: 2.0, Y: 2}, {X: 3.0, Y: 5},
				}},
				{Name: "b", Values: []spec.DataPoint{
					{X: 0.0, Y: 4}, {X: 1.0, Y: 2}, {X: 2.0, Y: 4}, {X: 3.0, Y: 1},
				}},
			},
		},
	}

	term, err := spec.Build(s)
	if err != nil {
		fmt.Println("build error:", err)
		return
	}
	m := term.(*wavelinechart.Model)
	fmt.Println(m.View())
	// Output:
	// 5│                                     ╭
	//  │                                     │
	// 4├╮                       ╭╮           │
	//  ││           ╭╮          ││           │
	// 2││           ││          ││           │
	//  ││           ├┤          ├┤           │
	// 1├┤           ││          ││           ├
	//  ││           ││          ││           │
	// 0└┴───────────┴┴──────────┴┴───────────┴
	//  0       1           2           3
}

// ExampleBuild_scatter shows describing a scatter chart and rendering it via
// Build to a *linechart.Model; ntcharts has no dedicated scatter model, so
// points are drawn directly onto a base linechart canvas.
func ExampleBuild_scatter() {
	s := spec.Spec{
		Type:   spec.ChartTypeScatter,
		Title:  "Fuel Efficiency",
		Width:  40,
		Height: 12,
		Data: spec.Data{
			Series: []spec.Series{
				{Name: "japan", Values: []spec.DataPoint{
					{X: 2100.0, Y: 31.5}, {X: 1980.0, Y: 33.1},
				}},
				{Name: "usa", Values: []spec.DataPoint{
					{X: 2875.0, Y: 24.0}, {X: 3200.0, Y: 19.2}, {X: 3600.0, Y: 16.5},
				}},
			},
		},
	}

	term, err := spec.Build(s)
	if err != nil {
		fmt.Println("build error:", err)
		return
	}
	m := term.(*linechart.Model)
	for _, line := range strings.Split(m.View(), "\n") {
		fmt.Println(strings.TrimRight(line, " "))
	}
	// Output:
	// 33│•
	//   │   •
	// 30│
	//   │
	// 26│
	//   │                    •
	// 23│
	//   │
	// 20│                           •
	//   │                                    •
	// 16└─────────────────────────────────────
	//   1980  2243  2505  2768  3031  3294
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

// ExampleBuild_sparkline shows describing a sparkline chart (single series)
// and rendering it via Build to a *sparkline.Model. The sparkline is rendered
// using plain runes with no color styling applied to keep the Example golden
// escape-free.
func ExampleBuild_sparkline() {
	s := spec.Spec{
		Type:   spec.ChartTypeSparkline,
		Title:  "CPU Usage",
		Width:  30,
		Height: 4,
		Data: spec.Data{
			Series: []spec.Series{{
				Name: "cpu",
				Values: []spec.DataPoint{
					{Y: 1}, {Y: 4}, {Y: 2}, {Y: 7}, {Y: 5}, {Y: 9}, {Y: 3},
				},
			}},
		},
	}

	term, err := spec.Build(s)
	if err != nil {
		fmt.Println("build error:", err)
		return
	}
	m := term.(*sparkline.Model)
	for _, line := range strings.Split(m.View(), "\n") {
		fmt.Println(strings.TrimRight(line, " "))
	}
	// Output:
	//                           ▁ █
	//                           █▂█
	//                         ▆ ███▃
	//                        ▄█▇████
}

// ExampleBuild_heatmap shows describing a heatmap with a custom gradient
// color scale (Theme.Gradient) and rendering it via Build to a
// *heatmap.Model.
//
// heatmap cells are colored by setting each cell's background style (see
// heatmap.Model.DrawPoint), so View() always embeds ANSI escape sequences —
// even with the package's grayscale default scale. Because go/doc Example
// goldens must be escape-free, this Example cannot print View() output as
// other Example functions in this file do. Instead it follows this
// package's established non-View Example pattern: assert the concrete
// return type and confirm the rendered view is non-empty.
func ExampleBuild_heatmap() {
	s := spec.Spec{
		Type:   spec.ChartTypeHeatmap,
		Title:  "Correlation Matrix",
		Width:  30,
		Height: 10,
		Heat: &spec.HeatData{
			Cells: []spec.HeatCell{
				{X: 0, Y: 0, Z: 1}, {X: 1, Y: 0, Z: 5}, {X: 2, Y: 0, Z: 9},
				{X: 0, Y: 1, Z: 3}, {X: 1, Y: 1, Z: 7}, {X: 2, Y: 1, Z: 2},
			},
		},
		Theme: spec.Theme{Gradient: []string{"#000044", "#ff4400"}},
	}

	term, err := spec.Build(s)
	if err != nil {
		fmt.Println("build error:", err)
		return
	}
	m := term.(*heatmap.Model)
	fmt.Printf("terminal: %T, non-empty view: %v\n", term, strings.TrimSpace(m.View()) != "")
	// Output: terminal: *heatmap.Model, non-empty view: true
}

// ExampleBuild_ohlc shows describing an open/high/low/close chart and
// rendering it via Build to a *canvas.Model. ntcharts has no dedicated OHLC
// model; this surface owns price scaling and draws candlesticks directly
// onto a raw canvas via canvas/graph.DrawCandlestickBottomToTop.
//
// Candles are always foreground-styled (Theme.Palette[0] for up candles,
// Theme.Palette[1] for down, defaulting to teal/red), so View() always
// embeds ANSI escape sequences — even with no Theme set. Because go/doc
// Example goldens must be escape-free, this Example follows the same
// non-View pattern as ExampleBuild_heatmap: assert the concrete return type
// and confirm the rendered view is non-empty.
func ExampleBuild_ohlc() {
	s := spec.Spec{
		Type:   spec.ChartTypeOHLC,
		Title:  "Daily Close",
		Width:  40,
		Height: 12,
		Data: spec.Data{
			Series: []spec.Series{{
				Name: "px",
				OHLC: []spec.OHLCPoint{
					{T: "2026-01-05", O: 100, H: 108, L: 97, C: 105},
					{T: "2026-01-06", O: 105, H: 112, L: 103, C: 110},
					{T: "2026-01-07", O: 110, H: 111, L: 98, C: 99},
					{T: "2026-01-08", O: 99, H: 106, L: 96, C: 104},
				},
			}},
		},
	}

	term, err := spec.Build(s)
	if err != nil {
		fmt.Println("build error:", err)
		return
	}
	m := term.(*canvas.Model)
	fmt.Printf("terminal: %T, non-empty view: %v\n", term, strings.TrimSpace(m.View()) != "")
	// Output: terminal: *canvas.Model, non-empty view: true
}
