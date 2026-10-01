// ntcharts - Copyright (c) 2026 Neomantra Corp.

package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/charmbracelet/x/ansi"
)

func press(m model, s string) model {
	next, _ := m.Update(tea.KeyPressMsg{Code: rune(s[0]), Text: s})
	return next.(model)
}

func view(m model) string { return ansi.Strip(m.View().Content) }

func TestToggleSwitchesScale(t *testing.T) {
	m := newModel()
	if got := m.chart.YScale(); got != linechart.ScaleLinear {
		t.Fatalf("starts on %v, want linear", got)
	}
	m = press(m, "l")
	if got := m.chart.YScale(); got != linechart.ScaleLog {
		t.Fatalf("after l: %v, want log", got)
	}
	if min, max := m.chart.ViewMinY(), m.chart.ViewMaxY(); min != logMin || max != logMax {
		t.Fatalf("log view range %v..%v, want %v..%v", min, max, logMin, logMax)
	}
	m = press(m, "l")
	if got := m.chart.YScale(); got != linechart.ScaleLinear {
		t.Fatalf("after second l: %v, want linear", got)
	}
	if min, max := m.chart.ViewMinY(), m.chart.ViewMaxY(); min != linearMin || max != linearMax {
		t.Fatalf("linear view range %v..%v, want %v..%v", min, max, linearMin, linearMax)
	}
}

func TestLogViewLabelsPowersOfTen(t *testing.T) {
	m := newModel()
	linear := view(m)
	if strings.Contains(linear, "$100k") {
		t.Fatalf("linear view should not label $100k:\n%s", linear)
	}
	if !strings.Contains(linear, "Y axis: linear") {
		t.Fatalf("linear view lacks its heading:\n%s", linear)
	}

	log := view(press(m, "l"))
	for _, want := range []string{"$100k", "$10k", "$1k", "$100", "$10", "Y axis: log"} {
		if !strings.Contains(log, want) {
			t.Errorf("log view lacks %q:\n%s", want, log)
		}
	}
}

func TestLogZoomMultiplies(t *testing.T) {
	m := press(newModel(), "l")
	m = press(m, "+")
	min, max := m.chart.ViewMinY(), m.chart.ViewMaxY()
	// a quarter decade in from each end: 10*10^0.25 .. 1e5*10^-0.25
	if min < 17 || min > 18 || max < 56000 || max > 57000 {
		t.Fatalf("log zoom in gave %.4g..%.4g, want about 17.8..56234", min, max)
	}
}

func TestDollars(t *testing.T) {
	for v, want := range map[float64]string{10: "$10", 100: "$100", 1000: "$1k", 1500: "$1.5k", 100000: "$100k"} {
		if got := dollars(0, v); got != want {
			t.Errorf("dollars(%v) = %q, want %q", v, got, want)
		}
	}
}
