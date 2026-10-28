// ntcharts - Copyright (c) 2026 Neomantra Corp.

package echarts_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/spec/echarts/v2"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func TestOHLCVolumeExport(t *testing.T) {
	v := 12300.0
	zero := 0.0
	s := spec.Spec{Type: spec.ChartTypeOHLC, Width: 800, Height: 600, Options: spec.Options{Volume: true}, Theme: spec.Theme{Palette: []string{"green", "red"}}, Data: spec.Data{Series: []spec.Series{{Name: "Price", OHLC: []spec.OHLCPoint{
		{T: "2026-01-02", O: 1, H: 3, L: 1, C: 2, V: &v},
		{T: "2026-01-05", O: 2, H: 3, L: 1, C: 1, V: &zero},
		{T: "2026-01-06", O: 2, H: 3, L: 1, C: 2},
	}}}}}
	before, _ := json.Marshal(s)
	got, err := echarts.ToECharts(s)
	if err != nil {
		t.Fatal(err)
	}
	k := got.(*charts.Kline)
	if len(k.GridList) != 2 || len(k.XAxisList) != 2 || len(k.YAxisList) != 2 {
		t.Fatal("missing panel grids/axes")
	}
	if k.XAxisList[1].GridIndex != 1 || k.YAxisList[1].GridIndex != 1 {
		t.Fatal("wrong grid indexes")
	}
	if len(k.MultiSeries) != 2 || k.MultiSeries[0].Type != "candlestick" || k.MultiSeries[1].Type != "bar" || k.MultiSeries[1].XAxisIndex != 1 || k.MultiSeries[1].YAxisIndex != 1 {
		t.Fatalf("wrong series: %+v", k.MultiSeries)
	}
	bars := k.MultiSeries[1].Data.([]opts.BarData)
	if bars[0].ItemStyle.Color != "green" || bars[1].ItemStyle.Color != "red" || bars[2].ItemStyle.Color != "green" {
		t.Fatal("wrong candle colours")
	}
	data, _ := json.Marshal(bars)
	if !strings.Contains(string(data), `"value":null`) || !strings.Contains(string(data), `"value":0`) {
		t.Fatalf("lost zero/missing volumes: %s", data)
	}
	if !reflect.DeepEqual(k.AxisPointer.Link[0].XAxisIndex, []int{0, 1}) {
		t.Fatal("axes not linked")
	}
	for _, z := range k.DataZoomList {
		if !reflect.DeepEqual(z.XAxisIndex, []int{0, 1}) {
			t.Fatal("zoom not linked")
		}
	}
	if k.Tooltip.Trigger != "axis" {
		t.Fatal("no axis tooltip")
	}
	var html bytes.Buffer
	if err := k.Render(&html); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"axisPointer"`, `"link"`, `"xAxisIndex":[0,1]`, `"dataZoom"`, `"Volume"`, `"candlestick"`} {
		if !strings.Contains(html.String(), want) {
			t.Fatalf("render missing %s", want)
		}
	}
	after, _ := json.Marshal(s)
	if string(before) != string(after) {
		t.Fatal("export mutated spec")
	}
	s.Options.Volume = false
	got, err = echarts.ToECharts(s)
	if err != nil {
		t.Fatal(err)
	}
	k = got.(*charts.Kline)
	off, _ := json.Marshal([]any{k.GridList, k.XAxisList, k.YAxisList, k.MultiSeries})
	for i := range s.Data.Series[0].OHLC {
		s.Data.Series[0].OHLC[i].V = nil
	}
	s.Options.Volume = true
	got, err = echarts.ToECharts(s)
	if err != nil {
		t.Fatal(err)
	}
	k = got.(*charts.Kline)
	if len(k.GridList) != 1 || len(k.MultiSeries) != 1 {
		t.Fatal("empty volume emitted panel")
	}
	// Chart IDs differ per construction; compare the structural chart fields.
	empty, _ := json.Marshal([]any{k.GridList, k.XAxisList, k.YAxisList, k.MultiSeries})
	if string(off) != string(empty) {
		t.Fatal("absent volume changed export")
	}
}

func TestOHLCVolumeShowGrid(t *testing.T) {
	for _, show := range []bool{false, true} {
		v := 100.0
		s := spec.Spec{Type: spec.ChartTypeOHLC, Width: 800, Height: 600, Options: spec.Options{Volume: true, ShowGrid: show}, Data: spec.Data{Series: []spec.Series{{Name: "Price", OHLC: []spec.OHLCPoint{{T: "2026-01-02", O: 1, H: 3, L: 1, C: 2, V: &v}}}}}}
		result, err := echarts.ToECharts(s)
		if err != nil {
			t.Fatal(err)
		}
		k := result.(*charts.Kline)
		raw, err := json.Marshal(k.JSON())
		if err != nil {
			t.Fatal(err)
		}
		var option struct {
			Grids []struct {
				Show *bool `json:"show"`
			} `json:"grid"`
		}
		if err := json.Unmarshal(raw, &option); err != nil {
			t.Fatal(err)
		}
		if len(option.Grids) != 2 {
			t.Fatal("missing grids")
		}
		for i, g := range option.Grids {
			if g.Show == nil || *g.Show != show {
				t.Fatalf("grid %d does not preserve ShowGrid=%v: %s", i, show, raw)
			}
		}
	}
}
