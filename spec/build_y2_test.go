// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden from the current output")

// golden compares got with testdata/<name>.golden, or rewrites the file
// when the tests run with -update.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test -run %s -update)", err, t.Name())
	}
	if got != string(want) {
		t.Fatalf("%s changed:\n%s\n--- want ---\n%s", path, got, want)
	}
}

var y2Start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// leftOnlySpecs are one-series specs of every chart type that will learn a
// right axis. Their renders must never change.
func leftOnlySpecs() map[string]Spec {
	load := []float64{0.1, 0.3, 0.2, 0.6, 0.5, 0.8, 0.7, 0.9, 0.4, 0.6, 0.2}
	var l []DataPoint
	for i := range load {
		l = append(l, DataPoint{X: y2Start.Add(time.Duration(i) * time.Hour), Y: load[i]})
	}
	line := Spec{Type: ChartTypeLine, Width: 40, Height: 12, Data: Data{Series: []Series{
		{Name: "a", Color: "#00ff00", Values: []DataPoint{{X: 0.0, Y: 1}, {X: 1.0, Y: 3}, {X: 2.0, Y: 2}, {X: 3.0, Y: 5}}},
	}}}
	scatter := line
	scatter.Type = ChartTypeScatter
	return map[string]Spec{
		"timeseries": {Type: ChartTypeTimeSeries, Width: 44, Height: 12,
			YAxis: YAxis{Min: f64(0), Max: f64(1), Format: Format{Precision: f64i(1)}},
			Data:  Data{Series: []Series{{Name: "load", Color: "#00ff00", Values: l}}}},
		"line":    line,
		"scatter": scatter,
	}
}

// TestGoldenLeftOnly pins the renders of specs without a right Y axis.
func TestGoldenLeftOnly(t *testing.T) {
	for name, s := range leftOnlySpecs() {
		t.Run(name, func(t *testing.T) {
			golden(t, "leftonly_"+name, rawView(t, s))
		})
	}
}

func TestBuildRejectsY2AxisOnUnsupportedTypes(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeBar, ChartTypeOHLC, ChartTypeSparkline} {
		s := rangeSpec(kind)
		s.Y2Axis = &YAxis{}
		wantBuildError(t, s, `spec: y2_axis is not supported for chart type "`+string(kind)+`"`)
	}
	h := heatSpec()
	h.Y2Axis = &YAxis{}
	wantBuildError(t, h, `spec: y2_axis is not supported for chart type "heatmap"`)
}

// y2TimeSeriesSpec has a load series on the left axis (0..1) and a
// temperature series on the right axis (5..20 C).
func y2TimeSeriesSpec() Spec {
	load := []float64{0.1, 0.3, 0.2, 0.6, 0.5, 0.8, 0.7, 0.9, 0.4, 0.6, 0.2}
	temp := []float64{19, 17, 18, 14, 15, 11, 12, 8, 9, 6, 5}
	var l, r []DataPoint
	for i := range load {
		x := y2Start.Add(time.Duration(i) * time.Hour)
		l = append(l, DataPoint{X: x, Y: load[i]})
		r = append(r, DataPoint{X: x, Y: temp[i]})
	}
	return Spec{
		Type: ChartTypeTimeSeries, Width: 44, Height: 12,
		YAxis:  YAxis{Min: f64(0), Max: f64(1), Format: Format{Precision: f64i(1)}},
		Y2Axis: &YAxis{Min: f64(5), Max: f64(20)},
		Data: Data{Series: []Series{
			{Name: "load", Color: "#00ff00", Values: l},
			{Name: "temp", Color: "#ff0000", YAxis: YAxisRight, Values: r},
		}},
	}
}

// rightLabels returns the non-empty text right of the chart's right axis
// column, top to bottom.
func rightLabels(m *linechart.Model) []string {
	var labels []string
	for i, line := range strings.Split(stripAnsi(m.View()), "\n") {
		if i > m.Origin().Y {
			break
		}
		r := []rune(line)
		if m.GraphRight()+1 >= len(r) {
			continue
		}
		if l := strings.TrimSpace(string(r[m.GraphRight()+1:])); l != "" {
			labels = append(labels, l)
		}
	}
	return labels
}

func TestBuildTimeSeriesTwoAxes(t *testing.T) {
	model, err := Build(y2TimeSeriesSpec())
	if err != nil {
		t.Fatal(err)
	}
	m := model.(*timeserieslinechart.Model)
	a, ok := m.Y2Axis()
	if !ok || a.Min != 5 || a.Max != 20 {
		t.Fatalf("right axis %+v ok=%v, want 5..20", a, ok)
	}
	if m.MinY() != 0 || m.MaxY() != 1 {
		t.Fatalf("left range %v..%v, want 0..1", m.MinY(), m.MaxY())
	}
	view := stripAnsi(m.View())
	if got := yLabels(view); got[0] != "1.0" || got[len(got)-1] != "0.0" {
		t.Fatalf("left labels %v", got)
	}
	if got := rightLabels(&m.Model); len(got) < 2 || got[0] != "20" || got[len(got)-1] != "5" {
		t.Fatalf("right labels %v, want 20 .. 5", got)
	}
	// the temperature line falls from top-left to bottom-right: its first
	// column has ink near the top and its last near the bottom
	rows := strings.Split(view, "\n")
	ink := func(x int) (top, bottom int) {
		top, bottom = -1, -1
		for y := 0; y < m.Origin().Y; y++ {
			if r := []rune(rows[y]); x < len(r) && r[x] >= 0x2800 && r[x] <= 0x28ff && r[x] != 0x2800 {
				if top < 0 {
					top = y
				}
				bottom = y
			}
		}
		return top, bottom
	}
	if top, _ := ink(m.Origin().X + 1); top > m.Origin().Y/3 {
		t.Fatalf("first column ink starts at row %d, want near the top:\n%s", top, view)
	}
	if _, bottom := ink(m.GraphRight() - 1); bottom < 2*m.Origin().Y/3 {
		t.Fatalf("last column ink ends at row %d, want near the bottom:\n%s", bottom, view)
	}
}

func TestBuildTimeSeriesLeftOnlyUnchanged(t *testing.T) {
	s := y2TimeSeriesSpec()
	s.Y2Axis = nil
	s.Data.Series = s.Data.Series[:1]
	want := rawView(t, s)
	// the same spec through the right-axis machinery with nothing on the
	// right must render exactly as the plain left-only build
	s2 := s
	s2.Data.Series = append([]Series(nil), s.Data.Series...)
	s2.Data.Series[0].YAxis = YAxisLeft
	if got := rawView(t, s2); got != want {
		t.Fatalf("explicit y_axis left changed the render:\n%s\n--- want ---\n%s", got, want)
	}
}

func TestBuildTimeSeriesY2AxisWithoutRightSeries(t *testing.T) {
	s := y2TimeSeriesSpec()
	s.Data.Series = s.Data.Series[:1]
	model, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := model.(*timeserieslinechart.Model)
	if a, ok := m.Y2Axis(); !ok || a.Min != 5 || a.Max != 20 {
		t.Fatalf("right axis %+v ok=%v, want an empty 5..20 axis", a, ok)
	}
	if got := rightLabels(&m.Model); len(got) == 0 || got[0] != "20" {
		t.Fatalf("right labels %v", got)
	}
}

func TestBuildTimeSeriesY2Pins(t *testing.T) {
	for _, tc := range []struct {
		name     string
		axis     YAxis
		min, max float64
	}{
		{"min", YAxis{Min: f64(6)}, 6, 19},
		{"max", YAxis{Max: f64(18)}, 5, 18},
		{"both", YAxis{Min: f64(6), Max: f64(18)}, 6, 18},
		{"min at data max", YAxis{Min: f64(19)}, 19, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := y2TimeSeriesSpec()
			s.Y2Axis = &tc.axis
			model, err := Build(s)
			if err != nil {
				t.Fatal(err)
			}
			a, _ := model.(*timeserieslinechart.Model).Y2Axis()
			if a.Min != tc.min || a.Max != tc.max {
				t.Fatalf("right range %g..%g, want %g..%g", a.Min, a.Max, tc.min, tc.max)
			}
		})
	}
	s := y2TimeSeriesSpec()
	s.Y2Axis = &YAxis{Min: f64(30)}
	wantBuildError(t, s, "spec: y2_axis min 30 exceeds max 19")
	s.Y2Axis = &YAxis{Min: f64(7), Max: f64(7)}
	wantBuildError(t, s, "spec: y2_axis min and max must differ")
}

func TestBuildTimeSeriesY2AutoRangesWhenUnpinned(t *testing.T) {
	s := y2TimeSeriesSpec()
	s.Y2Axis = &YAxis{}
	model, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := model.(*timeserieslinechart.Model).Y2Axis()
	if a.Min != 0 || a.Max != 19 || !a.AutoMin || !a.AutoMax {
		t.Fatalf("right axis %+v, want 0..19 auto-ranged", a)
	}
}

func TestBuildTimeSeriesY2Log(t *testing.T) {
	s := y2TimeSeriesSpec()
	s.Y2Axis = &YAxis{Scale: ScaleLog}
	for i := range s.Data.Series[1].Values {
		s.Data.Series[1].Values[i].Y = float64(i+1) * 3
	}
	model, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := model.(*timeserieslinechart.Model).Y2Axis()
	if a.Scale != linechart.ScaleLog || a.Min != 1 || a.Max != 100 || a.AutoMin || a.AutoMax {
		t.Fatalf("right axis %+v, want log 1..100", a)
	}
	s.Data.Series[1].Values[2].Y = 0
	wantBuildError(t, s, `spec: y2_axis scale "log" requires positive finite values; series "temp" point 2 has y = 0`)
	s = y2TimeSeriesSpec() // positive values again, so the pin is what fails
	s.Y2Axis = &YAxis{Scale: ScaleLog, Min: f64(0)}
	wantBuildError(t, s, `spec: y2_axis scale "log" requires a positive finite min; got 0`)
}

func TestBuildTimeSeriesY2LabelStyleFollowsFirstRightSeries(t *testing.T) {
	model, err := Build(y2TimeSeriesSpec())
	if err != nil {
		t.Fatal(err)
	}
	a, _ := model.(*timeserieslinechart.Model).Y2Axis()
	if got := a.LabelStyle.GetForeground(); got == nil || fmt.Sprint(got) != fmt.Sprint(lipgloss.Color("#ff0000")) {
		t.Fatalf("right label style foreground %v, want #ff0000", got)
	}
}

// y2LineSpec: "a" on the left (1..5), "b" on the right (100..500).
func y2LineSpec(kind ChartType) Spec {
	return Spec{
		Type: kind, Width: 40, Height: 12,
		Y2Axis: &YAxis{},
		Data: Data{Series: []Series{
			{Name: "a", Color: "#00ff00", Values: []DataPoint{{X: 0.0, Y: 1}, {X: 1.0, Y: 3}, {X: 2.0, Y: 2}, {X: 3.0, Y: 5}}},
			{Name: "b", Color: "#ff0000", YAxis: YAxisRight, Values: []DataPoint{{X: 0.0, Y: 500}, {X: 1.0, Y: 300}, {X: 2.0, Y: 400}, {X: 3.0, Y: 100}}},
		}},
	}
}

func TestBuildLineAndScatterTwoAxes(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeScatter} {
		t.Run(string(kind), func(t *testing.T) {
			model, err := Build(y2LineSpec(kind))
			if err != nil {
				t.Fatal(err)
			}
			m := model.(*linechart.Model)
			a, ok := m.Y2Axis()
			if !ok {
				t.Fatal("no right axis")
			}
			switch kind {
			case ChartTypeLine: // 0..1 widened around the data, as the left axis is
				if a.Min != 0 || a.Max != 500 {
					t.Fatalf("right range %g..%g, want 0..500", a.Min, a.Max)
				}
			case ChartTypeScatter: // the data's extremes
				if a.Min != 100 || a.Max != 500 {
					t.Fatalf("right range %g..%g, want 100..500", a.Min, a.Max)
				}
			}
			if got := rightLabels(m); len(got) < 2 || got[0] != "500" {
				t.Fatalf("right labels %v", got)
			}
			// series b's last point (X 3, right 100) is low on the right axis:
			// at the bottom of a 100..500 scatter axis, a fifth of the way up
			// a 0..500 line axis. Series a's last point (5) is at the top of
			// the left axis, so the lowest ink in the last plot column is b's.
			view := stripAnsi(m.View())
			rows := strings.Split(view, "\n")
			x := m.GraphRight() - 1
			bottom := -1
			for y := 0; y < m.Origin().Y; y++ {
				if r := []rune(rows[y]); x < len(r) && r[x] != ' ' && r[x] != 0x2800 {
					bottom = y
				}
			}
			wantBelow := m.Origin().Y / 2 // line: below the middle
			if kind == ChartTypeScatter {
				wantBelow = m.Origin().Y - 2 // scatter: on the bottom rows
			}
			if bottom < wantBelow {
				t.Fatalf("last column ink ends at row %d, want >= %d (origin %d):\n%s", bottom, wantBelow, m.Origin().Y, view)
			}
		})
	}
}

func TestBuildLineLeftOnlyUnchanged(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeScatter} {
		s := y2LineSpec(kind)
		s.Y2Axis = nil
		s.Data.Series = s.Data.Series[:1]
		want := rawView(t, s)
		s2 := s
		s2.Data.Series = []Series{s.Data.Series[0]}
		s2.Data.Series[0].YAxis = YAxisLeft
		if got := rawView(t, s2); got != want {
			t.Fatalf("%s: explicit y_axis left changed the render", kind)
		}
	}
}

func TestBuildLineY2Pins(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeScatter} {
		s := y2LineSpec(kind)
		s.Y2Axis = &YAxis{Min: f64(200)}
		model, err := Build(s)
		if err != nil {
			t.Fatal(err)
		}
		if a, _ := model.(*linechart.Model).Y2Axis(); a.Min != 200 || a.Max != 500 {
			t.Fatalf("%s: right range %g..%g, want 200..500", kind, a.Min, a.Max)
		}
		s.Y2Axis = &YAxis{Min: f64(900)}
		wantBuildError(t, s, "spec: y2_axis min 900 exceeds max 500")
	}
}

func TestBuildLineY2LogRejectsNonPositive(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeScatter} {
		s := y2LineSpec(kind)
		s.Y2Axis = &YAxis{Scale: ScaleLog}
		model, err := Build(s)
		if err != nil {
			t.Fatal(err)
		}
		if a, _ := model.(*linechart.Model).Y2Axis(); a.Scale != linechart.ScaleLog || a.Min != 100 || a.Max != 1000 {
			t.Fatalf("%s: right axis %+v, want log 100..1000", kind, a)
		}
		s.Data.Series[1].Values[3].Y = -1
		wantBuildError(t, s, `spec: y2_axis scale "log" requires positive finite values; series "b" point 3 has y = -1`)
	}
}

func TestBuildLineMixedScales(t *testing.T) {
	s := y2LineSpec(ChartTypeLine)
	s.YAxis = YAxis{Scale: ScaleLog}
	model, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := model.(*linechart.Model)
	if a, _ := m.Y2Axis(); m.YScale() != linechart.ScaleLog || a.Scale != linechart.ScaleLinear {
		t.Fatalf("left %v right %v, want log/linear", m.YScale(), a.Scale)
	}
}

func TestBuildScatterEmptyLogLeftAxisWithPin(t *testing.T) {
	s := y2LineSpec(ChartTypeScatter)
	s.YAxis = YAxis{Scale: ScaleLog, Min: f64(5)}
	s.Data.Series = s.Data.Series[1:] // every series on the right
	model, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := model.(*linechart.Model)
	if m.MinY() != 5 || m.MaxY() != 100 {
		t.Fatalf("left range %g..%g, want 5..100 (as line and timeseries give)", m.MinY(), m.MaxY())
	}
}

func TestBuildRejectsSameNameOnDifferentYAxes(t *testing.T) {
	s := y2TimeSeriesSpec()
	s.Data.Series = append(s.Data.Series, Series{Name: "temp", Values: []DataPoint{
		{X: y2Start, Y: 2}, {X: y2Start.Add(time.Hour), Y: 3},
	}})
	wantBuildError(t, s, `spec: series[2] "temp" is also series[1] but on a different y_axis`)
}
