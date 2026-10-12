package spec

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/charmbracelet/x/ansi"
)

func categoricalSpec(kind ChartType) Spec {
	return Spec{Type: kind, Width: 44, Height: 15, XAxis: XAxis{Type: XAxisCategory, Labels: []string{"Before", "Middle", "After"}},
		YAxis: YAxis{Min: f64(0), Max: f64(10)}, Data: Data{Series: []Series{{Name: "sample", Values: []DataPoint{{X: "After", Y: 5}, {X: "Before", Y: 5}}}}}}
}

func TestCategoricalSlotsAndSparseLine(t *testing.T) {
	s := categoricalSpec(ChartTypeLine)
	before := s.Data.Series[0].Values[0].X
	built, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := built.(*CategoricalModel)
	if s.Data.Series[0].Values[0].X != before {
		t.Fatal("Build mutated input")
	}
	for _, label := range s.XAxis.Labels {
		if !strings.Contains(ansi.Strip(m.View()), label) {
			t.Fatalf("missing %s", label)
		}
	}
	centers := categoryCenters(m.GraphWidth(), 3)
	x := m.Origin().X + 1 + centers[1]
	for y := 0; y < m.GraphHeight(); y++ {
		if r := m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune; r != 0 && r != ' ' {
			t.Fatalf("line bridged empty slot at %d,%d: %q", x, y, r)
		}
	}
	s.Data.Series[0].Values = append(s.Data.Series[0].Values, DataPoint{X: "Middle", Y: 5})
	built, err = Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m = built.(*CategoricalModel)
	found := false
	for y := 0; y < m.GraphHeight(); y++ {
		r := m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune
		found = found || (r >= '\u2801' && r <= '\u28ff')
	}
	if !found {
		t.Fatal("full line not drawn through middle slot")
	}
}

func TestCategoricalScatterRepeatsAndResize(t *testing.T) {
	s := categoricalSpec(ChartTypeScatter)
	s.Data.Series[0].Values = append(s.Data.Series[0].Values, DataPoint{X: "Before", Y: 8})
	built, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m := built.(*CategoricalModel)
	for _, size := range [][2]int{{80, 24}, {12, 6}, {44, 15}, {0, 0}, {44, 15}} {
		updated, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		*m = updated
		if size[0] < 20 {
			continue
		}
		fresh := s
		fresh.Width, fresh.Height = size[0], size[1]
		model, err := Build(fresh)
		if err != nil {
			t.Fatal(err)
		}
		if m.View() != model.(*CategoricalModel).View() {
			t.Fatal("resize differs from fresh build")
		}
	}
	if got := strings.Count(ansi.Strip(m.View()), "•"); got != 3 {
		t.Fatalf("scatter dots=%d, want 3", got)
	}
}

func TestCategoricalOrderSources(t *testing.T) {
	s := categoricalSpec(ChartTypeLine)
	s.XAxis.Labels = nil
	m, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.(*CategoricalModel).labels, []string{"After", "Before"}) {
		t.Fatal("first appearance order lost")
	}
	s.Data.XAxisData = []any{"Before", "Middle", "After"}
	s.Data.Series[0].Values = []DataPoint{{Y: 2}, {Y: 4}, {Y: 6}}
	m, err = Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.(*CategoricalModel).labels, []string{"Before", "Middle", "After"}) {
		t.Fatal("shared order lost")
	}
}

func TestCategoricalValidation(t *testing.T) {
	cases := map[string]func(*Spec){
		"log X":            func(s *Spec) { s.XAxis.Scale = ScaleLog },
		"duplicate labels": func(s *Spec) { s.XAxis.Labels = []string{"Before", "Before"} },
		"unknown category": func(s *Spec) { s.Data.Series[0].Values[0].X = "unknown" },
		"numeric category": func(s *Spec) { s.Data.Series[0].Values[0].X = 42 },
		"duplicate line":   func(s *Spec) { s.Data.Series[0].Values = append(s.Data.Series[0].Values, s.Data.Series[0].Values[0]) },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			s := categoricalSpec(ChartTypeLine)
			change(&s)
			if _, err := Build(s); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestCategoricalDualLogY(t *testing.T) {
	s := categoricalSpec(ChartTypeScatter)
	s.YAxis = YAxis{Scale: ScaleLog, Min: f64(1), Max: f64(10)}
	s.Y2Axis = &YAxis{Scale: ScaleLog, Min: f64(100), Max: f64(1000)}
	s.Data.Series = append(s.Data.Series, Series{Name: "right", YAxis: YAxisRight, Values: []DataPoint{{X: "Middle", Y: 300}}})
	model, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(ansi.Strip(model.(*CategoricalModel).View()), "•"); got != 3 {
		t.Fatalf("dots=%d", got)
	}
}
