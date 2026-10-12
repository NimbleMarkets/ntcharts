package picture

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"

	"github.com/NimbleMarkets/pixterm/pkg/ansimage"
)

func referenceGlyph(t *testing.T, src image.Image, bg color.Color) string {
	t.Helper()
	ai, err := ansimage.NewFromImage(src, bg, ansimage.NoDithering)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimRight(ai.RenderExt(false, false), "\n")
}

// The golden covers both crop axes and every anchor. Generate the reference
// with the shared fit implementation and pixterm, independently of appendGlyph.
func TestGlyphGolden(t *testing.T) {
	data, err := os.ReadFile("testdata/glyph.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden map[string]string
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	for _, size := range []image.Point{{8, 4}, {4, 8}} {
		for fit := FitContain; fit <= FitCover; fit++ {
			for anchor := AnchorCenter; anchor <= AnchorRight; anchor++ {
				key := fmt.Sprintf("%dx%d/fit%d/anchor%d", size.X, size.Y, fit, anchor)
				t.Run(key, func(t *testing.T) {
					src := benchImage(size.X, size.Y)
					bg := color.RGBA{R: 20, G: 30, B: 40, A: 255}
					prepared := prepareSource(src, fit, 4, 2, 1, 2, bg, anchor)
					want, ok := golden[key]
					if !ok {
						t.Fatal("missing golden")
					}
					if referenceGlyph(t, prepared, bg) != want {
						t.Fatal("reference differs from golden")
					}
					m := NewWithConfig(Config{Fit: fit, Anchor: anchor, Background: bg})
					m.SetSize(4, 2)
					m.SetImage(src)
					if got := m.View().Content; got != want {
						t.Fatalf("got %q, want %q", got, want)
					}
				})
			}
		}
	}
}

func TestGlyphMatchesPixterm(t *testing.T) {
	rgba := benchImage(8, 8).SubImage(image.Rect(2, 2, 6, 6)).(*image.RGBA)
	translucent := image.NewNRGBA(image.Rect(3, 5, 7, 9))
	for y := 5; y < 9; y++ {
		for x := 3; x < 7; x++ {
			translucent.SetNRGBA(x, y, color.NRGBA{R: 200, G: 100, B: 50, A: uint8((x + y) * 17)})
		}
	}
	for _, src := range []image.Image{rgba, translucent, image.NewGray(image.Rect(-2, -2, 2, 2)), image.NewRGBA(image.Rectangle{})} {
		for _, bg := range []color.Color{color.Transparent, color.Black, color.NRGBA{R: 30, G: 80, B: 100, A: 100}} {
			for fit := FitContain; fit <= FitCover; fit++ {
				m := NewWithConfig(Config{Fit: fit, Background: bg})
				m.SetSize(4, 2)
				m.SetImage(src)
				prepared := prepareSource(src, fit, 4, 2, 1, 2, bg, AnchorCenter)
				want := referenceGlyph(t, prepared, bg)
				if got := m.View().Content; got != want {
					t.Fatalf("%T bg %v fit %v: got %q, want %q", src, bg, fit, got, want)
				}
			}
		}
	}
}

func TestGlyphScratchAcrossFrames(t *testing.T) {
	m := NewWithConfig(Config{Fit: FitFill, Background: color.Black})
	m.SetSize(4, 2)
	m.SetImage(benchImage(9, 7))
	first := m.View().Content
	saved := strings.Clone(first)
	target, scaler := m.glyphFit.rgba, m.glyphFit.scaler
	for i := 0; i < 3; i++ {
		m.SetImage(solidImage(9, 7, color.RGBA{G: 100, A: 255}))
		m.View()
		if m.glyphFit.rgba != target || m.glyphFit.scaler != scaler {
			t.Fatal("scratch was not reused")
		}
	}
	// Force different scaler and buffer sizes, translucent pixels and letterbox
	// bars; compare with a fresh model to detect retained pixels from old frames.
	for _, size := range []image.Point{{2, 1}, {8, 4}, {4, 2}} {
		for fit := FitContain; fit <= FitCover; fit++ {
			src := solidImage(3, 5, color.RGBA{R: 60, A: 100})
			m.SetSize(size.X, size.Y)
			m.SetFit(fit)
			m.SetImage(src)
			fresh := NewWithConfig(Config{Fit: fit, Background: color.Black})
			fresh.SetSize(size.X, size.Y)
			fresh.SetImage(src)
			if m.View().Content != fresh.View().Content {
				t.Fatal("stale scratch pixels")
			}
		}
	}
	if first != saved {
		t.Fatal("render mutated a previously returned View")
	}
}

func TestGlyphExactSizeSkipsPreparation(t *testing.T) {
	for fit := FitContain; fit <= FitCover; fit++ {
		m := NewWithConfig(Config{Fit: fit})
		m.SetSize(4, 2)
		src := benchImage(6, 6).SubImage(image.Rect(1, 1, 5, 5))
		m.SetImage(src)
		m.View()
		if m.glyphFit.rgba != nil || m.glyphFit.scaler != nil {
			t.Fatal("exact-size opaque source should not need preparation")
		}
	}
}

// Uniform images with integral letterbox boundaries retain the legacy output
// byte-for-byte. More detailed images are now sampled once at glyph resolution
// rather than CatmullRom-upscaled and then box-downsampled.
func TestGlyphLegacyUniform(t *testing.T) {
	for _, size := range []image.Point{{8, 4}, {4, 8}} {
		for fit := FitContain; fit <= FitCover; fit++ {
			for anchor := AnchorCenter; anchor <= AnchorRight; anchor++ {
				bg := color.RGBA{R: 20, G: 30, B: 40, A: 255}
				src := solidImage(size.X, size.Y, color.RGBA{R: 80, G: 120, B: 160, A: 255})
				legacy := prepareSource(src, fit, 4, 2, 8, 16, bg, anchor)
				ai, err := ansimage.NewScaledFromImage(legacy, 4, 4, bg, ansimage.ScaleModeResize, ansimage.NoDithering)
				if err != nil {
					t.Fatal(err)
				}
				want := strings.TrimRight(ai.RenderExt(false, false), "\n")
				m := NewWithConfig(Config{Fit: fit, Anchor: anchor, Background: bg})
				m.SetSize(4, 2)
				m.SetImage(src)
				if got := m.View().Content; got != want {
					t.Fatalf("size %v fit %d anchor %d: got %q want %q", size, fit, anchor, got, want)
				}
			}
		}
	}
}
