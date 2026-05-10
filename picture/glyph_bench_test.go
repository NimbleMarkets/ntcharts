package picture

import (
	"image"
	"image/color"
	"math/rand/v2"
	"testing"

	"github.com/NimbleMarkets/pixterm/pkg/ansimage"
)

// benchImage builds a w×h *image.RGBA filled with a deterministic
// pseudo-random color pattern. Fixed seed so the same bytes are scaled
// across runs and across pixterm versions — only the implementation
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

// BenchmarkGlyph_AnsiImage_Direct isolates pixterm. No picture wrapper,
// no prepareSource — the scaled image is fed straight to ansimage.
// Use this to measure the delta from pixterm-internal changes alone.
//
// Render size 80 cols × 40 rows-of-half-blocks ≈ a typical demo pane.
func BenchmarkGlyph_AnsiImage_Direct(b *testing.B) {
	img := benchImage(800, 600)
	b.ReportAllocs()
	for b.Loop() {
		ai, err := ansimage.NewScaledFromImage(
			img, 40, 80,
			color.Transparent,
			ansimage.ScaleModeFit,
			ansimage.NoDithering,
		)
		if err != nil {
			b.Fatal(err)
		}
		_ = ai.RenderExt(false, false)
	}
}

// BenchmarkGlyph_AnsiImage_DitherBlocks measures the dithered path.
// Same input image; only the dither knob changes from NoDithering.
// (See pixterm itself for finer-grained block-vs-char benchmarks.)
func BenchmarkGlyph_AnsiImage_DitherBlocks(b *testing.B) {
	img := benchImage(800, 600)
	b.ReportAllocs()
	for b.Loop() {
		ai, err := ansimage.NewScaledFromImage(
			img, 40, 80,
			color.Transparent,
			ansimage.ScaleModeFit,
			ansimage.DitheringWithBlocks,
		)
		if err != nil {
			b.Fatal(err)
		}
		_ = ai.RenderExt(false, false)
	}
}

// BenchmarkGlyph_PictureView is the integrated path: View() runs
// prepareSource (CatmullRom resample + bg composite) then hands the
// result to ansimage. Use this to confirm picture-layer behavior
// doesn't drown out pixterm-level wins, and to see the user-visible
// delta a picture consumer will feel.
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
