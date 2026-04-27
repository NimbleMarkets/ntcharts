// ntcharts - Copyright (c) 2024 Neomantra Corp.

package timeserieslinechart

import (
	"math"
	"testing"
	"time"
)

func TestHourTimeLabelFormatterRoundsToNearestMillisecond(t *testing.T) {
	f := HourTimeLabelFormatter()
	base := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

	v := float64(base.Add(-400*time.Microsecond).UnixMilli())/1e3 + 0.0006
	if got, want := f(0, v), "03:04:05"; got != want {
		t.Fatalf("HourTimeLabelFormatter() = %q, want %q", got, want)
	}
}

func TestDateTimeLabelFormatterRoundsToNearestMillisecond(t *testing.T) {
	f := DateTimeLabelFormatter()
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	v := float64(base.Add(-400*time.Microsecond).UnixMilli())/1e3 + 0.0006
	if got, want := f(0, v), "'26 01/01"; got != want {
		t.Fatalf("DateTimeLabelFormatter() = %q, want %q", got, want)
	}
}

func TestPushPreservesMillisecondResolution(t *testing.T) {
	m := New(40, 10)
	start := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	end := start.Add(20 * time.Millisecond)
	m.SetTimeRange(start, end)
	m.SetViewTimeRange(start, end)

	m.Push(TimePoint{Time: start.Add(1 * time.Millisecond), Value: 1})
	m.Push(TimePoint{Time: start.Add(2 * time.Millisecond), Value: 2})

	raw := m.dSets[DefaultDataSetName].tBuf.ReadAllRaw()
	if len(raw) != 2 {
		t.Fatalf("expected 2 raw points, got %d", len(raw))
	}

	firstMillis := int64(math.Round(raw[0].X * 1e3))
	secondMillis := int64(math.Round(raw[1].X * 1e3))
	if diff := secondMillis - firstMillis; diff != 1 {
		t.Fatalf("rounded raw X diff = %dms, want 1ms", diff)
	}
}
