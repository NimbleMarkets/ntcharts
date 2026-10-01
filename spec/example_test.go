// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec_test

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/wavelinechart"
	"github.com/NimbleMarkets/ntcharts/v2/sparkline"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

// ExampleBuild_bar shows how to describe a bar chart and render it to a
// terminal via Build. The same Spec renders to the web with
// echarts.ToECharts from the spec/echarts module.
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
	fmt.Printf("terminal: %T\n", term)
	// Output: terminal: *barchart.Model
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

// ExampleBuild_logScale shows a logarithmic Y axis: YAxis.Scale = ScaleLog
// gives every decade the same height, so growth from 3 to 1,000,000 stays
// readable. The unpinned range widens to whole decades (1 to 1M), and with
// no YAxis.Format the labels use a short k/M suffix. Ticks sit on powers of
// ten whatever the height: here six decades share eight rows, so every
// second decade is labelled, on the row where it falls.
func ExampleBuild_logScale() {
	s := spec.Spec{
		Type:   spec.ChartTypeLine,
		Title:  "Signups",
		Width:  40,
		Height: 10,
		YAxis:  spec.YAxis{Scale: spec.ScaleLog},
		Data: spec.Data{
			Series: []spec.Series{{Name: "signups", Values: []spec.DataPoint{
				{X: 0.0, Y: 3}, {X: 1.0, Y: 40}, {X: 2.0, Y: 150},
				{X: 3.0, Y: 2500}, {X: 4.0, Y: 30000}, {X: 5.0, Y: 1e6},
			}}},
		},
	}

	term, err := spec.Build(s)
	if err != nil {
		fmt.Println("build error:", err)
		return
	}
	m := term.(*wavelinechart.Model)
	for _, line := range strings.Split(m.View(), "\n") {
		fmt.Println(strings.TrimRight(line, " "))
	}
	// Output:
	//  1M│                                   ╭
	//    │                                   │
	//    │                            ╭╮     │
	// 10k│                     ╭╮     ││     │
	//    │                     ││     ││     │
	// 100│             ╭╮      ││     ││     │
	//    │      ╭╮     ││      ││     ││     │
	//    ├╮     ││     ││      ││     ││     │
	//   1└┴─────┴┴─────┴┴──────┴┴─────┴┴─────┴
	//    0   1       2       3     4       5
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
	fmt.Printf("terminal: %T\n", term)
	// Output: terminal: *timeserieslinechart.Model
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
// This compact example prints model metadata. The rendering tests compare
// cell background colours directly, independently of terminal colour support.
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
// rendering it via Build to a *timeserieslinechart.Model. buildOHLC pushes
// open/high/low/close data sets keyed by each point's parsed time and draws
// candles at time-scaled X positions across the full graph width via
// (*timeserieslinechart.Model).DrawCandleWithOpts, with a body width chosen
// automatically from chart density.
//
// This compact example prints model metadata. The rendering tests inspect
// candle glyphs with ANSI styling removed.
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
	m := term.(*timeserieslinechart.Model)
	fmt.Printf("terminal: %T, non-empty view: %v\n", term, strings.TrimSpace(m.View()) != "")
	// Output: terminal: *timeserieslinechart.Model, non-empty view: true
}

// ExampleBuild_heatmapLabels shows a labelled heatmap. Cells are drawn with
// background colours, which the Example strips to keep its golden plain: each
// block of blank cells below is one coloured cell, and the first row label
// (Mon) is the top row.
func ExampleBuild_heatmapLabels() {
	days := []string{"Mon", "Tue", "Wed"}
	hours := []string{"09", "12", "15", "18"}
	var cells []spec.HeatCell
	for y := range days {
		for x := range hours {
			cells = append(cells, spec.HeatCell{X: float64(x), Y: float64(y), Z: float64(x + y*4)})
		}
	}
	s := spec.Spec{
		Type: spec.ChartTypeHeatmap, Width: 28, Height: 9,
		XAxis: spec.XAxis{Labels: hours},
		YAxis: spec.YAxis{Labels: days},
		Heat:  &spec.HeatData{Cells: cells},
		Theme: spec.Theme{Gradient: []string{"#000044", "#ff4400"}},
	}
	term, err := spec.Build(s)
	if err != nil {
		fmt.Println("build error:", err)
		return
	}
	m := term.(*heatmap.Model)
	for _, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		fmt.Println(strings.TrimRight(line, " ") + "|") // | marks each row's end, so blank rows show
	}
	// Output:
	// |
	// Mon|
	// |
	// Tue|
	// |
	// |
	// Wed|
	//     09    12    15    18|
	// |
}
