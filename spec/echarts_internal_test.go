// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import "testing"

func TestGoLayoutToECharts(t *testing.T) {
	cases := []struct {
		name, layout, want string
	}{
		{"go layout translated", "2006-01", "{yyyy}-{MM}"},
		{"go layout with day", "2006-01-02", "{yyyy}-{MM}-{dd}"},
		{"go layout with time", "15:04:05", "{HH}:{mm}:{ss}"},
		{"already echarts template passes through", "{yyyy}/{MM}", "{yyyy}/{MM}"},
		{"empty passes through", "", ""},
		{"unrecognized passthrough", "Q1", "Q1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := goLayoutToECharts(c.layout); got != c.want {
				t.Fatalf("goLayoutToECharts(%q) = %q, want %q", c.layout, got, c.want)
			}
		})
	}
}

func TestXAxisTimeLabelOptOmitsWhenNothingTranslated(t *testing.T) {
	if lbl := xAxisTimeLabelOpt(""); lbl != nil {
		t.Fatalf("xAxisTimeLabelOpt(\"\") = %#v, want nil", lbl)
	}
	if lbl := xAxisTimeLabelOpt("Q1"); lbl != nil {
		t.Fatalf("xAxisTimeLabelOpt(%q) = %#v, want nil (nothing recognizable translated)", "Q1", lbl)
	}
	lbl := xAxisTimeLabelOpt("2006-01")
	if lbl == nil || lbl.Formatter == "" {
		t.Fatal("xAxisTimeLabelOpt(\"2006-01\") should produce a formatter")
	}
}
