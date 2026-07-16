// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec_test

import (
	"testing"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/types"
)

func TestToEChartsTimeSeriesComboLineBar(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []spec.DataPoint{
		{X: start, Y: 10},
		{X: start.Add(24 * time.Hour), Y: 12},
	}

	chart, err := spec.Spec{
		Type:   spec.ChartTypeTimeSeries,
		Title:  "Combo",
		Width:  80,
		Height: 24,
		XAxis:  spec.XAxis{Type: spec.XAxisTime},
		Data: spec.Data{
			Series: []spec.Series{
				{Name: "Close", Type: "line", Color: "#ff0000", Values: points},
				{Name: "Average", Color: "#0000ff", Values: points},
				{Name: "Volume", Type: "bar", Color: "#00ff00", Values: points},
			},
		},
	}.ToECharts()
	if err != nil {
		t.Fatalf("ToECharts() error = %v", err)
	}

	line, ok := chart.(*charts.Line)
	if !ok {
		t.Fatalf("ToECharts() = %T, want *charts.Line", chart)
	}
	if got, want := len(line.YAxisList), 2; got != want {
		t.Fatalf("len(YAxisList) = %d, want %d", got, want)
	}
	if got, want := line.YAxisList[1].Type, spec.XAxisValue; got != want {
		t.Fatalf("secondary y-axis type = %q, want %q", got, want)
	}
	if got, want := len(line.MultiSeries), 3; got != want {
		t.Fatalf("len(MultiSeries) = %d, want %d", got, want)
	}

	closeSeries := line.MultiSeries[0]
	if got, want := closeSeries.Type, string(types.ChartLine); got != want {
		t.Fatalf("Close type = %q, want %q", got, want)
	}
	if closeSeries.LineStyle == nil || closeSeries.LineStyle.Color != "#ff0000" {
		t.Fatalf("Close LineStyle = %#v, want color #ff0000", closeSeries.LineStyle)
	}
	if closeSeries.ItemStyle == nil || closeSeries.ItemStyle.Color != "#ff0000" {
		t.Fatalf("Close ItemStyle = %#v, want color #ff0000", closeSeries.ItemStyle)
	}

	averageSeries := line.MultiSeries[1]
	if got, want := averageSeries.Type, string(types.ChartLine); got != want {
		t.Fatalf("Average type = %q, want %q", got, want)
	}
	if averageSeries.LineStyle == nil || averageSeries.LineStyle.Color != "#0000ff" {
		t.Fatalf("Average LineStyle = %#v, want color #0000ff", averageSeries.LineStyle)
	}

	volumeSeries := line.MultiSeries[2]
	if got, want := volumeSeries.Type, string(types.ChartBar); got != want {
		t.Fatalf("Volume type = %q, want %q", got, want)
	}
	if got, want := volumeSeries.YAxisIndex, 1; got != want {
		t.Fatalf("Volume YAxisIndex = %d, want %d", got, want)
	}
	if volumeSeries.ItemStyle == nil || volumeSeries.ItemStyle.Color != "#00ff00" {
		t.Fatalf("Volume ItemStyle = %#v, want color #00ff00", volumeSeries.ItemStyle)
	}
}

func TestToEChartsTimeSeriesXAxisLabelFormatter(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []spec.DataPoint{{X: start, Y: 10}}

	t.Run("Go layout translated into formatter", func(t *testing.T) {
		chart, err := spec.Spec{
			Type: spec.ChartTypeTimeSeries, Width: 80, Height: 24,
			XAxis: spec.XAxis{Type: spec.XAxisTime, Format: spec.Format{Kind: "time", Layout: "2006-01"}},
			Data:  spec.Data{Series: []spec.Series{{Name: "a", Values: points}}},
		}.ToECharts()
		if err != nil {
			t.Fatalf("ToECharts() error = %v", err)
		}
		line := chart.(*charts.Line)
		lbl := line.XAxisList[0].AxisLabel
		if lbl == nil || lbl.Formatter == "" {
			t.Fatalf("expected an AxisLabel formatter for a Go layout, got %#v", lbl)
		}
		if got, want := string(lbl.Formatter), "{yyyy}-{MM}"; got != want {
			t.Fatalf("formatter = %q, want %q", got, want)
		}
	})

	t.Run("no layout omits formatter", func(t *testing.T) {
		chart, err := spec.Spec{
			Type: spec.ChartTypeTimeSeries, Width: 80, Height: 24,
			XAxis: spec.XAxis{Type: spec.XAxisTime},
			Data:  spec.Data{Series: []spec.Series{{Name: "a", Values: points}}},
		}.ToECharts()
		if err != nil {
			t.Fatalf("ToECharts() error = %v", err)
		}
		line := chart.(*charts.Line)
		if lbl := line.XAxisList[0].AxisLabel; lbl != nil && lbl.Formatter != "" {
			t.Fatalf("expected no AxisLabel formatter when Layout is unset, got %#v", lbl.Formatter)
		}
	})
}

func TestToEChartsTimeSeriesDoesNotAddSecondaryAxisWithoutBars(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	chart, err := spec.Spec{
		Type:   spec.ChartTypeTimeSeries,
		Title:  "Lines",
		Width:  80,
		Height: 24,
		XAxis:  spec.XAxis{Type: spec.XAxisTime},
		Data: spec.Data{
			Series: []spec.Series{{
				Name:   "Close",
				Type:   "line",
				Values: []spec.DataPoint{{X: start, Y: 10}},
			}},
		},
	}.ToECharts()
	if err != nil {
		t.Fatalf("ToECharts() error = %v", err)
	}

	line := chart.(*charts.Line)
	if got, want := len(line.YAxisList), 1; got != want {
		t.Fatalf("len(YAxisList) = %d, want %d", got, want)
	}
	if got, want := len(line.MultiSeries), 1; got != want {
		t.Fatalf("len(MultiSeries) = %d, want %d", got, want)
	}
	if got, want := line.MultiSeries[0].Type, string(types.ChartLine); got != want {
		t.Fatalf("series type = %q, want %q", got, want)
	}
}
