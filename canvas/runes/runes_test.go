// ntcharts - Copyright (c) 2026 Neomantra Corp.

package runes

import "testing"

func TestInverseLowerBlockElement(t *testing.T) {
	cases := []struct{ in, want rune }{
		{Null, FullBlock},
		{FullBlock, Null},
		{LowerBlockOne, LowerBlockSeven},
		{LowerBlockFour, LowerBlockFour},
		{LowerBlockSeven, LowerBlockOne},
		{'x', Null},
	}
	for _, c := range cases {
		if got := InverseLowerBlockElement(c.in); got != c.want {
			t.Errorf("InverseLowerBlockElement(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestInverseLeftBlockElement(t *testing.T) {
	cases := []struct{ in, want rune }{
		{Null, FullBlock},
		{FullBlock, Null},
		{LeftBlockOne, LeftBlockSeven},
		{LeftBlockFour, LeftBlockFour},
		{LeftBlockSeven, LeftBlockOne},
		{'x', Null},
	}
	for _, c := range cases {
		if got := InverseLeftBlockElement(c.in); got != c.want {
			t.Errorf("InverseLeftBlockElement(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
