package heatpicture

import (
	"image/color"
	"math"
	"testing"

	"charm.land/lipgloss/v2"
)

// BenchmarkSampleField_TerminalSize measures the cost of one frame at a
// few representative terminal cell dimensions, with the same Perlin
// sampler the demo uses. If a frame at a "large" size takes longer than
// the inter-event spacing (typically tens of ms for a startup-heavy
// program), the demo's seq churn can keep invalidating frames before
// they land.
func BenchmarkSampleField_TerminalSize(b *testing.B) {
	scale := toRGBA([]color.Color{
		lipgloss.Color("#FFFFFF"),
		lipgloss.Color("#FF7216"),
		lipgloss.Color("#660000"),
	})
	// Deterministic value field in ~[-1,1]; replaces a Perlin sampler so the
	// library carries no go-perlin dependency. The benchmark measures
	// sampleField cost, not noise quality.
	sampler := func(x, y float64) float64 { return math.Sin(x*20) * math.Cos(y*20) }

	for _, sz := range []struct {
		name         string
		cellW, cellH int
		cols, rows   int
	}{
		{"80x24_8x16", 8, 16, 80, 24},
		{"160x40_8x16", 8, 16, 160, 40},
		{"200x60_8x16", 8, 16, 200, 60},
		{"240x80_8x16", 8, 16, 240, 80},
		{"320x100_8x16", 8, 16, 320, 100},
	} {
		b.Run(sz.name+"_Kitty", func(b *testing.B) {
			pixelW := sz.cols * sz.cellW
			pixelH := sz.rows * sz.cellH
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = sampleField(sampler, scale, -1, 1, 0, 1, 0, 1, pixelW, pixelH)
			}
		})
		b.Run(sz.name+"_Glyph", func(b *testing.B) {
			pixelW := sz.cols
			pixelH := sz.rows * 2
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = sampleField(sampler, scale, -1, 1, 0, 1, 0, 1, pixelW, pixelH)
			}
		})
	}
}
