// ntcharts - Copyright (c) 2024 Neomantra Corp.

package buffer

import (
	"math/rand"
	"testing"

	"github.com/NimbleMarkets/ntcharts/canvas"
)

func TestFloat64ScaleRingBuffer(t *testing.T) {
	offset := 10.0
	sz := 5
	scale := .5
	buf := NewFloat64ScaleRingBuffer(sz, offset, scale)
	if buf.Size() != 5 {
		t.Errorf("Float64ScaleRingBuffer wrong size:%d", buf.Size())
	}
	if buf.Length() != 0 {
		t.Errorf("Float64ScaleRingBuffer wrong length:%d", buf.Length())
	}
	if buf.Offset() != offset {
		t.Errorf("Float64ScaleRingBuffer wrong offset:%f", buf.Offset())
	}
	if buf.Scale() != scale {
		t.Errorf("Float64ScaleRingBuffer wrong scale:%f", buf.Scale())
	}
	n := 23
	max := 100.0
	seq := []float64{}
	for i := 0; i < n; i++ {
		seq = append(seq, rand.Float64()*max)
	}
	// fill before reaching capacity
	for i := 0; i < 2; i++ {
		buf.Push(seq[i])
	}
	if buf.Length() != 2 {
		t.Errorf("Float64ScaleRingBuffer wrong length:%d", buf.Length())
	}
	for i, v := range buf.ReadAllRaw() {
		if v != seq[i] {
			t.Errorf("Float64ScaleRingBuffer returned wrong value:%f, expected %f", v, seq[i])
		}
	}
	for i, v := range buf.ReadAll() {
		if v != (seq[i]-offset)*scale {
			t.Errorf("Float64ScaleRingBuffer returned wrong scaled value:%f, expected %f", v, (seq[i]-offset)*scale)
		}
	}
	buf.Pop()
	if buf.Length() != 1 {
		t.Errorf("Float64ScaleRingBuffer wrong length after pop:%d", buf.Length())
	}
	// fill reaching capacity
	for i := 2; i < 6; i++ {
		buf.Push(seq[i])
	}
	if buf.Length() != 5 {
		t.Errorf("Float64ScaleRingBuffer wrong length:%d", buf.Length())
	}
	for i, v := range buf.ReadAllRaw() {
		if v != seq[i+1] {
			t.Errorf("Float64ScaleRingBuffer returned wrong value:%f, expected %f", v, seq[i+1])
		}
	}
	for i, v := range buf.ReadAll() {
		expected := (seq[i+1] - offset) * scale
		if v != expected {
			t.Errorf("Float64ScaleRingBuffer returned wrong scaled value:%f, expected %f", v, expected)
		}
	}
	// fill rest
	for i := 5; i < n; i++ {
		buf.Push(seq[i])
	}
	if buf.Length() != 5 {
		t.Errorf("Float64ScaleRingBuffer wrong length:%d", buf.Length())
	}
	seqIdx := n - buf.Length()
	for i, v := range buf.ReadAllRaw() {
		if v != seq[seqIdx+i] {
			t.Errorf("Float64ScaleRingBuffer returned wrong value:%f, expected %f", v, seq[seqIdx+i])
		}
	}
	for i, v := range buf.ReadAll() {
		expected := (seq[seqIdx+i] - offset) * scale
		if v != expected {
			t.Errorf("Float64ScaleRingBuffer returned wrong scaled value:%f, expected %f", v, expected)
		}
	}
	// empty after clear
	buf.Clear()
	if sz := len(buf.ReadAllRaw()); sz != 0 {
		t.Errorf("Float64ScaleRingBuffer contains values after clearing %d", sz)
	}
	// contains values after clear
	for i := 0; i < buf.Size(); i++ {
		buf.Push(seq[i])
	}
	for i, v := range buf.ReadAllRaw() {
		if v != seq[i] {
			t.Errorf("Float64ScaleRingBuffer returned wrong value after clear:%f, expected %f", v, seq[i])
		}
	}
	for i, v := range buf.ReadAll() {
		expected := (seq[i] - offset) * scale
		if v != expected {
			t.Errorf("Float64ScaleRingBuffer returned wrong value scaled after clear:%f, expected %f", v, expected)
		}
	}
	buf.Pop()
	buf.Pop()
	for i, v := range buf.ReadAllRaw() {
		if v != seq[i+2] {
			t.Errorf("Float64ScaleRingBuffer returned wrong value after pop:%f, expected %f", v, seq[i+2])
		}
	}
	for i, v := range buf.ReadAll() {
		expected := (seq[i+2] - offset) * scale
		if v != expected {
			t.Errorf("Float64ScaleRingBuffer returned wrong value scaled after pop:%f, expected %f", v, expected)
		}
	}
}

func TestFloat64ScaleRingBufferRescale(t *testing.T) {
	sz := 5
	scale := .5
	buf := NewFloat64ScaleRingBuffer(sz, 0, scale)

	n := 15
	max := 1000.0
	seq := []float64{}
	for i := 0; i < n; i++ {
		seq = append(seq, rand.Float64()*max)
		buf.Push(seq[i])
	}
	seqIdx := n - buf.Length()
	for i, v := range buf.ReadAllRaw() {
		if v != seq[seqIdx+i] {
			t.Errorf("Float64ScaleRingBuffer returned wrong value:%f, expected %f", v, seq[seqIdx+i])
		}
	}
	for i, v := range buf.ReadAll() {
		if v != seq[seqIdx+i]*scale {
			t.Errorf("Float64ScaleRingBuffer returned wrong scaled value:%f, expected %f", v, seq[seqIdx+i]*scale)
		}
	}

	newScale := .1235
	buf.SetScale(newScale)
	for i, v := range buf.ReadAll() {
		if v != seq[seqIdx+i]*newScale {
			t.Errorf("Float64ScaleRingBuffer returned wrong rescaled value:%f, expected %f", v, seq[seqIdx+i]*newScale)
		}
	}

	newScale = 1.8
	buf.SetScale(newScale)
	for i, v := range buf.ReadAll() {
		if v != seq[seqIdx+i]*newScale {
			t.Errorf("Float64ScaleRingBuffer returned wrong rescaled value:%f, expected %f", v, seq[seqIdx+i]*newScale)
		}
	}
}

func TestFloat64ScaleBuffer(t *testing.T) {
	offset := -10.0
	scale := .5
	buf := NewFloat64ScaleBuffer(offset, scale)
	if buf.Length() != 0 {
		t.Errorf("Float64ScaleBuffer wrong length:%d", buf.Length())
	}

	n := 100
	max := 100.0
	seq := []float64{}
	for i := 0; i < n; i++ {
		y := rand.Float64() * max
		seq = append(seq, y)
		buf.Push(y)
	}
	if buf.Length() != n {
		t.Errorf("Float64ScaleBuffer wrong length:%d", buf.Length())
	}
	if buf.Offset() != offset {
		t.Errorf("Float64ScaleBuffer wrong offset:%f", buf.Offset())
	}
	if buf.Scale() != scale {
		t.Errorf("Float64ScaleBuffer wrong Scale:%f", buf.Scale())
	}
	for i, v := range seq {
		if (v-offset)*scale != buf.At(i) {
			t.Errorf("Float64ScaleBuffer returned wrong scaled value:%f, expected %f", buf.At(i), (v-offset)*scale)
		}
		if v != buf.AtRaw(i) {
			t.Errorf("Float64ScaleBuffer returned wrong original value:%f, expected %f", buf.AtRaw(i), v)
		}
	}
	buf.Pop()
	if buf.Length() != n-1 {
		t.Errorf("Float64ScaleBuffer wrong length after pop:%d", buf.Length())
	}
	for bufIdx := 0; bufIdx < buf.Length(); bufIdx++ {
		seqf := seq[bufIdx+1]
		if (seqf-offset)*scale != buf.At(bufIdx) {
			t.Errorf("Float64ScaleBuffer returned wrong scaled value after pop:%f, expected %f", buf.At(bufIdx), (seqf-offset)*scale)
		}
		if seqf != buf.AtRaw(bufIdx) {
			t.Errorf("Float64ScaleBuffer returned wrong original value after pop:%f, expected %f", buf.AtRaw(bufIdx), seqf)
		}
	}

	buf.Clear()
	if buf.Length() != 0 {
		t.Errorf("Float64ScaleBuffer failed to clear, length:%d", buf.Length())
	}
	buf.SetData(seq)
	if buf.Length() != n {
		t.Errorf("Float64ScaleBuffer SetData wrong length:%d", buf.Length())
	}
	for i, v := range seq {
		if (v-offset)*scale != buf.At(i) {
			t.Errorf("Float64ScaleBuffer returned wrong scaled value:%f, expected %f", buf.At(i), (v-offset)*scale)
		}
		if v != buf.AtRaw(i) {
			t.Errorf("Float64ScaleBuffer returned wrong original value:%f, expected %f", buf.AtRaw(i), v)
		}
	}
	for i := 0; i < 10; i++ {
		buf.Pop()
	}
	for bufIdx := 0; bufIdx < buf.Length(); bufIdx++ {
		seqf := seq[bufIdx+10]
		if (seqf-offset)*scale != buf.At(bufIdx) {
			t.Errorf("Float64ScaleBuffer returned wrong scaled value after pop:%f, expected %f", buf.At(bufIdx), (seqf-offset)*scale)
		}
		if seqf != buf.AtRaw(bufIdx) {
			t.Errorf("Float64ScaleBuffer returned wrong original value after pop:%f, expected %f", buf.AtRaw(bufIdx), seqf)
		}
	}

}

func TestFloat64ScaleBufferRescale(t *testing.T) {
	offset := 20.0
	scale := .22
	buf := NewFloat64ScaleBuffer(offset, scale)

	n := 100
	max := 560.0
	seq := []float64{}
	for i := 0; i < n; i++ {
		y := rand.Float64() * max
		seq = append(seq, y)
		buf.Push(y)
	}
	for i, v := range seq {
		if (v-offset)*scale != buf.At(i) {
			t.Errorf("Float64ScaleBuffer returned wrong scaled value:%f, expected %f", buf.AtRaw(i), (v-offset)*scale)
		}
		if v != buf.AtRaw(i) {
			t.Errorf("Float64ScaleBuffer returned wrong original value:%f, expected %f", buf.AtRaw(i), v)
		}
	}

	newScale := 2.34
	buf.SetScale(newScale)
	for i, v := range seq {
		if (v-offset)*newScale != buf.At(i) {
			t.Errorf("Float64ScaleBuffer returned wrong scaled value:%f, expected %f", buf.AtRaw(i), (v-offset)*newScale)
		}
		if v != buf.AtRaw(i) {
			t.Errorf("Float64ScaleBuffer returned wrong original value:%f, expected %f", buf.AtRaw(i), v)
		}
	}

	newScale = .25
	buf.SetScale(newScale)
	for i, v := range seq {
		if (v-offset)*newScale != buf.At(i) {
			t.Errorf("Float64ScaleBuffer returned wrong scaled value:%f, expected %f", buf.AtRaw(i), (v-offset)*newScale)
		}
		if v != buf.AtRaw(i) {
			t.Errorf("Float64ScaleBuffer returned wrong original value:%f, expected %f", buf.AtRaw(i), v)
		}
	}
}

func TestFloat64PointScaleBuffer(t *testing.T) {
	sf := .5
	scale := canvas.Float64Point{X: sf, Y: sf}
	add := canvas.Float64Point{X: 0, Y: 0}
	buf := NewFloat64PointScaleBuffer(add, scale)
	if buf.Length() != 0 {
		t.Errorf("Float64PointScaleBuffer wrong length:%d", buf.Length())
	}

	n := 100
	max := 100.0
	seq := []canvas.Float64Point{}
	for i := 0; i < n; i++ {
		x := rand.Float64() * max
		y := rand.Float64() * max
		f := canvas.Float64Point{X: x, Y: y}
		seq = append(seq, f)
		buf.Push(f)
	}
	if buf.Length() != n {
		t.Errorf("Float64PointScaleBuffer wrong length:%d", buf.Length())
	}
	for i, v := range seq {
		if v.Mul(scale) != buf.At(i) {
			t.Errorf("Float64PointScaleBuffer returned wrong scaled value:%f, expected %f", buf.At(i), v.Mul(scale))
		}
		if v != buf.AtRaw(i) {
			t.Errorf("Float64PointScaleBuffer returned wrong original value:%f, expected %f", buf.AtRaw(i), v)
		}
	}
	buf.Pop()
	if buf.Length() != n-1 {
		t.Errorf("Float64PointScaleBuffer wrong length after pop:%d", buf.Length())
	}
	for bufIdx := 0; bufIdx < buf.Length(); bufIdx++ {
		seqf := seq[bufIdx+1]
		if seqf.Mul(scale) != buf.At(bufIdx) {
			t.Errorf("Float64PointScaleBuffer returned wrong scaled value after pop:%f, expected %f", buf.At(bufIdx), seqf.Mul(scale))
		}
		if seqf != buf.AtRaw(bufIdx) {
			t.Errorf("Float64PointScaleBuffer returned wrong original value after pop:%f, expected %f", buf.AtRaw(bufIdx), seqf)
		}
	}

	buf.Clear()
	if buf.Length() != 0 {
		t.Errorf("Float64PointScaleBuffer failed to clear, length:%d", buf.Length())
	}
	buf.SetData(seq)
	if buf.Length() != n {
		t.Errorf("Float64PointScaleBuffer SetData wrong length:%d", buf.Length())
	}
	for i, v := range seq {
		if v.Mul(scale) != buf.At(i) {
			t.Errorf("Float64PointScaleBuffer returned wrong scaled value:%f, expected %f", buf.At(i), v.Mul(scale))
		}
		if v != buf.AtRaw(i) {
			t.Errorf("Float64PointScaleBuffer returned wrong original value:%f, expected %f", buf.AtRaw(i), v)
		}
	}
	for i := 0; i < 10; i++ {
		buf.Pop()
	}
	for bufIdx := 0; bufIdx < buf.Length(); bufIdx++ {
		seqf := seq[bufIdx+10]
		if seqf.Mul(scale) != buf.At(bufIdx) {
			t.Errorf("Float64PointScaleBuffer returned wrong scaled value after pop:%f, expected %f", buf.At(bufIdx), seqf.Mul(scale))
		}
		if seqf != buf.AtRaw(bufIdx) {
			t.Errorf("Float64PointScaleBuffer returned wrong original value after pop:%f, expected %f", buf.AtRaw(bufIdx), seqf)
		}
	}

}

func TestFloat64PointScaleBufferRescale(t *testing.T) {
	sf := .22
	scale := canvas.Float64Point{X: sf, Y: sf}
	add := canvas.Float64Point{X: 0, Y: 0}
	buf := NewFloat64PointScaleBuffer(add, scale)

	n := 100
	max := 560.0
	seq := []canvas.Float64Point{}
	for i := 0; i < n; i++ {
		x := rand.Float64() * max
		y := rand.Float64() * max
		f := canvas.Float64Point{X: x, Y: y}
		seq = append(seq, f)
		buf.Push(f)
	}
	for i, v := range seq {
		if v.Mul(scale) != buf.At(i) {
			t.Errorf("Float64PointScaleBuffer returned wrong scaled value:%f, expected %f", buf.AtRaw(i), v.Mul(scale))
		}
		if v != buf.AtRaw(i) {
			t.Errorf("Float64PointScaleBuffer returned wrong original value:%f, expected %f", buf.AtRaw(i), v)
		}
	}

	nsf := 2.34
	newScale := canvas.Float64Point{X: nsf, Y: nsf}
	buf.SetScale(newScale)
	for i, v := range seq {
		if v.Mul(newScale) != buf.At(i) {
			t.Errorf("Float64PointScaleBuffer returned wrong scaled value:%f, expected %f", buf.AtRaw(i), v.Mul(newScale))
		}
		if v != buf.AtRaw(i) {
			t.Errorf("Float64PointScaleBuffer returned wrong original value:%f, expected %f", buf.AtRaw(i), v)
		}
	}

	nsf = .25
	newScale = canvas.Float64Point{X: nsf, Y: nsf}
	buf.SetScale(newScale)
	for i, v := range seq {
		if v.Mul(newScale) != buf.At(i) {
			t.Errorf("Float64PointScaleBuffer returned wrong scaled value:%f, expected %f", buf.AtRaw(i), v.Mul(newScale))
		}
		if v != buf.AtRaw(i) {
			t.Errorf("Float64PointScaleBuffer returned wrong original value:%f, expected %f", buf.AtRaw(i), v)
		}
	}
}
