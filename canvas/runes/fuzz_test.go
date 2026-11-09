// ntcharts - Copyright (c) 2026 Neomantra Corp.

package runes

import (
	"math"
	"testing"
	"unicode/utf8"
)

func FuzzBlockElements(f *testing.F) {
	seeds := []float64{
		-1.0, 0.0, math.Copysign(0, -1), 0.03, 0.0625, 0.125, 0.25, 0.5,
		0.875, 0.9375, 1.0, 2.5, math.NaN(), math.Inf(1), math.Inf(-1),
	}
	for _, s := range seeds {
		f.Add(s, uint8(0x5))
	}

	f.Fuzz(func(t *testing.T, v float64, qmask uint8) {
		lb := LowerBlockElementFromFloat64(v)
		if lb != Null && !IsLowerBlockElement(lb) {
			t.Fatalf("LowerBlockElementFromFloat64(%v) = %U, not a lower block element", v, lb)
		}
		invLB := InverseLowerBlockElement(lb)
		if invLB != Null && !IsLowerBlockElement(invLB) {
			t.Fatalf("InverseLowerBlockElement(%U) = %U, not a lower block element", lb, invLB)
		}
		if InverseLowerBlockElement(invLB) != lb {
			t.Fatalf("InverseLowerBlockElement not involutive on %U -> %U", lb, invLB)
		}

		left := LeftBlockElementFromFloat64(v)
		if left != Null && !IsLeftBlockElement(left) {
			t.Fatalf("LeftBlockElementFromFloat64(%v) = %U, not a left block element", v, left)
		}
		invLeft := InverseLeftBlockElement(left)
		if invLeft != Null && !IsLeftBlockElement(invLeft) {
			t.Fatalf("InverseLeftBlockElement(%U) = %U, not a left block element", left, invLeft)
		}
		if InverseLeftBlockElement(invLeft) != left {
			t.Fatalf("InverseLeftBlockElement not involutive on %U -> %U", left, invLeft)
		}

		if math.IsNaN(v) || v <= 0 {
			if lb != Null || left != Null {
				t.Fatalf("expected Null for %v, got lower=%U left=%U", v, lb, left)
			}
		} else if v >= 1 {
			if lb != FullBlock || left != FullBlock {
				t.Fatalf("expected FullBlock for %v, got lower=%U left=%U", v, lb, left)
			}
		}

		ul := qmask&1 != 0
		ur := qmask&2 != 0
		ll := qmask&4 != 0
		lr := qmask&8 != 0
		qb := QuadrantBlock(ul, ur, ll, lr)
		if !ul && !ur && !ll && !lr {
			if qb != Null {
				t.Fatalf("expected Null quadrant for all-false, got %U", qb)
			}
		} else if !utf8.ValidRune(qb) || qb == Null {
			t.Fatalf("QuadrantBlock(%v,%v,%v,%v) = %U, invalid", ul, ur, ll, lr, qb)
		}
	})
}

func FuzzBraillePatterns(f *testing.F) {
	f.Add(uint8(4), uint8(3), []byte{0, 0, 1, 2, 3, 5, 7, 11})
	f.Add(uint8(1), uint8(1), []byte{0, 0, 0, 1, 1, 0, 1, 3})
	f.Add(uint8(0), uint8(0), []byte{})

	f.Fuzz(func(t *testing.T, wRaw, hRaw uint8, coords []byte) {
		w := int(wRaw % 16)
		h := int(hRaw % 16)
		grid := NewPatternDotsGrid(w, h)

		for i := 0; i+1 < len(coords); i += 2 {
			x := int(int8(coords[i]))
			y := int(int8(coords[i+1]))
			if i%4 == 2 {
				grid.Unset(x, y)
			} else {
				grid.Set(x, y)
			}
		}

		patterns := grid.BraillePatterns()
		wantRows := (h + 3) / 4
		wantCols := (w + 1) / 2
		if len(patterns) != wantRows {
			t.Fatalf("unexpected row count: patterns=%d want=%d", len(patterns), wantRows)
		}
		for y := 0; y < wantRows; y++ {
			if len(patterns[y]) != wantCols {
				t.Fatalf("unexpected col count at row %d: patterns=%d want=%d", y, len(patterns[y]), wantCols)
			}
			for x := 0; x < wantCols; x++ {
				r := patterns[y][x]
				if !IsBraillePattern(r) || !utf8.ValidRune(r) {
					t.Fatalf("invalid braille rune %U at (%d,%d)", r, x, y)
				}
				var dots PatternDots
				SetPatternDots(r, &dots)
				if fromDots := BraillePatternFromPatternDots(dots); fromDots != r {
					t.Fatalf("BraillePatternFromPatternDots round-trip mismatch: got %U want %U", fromDots, r)
				}
				if x > 0 {
					prev := patterns[y][x-1]
					comb := CombineBraillePatterns(prev, r)
					if !IsBraillePattern(comb) {
						t.Fatalf("CombineBraillePatterns(%U, %U) = %U not braille", prev, r, comb)
					}
					if (comb-BrailleBlockOffset)&((r-BrailleBlockOffset)|(prev-BrailleBlockOffset)) != ((r - BrailleBlockOffset) | (prev - BrailleBlockOffset)) {
						t.Fatalf("CombineBraillePatterns lost dots: %U | %U -> %U", prev, r, comb)
					}
				}
			}
		}
		grid.Reset()
	})
}

func FuzzCombineRunes(f *testing.F) {
	f.Add(int32(LineHorizontal), int32(LineVertical), true)
	f.Add(int32(ArcUpRight), int32(ArcDownLeft), false)
	f.Add(int32(LineUp), int32(LineDownHeavy), true)
	f.Add(int32('x'), int32('y'), false)

	f.Fuzz(func(t *testing.T, r1, r2 int32, arc bool) {
		ls := ThinLineStyle
		if arc {
			ls = ArcLineStyle
		}
		a, b := rune(r1), rune(r2)
		if !utf8.ValidRune(a) {
			a = LineHorizontal
		}
		if !utf8.ValidRune(b) {
			b = LineVertical
		}

		cl := CombineLines(a, b, ls)
		if (IsLine(a) || IsLine(b)) && cl != Null && !IsLine(cl) {
			t.Fatalf("CombineLines(%U, %U, %v) = %U, expected Null or line rune", a, b, ls, cl)
		}

		seg := LineSegments{
			Up:    r1&1 != 0,
			Down:  r1&2 != 0,
			Left:  r1&4 != 0,
			Right: r1&8 != 0,
		}
		tr := ThinLineFromLineSegments(seg)
		ar := ArcLineFromLineSegments(seg)
		if seg != (LineSegments{}) {
			if !IsLine(tr) || !IsLine(ar) {
				t.Fatalf("line from segments %+v produced non-line rune thin=%U arc=%U", seg, tr, ar)
			}
			var got LineSegments
			SetLineSegments(tr, &got)
			if got != seg {
				t.Fatalf("SetLineSegments(%U) = %+v, want %+v", tr, got, seg)
			}
		}

		cc := CombineCandlesticks(a, b)
		if (IsCandlestick(a) || IsCandlestick(b)) && !IsCandlestick(cc) {
			t.Fatalf("CombineCandlesticks(%U, %U) = %U, expected candlestick rune", a, b, cc)
		}
	})
}
