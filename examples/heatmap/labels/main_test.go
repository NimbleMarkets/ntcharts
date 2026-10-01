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

// chartText is the chart's own text, without the heading.
func chartText(m model) string {
	hm := m.chart()
	return ansi.Strip(hm.View())
}

func TestLabelsNameRowsAndColumns(t *testing.T) {
	rows := strings.Split(chartText(model{w: 48, h: 14, labels: true}), "\n")
	// rows read Mon..Sun from the top
	last := -1
	for _, d := range days {
		found := -1
		for i, r := range rows {
			if strings.Contains(r, d) {
				found = i
			}
		}
		if found <= last {
			t.Fatalf("%s on row %d, not below the previous day (row %d):\n%s", d, found, last, strings.Join(rows, "\n"))
		}
		last = found
	}
	under := strings.Join(rows[len(rows)-3:], " ")
	for _, h := range hours {
		if !strings.Contains(under, h) {
			t.Errorf("hour %q missing under the grid:\n%s", h, strings.Join(rows, "\n"))
		}
	}
}

func TestToggleRemovesLabelsAndFillsTheChart(t *testing.T) {
	m := model{w: 48, h: 14, labels: true}
	off := press(m, "l")
	if off.labels {
		t.Fatal("l did not switch labels off")
	}
	if text := strings.TrimSpace(chartText(off)); text != "" {
		t.Fatalf("an unlabelled chart drew text: %q", text)
	}
	if press(off, "l").labels != true {
		t.Fatal("l did not switch labels back on")
	}
}

func TestResizeKeepsWithinBounds(t *testing.T) {
	m := model{w: 48, h: 12, labels: true}
	for i := 0; i < 20; i++ {
		m = press(m, "-")
	}
	if m.w != minW || m.h != minH {
		t.Fatalf("shrunk to %dx%d, want the minimum %dx%d", m.w, m.h, minW, minH)
	}
	for i := 0; i < 40; i++ {
		m = press(m, "+")
	}
	if m.w != maxW || m.h != maxH {
		t.Fatalf("grown to %dx%d, want the maximum %dx%d", m.w, m.h, maxW, maxH)
	}
}

// At the smallest size there is no room to name every column; labels are
// dropped, not overdrawn: no two hours run together.
func TestSmallChartDropsLabelsInsteadOfOverdrawing(t *testing.T) {
	text := chartText(model{w: minW, h: minH, labels: true})
	for _, run := range strings.Fields(text) {
		ok := false
		for _, l := range append(append([]string{}, days...), hours...) {
			if run == l {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("%q is not a whole label; labels collided:\n%s", run, text)
		}
	}
}

func TestVisitsPeakAtMiddayOnWeekdays(t *testing.T) {
	if visits(0, 4) <= visits(0, 0) || visits(0, 4) <= visits(0, 7) {
		t.Fatal("weekday peak should be around midday")
	}
	if visits(6, 6) <= visits(6, 4) {
		t.Fatal("weekend peak should be in the evening")
	}
}
