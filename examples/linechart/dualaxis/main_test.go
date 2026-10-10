// ntcharts - Copyright (c) 2026 Neomantra Corp.

package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func press(m model, s string) model {
	next, _ := m.Update(tea.KeyPressMsg{Code: rune(s[0]), Text: s})
	return next.(model)
}

func view(m model) string { return ansi.Strip(m.View().Content) }

func TestRightAxisFollowsZoom(t *testing.T) {
	m := newModel()
	a, ok := m.chart.Y2Axis()
	if !ok || a.ViewMin != tempMin || a.ViewMax != tempMax {
		t.Fatalf("right view %v..%v ok %v, want %v..%v", a.ViewMin, a.ViewMax, ok, tempMin, tempMax)
	}
	m = press(m, "+")
	if a, _ = m.chart.Y2Axis(); !(a.ViewMin > tempMin && a.ViewMax < tempMax) {
		t.Fatalf("right view %v..%v did not follow the zoom", a.ViewMin, a.ViewMax)
	}
}

func TestViewShowsBothAxes(t *testing.T) {
	v := view(newModel())
	for _, label := range []string{"1.0", "0.0", "20C", "5C"} {
		if !strings.Contains(v, label) {
			t.Fatalf("view lacks label %q:\n%s", label, v)
		}
	}
}
