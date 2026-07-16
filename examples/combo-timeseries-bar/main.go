// ntcharts - Copyright (c) 2026 Neomantra Corp.

// Command combo-timeseries-bar demonstrates ntcharts' multi-surface spec
// package by building a single chart specification and rendering it to both
// a terminal and an ECharts HTML file.
//
// It downloads daily AAPL price / volume data from the plotly public dataset,
// aggregates it to monthly buckets (last-Close price as a line series, summed
// Volume in millions as a bar series), then:
//
//   - prints the terminal rendering via spec.Build()
//   - writes combo_chart.html via spec.ToECharts()
//   - opens combo_chart.html in the default browser
//
// Run it with:
//
//	go run ./examples/combo-timeseries-bar
package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	"github.com/NimbleMarkets/ntcharts/v2/spec"

	"github.com/go-echarts/go-echarts/v2/charts"
)

const (
	aaplCSVURL = "https://raw.githubusercontent.com/plotly/datasets/master/finance-charts-apple.csv"
	htmlFile   = "combo_chart.html"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("Downloading AAPL daily data…")
	daily, err := fetchAAPL(aaplCSVURL)
	if err != nil {
		return fmt.Errorf("fetch AAPL: %w", err)
	}
	fmt.Printf("  %d daily rows\n", len(daily))

	monthly := aggregateMonthly(daily)
	sort.Slice(monthly, func(i, j int) bool { return monthly[i].Date.Before(monthly[j].Date) })
	fmt.Printf("  %d monthly buckets\n\n", len(monthly))

	s := buildSpec(monthly)

	// Terminal rendering.
	term, err := spec.Build(s)
	if err != nil {
		return fmt.Errorf("spec.Build: %w", err)
	}
	tslc, ok := term.(*timeserieslinechart.Model)
	if !ok {
		// This example only handles the TimeSeries case; bail with a clear
		// message if Build returns something unexpected (e.g. *barchart.Model).
		return fmt.Errorf("spec.Build: unexpected model type %T", term)
	}
	fmt.Println(tslc.View())

	// Web rendering.
	web, err := s.ToECharts()
	if err != nil {
		return fmt.Errorf("spec.ToECharts: %w", err)
	}
	line, ok := web.(*charts.Line)
	if !ok {
		return fmt.Errorf("spec.ToECharts: unexpected chart type %T", web)
	}
	if err := writeHTML(htmlFile, line); err != nil {
		return fmt.Errorf("write HTML: %w", err)
	}
	abs, _ := filepath.Abs(htmlFile)
	fmt.Printf("\nWrote %s — opening in browser…\n", abs)
	_ = openBrowser(abs) // best effort; ignore failure on headless envs
	return nil
}

// monthlyPoint is one aggregated month of AAPL data.
type monthlyPoint struct {
	Date       time.Time // first of the month (UTC)
	CloseLast  float64   // last Close price observed in the month
	VolumeSumM float64   // sum of Volume for the month, in millions
}

// dailyRow is a single CSV row we care about (Date, Close, Volume).
type dailyRow struct {
	Date   time.Time
	Close  float64
	Volume float64
}

// fetchAAPL downloads and parses the plotly AAPL CSV into a slice of
// dailyRow sorted by date ascending.
func fetchAAPL(url string) ([]dailyRow, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	r := csv.NewReader(resp.Body)
	r.FieldsPerRecord = -1 // plotly CSVs occasionally have trailing columns

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	cols := map[string]int{}
	for i, name := range header {
		cols[name] = i
	}
	dateIdx, ok1 := cols["Date"]
	closeIdx, ok2 := cols["AAPL.Close"]
	volIdx, ok3 := cols["AAPL.Volume"]
	if !ok1 || !ok2 || !ok3 {
		return nil, errors.New("CSV missing expected columns")
	}

	var rows []dailyRow
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		t, err := time.Parse("2006-01-02", rec[dateIdx])
		if err != nil {
			continue // skip malformed dates
		}
		close, err := strconv.ParseFloat(rec[closeIdx], 64)
		if err != nil {
			continue
		}
		vol, err := strconv.ParseFloat(rec[volIdx], 64)
		if err != nil {
			continue
		}
		rows = append(rows, dailyRow{Date: t, Close: close, Volume: vol})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Date.Before(rows[j].Date) })
	return rows, nil
}

// aggregateMonthly groups daily rows by calendar month. The Close value of
// each month is the last observed Close; the Volume is the sum of daily
// volumes, scaled to millions for legibility.
func aggregateMonthly(daily []dailyRow) []monthlyPoint {
	type key struct{ year, month int }
	buckets := map[key]*monthlyPoint{}
	for _, d := range daily {
		k := key{d.Date.Year(), int(d.Date.Month())}
		b, ok := buckets[k]
		if !ok {
			b = &monthlyPoint{Date: time.Date(k.year, time.Month(k.month), 1, 0, 0, 0, 0, time.UTC)}
			buckets[k] = b
		}
		b.CloseLast = d.Close // rows are sorted ascending, so the last write wins
		b.VolumeSumM += d.Volume / 1e6
	}
	out := make([]monthlyPoint, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, *b)
	}
	return out
}

// buildSpec turns the aggregated monthly points into a neutral Spec with
// two series: a line (Close) and a bar (Volume in millions). ECharts will
// render the bar series on a secondary Y axis; the terminal will draw both
// as lines (with a distinct style on the bar series).
func buildSpec(monthly []monthlyPoint) spec.Spec {
	closePts := make([]spec.DataPoint, len(monthly))
	volumePts := make([]spec.DataPoint, len(monthly))
	for i, m := range monthly {
		closePts[i] = spec.DataPoint{X: m.Date, Y: m.CloseLast}
		volumePts[i] = spec.DataPoint{X: m.Date, Y: m.VolumeSumM}
	}
	return spec.Spec{
		Type:     spec.ChartTypeTimeSeries,
		Title:    "AAPL Monthly Close & Volume",
		Subtitle: "Source: plotly datasets · combo line + bar, rendered from one spec",
		Width:    100,
		Height:   24,
		XAxis: spec.XAxis{
			Type:   spec.XAxisTime,
			Format: spec.Format{Kind: "time", Layout: "{yyyy}-{MM}"},
		},
		Data: spec.Data{
			Series: []spec.Series{
				{
					Name:   "Close (USD)",
					Type:   "line",
					Color:  "#5ad0ff",
					Values: closePts,
				},
				{
					Name:   "Volume (M)",
					Type:   "bar",
					Color:  "#ff9e64",
					Values: volumePts,
				},
			},
		},
		Options: spec.Options{
			ShowLegend: true,
			ShowGrid:   true,
		},
		Theme: Theme(),
	}
}

// Theme returns a simple dark theme for the web rendering.
func Theme() spec.Theme {
	return spec.Theme{
		Background: "#1a1b26",
		Foreground: "#c0caf5",
		Palette:    []string{"#5ad0ff", "#ff9e64", "#9ece6a", "#bb9af7"},
	}
}

// writeHTML renders the ECharts chart to path. The *charts.Line Renderer
// produces a complete HTML document that can be opened directly in a browser.
func writeHTML(path string, chart *charts.Line) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return chart.Render(f)
}

// openBrowser opens url in the user's default browser. Returns nil on a
// best-effort basis: failures (headless CI, missing command) are non-fatal.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // linux, bsd, etc.
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
