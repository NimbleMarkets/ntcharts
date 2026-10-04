package timeserieslinechart

import (
	"math"
	"testing"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

func TestFitYToViewSpikeLeavesWindow(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := New(60, 15, WithTimeRange(base, base.Add(2*time.Minute)))
	m.Push(TimePoint{base, 3500})
	m.Push(TimePoint{base.Add(time.Minute), 80})
	m.PushDataSet("other", TimePoint{base.Add(2 * time.Minute), 100})
	m.SetViewTimeRange(base.Add(time.Minute), base.Add(2*time.Minute))
	if m.ViewMaxY() != 3500 {
		t.Fatal("changing X unexpectedly fitted Y")
	}
	if !m.FitYToViewWithOpts(FitYOpts{IncludeZero: true}) || m.ViewMinY() != 0 || m.ViewMaxY() != 100 {
		t.Fatalf("fit = %v..%v, want 0..100", m.ViewMinY(), m.ViewMaxY())
	}
	if m.MaxY() != 3500 || !m.AutoMinY || !m.AutoMaxY {
		t.Fatal("fit changed historical bounds or automatic range flags")
	}
	for _, name := range []string{DefaultDataSetName, "other"} {
		ds := m.dSets[name]
		for i, p := range ds.tBuf.ReadAllRaw() {
			if got, want := ds.tBuf.At(i), m.ScaleFloat64PointForLine(p); got != want {
				t.Fatalf("%s point %d scaled to %v, want %v", name, i, got, want)
			}
		}
	}
	// Fitting still works when automatic growth is disabled.
	m.AutoMinY, m.AutoMaxY = false, false
	m.SetViewTimeRange(base, base.Add(2*time.Minute))
	if !m.FitYToView() || m.ViewMinY() != 80 || m.ViewMaxY() != 3500 {
		t.Fatalf("fit retained history = %v..%v", m.ViewMinY(), m.ViewMaxY())
	}
}

func TestFitYToViewSelectionAndEmptyWindow(t *testing.T) {
	base := time.Unix(0, 0).UTC()
	m := New(40, 12, WithTimeRange(base, base.Add(10*time.Second)))
	m.Push(TimePoint{base, -10})
	m.Push(TimePoint{base.Add(time.Second), 10})
	m.PushDataSet("hidden", TimePoint{base, 1000})
	if !m.FitYToViewWithOpts(FitYOpts{DataSets: []string{DefaultDataSetName, "missing"}}) || m.ViewMinY() != -10 || m.ViewMaxY() != 10 {
		t.Fatalf("selected fit = %v..%v", m.ViewMinY(), m.ViewMaxY())
	}
	if m.FitYToViewWithOpts(FitYOpts{DataSets: []string{}}) {
		t.Fatal("fit succeeded with no selected datasets")
	}
	m.SetViewTimeRange(base.Add(2*time.Second), base.Add(10*time.Second))
	if m.FitYToView() || m.ViewMinY() != -10 || m.ViewMaxY() != 10 {
		t.Fatal("empty window changed Y range")
	}
}

func TestFitYToViewConstantAndInvalidValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		scale linechart.Scale
		value float64
		zero  bool
	}{
		{"linear", linechart.ScaleLinear, 80, false},
		{"zero", linechart.ScaleLinear, 0, false},
		{"zero baseline", linechart.ScaleLinear, 0, true},
		{"negative baseline", linechart.ScaleLinear, -80, true},
		{"log", linechart.ScaleLog, 80, true},
		{"tiny log", linechart.ScaleLog, math.SmallestNonzeroFloat64, false},
		{"large log", linechart.ScaleLog, math.MaxFloat64, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := time.Unix(0, 0).UTC()
			m := New(40, 12, WithTimeRange(base, base.Add(time.Second)), WithYScale(tc.scale))
			// Insert raw points to isolate fitting from Push's automatic growth.
			ds := m.dSets[DefaultDataSetName]
			for _, v := range []float64{tc.value, math.NaN(), math.Inf(1), math.Inf(-1)} {
				ds.tBuf.Push(canvas.Float64Point{X: float64(base.UnixMilli()) / 1e3, Y: v})
			}
			if !m.FitYToViewWithOpts(FitYOpts{IncludeZero: tc.zero}) {
				t.Fatal("no fit")
			}
			lo, hi := m.ViewMinY(), m.ViewMaxY()
			if !(lo < hi && lo <= tc.value && hi >= tc.value) || math.IsInf(lo, 0) || math.IsInf(hi, 0) {
				t.Fatalf("invalid range %v..%v for %v", lo, hi, tc.value)
			}
			if tc.scale == linechart.ScaleLog && lo <= 0 {
				t.Fatal("log range includes zero")
			}
			if tc.zero && tc.scale == linechart.ScaleLinear && !(lo <= 0 && hi >= 0) {
				t.Fatal("baseline omitted")
			}
		})
	}
}

func TestFitYToViewIgnoresNonPositiveLogValues(t *testing.T) {
	base := time.Unix(0, 0).UTC()
	m := New(40, 12, WithTimeRange(base, base.Add(time.Second)), WithYScale(linechart.ScaleLog))
	m.Push(TimePoint{base, -10})
	m.Push(TimePoint{base, 0})
	lo, hi := m.ViewMinY(), m.ViewMaxY()
	if m.FitYToView() || m.ViewMinY() != lo || m.ViewMaxY() != hi {
		t.Fatal("invalid log values changed range")
	}
}

func TestTrimBefore(t *testing.T) {
	base := time.Unix(0, 0).UTC()
	m := New(40, 12, WithTimeRange(base, base.Add(2*time.Second)))
	for _, name := range []string{DefaultDataSetName, "other"} {
		for i := 0; i < 3; i++ {
			m.PushDataSet(name, TimePoint{base.Add(time.Duration(i) * time.Second), float64(i)})
		}
	}
	m.SetViewTimeRange(base.Add(time.Second), base.Add(2*time.Second))
	minX, maxX, minY, maxY := m.ViewMinX(), m.ViewMaxX(), m.ViewMinY(), m.ViewMaxY()
	old := m.dSets[DefaultDataSetName].tBuf.ReadAllRaw()
	if got := m.TrimBefore(base.Add(time.Second + 400*time.Microsecond)); got != 2 {
		t.Fatalf("removed %d, want 2", got)
	}
	for _, ds := range m.dSets {
		if ds.tBuf.Length() != 2 || ds.tBuf.AtRaw(0).X != 1 || ds.tBuf.AtRaw(0).Y != 1 {
			t.Fatal("wrong survivors")
		}
		for i, raw := range ds.tBuf.ReadAllRaw() {
			if ds.tBuf.At(i) != ds.tBuf.ScaleDatum(raw) {
				t.Fatal("scaled survivors do not match raw data")
			}
		}
	}
	if &old[1] == &m.dSets[DefaultDataSetName].tBuf.ReadAllRaw()[0] {
		t.Fatal("trim retained discarded backing storage")
	}
	if m.ViewMinX() != minX || m.ViewMaxX() != maxX || m.ViewMinY() != minY || m.ViewMaxY() != maxY {
		t.Fatal("trim changed viewport")
	}
	if m.TrimBefore(base) != 0 || m.TrimBefore(base.Add(3*time.Second)) != 4 || m.TrimBefore(base.Add(3*time.Second)) != 0 {
		t.Fatal("incorrect no-op or full trim")
	}
	m.Push(TimePoint{base.Add(3 * time.Second), 5})
	if m.dSets[DefaultDataSetName].tBuf.Length() != 1 {
		t.Fatal("cannot push after full trim")
	}
}
