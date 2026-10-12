package picture

import (
	"image"
	"image/color"
	"math/rand/v2"
	"testing"
)

// benchImage builds a w×h *image.RGBA filled with a deterministic
// pseudo-random color pattern. Fixed seed so the same bytes are scaled
// across runs — only the implementation
// changes, not the input.
func benchImage(w, h int) *image.RGBA {
	r := rand.New(rand.NewPCG(0xc0ffee, 0xfacade))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(r.UintN(256)),
				G: uint8(r.UintN(256)),
				B: uint8(r.UintN(256)),
				A: 255,
			})
		}
	}
	return img
}

// BenchmarkGlyph_EncodeDirect isolates ANSI encoding of a prepared image,
// reusing the output buffer as Model.View does between frames.
func BenchmarkGlyph_EncodeDirect(b *testing.B) {
	img := benchImage(80, 40)
	var buf []byte
	b.ReportAllocs()
	for b.Loop() {
		buf = appendGlyph(buf[:0], img)
		if len(buf) == 0 {
			b.Fatal("empty output")
		}
	}
}

// BenchmarkGlyph_PictureView measures fitting and ANSI half-block encoding
// together, including per-model scratch reuse across frames.
//
// invalidateGlyph each iteration so the cache doesn't short-circuit
// and we measure the full render every time.
func BenchmarkGlyph_PictureView(b *testing.B) {
	m := New()
	m.SetSize(80, 20)
	m.SetImage(benchImage(800, 600))
	b.ReportAllocs()
	for b.Loop() {
		m.invalidateGlyph()
		if got := m.View().Content; got == "" {
			b.Fatal("empty View")
		}
	}
}

func BenchmarkGlyph_PictureView_240x68(b *testing.B) {
	m := NewWithConfig(Config{Fit: FitFill})
	m.SetSize(240, 68)
	m.SetImage(benchImage(240, 135))
	b.ReportAllocs()
	for b.Loop() {
		m.invalidateGlyph()
		if got := m.View().Content; got == "" {
			b.Fatal("empty View")
		}
	}
}
