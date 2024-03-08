package canvas

import (
	"testing"
)

func TestNew(t *testing.T) {
	w := 30
	h := 15
	c := New(w, h)

	p := c.Cursor()
	if (p.X != 0) || (p.Y != 0) {
		t.Error("Cursor not initialized to (0,0)")
	}

	if c.Width() != w {
		t.Errorf("Width not initialized:%d", w)
	}
	if c.Height() != h {
		t.Errorf("Height not initialized:%d", h)
	}

	if c.ViewWidth != w {
		t.Errorf("ViewWidth not initialized:%d", w)
	}
	if c.ViewHeight != h {
		t.Errorf("ViewHeight not initialized:%d", h)
	}

}

func TestCursor(t *testing.T) {
	w := 30
	h := 15
	c := New(w, h)

	c.SetCursor(Point{w / 2, h / 2})
	p := c.Cursor()
	if p.X != w/2 {
		t.Errorf("CursorX not set correctly:%d", p.X)
	}
	if p.Y != h/2 {
		t.Errorf("CursorY not set correctly:%d", p.Y)
	}

	c.Clear()
	p = c.Cursor()
	if p.X != w/2 {
		t.Errorf("CursorX not set correctly after clear:%d", p.X)
	}
	if p.Y != h/2 {
		t.Errorf("CursorY not set correctly after clear:%d", p.Y)
	}

	c.SetCursor(Point{-1, -1})
	p = c.Cursor()
	if p.X != 0 {
		t.Errorf("CursorX not bounded:%d", p.X)
	}
	if p.Y != 0 {
		t.Errorf("CursorY not bounded:%d", p.Y)
	}

	c.SetCursor(Point{w, h})
	p = c.Cursor()
	if p.X != w-1 {
		t.Errorf("CursorX not bounded:%d", p.X)
	}
	if p.Y != h-1 {
		t.Errorf("CursorY not bounded:%d", p.Y)
	}
}

func TestResize(t *testing.T) {
	w := 30
	h := 15
	c := New(w, h)
	c.SetCursor(Point{w / 2, h / 2})

	nW := w + 5
	nH := h + 5
	c.Resize(nW, nH)

	p := c.Cursor()
	if (p.X != 0) || (p.Y != 0) {
		t.Error("Cursor not set to (0,0) after Resize()")
	}

	c.SetCursor(Point{w, h})
	p = c.Cursor()
	if p.X != w {
		t.Errorf("CursorX not set correctly after resize:%d", p.X)
	}
	if p.Y != h {
		t.Errorf("CursorY not set correctly after resize:%d", p.Y)
	}
}

func TestSetRune(t *testing.T) {
	w := 10
	h := 5
	c := New(w, h)

	r := c.Cell(Point{w / 2, h / 2}).Rune
	if r != 0 {
		t.Errorf("Rune not initialized:'%c'", r)
	}

	if c.SetRune(Point{w, h}, 'A') {
		t.Error("SetRune not bounded")
	}

	if !c.SetRune(Point{w / 2, h / 2}, 'A') {
		t.Error("SetRune not set correctly")
	}

	r = c.Cell(Point{w / 2, h / 2}).Rune
	if r != 'A' {
		t.Errorf("Rune not set correctly:'%c'", r)
	}

	c.Clear()
	r = c.Cell(Point{w / 2, h / 2}).Rune
	if r != 0 {
		t.Errorf("Rune not clear correctly:'%c'", r)
	}
}

func TestFill(t *testing.T) {
	w := 10
	h := 5
	c := New(w, h)

	c.Fill(NewCell('B'))

	for y := 0; y < c.Height(); y++ {
		for x := 0; x < c.Width(); x++ {
			r := c.Cell(Point{x, y}).Rune
			if r != 'B' {
				t.Errorf("Rune did not fill correctly:'%c'", r)
				return
			}
		}
	}
}

func TestFloat64Point(t *testing.T) {
	x := -1.5
	y := 2.5
	f := Float64Point{x, y}

	scale := 2.0
	sc := Float64Point{scale, scale}
	nf := f.Mul(sc)
	if nf.X != x*scale {
		t.Errorf("Float64Point X value did not Mul correctly:%f", nf.X)
	}
	if nf.Y != y*scale {
		t.Errorf("Float64Point Y value did not Mul correctly:%f", nf.Y)
	}

	xOffset := -10.0
	yOffset := 3.0
	offset := Float64Point{X: xOffset, Y: yOffset}
	nf = f.Add(offset)
	if nf.X != x+xOffset {
		t.Errorf("Float64Point X value did not Add correctly:%f", nf.X)
	}
	if nf.Y != y+yOffset {
		t.Errorf("Float64Point Y value did not Add correctly:%f", nf.Y)
	}

	offset = Float64Point{X: xOffset, Y: yOffset}
	nf = f.Sub(offset)
	if nf.X != x-xOffset {
		t.Errorf("Float64Point X value did not Sub correctly:%f", nf.X)
	}
	if nf.Y != y-yOffset {
		t.Errorf("Float64Point Y value did not Sub correctly:%f", nf.Y)
	}
}
