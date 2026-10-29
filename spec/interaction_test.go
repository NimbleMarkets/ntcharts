package spec

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/charmbracelet/x/ansi"
	"reflect"
	"strings"
	"testing"
)

func meta(id string) *MarkMetadata {
	return &MarkMetadata{ID: id, Fields: map[string]any{"Country": id, "nested": map[string]any{"v": "original"}}}
}
func interactionSpecs() []Spec {
	bar := Spec{Type: ChartTypeBar, Width: 40, Height: 12, XAxis: XAxis{Labels: []string{"A", "B", "C"}}, Data: Data{Series: []Series{{Name: "sales", Values: []DataPoint{{Y: 5, Mark: meta("a")}, {Y: -3, Mark: meta("b")}, {Y: 0, Mark: meta("zero")}}}}}}
	horizontal := bar
	horizontal.Options.Orientation = OrientationHorizontal
	scatter := Spec{Type: ChartTypeScatter, Width: 40, Height: 12, Data: Data{Series: []Series{{Name: "s", Values: []DataPoint{{X: 1.0, Y: 2, Mark: meta("a")}, {X: 1.0, Y: 2, Mark: meta("b")}, {X: 3.0, Y: 4, Mark: meta("c")}}}}}}
	heat := Spec{Type: ChartTypeHeatmap, Width: 40, Height: 12, XAxis: XAxis{Labels: []string{"Mon", "Tue"}}, YAxis: YAxis{Labels: []string{"AM", "PM"}}, Heat: &HeatData{Cells: []HeatCell{{X: 0, Y: 0, Z: 1, Mark: meta("a")}, {X: 1, Y: 1, Z: 2, Mark: meta("b")}}}}
	return []Spec{bar, horizontal, scatter, heat}
}
func TestInteractionLifecycle(t *testing.T) {
	for _, s := range interactionSpecs() {
		m, err := BuildInteractive(s)
		if err != nil {
			t.Fatal(err)
		}
		original := m.View()
		cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		if cmd == nil {
			t.Fatal("missing inspection event")
		}
		event := cmd().(InspectionMsg)
		if event.Mark.Fields["Country"] != "a" {
			t.Fatal(event)
		}
		event.Mark.Fields["nested"].(map[string]any)["v"] = "mutated"
		focused, _ := m.Focused()
		if focused.Fields["nested"].(map[string]any)["v"] != "original" {
			t.Fatal("event aliases model")
		}
		cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		selection := cmd().(SelectionMsg)
		if len(selection.Marks) != 1 || selection.Marks[0].ID != "a" {
			t.Fatal(selection)
		}
		if m.View() == original {
			t.Fatal("selection/focus has no visible effect")
		}
		if err := m.Resize(60, 18); err != nil {
			t.Fatal(err)
		}
		focused, _ = m.Focused()
		if focused.ID != "a" || len(m.Selection().Marks) != 1 {
			t.Fatal("resize lost state")
		}
		before := m.Selection()
		if _, err := m.Highlight("b", "missing"); err == nil {
			t.Fatal("unknown id accepted")
		}
		if !reflect.DeepEqual(before, m.Selection()) {
			t.Fatal("failed highlight changed selection")
		}
		selected, err := m.Highlight("b")
		if err != nil || selected.Marks[0].ID != "b" {
			t.Fatal(selected, err)
		}
		if len(m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})().(SelectionMsg).Marks) != 0 {
			t.Fatal("escape did not clear")
		}
		// Every hit refers to a real mark; axes and margins are not hit targets.
		if len(m.HitTest(canvas.Point{X: -1, Y: 0})) != 0 {
			t.Fatal("outside hit")
		}
		hits := 0
		for p := range m.cells {
			hits += len(m.HitTest(p))
		}
		if hits == 0 {
			t.Fatal("no rendered hit targets")
		}
	}
}
func TestInteractionOverlapAndOwnership(t *testing.T) {
	s := interactionSpecs()[2]
	m, err := BuildInteractive(s)
	if err != nil {
		t.Fatal(err)
	}
	s.Data.Series[0].Values[0].Mark.Fields["Country"] = "changed"
	if m.Marks()[0].Fields["Country"] != "a" {
		t.Fatal("input aliases model")
	}
	found := false
	for p := range m.cells {
		if len(m.HitTest(p)) == 2 {
			found = true
		}
	}
	if !found {
		t.Fatal("overlapping scatter observations lost")
	}
	_, _ = m.Inspect("b")
	if f, _ := m.Focused(); f.ID != "b" {
		t.Fatal("cannot inspect overlap")
	}
}
func TestInteractionMetadataValidationAndSyntheticSlots(t *testing.T) {
	s := interactionSpecs()[0]
	s.Data.Series[0].Values[1].Mark.ID = "a"
	if _, err := BuildInteractive(s); err == nil {
		t.Fatal("duplicate ID accepted")
	}
	s = interactionSpecs()[0]
	s.Data.Series[0].Values[1].Mark = nil
	m, err := BuildInteractive(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Marks()) != 2 {
		t.Fatal("synthetic slot became selectable")
	}
	if _, err := BuildInteractive(Spec{Type: ChartTypeLine}); err == nil {
		t.Fatal("unsupported chart accepted")
	}
}
func TestInteractiveClearRestoresStaticView(t *testing.T) {
	for _, s := range interactionSpecs() {
		m, err := BuildInteractive(s)
		if err != nil {
			t.Fatal(err)
		}
		before := m.View()
		_, _ = m.Highlight("a")
		m.Clear()
		if m.View() != before {
			t.Fatal("clear did not restore original styles")
		}
	}
}

func TestInteractionMouseAndStackSegments(t *testing.T) {
	s := interactionSpecs()[0]
	s.Options.Stacked = true
	s.Data.Series[0].Values = []DataPoint{{Y: 2, Mark: meta("a")}, {Y: -2, Mark: meta("b")}}
	s.Data.Series = append(s.Data.Series, Series{Name: "second", Values: []DataPoint{{Y: 3, Mark: meta("c")}, {Y: -3, Mark: meta("d")}, {Y: 4, Mark: meta("e")}}})
	for _, orientation := range []string{"vertical", "horizontal"} {
		s.Options.Orientation = orientation
		m, err := BuildInteractive(s)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for p := range m.cells {
			hits := m.HitTest(p)
			for _, mark := range hits {
				seen[mark.ID] = true
			}
		}
		for _, id := range []string{"a", "b", "c", "d", "e"} {
			if !seen[id] {
				t.Fatalf("%s: missing hit region for %s", orientation, id)
			}
		}
		for p := range m.cells {
			cmd := m.Update(tea.MouseClickMsg{X: p.X, Y: p.Y, Button: tea.MouseLeft})
			if cmd == nil || len(cmd().(SelectionMsg).Marks) != 1 {
				t.Fatal("click did not select")
			}
			break
		}
	}
}

// Full blocks conceal underline and reverse-video cues. Badges must survive
// stripping all terminal styling, including on default-color currency bars.
func TestInteractionBadgesWithoutColor(t *testing.T) {
	for _, s := range interactionSpecs() {
		m, err := BuildInteractive(s)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = m.Inspect("a")
		if !strings.Contains(ansi.Strip(m.View()), ">") {
			t.Fatalf("%s: focus invisible without color", s.Type)
		}
		_, _ = m.Highlight("a")
		if !strings.Contains(ansi.Strip(m.View()), "@") {
			t.Fatalf("%s: focused selection invisible", s.Type)
		}
		m.Clear()
		view := ansi.Strip(m.View())
		if !strings.Contains(view, ">") || strings.Contains(view, "@") {
			t.Fatalf("%s: stale selection badge", s.Type)
		}
		// The focus-free programmatic selection has a distinct glyph.
		m.focus = -1
		_, _ = m.Highlight("a")
		if !strings.Contains(ansi.Strip(m.View()), "*") {
			t.Fatalf("%s: selection invisible without color", s.Type)
		}
		if err := m.Resize(60, 18); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(ansi.Strip(m.View()), "*") {
			t.Fatalf("%s: resize lost selection badge", s.Type)
		}
	}
}
