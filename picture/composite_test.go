package picture

import (
	"image"
	"image/color"
	"testing"
)

// solidImage returns a w×h *image.RGBA filled with c.
func solidImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestComposite_TransparentBackground_ReturnsInputUnchanged(t *testing.T) {
	src := solidImage(2, 2, color.RGBA{R: 100, G: 150, B: 200, A: 128})
	out := composite(src, color.Transparent)
	// Short-circuit contract: returns the input image, not a copy.
	if out != image.Image(src) {
		t.Fatalf("expected composite to return input unchanged for transparent bg, got different image")
	}
}

func TestComposite_FullyTransparentSource_ReturnsBackground(t *testing.T) {
	src := solidImage(2, 2, color.RGBA{R: 0, G: 0, B: 0, A: 0}) // fully transparent
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	out := composite(src, red)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			r, g, b, a := out.At(x, y).RGBA()
			// RGBA() returns 16-bit values; 255 -> 65535.
			if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
				t.Fatalf("pixel (%d,%d) = (%d,%d,%d,%d), want red", x, y, r, g, b, a)
			}
		}
	}
}

func TestComposite_FullyOpaqueSource_PreservesSource(t *testing.T) {
	src := solidImage(2, 2, color.RGBA{R: 50, G: 100, B: 150, A: 255})
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	out := composite(src, red)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			r, g, b, a := out.At(x, y).RGBA()
			// Source dominates because alpha=1.0.
			wantR, wantG, wantB := uint32(50)<<8|50, uint32(100)<<8|100, uint32(150)<<8|150
			if r != wantR || g != wantG || b != wantB || a != 0xffff {
				t.Fatalf("pixel (%d,%d) = (%d,%d,%d,%d), want (%d,%d,%d,65535)",
					x, y, r, g, b, a, wantR, wantG, wantB)
			}
		}
	}
}

func TestComposite_HalfAlphaSource_BlendsHalfwayWithBackground(t *testing.T) {
	// Source is white at 50% alpha. Background is solid black.
	// Expected: roughly mid-gray.
	src := solidImage(2, 2, color.RGBA{R: 128, G: 128, B: 128, A: 128}) // premultiplied: src color is 128/255 * 255 ≈ 128
	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	out := composite(src, black)
	r, g, b, a := out.At(0, 0).RGBA()
	// With premultiplied 'src' = (128, 128, 128, 128) over opaque black,
	// result = src + (1-srcA)*dst = (128,128,128,128) + (127/255)*(0,0,0,255)
	//        = (128, 128, 128, 128 + 127) ≈ (128, 128, 128, 255) in 8-bit;
	// in 16-bit RGBA() -> (32896, 32896, 32896, 65535) approximately.
	// Allow a small tolerance.
	wantMid := uint32(128) << 8
	if absDiff(r, wantMid) > 0x0200 || absDiff(g, wantMid) > 0x0200 || absDiff(b, wantMid) > 0x0200 {
		t.Fatalf("pixel (0,0) = (%d,%d,%d,%d), want roughly (%d,%d,%d,65535)",
			r, g, b, a, wantMid, wantMid, wantMid)
	}
	if a != 0xffff {
		t.Fatalf("alpha = %d, want 65535 (composited over opaque background)", a)
	}
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
