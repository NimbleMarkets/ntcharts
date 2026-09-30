// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
)

func ohlcSpec() Spec {
	return Spec{
		Type: ChartTypeOHLC, Width: 40, Height: 12,
		Data: Data{Series: []Series{{Name: "px", OHLC: []OHLCPoint{
			{T: "2026-01-05", O: 100, H: 108, L: 97, C: 105},
			{T: "2026-01-06", O: 105, H: 112, L: 103, C: 110},
			{T: "2026-01-07", O: 110, H: 111, L: 98, C: 99},
			{T: "2026-01-08", O: 99, H: 106, L: 96, C: 104},
		}}}},
	}
}

// stripAnsi removes SGR escape sequences so rune positions equal columns —
// candle styles emit color codes that would otherwise inflate byte indices
// and break substring adjacency checks.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiRe.ReplaceAllString(s, "") }

func TestBuildOHLC(t *testing.T) {
	s := ohlcSpec()
	s.Width = 60
	s.XAxis.Format = Format{Kind: "time", Layout: "Jan 02"}
	got, err := Build(s)
	if err != nil {
		t.Fatalf("Build(ohlc): %v", err)
	}
	m, ok := got.(*timeserieslinechart.Model)
	if !ok {
		t.Fatalf("Build(ohlc) returned %T, want *timeserieslinechart.Model", got)
	}
	view := stripAnsi(m.View())
	if !strings.ContainsAny(view, "│┃╽╿") {
		t.Fatalf("no candle runes found in view:\n%s", view)
	}
	// Axis labels render now (the old canvas path drew a bare axis).
	if !strings.Contains(view, "Jan") {
		t.Fatalf("no x-axis time label in view:\n%s", view)
	}
	// Time-scaled placement: candle bodies must reach the right half of
	// the chart, not pack against the left edge. Count in runes (ANSI is
	// stripped above) so index == column.
	rightmost := -1
	for _, line := range strings.Split(view, "\n") {
		runes := []rune(line)
		for i := len(runes) - 1; i >= 0; i-- {
			if runes[i] == '┃' || runes[i] == '╽' || runes[i] == '╿' {
				if i > rightmost {
					rightmost = i
				}
				break
			}
		}
	}
	if rightmost <= s.Width/2 {
		t.Fatalf("all candle bodies in the left half (rightmost col %d) — packed-left bug:\n%s", rightmost, view)
	}

	lines := strings.Split(view, "\n")

	// The y-axis column must survive wide candle drawing (Critical 1): find
	// the column where '│' appears most often, then require every mid-chart
	// row (all rows except the bottom axis-rule row and the label row below
	// it) to still carry it there.
	counts := map[int]int{}
	for _, line := range lines {
		for i, r := range []rune(line) {
			if r == '│' {
				counts[i]++
			}
		}
	}
	axisCol, best := -1, 0
	for col, c := range counts {
		if c > best {
			best, axisCol = c, col
		}
	}
	if axisCol < 0 {
		t.Fatalf("no axis column found in view:\n%s", view)
	}
	bodyRows := lines
	if len(bodyRows) > 2 {
		bodyRows = bodyRows[:len(bodyRows)-2] // drop x-axis rule row + label row
	}
	for i, line := range bodyRows {
		runes := []rune(line)
		if axisCol >= len(runes) || runes[axisCol] != '│' {
			t.Fatalf("row %d missing y-axis rune at col %d (axis overdrawn):\n%s", i, axisCol, view)
		}
	}

	// The newest candle must render, not be clipped past the graph edge
	// (Critical 2): body/wick runes must appear in the last 4 columns of at
	// least one row.
	newestFound := false
	for _, line := range lines {
		runes := []rune(line)
		start := len(runes) - 4
		if start < 0 {
			start = 0
		}
		for _, r := range runes[start:] {
			switch r {
			case '┃', '╽', '╿', '│':
				newestFound = true
			}
		}
	}
	if !newestFound {
		t.Fatalf("newest candle missing from last 4 columns of every row (clipped bug):\n%s", view)
	}
}

// A single-point OHLC spec has tMin == tMax, which would divide by zero in
// the X scale factor without the degenerate-range guard (Important 3).
func TestBuildOHLCSingleCandle(t *testing.T) {
	s := ohlcSpec()
	s.Data.Series[0].OHLC = s.Data.Series[0].OHLC[:1]
	got, err := Build(s)
	if err != nil {
		t.Fatalf("Build(ohlc, single candle): %v", err)
	}
	view := stripAnsi(got.(*timeserieslinechart.Model).View())
	if !strings.ContainsAny(view, "┃╽╿") {
		t.Fatalf("no candle body runes for single-candle spec:\n%s", view)
	}
}

func TestAutoCandleWidth(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		graphWidth int
		times      []time.Time
		want       int
	}{
		{
			name:       "no candles",
			graphWidth: 100,
			times:      nil,
			want:       1,
		},
		{
			name:       "single candle on a wide chart caps at 7",
			graphWidth: 200,
			times:      []time.Time{base},
			want:       7,
		},
		{
			name:       "dense candles on a narrow chart floor at 1",
			graphWidth: 20,
			times:      makeTimes(base, 100, time.Hour),
			want:       1,
		},
		{
			name:       "gap-clamped: one close pair narrows the estimate",
			graphWidth: 100,
			times:      []time.Time{base, base.Add(6 * time.Hour), base.Add(100 * time.Hour)},
			want:       5,
		},
		{
			name:       "even share forced odd",
			graphWidth: 10,
			times:      []time.Time{base, base.Add(100 * time.Hour)},
			want:       3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tMin, tMax := base, base
			if len(tt.times) > 0 {
				tMin, tMax = tt.times[0], tt.times[0]
				for _, tm := range tt.times[1:] {
					if tm.Before(tMin) {
						tMin = tm
					}
					if tm.After(tMax) {
						tMax = tm
					}
				}
			}
			got := autoCandleWidth(tt.graphWidth, tt.times, tMin, tMax)
			if got != tt.want {
				t.Fatalf("autoCandleWidth(%d, %d times) = %d, want %d",
					tt.graphWidth, len(tt.times), got, tt.want)
			}
		})
	}
}

func makeTimes(base time.Time, n int, step time.Duration) []time.Time {
	times := make([]time.Time, n)
	for i := range times {
		times[i] = base.Add(time.Duration(i) * step)
	}
	return times
}

func TestBuildOHLCDenseDataStillBuilds(t *testing.T) {
	s := ohlcSpec()
	s.Width = 12
	pts := s.Data.Series[0].OHLC
	for i := 0; i < 40; i++ {
		pts = append(pts, OHLCPoint{
			T: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i).Format("2006-01-02"),
			O: 100, H: 101, L: 99, C: 100})
	}
	s.Data.Series[0].OHLC = pts
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(ohlc, dense): %v", err)
	}
}

func TestBuildOHLCWideBodies(t *testing.T) {
	s := ohlcSpec()
	s.Width = 60 // 4 candles on a wide chart → auto width > 1
	got, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	view := stripAnsi(got.(*timeserieslinechart.Model).View())
	// A widened body draws the same heavy rune in horizontally adjacent
	// cells; single-column candles never do. (ANSI stripped, or the escape
	// codes between styled cells would defeat the adjacency check.)
	if !strings.Contains(view, "┃┃") {
		t.Fatalf("expected multi-column candle bodies (adjacent heavy runes):\n%s", view)
	}
}

func TestBuildOHLCUnparsableTimeErrors(t *testing.T) {
	s := ohlcSpec()
	s.Data.Series[0].OHLC[0].T = "not-a-time"
	if _, err := Build(s); err == nil {
		t.Fatal("expected unparsable-time error")
	}
}

func TestBuildOHLCInvertedPinErrors(t *testing.T) {
	s := ohlcSpec()
	s.YAxis.Min = f64(500) // above all highs
	if _, err := Build(s); err == nil {
		t.Fatal("expected inverted-range error")
	}
}

func TestPointTimeDateOnly(t *testing.T) {
	tm, ok := PointTime("2026-01-05")
	if !ok || !tm.Equal(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("PointTime(date-only) = %v, %v", tm, ok)
	}
}

func TestBuildOHLCBlockStyle(t *testing.T) {
	s := ohlcSpec()
	s.Width = 60
	s.Options.CandleStyle = CandleStyleBlock
	got, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	view := stripAnsi(got.(*timeserieslinechart.Model).View())
	if !strings.ContainsRune(view, '█') {
		t.Fatalf("candle_style block: no full blocks in view:\n%s", view)
	}

	s.Options.CandleStyle = CandleStyleLine
	got, err = Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(stripAnsi(got.(*timeserieslinechart.Model).View()), '█') {
		t.Fatal("candle_style line must not render full blocks")
	}
}

func TestBuildOHLCUnknownCandleStyleErrors(t *testing.T) {
	s := ohlcSpec()
	s.Options.CandleStyle = "bogus"
	if _, err := Build(s); err == nil {
		t.Fatal("expected unknown candle_style error")
	}
}
