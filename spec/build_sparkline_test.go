package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/sparkline"
)

func sparkSpec() Spec {
	return Spec{
		Type: ChartTypeSparkline, Width: 30, Height: 4,
		Data: Data{Series: []Series{{Name: "cpu", Values: []DataPoint{
			{Y: 1}, {Y: 4}, {Y: 2}, {Y: 7}, {Y: 5}, {Y: 9}, {Y: 3},
		}}}},
	}
}

func TestBuildSparkline(t *testing.T) {
	got, err := Build(sparkSpec())
	if err != nil {
		t.Fatalf("Build(sparkline): %v", err)
	}
	m, ok := got.(*sparkline.Model)
	if !ok {
		t.Fatalf("Build(sparkline) returned %T, want *sparkline.Model", got)
	}
	if strings.TrimSpace(m.View()) == "" {
		t.Fatal("sparkline view is empty")
	}
}

func TestBuildSparklineSingleSeriesOnly(t *testing.T) {
	s := sparkSpec()
	s.Data.Series = append(s.Data.Series, Series{Name: "b", Values: []DataPoint{{Y: 1}}})
	if _, err := Build(s); err == nil || !strings.Contains(err.Error(), "one series") {
		t.Fatalf("expected single-series error, got %v", err)
	}
}

func TestBuildSparklineMaxPin(t *testing.T) {
	s := sparkSpec()
	s.YAxis.Max = f64(20)
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(sparkline, max pin): %v", err)
	}
}
