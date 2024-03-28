// ntcharts - Copyright (c) 2024 Neomantra Corp.

package sparkline

import (
	"testing"
)

func TestNew(t *testing.T) {
	w := 30
	h := 15
	max := 100.0
	scale := float64(h) / max

	sl := New(w, h, WithMaxValue(max))

	if sl.Width() != w {
		t.Errorf("Width not initialized:%d", sl.Width())
	}
	if sl.Height() != h {
		t.Errorf("Height not initialized:%d", sl.Height())
	}
	if sl.MaxValue() != max {
		t.Errorf("MaxValue not initialized:%f", sl.MaxValue())
	}
	if sl.Scale() != scale {
		t.Errorf("Scale not initialized:%f", sl.Scale())
	}
}

func TestAutoMaxValue(t *testing.T) {
	w := 30
	h := 15
	max := 81.4

	sl := New(w, h)

	sl.Push(max / 2)
	scale := float64(h) / (max / 2)
	if sl.Scale() != scale {
		t.Errorf("Scale not correct with AutoMaxValue true after greater value than max:%f", sl.Scale())
	}

	max = 99.2
	scale = float64(h) / max
	sl.Push(max)
	if sl.Scale() != scale {
		t.Errorf("Scale not correct with AutoMaxValue true after greater value than max:%f", sl.Scale())
	}

	sl.Push(max / 2)
	scale = float64(h) / (max / 2)
	if sl.Scale() == scale {
		t.Errorf("Scale changed after lesser value than max:%f", sl.Scale())
	}

	sl.AutoMaxValue = false
	max = 104.7
	scale = float64(h) / max
	sl.Push(max)
	if sl.Scale() == scale {
		t.Errorf("Scale changed with AutoMaxValue false after greater value than max:%f", sl.Scale())
	}
}

func TestNoAutoMaxValue(t *testing.T) {
	w := 30
	h := 15
	max := 100.0
	newMax := 150.0
	scale := float64(h) / max

	sl := New(w, h, WithMaxValue(max), WithNoAutoMaxValue())

	sl.Push(max / 2)
	if sl.Scale() != scale {
		t.Errorf("Scale not correct:%f", sl.Scale())
	}

	sl.Push(newMax)
	if sl.Scale() != scale {
		t.Errorf("Scale changed with AutoMaxValue false after greater value than max:%f", sl.Scale())
	}

	sl.AutoMaxValue = true
	scale = float64(h) / newMax
	sl.Push(newMax)
	if sl.Scale() != scale {
		t.Errorf("Scale failed to changed with AutoMaxValue true after greater value than max:%f", sl.Scale())
	}

}
