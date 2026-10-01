// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"encoding/json"
	"math"
	"slices"
	"strings"
	"testing"
	"time"
)

// logKinds are the chart types that honour a log Y axis.
var logKinds = []ChartType{ChartTypeLine, ChartTypeScatter, ChartTypeTimeSeries, ChartTypeOHLC}

// logSpec returns a chart of the given kind whose Y values (and OHLC
// low/high) are ys, at X = 1, 10, 100, ... (ms-since-epoch on time axes).
func logSpec(kind ChartType, ys ...float64) Spec {
	ser := Series{Name: "a"}
	lo, hi := slices.Min(ys), slices.Max(ys)
	for i, y := range ys {
		x := math.Pow10(i)
		ser.Values = append(ser.Values, DataPoint{X: x, Y: y})
		ser.OHLC = append(ser.OHLC, OHLCPoint{T: x, O: y, H: hi, L: lo, C: y})
	}
	return Spec{
		Type: kind, Width: 40, Height: 14,
		YAxis: YAxis{Scale: ScaleLog},
		Data:  Data{Series: []Series{ser}},
	}
}

func rawView(t *testing.T, s Spec) string {
	t.Helper()
	m, err := Build(s)
	if err != nil {
		t.Fatalf("Build(%s): %v", s.Type, err)
	}
	return m.(interface{ View() string }).View()
}

func viewOf(t *testing.T, s Spec) string {
	t.Helper()
	return stripAnsi(rawView(t, s))
}

// yLabels returns the non-empty Y tick labels of a view, top to bottom: the
// text left of the first box-drawing or braille rune on each row.
func yLabels(view string) []string {
	var labels []string
	lines := strings.Split(view, "\n")
	for _, line := range lines[:len(lines)-1] { // last row holds the X labels
		i := strings.IndexFunc(line, func(r rune) bool { return r >= 0x2500 })
		if i < 0 {
			continue
		}
		if l := strings.TrimSpace(line[:i]); l != "" {
			labels = append(labels, l)
		}
	}
	return labels
}

// plotRows returns the view without its X label row.
func plotRows(view string) string {
	return view[:strings.LastIndex(view, "\n")]
}

func yRange(t *testing.T, s Spec) (float64, float64) {
	t.Helper()
	model, err := Build(s)
	if err != nil {
		t.Fatalf("Build(%s): %v", s.Type, err)
	}
	m := model.(interface {
		ViewMinY() float64
		ViewMaxY() float64
	})
	return m.ViewMinY(), m.ViewMaxY()
}

func wantBuildError(t *testing.T, s Spec, want string) {
	t.Helper()
	if _, err := Build(s); err == nil || err.Error() != want {
		t.Fatalf("Build(%s) error:\n got %v\nwant %s", s.Type, err, want)
	}
}

// TestLogYDecadeLabels spans six decades on a 12-row graph, so every other
// row is a decade tick. Each must print its power of ten exactly, in every
// format, on every chart type that honours a log Y axis.
func TestLogYDecadeLabels(t *testing.T) {
	for _, tc := range []struct {
		name   string
		format Format
		want   []string
	}{
		{"default", Format{}, []string{"1M", "100k", "10k", "1k", "100", "10", "1"}},
		{"number", Format{Kind: "number"}, []string{"1000000", "100000", "10000", "1000", "100", "10", "1"}},
		{"currency", Format{Kind: "currency"}, []string{"$1000000.00", "$100000.00", "$10000.00", "$1000.00", "$100.00", "$10.00", "$1.00"}},
		{"si", Format{Kind: "si"}, []string{"1.0M", "100.0k", "10.0k", "1.0k", "100.0", "10.0", "1.0"}},
		{"percent", Format{Kind: "percent"}, []string{"100000000%", "10000000%", "1000000%", "100000%", "10000%", "1000%", "100%"}},
	} {
		for _, kind := range logKinds {
			t.Run(tc.name+"/"+string(kind), func(t *testing.T) {
				s := logSpec(kind, 1, 30, 2000, 1e6)
				s.YAxis.Format = tc.format
				view := viewOf(t, s)
				if got := yLabels(view); !slices.Equal(got, tc.want) {
					t.Fatalf("labels %q, want %q\n%s", got, tc.want, view)
				}
			})
		}
	}
}

func TestLogYDecadeLabelsBelowOne(t *testing.T) {
	s := logSpec(ChartTypeScatter, 0.001, 0.5, 40, 1000)
	s.YAxis.Format = Format{Kind: "number"}
	want := []string{"1000", "100", "10", "1", "0.1", "0.01", "0.001"}
	if got := yLabels(viewOf(t, s)); !slices.Equal(got, want) {
		t.Fatalf("labels %q, want %q", got, want)
	}
}

// math.Log10(1000) is 2.9999999999999996; its floor must not start the axis
// a decade low. Ranges that are already whole decades stay put, in data units.
func TestLogDecadesAreExact(t *testing.T) {
	for exp := -12; exp <= 15; exp++ {
		v := math.Pow10(exp)
		if lo, hi := logRange(v, v*1000, false, false); lo != v || hi != math.Pow10(exp+3) {
			t.Fatalf("logRange(1e%d, 1e%d) = %v..%v", exp, exp+3, lo, hi)
		}
	}
	for _, kind := range logKinds {
		if lo, hi := yRange(t, logSpec(kind, 1000, 1e6)); lo != 1000 || hi != 1e6 {
			t.Fatalf("%s: got %v..%v, want 1000..1e6", kind, lo, hi)
		}
	}
}

func TestLogDefaultLabels(t *testing.T) {
	label := axisLabelFormatter(Format{}, ScaleLog)
	for _, tc := range []struct {
		v    float64
		want string
	}{
		{1, "1"}, {10, "10"}, {100, "100"}, {1000, "1k"}, {1e6, "1M"}, {1e9, "1G"}, {1e12, "1T"},
		{0.1, "0.1"}, {0.001, "0.001"}, {0.0001, "0.0001"}, {1e-5, "1e-05"},
		{1e14, "100T"}, {1e15, "1e+15"},
		{math.Sqrt(10), "3.16"}, {31.6227766, "31.6"}, {316.227766, "316"},
		{3162.27766, "3.16k"}, {31622.7766, "31.6k"}, {0.0316227766, "0.0316"},
		{999.6, "1k"}, {999600, "1M"}, {2.5e6, "2.5M"}, {50, "50"},
	} {
		if got := label(0, tc.v); got != tc.want {
			t.Errorf("label(%v) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

// TestLogTicksAtUnevenHeight is the case plotting logs on a linear axis got
// wrong: 4 decades on 7 rows. Stepping every second row would label
// $1.00, $13.89, $193.07, ... ; the ticks sit on powers of ten instead
// (every second one, since adjacent decades are under two rows apart).
func TestLogTicksAtUnevenHeight(t *testing.T) {
	s := Spec{
		Type: ChartTypeTimeSeries, Width: 44, Height: 9,
		YAxis: YAxis{Scale: ScaleLog, Format: Format{Kind: "currency"}},
		Data:  Data{Series: []Series{{Name: "revenue"}}},
	}
	for i, y := range []float64{2.5, 4, 9, 30, 55, 180, 700, 1500, 4200, 9800} {
		s.Data.Series[0].Values = append(s.Data.Series[0].Values, DataPoint{
			X: time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC), Y: y,
		})
	}
	var trimmed []string
	for _, line := range strings.Split(viewOf(t, s), "\n") {
		trimmed = append(trimmed, strings.TrimRight(line, " "))
	}
	const want = `$10000.00│                             ⢀⣀⠤⠒⠉
         │                       ⢀⣀⡠⠤⠒⠊⠁
         │                   ⢀⡠⠔⠊⠁
  $100.00│              ⢀⣀⠤⠒⠉⠁
         │         ⣀⠤⠒⠊⠉⠁
         │  ⣀⣀⣀⠤⠤⠒⠉
         │⠉⠉
    $1.00└──────────────────────────────────
         '26 01/01   01/04   01/06   01/08`
	if got := strings.Join(trimmed, "\n"); got != want {
		t.Fatalf("view:\n%s\n--- want ---\n%s", got, want)
	}

	// The same holds at every height: only powers of ten are labelled.
	for h := 5; h <= 30; h++ {
		s.Height = h
		labels := yLabels(viewOf(t, s))
		for _, l := range labels {
			if !slices.Contains([]string{"$1.00", "$10.00", "$100.00", "$1000.00", "$10000.00"}, l) {
				t.Fatalf("height %d: label %s is not a power of ten (%q)", h, l, labels)
			}
		}
		if len(labels) < 2 {
			t.Fatalf("height %d: labels %q", h, labels)
		}
	}
}

func wantTrimmedView(t *testing.T, s Spec, want string) {
	t.Helper()
	var trimmed []string
	for _, line := range strings.Split(viewOf(t, s), "\n") {
		trimmed = append(trimmed, strings.TrimRight(line, " "))
	}
	if got := strings.Join(trimmed, "\n"); got != strings.Trim(want, "\n") {
		t.Fatalf("view:\n%s\n--- want ---\n%s", got, strings.Trim(want, "\n"))
	}
}

// When decades are too close to label each one, the kept labels count from
// the ends of the axis, so the range can be read off the chart: here
// $10 .. $100000 on seven rows, where exponent parity alone would label
// only $100 and $10000.
func TestLogThinnedTicksLabelTheEnds(t *testing.T) {
	s := Spec{
		Type: ChartTypeTimeSeries, Width: 56, Height: 9,
		YAxis: YAxis{Scale: ScaleLog, Format: Format{Kind: "currency"}},
		Data:  Data{Series: []Series{{Name: "revenue"}}},
	}
	for i, y := range []float64{12, 30, 25, 140, 600, 450, 3000, 9000, 20000, 85000} {
		s.Data.Series[0].Values = append(s.Data.Series[0].Values, DataPoint{
			X: time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC), Y: y,
		})
	}
	wantTrimmedView(t, s, `
$100000.00│                                         ⢀⡠⠔⠊
          │                                  ⣀⣀⠤⠤⠔⠒⠊⠁
          │                            ⢀⡠⠔⠒⠊⠉
  $1000.00│                   ⢀⣀⣀⡀  ⣀⠤⠊⠁
          │               ⣀⠤⠒⠊⠁  ⠈⠉⠉
          │           ⣀⠤⠒⠉
          │⠤⠔⠒⠒⠉⠉⠉⠉⠉⠉⠉
    $10.00└─────────────────────────────────────────────
          '26 01/01   01/03   01/05   01/06   01/08
`)
	if lo, hi := yRange(t, s); lo != 10 || hi != 1e5 {
		t.Fatalf("Y range %v..%v, want 10..100000", lo, hi)
	}

	// Scatter, both axes log: Y spans 0.1 .. 100k on ten rows.
	s = Spec{
		Type: ChartTypeScatter, Width: 44, Height: 12,
		XAxis: XAxis{Scale: ScaleLog},
		YAxis: YAxis{Scale: ScaleLog},
		Data: Data{Series: []Series{{Name: "a", Values: []DataPoint{
			{X: 2.0, Y: 0.3}, {X: 9.0, Y: 4}, {X: 40.0, Y: 2}, {X: 150.0, Y: 90},
			{X: 700.0, Y: 1200}, {X: 3000.0, Y: 800}, {X: 8000.0, Y: 60000},
		}}}},
	}
	wantTrimmedView(t, s, `
100k│                                     •
    │
    │
  1k│                           •     •
    │
    │                     •
    │
  10│         •     •
    │   •
    │
 0.1└───────────────────────────────────────
    1         10        100      1k      10k
`)
	if lo, hi := yRange(t, s); lo != 0.1 || hi != 1e5 {
		t.Fatalf("Y range %v..%v, want 0.1..100000", lo, hi)
	}
}

// TestLogYMatchesLinearPlotOfLogs draws each chart type twice: with a log Y
// axis, and with a linear Y axis over the same data's base-10 logs. The
// plotted cells must be identical. The label formats are chosen so both
// charts reserve six label columns ("$10000" vs "4.0000").
func TestLogYMatchesLinearPlotOfLogs(t *testing.T) {
	for _, kind := range logKinds {
		t.Run(string(kind), func(t *testing.T) {
			logged := logSpec(kind, 3, 400, 20, 9000, 70)
			logged.YAxis.Format = Format{Kind: "currency", Precision: f64i(0)}

			linear := logSpec(kind, 3, 400, 20, 9000, 70)
			linear.YAxis = YAxis{Min: f64(0), Max: f64(4), Format: Format{Kind: "number", Precision: f64i(4)}}
			ser := &linear.Data.Series[0]
			for i := range ser.Values {
				ser.Values[i].Y = math.Log10(ser.Values[i].Y)
				o := &ser.OHLC[i]
				o.O, o.H, o.L, o.C = math.Log10(o.O), math.Log10(o.H), math.Log10(o.L), math.Log10(o.C)
			}

			got, want := viewOf(t, logged), viewOf(t, linear)
			strip := func(view string) string {
				var b strings.Builder
				for _, line := range strings.Split(plotRows(view), "\n") {
					b.WriteString(string([]rune(line)[6:]) + "\n")
				}
				return b.String()
			}
			if strip(got) != strip(want) {
				t.Fatalf("log plot differs from linear plot of logs:\n%s\n--- want ---\n%s", got, want)
			}
			if labels := yLabels(got); labels[0] != "$10000" || labels[len(labels)-1] != "$1" {
				t.Fatalf("labels %q", labels)
			}
		})
	}
}

// TestLogXMatchesLinearPlotOfLogs is the X-axis counterpart for the two
// numeric-X chart types. X runs 1..1000, so the decade range is 0..3 in both.
func TestLogXMatchesLinearPlotOfLogs(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeScatter} {
		t.Run(string(kind), func(t *testing.T) {
			points := []DataPoint{{X: 1.0, Y: 2}, {X: 4.0, Y: 5}, {X: 30.0, Y: 1}, {X: 250.0, Y: 4}, {X: 1000.0, Y: 3}}
			logged := Spec{
				Type: kind, Width: 40, Height: 10,
				XAxis: XAxis{Scale: ScaleLog},
				Data:  Data{Series: []Series{{Name: "a", Values: points}}},
			}
			linear := logged
			linear.XAxis = XAxis{}
			linear.Data = Data{Series: []Series{{Name: "a"}}}
			for _, p := range points {
				p.X = math.Log10(p.X.(float64))
				linear.Data.Series[0].Values = append(linear.Data.Series[0].Values, p)
			}

			got, want := viewOf(t, logged), viewOf(t, linear)
			if plotRows(got) != plotRows(want) {
				t.Fatalf("log plot differs from linear plot of logs:\n%s\n--- want ---\n%s", got, want)
			}
			xLabels := strings.Fields(got[strings.LastIndex(got, "\n"):])
			if want := []string{"1", "10", "100", "1k"}; !slices.Equal(xLabels, want) {
				t.Fatalf("x labels %q, want %q\n%s", xLabels, want, got)
			}
			m, _ := Build(logged)
			x := m.(interface {
				ViewMinX() float64
				ViewMaxX() float64
			})
			if x.ViewMinX() != 1 || x.ViewMaxX() != 1000 {
				t.Fatalf("x range %v..%v, want 1..1000", x.ViewMinX(), x.ViewMaxX())
			}
		})
	}
}

func TestLogXExpandsToDecadesAndFormats(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeScatter} {
		s := Spec{
			Type: kind, Width: 40, Height: 10,
			XAxis: XAxis{Scale: ScaleLog, Format: Format{Kind: "si", Precision: f64i(0)}},
			YAxis: YAxis{Scale: ScaleLog},
			Data: Data{Series: []Series{{Name: "a", Values: []DataPoint{
				{X: 20.0, Y: 5}, {X: 700.0, Y: 50}, {X: 30000.0, Y: 500},
			}}}},
		}
		model, err := Build(s)
		if err != nil {
			t.Fatal(err)
		}
		m := model.(interface {
			ViewMinX() float64
			ViewMaxX() float64
			ViewMinY() float64
			ViewMaxY() float64
			View() string
		})
		if m.ViewMinX() != 10 || m.ViewMaxX() != 1e5 || m.ViewMinY() != 1 || m.ViewMaxY() != 1000 {
			t.Fatalf("%s: x %v..%v y %v..%v", kind, m.ViewMinX(), m.ViewMaxX(), m.ViewMinY(), m.ViewMaxY())
		}
		view := m.View()
		xLabels := strings.Fields(view[strings.LastIndex(view, "\n"):])
		if want := []string{"10", "100", "1k", "10k", "100k"}; !slices.Equal(xLabels, want) {
			t.Fatalf("%s: x labels %q\n%s", kind, xLabels, view)
		}
	}
}

// A single X value (or a flat Y series) on a power of ten has no decade to
// round out to, so the range widens by one decade on each side.
func TestLogFlatRangesWiden(t *testing.T) {
	for _, kind := range logKinds {
		if lo, hi := yRange(t, logSpec(kind, 100, 100)); lo != 10 || hi != 1000 {
			t.Fatalf("%s: flat Y got %v..%v, want 10..1000", kind, lo, hi)
		}
		if lo, hi := yRange(t, logSpec(kind, 50, 50)); lo != 10 || hi != 100 {
			t.Fatalf("%s: flat off-decade Y got %v..%v, want 10..100", kind, lo, hi)
		}
	}
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeScatter} {
		s := logSpec(kind, 5)
		s.XAxis.Scale = ScaleLog
		s.Data.Series[0].Values[0].X = 100.0
		m, err := Build(s)
		if err != nil {
			t.Fatal(err)
		}
		x := m.(interface {
			ViewMinX() float64
			ViewMaxX() float64
		})
		if x.ViewMinX() != 10 || x.ViewMaxX() != 1000 {
			t.Fatalf("%s: flat X got %v..%v, want 10..1000", kind, x.ViewMinX(), x.ViewMaxX())
		}
	}
}

// TestLogYPins applies the one-sided pin rule on a log axis: pins are data
// values, used exactly; an unpinned bound is the data's whole decade. The
// model reports its range in data units. The data spans 2..5000.
func TestLogYPins(t *testing.T) {
	for _, kind := range logKinds {
		for _, tc := range []struct {
			name     string
			min, max *float64
			lo, hi   float64
		}{
			{"none", nil, nil, 1, 1e4},
			{"min", f64(0.5), nil, 0.5, 1e4},
			{"max", nil, f64(20000), 1, 20000},
			{"both", f64(1), f64(1e5), 1, 1e5},
			{"both off-decade", f64(3), f64(300), 3, 300},
			{"min inside data", f64(50), nil, 50, 1e4},
			{"min at data max", f64(5000), nil, 5000, 1e4},
			{"max at data min", nil, f64(2), 1, 2},
		} {
			t.Run(string(kind)+"/"+tc.name, func(t *testing.T) {
				s := logSpec(kind, 2, 40, 5000)
				s.YAxis.Min, s.YAxis.Max = tc.min, tc.max
				if lo, hi := yRange(t, s); lo != tc.lo || hi != tc.hi {
					t.Fatalf("got %v..%v, want %v..%v", lo, hi, tc.lo, tc.hi)
				}
			})
		}

		// A lone pin on the same power of ten as the data's opposite extreme
		// leaves no decade to round to: only the unpinned side widens.
		t.Run(string(kind)+"/min at decade data max", func(t *testing.T) {
			s := logSpec(kind, 10, 1000)
			s.YAxis.Min = f64(1000)
			if lo, hi := yRange(t, s); lo != 1000 || hi != 1e4 {
				t.Fatalf("got %v..%v, want 1000..10000", lo, hi)
			}
		})
		t.Run(string(kind)+"/max at decade data min", func(t *testing.T) {
			s := logSpec(kind, 10, 1000)
			s.YAxis.Max = f64(10)
			if lo, hi := yRange(t, s); lo != 1 || hi != 10 {
				t.Fatalf("got %v..%v, want 1..10", lo, hi)
			}
		})

		t.Run(string(kind)+"/equal pins", func(t *testing.T) {
			s := logSpec(kind, 2, 40, 5000)
			s.YAxis.Min, s.YAxis.Max = f64(30), f64(30)
			wantBuildError(t, s, "spec: y_axis min and max must differ")
		})
		t.Run(string(kind)+"/inverted pins", func(t *testing.T) {
			s := logSpec(kind, 2, 40, 5000)
			s.YAxis.Min, s.YAxis.Max = f64(300), f64(30)
			wantBuildError(t, s, "spec: y_axis min 300 exceeds max 30")
		})
		t.Run(string(kind)+"/min above data", func(t *testing.T) {
			s := logSpec(kind, 2, 40, 5000)
			s.YAxis.Min = f64(6000)
			wantBuildError(t, s, "spec: y_axis min 6000 exceeds max 5000")
		})
		t.Run(string(kind)+"/max below data", func(t *testing.T) {
			s := logSpec(kind, 2, 40, 5000)
			s.YAxis.Max = f64(1.5)
			wantBuildError(t, s, "spec: y_axis min 2 exceeds max 1.5")
		})
	}
}

// Off-decade pins bound the axis, but the labels stay on the round values
// inside them.
func TestLogYPinnedBoundLabels(t *testing.T) {
	s := logSpec(ChartTypeScatter, 2, 40, 5000)
	s.YAxis.Min, s.YAxis.Max = f64(0.5), f64(20000)
	want := []string{"10k", "1k", "100", "10", "1"}
	if labels := yLabels(viewOf(t, s)); !slices.Equal(labels, want) {
		t.Fatalf("labels %q, want %q", labels, want)
	}
}

func TestLogRejectsNonPositivePins(t *testing.T) {
	for _, kind := range logKinds {
		for _, v := range []float64{0, -5} {
			s := logSpec(kind, 2, 40, 5000)
			s.YAxis.Min = f64(v)
			wantBuildError(t, s, `spec: y_axis scale "log" requires a positive finite min; got `+FormatValue(Format{}, v))
			s = logSpec(kind, 2, 40, 5000)
			s.YAxis.Max = f64(v)
			wantBuildError(t, s, `spec: y_axis scale "log" requires a positive finite max; got `+FormatValue(Format{}, v))
		}
	}
}

func TestLogRejectsNonPositiveValues(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeLine, ChartTypeScatter, ChartTypeTimeSeries} {
		for _, tc := range []struct {
			v    float64
			text string
		}{{0, "0"}, {-3, "-3"}, {math.NaN(), "NaN"}, {math.Inf(1), "+Inf"}} {
			s := logSpec(kind, 2, tc.v, 5000)
			wantBuildError(t, s, `spec: y_axis scale "log" requires positive finite values; series "a" point 1 has y = `+tc.text)
		}
	}

	for _, tc := range []struct {
		name  string
		set   func(*OHLCPoint)
		field string
	}{
		{"open", func(p *OHLCPoint) { p.O = 0 }, "open = 0"},
		{"high", func(p *OHLCPoint) { p.H = -1 }, "high = -1"},
		{"low", func(p *OHLCPoint) { p.L = -0.5 }, "low = -0.5"},
		{"close", func(p *OHLCPoint) { p.C = 0 }, "close = 0"},
	} {
		s := logSpec(ChartTypeOHLC, 2, 40, 5000)
		tc.set(&s.Data.Series[0].OHLC[2])
		wantBuildError(t, s, `spec: y_axis scale "log" requires positive finite values; ohlc point 2 has `+tc.field)
	}

	for _, kind := range []ChartType{ChartTypeLine, ChartTypeScatter} {
		s := logSpec(kind, 2, 40, 5000)
		s.XAxis.Scale = ScaleLog
		s.Data.Series[0].Values[1].X = -10.0
		wantBuildError(t, s, `spec: x_axis scale "log" requires positive finite values; series "a" point 1 has x = -10`)

		// An omitted X falls back to the point index, and index 0 has no log.
		s = logSpec(kind, 2, 40, 5000)
		s.XAxis.Scale = ScaleLog
		for i := range s.Data.Series[0].Values {
			s.Data.Series[0].Values[i].X = nil
		}
		wantBuildError(t, s, `spec: x_axis scale "log" requires positive finite values; series "a" point 0 has x = 0`)

		// The same values are fine while the axis is linear.
		s.XAxis.Scale = ScaleLinear
		if _, err := Build(s); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLogXRejectedOnTimeAxes(t *testing.T) {
	for _, kind := range []ChartType{ChartTypeTimeSeries, ChartTypeOHLC} {
		s := logSpec(kind, 2, 40, 5000)
		s.YAxis.Scale = ""
		s.XAxis.Scale = ScaleLog
		wantBuildError(t, s, `spec: `+string(kind)+` chart cannot draw x_axis scale "log": its X axis is time`)
	}
}

func TestLogRejectedByChartsWithoutAValueAxis(t *testing.T) {
	for _, tc := range []struct {
		spec Spec
		why  string
	}{
		{barSpec(1), "bars are measured linearly from a zero baseline"},
		{heatSpec(), "its axes are cell indices, not scaled values"},
		{sparkSpec(), "columns are measured linearly from zero and it draws no axis"},
	} {
		s := tc.spec
		if _, err := Build(s); err != nil {
			t.Fatalf("%s: linear build: %v", s.Type, err)
		}
		s.YAxis.Scale = ScaleLog
		wantBuildError(t, s, `spec: `+string(s.Type)+` chart cannot draw y_axis scale "log": `+tc.why)
		s.YAxis.Scale = ScaleLinear
		s.XAxis.Scale = ScaleLog
		wantBuildError(t, s, `spec: `+string(s.Type)+` chart cannot draw x_axis scale "log": `+tc.why)
	}
}

// TestLinearScaleUnchanged pins the linear renderings: an explicit "linear"
// scale draws exactly what an empty one does, and the line chart still
// matches the golden recorded before axes had a scale.
func TestLinearScaleUnchanged(t *testing.T) {
	for _, s := range []Spec{
		lineSpec(), scatterSpec(), rangeSpec(ChartTypeTimeSeries), ohlcSpec(),
		barSpec(1), heatSpec(), sparkSpec(),
	} {
		want := rawView(t, s) // styled: heatmap cells differ only in colour
		s.XAxis.Scale, s.YAxis.Scale = ScaleLinear, ScaleLinear
		if got := rawView(t, s); got != want {
			t.Fatalf("%s: explicit linear scale changed the view:\n%s\n--- want ---\n%s", s.Type, got, want)
		}
	}

	const golden = `5│                                     ╭
 │                                     │
4├╮                       ╭╮           │
 ││           ╭╮          ││           │
2││           ││          ││           │
 ││           ├┤          ├┤           │
1├┤           ││          ││           ├
 ││           ││          ││           │
0└┴───────────┴┴──────────┴┴───────────┴
 0       1           2           3      `
	if got := viewOf(t, lineSpec()); got != golden {
		t.Fatalf("linear line chart changed:\n%s", got)
	}
}

func TestLogBuildDoesNotMutateSpec(t *testing.T) {
	for _, kind := range logKinds {
		s := logSpec(kind, 2, 40, 5000)
		s.YAxis.Min = f64(1)
		if kind == ChartTypeLine || kind == ChartTypeScatter {
			s.XAxis.Scale = ScaleLog
		}
		before, _ := json.Marshal(s)
		if _, err := Build(s); err != nil {
			t.Fatal(err)
		}
		if after, _ := json.Marshal(s); string(before) != string(after) {
			t.Fatalf("%s: Build changed its input", kind)
		}
	}
}
