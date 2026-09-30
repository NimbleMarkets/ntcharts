// ntcharts - Copyright (c) 2026 Neomantra Corp.

package echarts_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/NimbleMarkets/ntcharts/spec/echarts/v2"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

func TestToEChartsSharedXAndPalette(t *testing.T) {
	t.Run("bar", func(t *testing.T) {
		s := spec.Spec{Type: spec.ChartTypeBar, Width: 40, Height: 10,
			Theme: spec.Theme{Palette: []string{"#123abc"}},
			Data: spec.Data{XAxisData: []any{"Q1", "unused"}, Series: []spec.Series{{Name: "sales",
				Values: []spec.DataPoint{{Y: 1}, {X: "override", Y: 2}},
			}}}}
		model, err := echarts.ToECharts(s)
		if err != nil {
			t.Fatal(err)
		}
		var html bytes.Buffer
		if err := model.(*charts.Bar).Render(&html); err != nil {
			t.Fatal(err)
		}
		for _, value := range []string{"Q1", "override", "#123abc"} {
			if !strings.Contains(html.String(), value) {
				t.Fatalf("HTML lacks %q", value)
			}
		}
		if strings.Contains(html.String(), "unused") {
			t.Fatal("shared X overrides point X")
		}
	})
	t.Run("time line and bar", func(t *testing.T) {
		s := spec.Spec{Type: spec.ChartTypeTimeSeries, Width: 40, Height: 10,
			Data: spec.Data{XAxisData: []any{"2026-01-01", "2026-01-02"}, Series: []spec.Series{
				{Name: "price", Values: []spec.DataPoint{{Y: 1}, {X: "2026-01-03", Y: 2}}},
				{Name: "volume", Type: "bar", Values: []spec.DataPoint{{Y: 10}, {Y: 20}}},
			}}}
		before, _ := json.Marshal(s)
		model, err := echarts.ToECharts(s)
		if err != nil {
			t.Fatal(err)
		}
		after, _ := json.Marshal(s)
		if string(before) != string(after) {
			t.Fatal("ToECharts mutated its input")
		}
		line := model.(*charts.Line)
		items := line.MultiSeries[0].Data.([]opts.LineData)
		bars := line.MultiSeries[1].Data.([]opts.BarData)
		if len(items) != 2 || len(bars) != 2 {
			t.Fatalf("missing shared-X points: %d line, %d bar", len(items), len(bars))
		}
		for _, tc := range []struct {
			value any
			want  string
		}{
			{items[0].Value, "2026-01-01T00:00:00Z"},
			{items[1].Value, "2026-01-03T00:00:00Z"},
			{bars[1].Value, "2026-01-02T00:00:00Z"},
		} {
			if got := tc.value.([]any)[0]; got != tc.want {
				t.Fatalf("timestamp %v, want %s", got, tc.want)
			}
		}
	})
}

func TestToEChartsBarDerivesLabelsAndColors(t *testing.T) {
	chart, err := echarts.ToECharts(spec.Spec{
		Type: spec.ChartTypeBar, Width: 60, Height: 20,
		Data: spec.Data{Series: []spec.Series{{
			Name:  "Revenue",
			Color: "#22aadd",
			Values: []spec.DataPoint{
				{X: "Q1", Y: 120}, {X: "Q2", Y: 180},
			},
		}}},
	})
	if err != nil {
		t.Fatalf("ToECharts() error = %v", err)
	}
	bar, ok := chart.(*charts.Bar)
	if !ok {
		t.Fatalf("ToECharts() = %T, want *charts.Bar", chart)
	}
	if got, want := len(bar.MultiSeries), 1; got != want {
		t.Fatalf("len(MultiSeries) = %d, want %d", got, want)
	}
	// go-echarts keeps SetXAxis data private until Render, so assert on the
	// rendered HTML: the derived category labels must reach the page.
	var html bytes.Buffer
	if err := bar.Render(&html); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, label := range []string{"Q1", "Q2"} {
		if !strings.Contains(html.String(), label) {
			t.Fatalf("rendered HTML lacks derived x-axis label %q", label)
		}
	}
	if !strings.Contains(html.String(), "#22aadd") {
		t.Fatal("rendered HTML lacks the series colour #22aadd")
	}
}

func TestToEChartsRejectsUnimplementedTypes(t *testing.T) {
	_, err := echarts.ToECharts(spec.Spec{
		Type: spec.ChartTypeScatter, Width: 60, Height: 20,
		Data: spec.Data{Series: []spec.Series{{Name: "a", Values: []spec.DataPoint{{X: 1.0, Y: 2}}}}},
	})
	if err == nil {
		t.Fatal("ToECharts(scatter) error = nil, want not-yet-implemented error")
	}
}

func TestToEChartsTimeSeriesComboLineBar(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []spec.DataPoint{
		{X: start, Y: 10},
		{X: start.Add(24 * time.Hour), Y: 12},
	}

	chart, err := echarts.ToECharts(spec.Spec{
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
	})
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
		chart, err := echarts.ToECharts(spec.Spec{
			Type: spec.ChartTypeTimeSeries, Width: 80, Height: 24,
			XAxis: spec.XAxis{Type: spec.XAxisTime, Format: spec.Format{Kind: "time", Layout: "2006-01"}},
			Data:  spec.Data{Series: []spec.Series{{Name: "a", Values: points}}},
		})
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
		chart, err := echarts.ToECharts(spec.Spec{
			Type: spec.ChartTypeTimeSeries, Width: 80, Height: 24,
			XAxis: spec.XAxis{Type: spec.XAxisTime},
			Data:  spec.Data{Series: []spec.Series{{Name: "a", Values: points}}},
		})
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

	chart, err := echarts.ToECharts(spec.Spec{
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
	})
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
