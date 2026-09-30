// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"strings"
	"testing"
	"time"
)

// TestTimeSeriesFinalXLabel guards the drawXLabel final-tick rule: the label
// under the newest data must render right-aligned when it does not fit
// left-anchored at the axis end, not silently vanish. Six monthly points at
// width 44 previously rendered labels Jan..May with Jun dropped.
func TestTimeSeriesFinalXLabel(t *testing.T) {
	ms := func(mo time.Month) float64 {
		return float64(time.Date(2026, mo, 1, 0, 0, 0, 0, time.UTC).UnixMilli())
	}
	s := Spec{
		Type: ChartTypeTimeSeries, Width: 44, Height: 12,
		Data: Data{Series: []Series{{Name: "px", Values: []DataPoint{
			{X: ms(time.January), Y: 104.2}, {X: ms(time.February), Y: 108.9},
			{X: ms(time.March), Y: 101.4}, {X: ms(time.April), Y: 115.7},
			{X: ms(time.May), Y: 119.3}, {X: ms(time.June), Y: 112.8},
		}}}},
		XAxis: XAxis{Type: "time", Format: Format{Kind: "time", Layout: "Jan"}},
	}
	got, err := Build(s) // Build draws the model; View() is ready immediately
	if err != nil {
		t.Fatalf("Build(timeseries): %v", err)
	}
	view := got.(interface{ View() string }).View()
	if !strings.Contains(view, "May") {
		t.Fatalf("sanity: expected an interior month label in view:\n%s", view)
	}
	if !strings.Contains(view, "Jun") {
		t.Fatalf("final month label missing — last x tick was dropped:\n%s", view)
	}
	// Right-aligned: Jun ends at (or within a cell of) the row's right edge.
	for _, line := range strings.Split(view, "\n") {
		if idx := strings.Index(line, "Jun"); idx >= 0 {
			if idx < len(line)-len("Jun ") {
				t.Fatalf("Jun not right-aligned (idx %d in %q)", idx, line)
			}
		}
	}
}
