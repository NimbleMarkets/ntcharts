// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
)

func rangeSpec(kind ChartType) Spec {
	return Spec{Type: kind, Width: 40, Height: 10, Data: Data{Series: []Series{{
		Name: "price", Values: []DataPoint{{X: 0.0, Y: 1}, {X: 1000.0, Y: 5}},
		OHLC: []OHLCPoint{{T: 0.0, O: 2, H: 5, L: 1, C: 4}, {T: 1000.0, O: 3, H: 5, L: 1, C: 2}},
	}}}}
}

func TestBuildPreservesYPins(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeTimeSeries, ChartTypeScatter, ChartTypeOHLC} {
		for _, tc := range []struct {
			name     string
			axis     YAxis
			min, max float64
		}{
			{"min", YAxis{Min: f64(2)}, 2, 5},
			{"max", YAxis{Max: f64(4)}, 1, 4},
			{"both", YAxis{Min: f64(2), Max: f64(4)}, 2, 4},
			{"min at data max", YAxis{Min: f64(5)}, 5, 6},
			{"max at data min", YAxis{Max: f64(1)}, 0, 1},
		} {
			t.Run(string(kind)+"/"+tc.name, func(t *testing.T) {
				s := rangeSpec(kind)
				s.YAxis = tc.axis
				model, err := Build(s)
				if err != nil {
					t.Fatal(err)
				}
				m := model.(interface {
					ViewMinY() float64
					ViewMaxY() float64
				})
				if m.ViewMinY() != tc.min || m.ViewMaxY() != tc.max {
					t.Fatalf("got %g..%g, want %g..%g", m.ViewMinY(), m.ViewMaxY(), tc.min, tc.max)
				}
			})
		}
		t.Run(string(kind)+"/equal pins", func(t *testing.T) {
			s := rangeSpec(kind)
			s.YAxis = YAxis{Min: f64(3), Max: f64(3)}
			if _, err := Build(s); err == nil || !strings.Contains(err.Error(), "must differ") {
				t.Fatalf("expected equal-pin error, got %v", err)
			}
		})
	}
}

func TestBuildRejectsInsufficientPlotSpace(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeTimeSeries, ChartTypeScatter, ChartTypeOHLC} {
		for _, size := range [][2]int{{1, 1}, {1, 2}, {2, 3}, {40, 2}} {
			s := rangeSpec(kind)
			s.Width, s.Height = size[0], size[1]
			if _, err := Build(s); err == nil || !strings.Contains(err.Error(), "too small") {
				t.Fatalf("%s %dx%d: expected layout error, got %v", kind, s.Width, s.Height, err)
			}
		}
	}
}

func TestBuildLineSizesFormattedLabels(t *testing.T) {
	s := rangeSpec(ChartTypeLine)
	s.YAxis = YAxis{Min: f64(0), Max: f64(10), Format: Format{Kind: "currency"}}
	s.Data.Series[0].Values = []DataPoint{{X: 0.0, Y: 2}, {X: 1.0, Y: 5}}
	m, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	view := m.(interface{ View() string }).View()
	if !strings.HasPrefix(view, "$10.00│") || !strings.Contains(view, "$0.00└") {
		t.Fatalf("formatted labels clipped:\n%s", view)
	}
}

func TestBuildTimeSeriesSingletonTime(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, points := range [][]DataPoint{
		{{X: base, Y: 5}},
		{{X: base, Y: 3}, {X: base, Y: 5}},
		{{X: base.Add(time.Nanosecond), Y: 3}, {X: base.Add(2 * time.Nanosecond), Y: 5}},
	} {
		s := rangeSpec(ChartTypeTimeSeries)
		s.Data.Series[0].Values = points
		model, err := Build(s)
		if err != nil {
			t.Fatal(err)
		}
		m := model.(*timeserieslinechart.Model)
		center := float64(base.Unix())
		if m.ViewMinX() != center-43200 || m.ViewMaxX() != center+43200 {
			t.Fatalf("unexpected time range: %v..%v", m.ViewMinX(), m.ViewMaxX())
		}
		found := false
		for _, r := range m.View() {
			if r > '\u2800' && r <= '\u28ff' {
				found = true
			}
		}
		if !found {
			t.Fatalf("singleton is not drawn:\n%s", m.View())
		}
	}
}

func TestBuildSharedXMatchesExplicitXWithoutMutation(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeScatter, ChartTypeTimeSeries, ChartTypeBar} {
		t.Run(string(kind), func(t *testing.T) {
			s := rangeSpec(kind)
			s.Data.XAxisData = []any{1000.0, 2000.0}
			if kind == ChartTypeBar {
				s.Data.XAxisData = []any{"Q1", "Q2"}
			}
			s.Data.Series[0].Values = []DataPoint{{Y: 1}, {X: s.Data.XAxisData[1], Y: 5}, {X: nil, Y: 3}}
			before, _ := json.Marshal(s)
			shared, err := Build(s)
			if err != nil {
				t.Fatal(err)
			}
			after, _ := json.Marshal(s)
			if string(before) != string(after) {
				t.Fatal("Build changed its input")
			}
			s.Data.Series[0].Values[0].X = s.Data.XAxisData[0]
			s.Data.XAxisData = nil
			explicit, err := Build(s)
			if err != nil {
				t.Fatal(err)
			}
			if shared.(interface{ View() string }).View() != explicit.(interface{ View() string }).View() {
				t.Fatal("shared X rendering differs from explicit X")
			}
			if kind == ChartTypeBar {
				bars := shared.(*barchart.Model).Data()
				if bars[0].Label != "Q1" || bars[1].Label != "Q2" || bars[2].Label != "3" {
					t.Fatalf("labels: %v", bars)
				}
			}
		})
	}
}

func TestBuildOHLCRejectsAdditionalSeries(t *testing.T) {
	s := rangeSpec(ChartTypeOHLC)
	s.Data.Series = append(s.Data.Series, Series{Name: "other", OHLC: s.Data.Series[0].OHLC})
	if _, err := Build(s); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("expected series-count error, got %v", err)
	}
}

func TestBuildOHLCTimePlacementDoesNotDependOnInputOrder(t *testing.T) {
	s := ohlcSpec()
	before, _ := json.Marshal(s)
	ordered, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(s)
	if string(before) != string(after) {
		t.Fatal("Build changed its OHLC input")
	}
	points := s.Data.Series[0].OHLC
	s.Data.Series[0].OHLC = []OHLCPoint{points[2], points[0], points[3], points[1]}
	shuffled, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if ordered.(interface{ View() string }).View() != shuffled.(interface{ View() string }).View() {
		t.Fatal("non-overlapping candles moved when input order changed")
	}
}
