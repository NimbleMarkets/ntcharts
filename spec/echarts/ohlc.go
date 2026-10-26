// ntcharts - Copyright (c) 2026 Neomantra Corp.

package echarts

import (
	"fmt"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func toEChartsOHLC(s spec.Spec) (*charts.Kline, error) {
	if len(s.Data.Series) != 1 {
		return nil, fmt.Errorf("spec: ohlc supports exactly one series; got %d", len(s.Data.Series))
	}
	if s.Y2Axis != nil {
		return nil, fmt.Errorf("spec: ohlc does not support y2_axis")
	}
	if s.XAxis.Scale == spec.ScaleLog {
		return nil, fmt.Errorf("spec: ohlc does not support log x_axis")
	}
	colors := []string{"#26a69a", "#ef5350"}
	for i := range colors {
		if i < len(s.Theme.Palette) && s.Theme.Palette[i] != "" {
			colors[i] = s.Theme.Palette[i]
		}
	}
	pts := s.Data.Series[0].OHLC
	labels := make([]string, len(pts))
	candles := make([]opts.KlineData, len(pts))
	volumes := make([]opts.BarData, len(pts))
	for i, p := range pts {
		t, ok := spec.PointTime(p.T)
		if !ok {
			return nil, fmt.Errorf("spec: ohlc point %d has unparsable time %v", i, p.T)
		}
		labels[i] = t.UTC().Format(time.RFC3339Nano)
		candles[i] = opts.KlineData{Value: []float64{p.O, p.C, p.L, p.H}}
		color := colors[0]
		if p.C < p.O {
			color = colors[1]
		}
		volumes[i] = opts.BarData{Value: p.V, ItemStyle: &opts.ItemStyle{Color: color}}
	}
	k := charts.NewKLine()
	k.SetGlobalOptions(titleOpt(s), initOpt(s), legendOpt(s), gridOpt(s),
		charts.WithXAxisOpts(opts.XAxis{Type: "category", Name: s.XAxis.Title}),
		charts.WithYAxisOpts(yAxisOpt(s.YAxis, "")),
		charts.WithTooltipOpts(opts.Tooltip{Trigger: "axis"}))
	k.SetXAxis(labels)
	k.AddSeries(s.Data.Series[0].Name, candles, charts.WithItemStyleOpts(opts.ItemStyle{Color: colors[0], Color0: colors[1], BorderColor: colors[0], BorderColor0: colors[1]}))
	if !s.HasVolume() {
		return k, nil
	}
	// Equal fixed margins keep the two category scales aligned even when their
	// Y labels have different lengths. Category slots preserve missing volumes.
	k.GridList = []opts.Grid{
		{Left: "12%", Right: "8%", Top: "10%", Height: "52%"},
		{Left: "12%", Right: "8%", Top: "68%", Height: "20%"},
	}
	k.XAxisList[0].AxisLabel = &opts.AxisLabel{Show: opts.Bool(false)}
	k.XAxisList[0].Name = ""
	k.XAxisList = append(k.XAxisList, opts.XAxis{Type: "category", GridIndex: 1, Data: labels, Name: s.XAxis.Title})
	k.YAxisList = append(k.YAxisList, opts.YAxis{Type: "value", GridIndex: 1, Name: "Volume", Min: 0})
	k.AxisPointer = &opts.AxisPointer{Link: []opts.AxisPointerLink{{XAxisIndex: []int{0, 1}}}}
	k.DataZoomList = []opts.DataZoom{{Type: "inside", XAxisIndex: []int{0, 1}, End: 100}}
	bars := charts.NewBar()
	bars.AddSeries("Volume", volumes, charts.WithBarChartOpts(opts.BarChart{XAxisIndex: 1, YAxisIndex: 1}))
	// Price is first in series order, followed by volume in the linked axis tooltip.
	k.Overlap(bars)
	return k, nil
}
