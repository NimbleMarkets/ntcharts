package echarts_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/spec/echarts/v2"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/go-echarts/go-echarts/v2/charts"
)

func categoricalCombo() spec.Spec {
	return spec.Spec{Type: spec.ChartTypeBar, Width: 60, Height: 15,
		XAxis: spec.XAxis{Type: spec.XAxisCategory, Labels: []string{"B", "A", "C"}},
		YAxis: spec.YAxis{Format: spec.Format{Kind: "currency"}}, Y2Axis: &spec.YAxis{Format: spec.Format{Kind: "percent"}},
		Data: spec.Data{Series: []spec.Series{
			{Name: "rate", Type: "line", YAxis: spec.YAxisRight, Color: "#ff9900", Values: []spec.DataPoint{{X: "C", Y: .8}, {X: "B", Y: .2}}},
			{Name: "sales", Type: "bar", Values: []spec.DataPoint{{X: "A", Y: -10}, {X: "B", Y: 30}}},
		}}}
}

func TestCategoricalComboWeb(t *testing.T) {
	s := categoricalCombo()
	before, _ := json.Marshal(s)
	result, err := echarts.ToECharts(s)
	if err != nil {
		t.Fatal(err)
	}
	bar := result.(*charts.Bar)
	if len(bar.MultiSeries) != 2 || bar.MultiSeries[0].Type != "bar" || bar.MultiSeries[1].Type != "line" || bar.MultiSeries[1].YAxisIndex != 1 {
		t.Fatalf("wrong marks or axes: %+v", bar.MultiSeries)
	}
	if bool(*bar.MultiSeries[1].ConnectNulls) {
		t.Fatal("connected gaps")
	}
	data, _ := json.Marshal(bar.MultiSeries)
	var series []struct {
		Data []struct {
			Value *float64 `json:"value"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &series); err != nil {
		t.Fatal(err)
	}
	if len(series[0].Data) != 3 || len(series[1].Data) != 3 || *series[0].Data[0].Value != 30 || *series[0].Data[1].Value != -10 || series[0].Data[2].Value != nil || series[1].Data[1].Value != nil || *series[1].Data[2].Value != .8 {
		t.Fatalf("shifted data: %s", data)
	}
	if !bytes.Contains(data, []byte(`"value":null`)) {
		t.Fatal("missing values should be explicit nulls")
	}
	if len(bar.YAxisList) != 2 || bar.YAxisList[0].Min != float64(-10) || bar.YAxisList[0].Max != float64(30) || bar.YAxisList[1].Min != .2 || bar.YAxisList[1].Max != .8 {
		t.Fatalf("wrong independent ranges: %+v", bar.YAxisList)
	}
	var html bytes.Buffer
	if err := bar.Render(&html); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"B","A","C"`, `function(v){return`, `.toFixed(2)`, `v*100`, `#ff9900`} {
		if !strings.Contains(html.String(), want) {
			t.Fatalf("HTML missing %q", want)
		}
	}
	after, _ := json.Marshal(s)
	if string(before) != string(after) {
		t.Fatal("mutated spec")
	}
}

func TestCategoricalComboWebRejectsUnsupported(t *testing.T) {
	for name, edit := range map[string]func(*spec.Spec){
		"stacked":          func(s *spec.Spec) { s.Options.Stacked = true },
		"wrong axis":       func(s *spec.Spec) { s.Data.Series[0].YAxis = spec.YAxisLeft },
		"unknown category": func(s *spec.Spec) { s.Data.Series[0].Values[0].X = "missing" },
		"unknown mix":      func(s *spec.Spec) { s.Data.Series[0].Type = "scatter" },
		"log":              func(s *spec.Spec) { s.Y2Axis.Scale = spec.ScaleLog },
	} {
		t.Run(name, func(t *testing.T) {
			s := categoricalCombo()
			edit(&s)
			if _, err := echarts.ToECharts(s); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
