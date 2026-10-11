// ntcharts - Copyright (c) 2026 Neomantra Corp.

// Command combo-categorical prints a categorical bar/line combo. Run from the
// repository root: go run ./examples/combo-categorical -width 72 -height 18
// Optionally pass -html /tmp/combo.html to render the same spec with ECharts.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/NimbleMarkets/ntcharts/spec/echarts/v2"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/go-echarts/go-echarts/v2/charts"
)

func main() {
	width := flag.Int("width", 72, "terminal columns")
	height := flag.Int("height", 18, "terminal rows")
	html := flag.String("html", "", "optional HTML output path")
	flag.Parse()
	s := spec.Spec{Type: spec.ChartTypeBar, Width: *width, Height: *height,
		XAxis:   spec.XAxis{Type: spec.XAxisCategory, Labels: []string{"Jan", "Feb", "Mar", "Apr", "May"}},
		YAxis:   spec.YAxis{Title: "Profit", Format: spec.Format{Kind: "currency"}},
		Y2Axis:  &spec.YAxis{Title: "Margin", Format: spec.Format{Kind: "percent"}},
		Options: spec.Options{ShowLegend: true},
		Data: spec.Data{Series: []spec.Series{
			{Name: "Profit", Type: "bar", YAxis: spec.YAxisLeft, Color: "#63b3ed", Values: []spec.DataPoint{{Y: 120}, {Y: -40}, {Y: 180}, {Y: 90}, {Y: 150}}},
			{Name: "Margin", Type: "line", YAxis: spec.YAxisRight, Color: "#f6ad55", Values: []spec.DataPoint{{X: "Jan", Y: .12}, {X: "Feb", Y: .08}, {X: "Apr", Y: .15}, {X: "May", Y: .22}}},
		}},
	}
	model, err := spec.Build(s)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Profit (bars, left) / Margin (line, right); March margin is missing")
	fmt.Println(model.(*spec.ComboModel).View())
	if *html != "" {
		chart, err := echarts.ToECharts(s)
		if err != nil {
			log.Fatal(err)
		}
		f, err := os.Create(*html)
		if err != nil {
			log.Fatal(err)
		}
		if err := chart.(*charts.Bar).Render(f); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
