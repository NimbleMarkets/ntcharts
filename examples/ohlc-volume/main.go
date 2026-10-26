// ntcharts - Copyright (c) 2026 Neomantra Corp.

// Command ohlc-volume renders one OHLCV spec in the terminal and optionally HTML.
// Run: go run ./examples/ohlc-volume [-html chart.html] [-width 80] [-height 24]
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/NimbleMarkets/ntcharts/spec/echarts/v2"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/go-echarts/go-echarts/v2/charts"
)

func main() {
	width := flag.Int("width", 80, "terminal columns")
	height := flag.Int("height", 24, "terminal rows")
	html := flag.String("html", "", "optional ECharts HTML output path")
	flag.Parse()
	if err := run(*width, *height, *html); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(w, h int, path string) error {
	start := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	var points []spec.OHLCPoint
	price := 100.0
	for i := range 18 {
		day := i + i/5*2 // leave weekend gaps
		close := price + float64((i*7)%9) - 4
		volume := 12000 + float64((i*13)%17)*1100
		points = append(points, spec.OHLCPoint{T: start.AddDate(0, 0, day), O: price, H: max(price, close) + 2, L: min(price, close) - 2, C: close, V: &volume})
		price = close
	}
	s := spec.Spec{Type: spec.ChartTypeOHLC, Title: "OHLC with volume", Width: w, Height: h,
		Options: spec.Options{Volume: true}, XAxis: spec.XAxis{Format: spec.Format{Kind: "time", Layout: "Jan 02"}},
		Data: spec.Data{Series: []spec.Series{{Name: "Price", OHLC: points}}}}
	model, err := spec.Build(s)
	if err != nil {
		return err
	}
	fmt.Println(model.(*spec.OHLCModel).View())
	if path == "" {
		return nil
	}
	s.Width, s.Height = 1000, 650
	web, err := echarts.ToECharts(s)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	renderErr := web.(*charts.Kline).Render(f)
	closeErr := f.Close()
	if renderErr != nil {
		return renderErr
	}
	return closeErr
}
