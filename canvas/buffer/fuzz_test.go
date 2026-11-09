// ntcharts - Copyright (c) 2026 Neomantra Corp.

package buffer

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

func fuzzFloat64s(data []byte, maxCount int) []float64 {
	n := len(data) / 8
	if n > maxCount {
		n = maxCount
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8 : (i+1)*8]))
	}
	return out
}

func FuzzScaleBuffers(f *testing.F) {
	f.Add(uint8(8), 0.0, 1.5, []byte{0, 0, 0, 0, 0, 0, 0xf0, 0x3f, 0, 0, 0, 0, 0, 0, 0x00, 0x40})
	f.Add(uint8(0), -10.0, 0.25, []byte{})
	f.Add(uint8(3), 5.0, -2.0, make([]byte, 48))

	f.Fuzz(func(t *testing.T, ringSizeRaw uint8, offset, scale float64, raw []byte) {
		vals := fuzzFloat64s(raw, 64)
		ringSize := int(ringSizeRaw%32) - 2

		sb := NewFloat64ScaleBuffer(offset, scale)
		rb := NewFloat64ScaleRingBuffer(ringSize, offset, scale)
		pb := NewFloat64PointScaleBuffer(
			canvas.Float64Point{X: offset, Y: offset},
			canvas.Float64Point{X: scale, Y: scale},
		)

		for i, v := range vals {
			sb.Push(v)
			rb.Push(v)
			pb.Push(canvas.Float64Point{X: float64(i), Y: v})
			if i%5 == 4 {
				sb.Pop()
				rb.Pop()
				pb.Pop()
			}
		}

		sb.SetScale(scale * 0.5)
		sb.SetOffset(offset + 1)
		rb.SetScale(scale * 0.5)
		rb.SetOffset(offset + 1)
		pb.SetScale(canvas.Float64Point{X: scale * 0.5, Y: scale * 0.5})
		pb.SetOffset(canvas.Float64Point{X: offset + 1, Y: offset + 1})

		if len(sb.ReadAll()) != sb.Length() || len(sb.ReadAllRaw()) != sb.Length() {
			t.Fatalf("Float64ScaleBuffer length mismatch: %d", sb.Length())
		}
		if len(rb.ReadAll()) != rb.Length() || len(rb.ReadAllRaw()) != rb.Length() {
			t.Fatalf("Float64ScaleRingBuffer length mismatch: %d", rb.Length())
		}
		if rb.Size() > 0 && rb.Length() > rb.Size() {
			t.Fatalf("Float64ScaleRingBuffer length %d exceeds size %d", rb.Length(), rb.Size())
		}
		if len(pb.ReadAll()) != pb.Length() || len(pb.ReadAllRaw()) != pb.Length() {
			t.Fatalf("Float64PointScaleBuffer length mismatch: %d", pb.Length())
		}
	})
}
