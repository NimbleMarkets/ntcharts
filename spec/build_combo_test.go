// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
)

func comboSpec() Spec {
	return Spec{Type: ChartTypeBar, Width: 64, Height: 16,
		XAxis: XAxis{Type: XAxisCategory, Labels: []string{"C", "A", "B"}},
		YAxis: YAxis{Format: Format{Kind: "currency"}}, Y2Axis: &YAxis{Format: Format{Kind: "percent"}},
		Data: Data{Series: []Series{
			{Name: "sales", Type: "bar", YAxis: YAxisLeft, Values: []DataPoint{{X: "B", Y: -20}, {X: "C", Y: 40}}},
			{Name: "rate", Type: "line", YAxis: YAxisRight, Values: []DataPoint{{X: "B", Y: .3}, {X: "C", Y: .1}, {X: "A", Y: .2}}},
		}}}
}

func TestComboResolution(t *testing.T) {
	s := comboSpec()
	before, _ := json.Marshal(s)
	c, err := ResolveCategoricalCombo(s)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.Labels, []string{"C", "A", "B"}) || *c.Bars[0] != 40 || c.Bars[1] != nil || *c.Bars[2] != -20 {
		t.Fatalf("categories/values shifted: %+v", c)
	}
	if c.Min != -20 || c.Max != 40 || c.Min2 != .1 || c.Max2 != .3 {
		t.Fatalf("scales not independent: %+v", c)
	}
	model, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := model.(*ComboModel)
	if !strings.Contains(m.View(), "$") || !strings.Contains(m.View(), "%") {
		t.Fatal("independent axis formats missing")
	}
	after, _ := json.Marshal(s)
	if string(before) != string(after) {
		t.Fatal("Build mutated input")
	}
	// Shared categories and reversed series order retain the same category order.
	s.Data.XAxisData = []any{"C", "A", "B"}
	s.XAxis.Labels = nil
	s.Data.Series[0], s.Data.Series[1] = s.Data.Series[1], s.Data.Series[0]
	c, err = ResolveCategoricalCombo(s)
	if err != nil {
		t.Fatal(err)
	}
	if c.LineIndex != 0 || c.BarIndex != 1 || *c.Bars[2] != -20 {
		t.Fatal("series order changed semantics")
	}
}

func comboCenter(m *ComboModel, i int) int {
	w, n := m.GraphWidth(), len(m.data.Labels)
	left, right := i*w/n, (i+1)*w/n
	return m.Origin().X + 1 + min(w-1, (left+max(left, right-1))/2)
}
func isBraille(r rune) bool { return r > runes.BrailleBlockOffset && r <= runes.BrailleBlockOffset+255 }

func TestComboMissingAndOverlay(t *testing.T) {
	s := comboSpec()
	s.YAxis = YAxis{}
	s.Y2Axis = &YAxis{}
	s.Data.Series[0].Values = []DataPoint{{Y: 10}, {Y: 10}, {Y: 10}}
	s.Data.Series[1].Values = []DataPoint{{X: "C", Y: 100}, {X: "B", Y: 100}}
	result, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(*ComboModel)
	for _, size := range [][2]int{{64, 16}, {30, 8}, {80, 20}} {
		m.Resize(size[0], size[1])
		for _, i := range []int{0, 2} {
			x := comboCenter(m, i)
			found := false
			for y := 0; y < m.GraphHeight(); y++ {
				cell := m.Canvas.Cell(canvas.Point{X: x, Y: y})
				if isBraille(cell.Rune) {
					found = true
					if !reflect.DeepEqual(cell.Style.GetForeground(), m.lineStyle.GetForeground()) {
						t.Fatal("line did not replace bar colour")
					}
				}
			}
			if !found {
				t.Fatalf("line not visible over bar center %d after resize %v", i, size)
			}
		}
		x := comboCenter(m, 1)
		for y := 0; y < m.GraphHeight(); y++ {
			if isBraille(m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune) {
				t.Fatal("line connected across missing category")
			}
		}
	}
	// A present middle value connects the line across the plot.
	s.Data.Series[1].Values = []DataPoint{{Y: 100}, {Y: 100}, {Y: 100}}
	result, err = Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m = result.(*ComboModel)
	x := (comboCenter(m, 0) + comboCenter(m, 1)) / 2
	found := false
	for y := 0; y < m.GraphHeight(); y++ {
		found = found || isBraille(m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune)
	}
	if !found {
		t.Fatal("adjacent points were not connected")
	}
}

func TestComboNegativeBarsAndZero(t *testing.T) {
	s := comboSpec()
	s.YAxis = YAxis{}
	s.Data.Series[0].Values = []DataPoint{{Y: 10}, {Y: 0}, {Y: -10}}
	s.Data.Series[1].Values = nil
	result, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(*ComboModel)
	for _, tc := range []struct {
		i, y int
		want rune
	}{{0, 0, '█'}, {0, m.GraphHeight() - 1, 0}, {1, 0, 0}, {2, 0, 0}, {2, m.GraphHeight() - 1, '█'}} {
		r := m.Canvas.Cell(canvas.Point{X: comboCenter(m, tc.i), Y: tc.y}).Rune
		if tc.want == '█' && r != '█' || tc.want == 0 && r == '█' {
			t.Fatalf("bar %d row %d: %q", tc.i, tc.y, r)
		}
	}
	for _, v := range []float64{-10, 0, 10} {
		s.Data.Series[0].Values = []DataPoint{{Y: v}}
		c, err := ResolveCategoricalCombo(s)
		if err != nil {
			t.Fatal(err)
		}
		if c.Min > 0 || c.Max < 0 || c.Min >= c.Max {
			t.Fatalf("zero baseline lost for %v: %+v", v, c)
		}
	}
}

func TestComboNarrowAndWindowResize(t *testing.T) {
	s := comboSpec()
	result, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(*ComboModel)
	for w := 0; w < 25; w++ {
		for h := 0; h < 6; h++ {
			m.Resize(w, h)
			if m.Width() != w || m.Height() != h {
				t.Fatal("resize dimensions")
			}
		}
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 64, Height: 16})
	fresh, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if updated.View() != fresh.(*ComboModel).View() {
		t.Fatal("resize failed to restore chart")
	}
	s.Width = 1
	s.Height = 1
	if _, err := Build(s); err == nil {
		t.Fatal("expected too-small build error")
	}
}

func TestComboPinsAndRejections(t *testing.T) {
	ptr := func(v float64) *float64 { return &v }
	s := comboSpec()
	s.YAxis.Min = ptr(-50)
	s.YAxis.Max = ptr(50)
	s.Y2Axis.Min = ptr(0)
	s.Y2Axis.Max = ptr(1)
	c, err := ResolveCategoricalCombo(s)
	if err != nil {
		t.Fatal(err)
	}
	if c.Min != -50 || c.Max != 50 || c.Min2 != 0 || c.Max2 != 1 {
		t.Fatal("pins ignored")
	}
	cases := map[string]func(*Spec){
		"no right axis":    func(s *Spec) { s.Y2Axis = nil },
		"wrong line axis":  func(s *Spec) { s.Data.Series[1].YAxis = YAxisLeft },
		"wrong bar axis":   func(s *Spec) { s.Data.Series[0].YAxis = YAxisRight },
		"extra series":     func(s *Spec) { s.Data.Series = append(s.Data.Series, Series{Name: "extra"}) },
		"stacked":          func(s *Spec) { s.Options.Stacked = true },
		"horizontal":       func(s *Spec) { s.Options.Orientation = OrientationHorizontal },
		"numeric x":        func(s *Spec) { s.XAxis.Type = XAxisValue },
		"log":              func(s *Spec) { s.Y2Axis.Scale = ScaleLog },
		"no labels":        func(s *Spec) { s.XAxis.Labels = nil },
		"duplicate labels": func(s *Spec) { s.XAxis.Labels[1] = "C" },
		"duplicate points": func(s *Spec) { s.Data.Series[0].Values = append(s.Data.Series[0].Values, DataPoint{X: "B", Y: 1}) },
		"unknown category": func(s *Spec) { s.Data.Series[0].Values[0].X = "other" },
		"nonfinite":        func(s *Spec) { s.Data.Series[0].Values[0].Y = math.NaN() },
		"nonfinite pin":    func(s *Spec) { s.Y2Axis.Min = ptr(math.Inf(-1)) },
		"exclude zero":     func(s *Spec) { s.YAxis.Min = ptr(1) },
		"inverted":         func(s *Spec) { s.Y2Axis.Min = ptr(2) },
		"equal pins":       func(s *Spec) { s.Y2Axis.Min = ptr(1); s.Y2Axis.Max = ptr(1) },
		"unsupported mark": func(s *Spec) { s.Data.Series[1].Type = "scatter" },
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			s := comboSpec()
			edit(&s)
			if _, err := Build(s); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestComboDoesNotChangePlainBars(t *testing.T) {
	s := comboSpec()
	s.Data.Series = s.Data.Series[:1]
	s.Y2Axis = nil
	model, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := model.(*barchart.Model); !ok {
		t.Fatalf("plain bars became %T", model)
	}
	s.Type = ChartTypeLine
	if _, err := Build(s); err == nil {
		t.Fatal("silently rendered a bar series as a line")
	}
}

func TestComboAxisLimitsAndClipping(t *testing.T) {
	s := comboSpec()
	lo, hi := 0.0, 1.0
	s.Y2Axis = &YAxis{Min: &lo, Max: &hi}
	s.Data.Series[1].Values = []DataPoint{{X: "C", Y: 0}, {X: "B", Y: 1}}
	result, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(*ComboModel)
	for _, tc := range []struct{ i, y int }{{0, m.GraphHeight() - 1}, {2, 0}} {
		if !isBraille(m.Canvas.Cell(canvas.Point{X: comboCenter(m, tc.i), Y: tc.y}).Rune) {
			t.Fatalf("lost endpoint at axis limit: %+v", tc)
		}
	}
	s.Data.Series[1].Values = []DataPoint{{Y: -1}, {Y: .5}, {Y: 2}}
	result, err = Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m = result.(*ComboModel)
	found := false
	for y := 0; y < m.GraphHeight(); y++ {
		found = found || isBraille(m.Canvas.Cell(canvas.Point{X: comboCenter(m, 1), Y: y}).Rune)
	}
	if !found {
		t.Fatal("crossing segments were lost during clipping")
	}
	// Positional trailing omissions keep slots, and explicit X beats shared X.
	s.Data.XAxisData = []any{"C", "A", "B"}
	s.Data.Series[0].Values = []DataPoint{{Y: 0}, {X: "B", Y: 3}}
	c, err := ResolveCategoricalCombo(s)
	if err != nil {
		t.Fatal(err)
	}
	if c.Bars[0] == nil || *c.Bars[0] != 0 || c.Bars[1] != nil || *c.Bars[2] != 3 {
		t.Fatal("zero, omission, or explicit category precedence lost")
	}
}

func TestComboOverlayPreservesBarBackground(t *testing.T) {
	s := comboSpec()
	lo, hi := -10.0, 10.0
	s.YAxis = YAxis{Min: &lo, Max: &hi}
	s.Y2Axis = &YAxis{Min: &lo, Max: &hi}
	s.Data.Series[0].Color = "#123abc"
	s.Data.Series[0].Values = []DataPoint{{Y: 10}, {Y: -10}, {Y: 10}}
	s.Data.Series[1].Color = "#ff9900"
	s.Data.Series[1].Values = []DataPoint{{Y: 5}, {Y: -5}, {Y: 5}}
	result, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	overlay := result.(*ComboModel)
	s.Data.Series[1].Values = nil
	result, err = Build(s)
	if err != nil {
		t.Fatal(err)
	}
	bars := result.(*ComboModel)
	for _, size := range [][2]int{{64, 16}, {30, 8}, {80, 20}} {
		overlay.Resize(size[0], size[1])
		bars.Resize(size[0], size[1])
		onBar, offBar := 0, 0
		for y := 0; y < overlay.GraphHeight(); y++ {
			for x := 0; x < overlay.Width(); x++ {
				p := canvas.Point{X: x, Y: y}
				cell := overlay.Canvas.Cell(p)
				if !isBraille(cell.Rune) {
					continue
				}
				under := bars.Canvas.Cell(p)
				if !reflect.DeepEqual(cell.Style.GetForeground(), overlay.lineStyle.GetForeground()) {
					t.Fatal("line foreground changed")
				}
				if under.Rune == '█' {
					onBar++
					if !reflect.DeepEqual(cell.Style.GetBackground(), under.Style.GetForeground()) {
						t.Fatalf("line punched a hole at %v, size %v", p, size)
					}
				} else {
					offBar++
					if !reflect.DeepEqual(cell.Style.GetBackground(), overlay.lineStyle.GetBackground()) {
						t.Fatalf("bar background leaked outside bar at %v", p)
					}
				}
			}
		}
		if onBar == 0 || offBar == 0 {
			t.Fatalf("test must cover line over bars and gaps: %d, %d", onBar, offBar)
		}
	}
}
