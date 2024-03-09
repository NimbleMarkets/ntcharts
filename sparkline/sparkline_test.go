package sparkline

import (
	"testing"
)

func TestNew(t *testing.T) {
	w := 30
	h := 15
	max := 100.0
	scale := float64(h) / max

	sl := New(w, h, max)

	if sl.Width() != w {
		t.Errorf("Width not initialized:%d", sl.Width())
	}
	if sl.Height() != h {
		t.Errorf("Height not initialized:%d", sl.Height())
	}
	if sl.Max() != max {
		t.Errorf("Max not initialized:%f", sl.Max())
	}
	if sl.Scale() != scale {
		t.Errorf("Scale not initialized:%f", sl.Scale())
	}
}

func TestAuto(t *testing.T) {
	w := 30
	h := 15
	max := 100.0
	newMax := 150.0
	scale := float64(h) / max

	sl := New(w, h, max)

	sl.Push(max / 2)
	if sl.Scale() != scale {
		t.Errorf("Scale not correct:%f", sl.Scale())
	}

	sl.Push(newMax)
	if sl.Scale() != scale {
		t.Errorf("Scale changed after greater value than max:%f", sl.Scale())
	}

	sl.Auto = true
	scale = float64(h) / newMax
	sl.Push(newMax)
	if sl.Scale() != scale {
		t.Errorf("Scale failed to changed after greater value than max:%f", sl.Scale())
	}

}
