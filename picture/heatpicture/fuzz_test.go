// ntcharts - Copyright (c) 2026 Neomantra Corp.

package heatpicture

import (
	"image/color"
	"math"
	"testing"
)

func FuzzColorScaleAndSampleField(f *testing.F) {
	f.Add(0.5, 0.0, 1.0, -1.0, 1.0, -1.0, 1.0, uint8(4), uint8(4), []byte{0, 0, 0, 255, 255, 128, 0, 255})
	f.Add(math.NaN(), 0.0, 0.0, 0.0, 1.0, 0.0, 1.0, uint8(2), uint8(2), []byte{10, 20, 30, 255})
	f.Add(math.Inf(1), -10.0, 10.0, 0.0, 0.0, 0.0, 0.0, uint8(1), uint8(1), []byte{})

	f.Fuzz(func(t *testing.T, pos, minV, maxV, minX, maxX, minY, maxY float64, wRaw, hRaw uint8, rawColors []byte) {
		var colors []color.Color
		for i := 0; i+3 < len(rawColors) && len(colors) < 16; i += 4 {
			colors = append(colors, color.RGBA{
				R: rawColors[i],
				G: rawColors[i+1],
				B: rawColors[i+2],
				A: rawColors[i+3],
			})
		}
		rgbaScale := toRGBA(colors)
		_ = colorAt(rgbaScale, pos)
		_ = colorAt(rgbaScale, math.NaN())
		_ = colorAt(rgbaScale, math.Inf(1))
		_ = colorAt(rgbaScale, math.Inf(-1))

		pw := int(wRaw%16) + 1
		ph := int(hRaw%16) + 1
		img, err := sampleField(
			func(x, y float64) float64 {
				return x*pos + y
			},
			rgbaScale,
			minV, maxV, minX, maxX, minY, maxY,
			pw, ph,
		)
		if err != nil {
			t.Fatalf("sampleField returned unexpected error for %dx%d: %v", pw, ph, err)
		}
		if img.Bounds().Dx() != pw || img.Bounds().Dy() != ph {
			t.Fatalf("sampleField bounds %v != %dx%d", img.Bounds(), pw, ph)
		}

		m := NewWithConfig(Config{SamplingFactor: pos})
		if sf := m.SamplingFactor(); !(sf > 0 && sf <= 1) {
			t.Fatalf("SamplingFactor out of (0,1]: %v", sf)
		}
		_ = m.SetSamplingFactor(pos)
		if sf := m.SamplingFactor(); !(sf > 0 && sf <= 1) {
			t.Fatalf("SetSamplingFactor out of (0,1]: %v", sf)
		}
	})
}
