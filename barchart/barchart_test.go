// bubbletea-charts - Copyright (c) 2024 Neomantra Corp.

package barchart

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestBarchart(t *testing.T) {
	w := 6
	h := 10
	bw := 3 // will be auto updated
	max := 50.0
	scale := float64(h) / max
	bc := New(w, h, WithMaxValue(max), WithBarWidth(bw))

	if bc.Width() != w {
		t.Errorf("Width not initialized:%d", bc.Width())
	}
	if bc.Height() != h {
		t.Errorf("Height not initialized:%d", bc.Height())
	}
	if bc.MaxValue() != max {
		t.Errorf("MaxValue not initialized:%f", bc.MaxValue())
	}
	if bc.Scale() != scale {
		t.Errorf("Scale not initialized:%f", bc.Scale())
	}
	if bc.BarWidth() != 1 {
		t.Errorf("BarWidth not initialized:%d", bc.BarWidth())
	}
	if bc.BarGap() != 1 {
		t.Errorf("BarGap not initialized:%d", bc.BarGap())
	}

	bc.Push(BarData{
		Label: "LesserThanMaxValue",
		Values: []BarValue{
			{Value: 11.1, Style: lipgloss.NewStyle()},
			{Value: 22.1, Style: lipgloss.NewStyle()},
		},
	})
	if bc.MaxValue() != max {
		t.Errorf("MaxValue changed when max value did not:%f", bc.MaxValue())
	}
	if bc.Scale() != scale {
		t.Errorf("Scale changed when max value did not:%f", bc.Scale())
	}
	if bc.BarWidth() != w {
		t.Errorf("BarWidth expected to be graph width when only 1 bar:%d", bc.BarWidth())
	}

	max = 66.6
	scale = float64(h) / max
	bc.Push(BarData{
		Label: "GreaterThanMaxValue",
		Values: []BarValue{
			{Value: 11.1, Style: lipgloss.NewStyle()},
			{Value: 22.2, Style: lipgloss.NewStyle()},
			{Value: 33.3, Style: lipgloss.NewStyle()},
		},
	})
	if bc.MaxValue() != max {
		t.Errorf("MaxValue not correct after new max value:%f", bc.MaxValue())
	}
	if bc.Scale() != scale {
		t.Errorf("Scale not correct after new max value:%f", bc.Scale())
	}
	if bc.BarWidth() != 2 { // 2 bars w/ 1 gap = [0][0][-1][1][1][-1] graph indices
		t.Errorf("BarWidth expected to be 2 ( 2 bars + 1 gap ):%d", bc.BarWidth())
	}

	bc.Push(BarData{
		Label: "EqualToMaxValue",
		Values: []BarValue{
			{Value: 11.1, Style: lipgloss.NewStyle()},
			{Value: 22.2, Style: lipgloss.NewStyle()},
			{Value: 33.3, Style: lipgloss.NewStyle()},
		},
	})
	if bc.MaxValue() != max {
		t.Errorf("MaxValue not correct after new max value:%f", bc.MaxValue())
	}
	if bc.Scale() != scale {
		t.Errorf("Scale not correct after new max value:%f", bc.Scale())
	}
	if bc.BarWidth() != 1 { // 3 bars w/ 1 gap = [0][-1][1][-1][2][-1] graph indices
		t.Errorf("BarWidth expected to be 1 ( 3 bars + 1 gap ):%d", bc.BarWidth())
	}
}

func TestBarNoAutoMaxValue(t *testing.T) {
	w := 6
	h := 10
	max := 50.0
	scale := float64(h) / max
	bc := New(w, h, WithMaxValue(max), WithNoAutoMaxValue())

	bc.Push(BarData{
		Label: "LesserThanMaxValue",
		Values: []BarValue{
			{Value: 11.1, Style: lipgloss.NewStyle()},
			{Value: 22.1, Style: lipgloss.NewStyle()},
		},
	})
	if bc.MaxValue() != max {
		t.Errorf("MaxValue changed when max value did not:%f", bc.MaxValue())
	}
	if bc.Scale() != scale {
		t.Errorf("Scale changed when max value did not:%f", bc.Scale())
	}

	bc.Push(BarData{
		Label: "GreaterThanMaxValue",
		Values: []BarValue{
			{Value: 11.1, Style: lipgloss.NewStyle()},
			{Value: 22.2, Style: lipgloss.NewStyle()},
			{Value: 33.3, Style: lipgloss.NewStyle()},
		},
	})
	if bc.MaxValue() != max {
		t.Errorf("MaxValue changed with AutoMaxValue disabled:%f", bc.MaxValue())
	}
	if bc.Scale() != scale {
		t.Errorf("Scale changed with AutoMaxValue disabled:%f", bc.Scale())
	}
}

func TestBarNoAutoBarWidth(t *testing.T) {
	w := 6
	h := 10
	bw := 3
	bc := New(w, h, WithNoAutoBarWidth(), WithBarWidth(bw))

	bc.Push(BarData{
		Label: "LesserThanMaxValue",
		Values: []BarValue{
			{Value: 11.1, Style: lipgloss.NewStyle()},
			{Value: 22.1, Style: lipgloss.NewStyle()},
		},
	})
	if bc.BarWidth() != bw { // 1 bars w/ 3 bar width and w/ 1 gap = [0][0][0][-1][-1][-1] graph indices
		t.Errorf("BarWidth changed with AutoBarWidth disabled:%d", bc.BarWidth())
	}

	bc.Push(BarData{
		Label: "GreaterThanMaxValue",
		Values: []BarValue{
			{Value: 11.1, Style: lipgloss.NewStyle()},
			{Value: 22.2, Style: lipgloss.NewStyle()},
			{Value: 33.3, Style: lipgloss.NewStyle()},
		},
	})
	if bc.BarWidth() != bw { // 2 bars w/ 3 bar width and w/ 1 gap = [0][0][0][-1][1][1] graph indices
		t.Errorf("BarWidth changed with AutoBarWidth disabled:%d", bc.BarWidth())
	}
}
