// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func view(t *testing.T, s Spec) string {
	t.Helper()
	m, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	return ansi.Strip(m.(interface{ View() string }).View())
}

func TestCategoryTickLabelsAreDrawnWholeAcrossEmptySlots(t *testing.T) {
	var keys, ticks []string
	var values []DataPoint
	for i := 1; i <= 24; i++ {
		key := "2026-W" + strconv.Itoa(i)
		keys = append(keys, key)
		tick := ""
		if i == 1 || i%6 == 0 {
			tick = "Week " + strconv.Itoa(i)
		}
		ticks = append(ticks, tick)
		values = append(values, DataPoint{X: key, Y: float64(i % 5)})
	}
	s := Spec{Type: ChartTypeLine, Width: 64, Height: 10, XAxis: XAxis{Type: XAxisCategory, Labels: keys, TickLabels: ticks},
		Data: Data{Series: []Series{{Name: "v", Values: values}}}}
	row := lastLine(view(t, s))
	for _, want := range []string{"Week 6", "Week 12", "Week 18", "Week 24"} {
		if !strings.Contains(row, want) {
			t.Fatalf("tick %q not drawn whole in %q", want, row)
		}
	}
	if strings.Contains(row, "2026-W") {
		t.Fatalf("category keys drawn instead of tick labels: %q", row)
	}
}

func TestBarTickLabelsReplaceTheDrawnNames(t *testing.T) {
	s := barsLabelledEvery(30, 1, 60)
	s.XAxis.TickLabels = make([]string, 30)
	for i := range s.XAxis.TickLabels {
		if (i+1)%10 == 0 {
			s.XAxis.TickLabels[i] = strconv.Itoa(i + 1)
		}
	}
	row := lastLine(view(t, s))
	if strings.Fields(row)[0] != "10" || !strings.Contains(row, "20") || !strings.Contains(row, "30") {
		t.Fatalf("row = %q, want only 10, 20 and 30", row)
	}
}

func TestBarValuesPrintAtTheEndOfEachBar(t *testing.T) {
	vertical := Spec{Type: ChartTypeBar, Width: 30, Height: 10, XAxis: XAxis{Labels: []string{"a", "b", "c"}},
		Options: Options{ShowValues: true},
		Data:    Data{Series: []Series{{Name: "v", Values: []DataPoint{{Y: 120}, {Y: 1800}, {Y: -40}}}}}}
	v := view(t, vertical)
	for _, want := range []string{"120", "1.8k", "-40"} {
		if !strings.Contains(v, want) {
			t.Fatalf("value %q missing:\n%s", want, v)
		}
	}
	if top := strings.Split(v, "\n")[0]; !strings.Contains(top, "1.8k") {
		t.Fatalf("the tallest bar's value must sit on the top row, above the bar:\n%s", v)
	}
	horizontal := vertical
	horizontal.Options.Orientation = OrientationHorizontal
	horizontal.Data.Series[0].Values = []DataPoint{{Y: 120}, {Y: 1800}, {Y: 40}}
	if h := view(t, horizontal); !strings.Contains(h, "1.8k") || !strings.Contains(h, "120") {
		t.Fatalf("horizontal values missing:\n%s", h)
	}
}

func TestScatterSeriesSymbols(t *testing.T) {
	s := Spec{Type: ChartTypeScatter, Width: 30, Height: 10, Data: Data{Series: []Series{
		{Name: "a", Symbol: "×", Values: []DataPoint{{X: 1.0, Y: 1}}},
		{Name: "b", Values: []DataPoint{{X: 2.0, Y: 2}}},
	}}}
	v := view(t, s)
	if !strings.Contains(v, "×") || !strings.Contains(v, "•") {
		t.Fatalf("want × and the default •:\n%s", v)
	}
	s.Data.Series[0].Symbol = "ab"
	if _, err := Build(s); err == nil {
		t.Fatal("a two-rune symbol must be rejected")
	}
}

func TestZeroLineWhenTheRangeCrossesZero(t *testing.T) {
	s := Spec{Type: ChartTypeLine, Width: 30, Height: 10, Options: Options{ZeroLine: true},
		Data: Data{Series: []Series{{Name: "v", Values: []DataPoint{{X: 0.0, Y: -5}, {X: 10.0, Y: 5}}}}}}
	if !strings.Contains(view(t, s), "┄") {
		t.Fatal("no zero line")
	}
	s.Data.Series[0].Values = []DataPoint{{X: 0.0, Y: 1}, {X: 10.0, Y: 5}}
	if strings.Contains(view(t, s), "┄") {
		t.Fatal("a zero line outside the range")
	}
}

func TestLegendPositionIsValidated(t *testing.T) {
	s := barSpec(1)
	s.Options.LegendPosition = LegendBottom
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Options.LegendPosition = "left"
	if err := s.Validate(); err == nil {
		t.Fatal("unknown legend position accepted")
	}
}

func TestBarValuesSkipSpacersAndNeverRunTogether(t *testing.T) {
	s := Spec{Type: ChartTypeBar, Width: 11, Height: 8, XAxis: XAxis{Labels: []string{"a", "b", "", "c"}},
		Options: Options{ShowValues: true},
		Data:    Data{Series: []Series{{Name: "v", Values: []DataPoint{{Y: 190}, {Y: 260}, {Y: 0}, {Y: 330}}}}}}
	v := view(t, s)
	if strings.Contains(v, "190260") || strings.Contains(v, " 0 ") {
		t.Fatalf("values ran together or a spacer was labelled:\n%s", v)
	}
}
