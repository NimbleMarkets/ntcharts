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

func TestPrepareSource_Fill_OutputDims(t *testing.T) {
	src := solidImage(200, 100, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	out := prepareSource(src, FitFill, 10, 10, 8, 16, color.Transparent)
	got := out.Bounds()
	want := image.Rect(0, 0, 80, 160)
	if got != want {
		t.Fatalf("FitFill output bounds = %v, want %v", got, want)
	}
}

func TestPrepareSource_Fill_NoLetterbox(t *testing.T) {
	// Solid red source stretched into a non-matching AR target should fill
	// every pixel red; no transparent bars.
	src := solidImage(200, 100, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	out := prepareSource(src, FitFill, 10, 10, 8, 16, color.Transparent)
	rgba := out.(*image.RGBA)
	for _, pt := range []image.Point{
		{X: 0, Y: 0},
		{X: 79, Y: 0},
		{X: 0, Y: 159},
		{X: 79, Y: 159},
		{X: 40, Y: 80},
	} {
		c := rgba.RGBAAt(pt.X, pt.Y)
		if c.A != 255 {
			t.Errorf("FitFill leaves pixel %v transparent (alpha %d); should be opaque red", pt, c.A)
		}
		if c.R < 100 {
			t.Errorf("FitFill pixel %v R=%d, want red-ish", pt, c.R)
		}
	}
}

func TestPrepareSource_Fill_FastPath_AlreadyTargetSize(t *testing.T) {
	// If src is already exactly target size, FitFill should return src unchanged.
	src := solidImage(80, 160, color.RGBA{R: 50, G: 100, B: 150, A: 255})
	out := prepareSource(src, FitFill, 10, 10, 8, 16, color.Transparent)
	if out != image.Image(src) {
		t.Fatal("FitFill with src already at target size should return src unchanged (fast path)")
	}
}

func TestPrepareSource_Contain_OutputDims(t *testing.T) {
	src := solidImage(200, 100, color.RGBA{R: 0, G: 0, B: 200, A: 255})
	out := prepareSource(src, FitContain, 10, 10, 8, 16, color.Transparent)
	got := out.Bounds()
	want := image.Rect(0, 0, 80, 160)
	if got != want {
		t.Fatalf("FitContain output bounds = %v, want %v", got, want)
	}
}

func TestPrepareSource_Contain_LetterboxesTransparent(t *testing.T) {
	// 200x100 source (AR 2.0) into 80x160 target (AR 0.5):
	// AR-preserving inscribed rect = 80x40, centered vertically (rows 60..100 occupied).
	src := solidImage(200, 100, color.RGBA{R: 0, G: 0, B: 200, A: 255})
	out := prepareSource(src, FitContain, 10, 10, 8, 16, color.Transparent)
	rgba := out.(*image.RGBA)

	// Letterbox bars (top and bottom) should be fully transparent.
	for _, pt := range []image.Point{{X: 40, Y: 0}, {X: 40, Y: 30}, {X: 40, Y: 130}, {X: 40, Y: 159}} {
		if a := rgba.RGBAAt(pt.X, pt.Y).A; a != 0 {
			t.Errorf("letterbox bar pixel %v should be transparent, got alpha=%d", pt, a)
		}
	}
	// Inscribed band should be opaque blue.
	for _, pt := range []image.Point{{X: 40, Y: 60}, {X: 40, Y: 80}, {X: 40, Y: 99}} {
		c := rgba.RGBAAt(pt.X, pt.Y)
		if c.A != 255 || c.B < 100 {
			t.Errorf("inscribed pixel %v should be opaque blue, got %+v", pt, c)
		}
	}
}

func TestPrepareSource_Contain_LetterboxesOpaqueRed(t *testing.T) {
	src := solidImage(200, 100, color.RGBA{R: 0, G: 0, B: 200, A: 255})
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	out := prepareSource(src, FitContain, 10, 10, 8, 16, red)
	rgba := out.(*image.RGBA)
	c := rgba.RGBAAt(40, 0) // top letterbox bar
	if c != red {
		t.Errorf("top letterbox pixel should be red, got %+v", c)
	}
}

func TestPrepareSource_Contain_PortraitInWideTarget(t *testing.T) {
	// 100x200 source (AR 0.5) into 160x80 target (AR 2.0):
	// inscribed = 40x80, centered horizontally (cols 60..100 occupied).
	src := solidImage(100, 200, color.RGBA{R: 0, G: 200, B: 0, A: 255})
	out := prepareSource(src, FitContain, 20, 5, 8, 16, color.Transparent)
	rgba := out.(*image.RGBA)
	// Left bar transparent.
	if a := rgba.RGBAAt(10, 40).A; a != 0 {
		t.Errorf("left letterbox should be transparent, got alpha=%d", a)
	}
	// Inscribed center opaque green.
	c := rgba.RGBAAt(80, 40)
	if c.A != 255 || c.G < 100 {
		t.Errorf("inscribed center should be opaque green, got %+v", c)
	}
}

func TestPrepareSource_Cover_OutputDims(t *testing.T) {
	src := solidImage(200, 100, color.RGBA{R: 200, G: 200, B: 0, A: 255})
	out := prepareSource(src, FitCover, 10, 10, 8, 16, color.Transparent)
	got := out.Bounds()
	want := image.Rect(0, 0, 80, 160)
	if got != want {
		t.Fatalf("FitCover output bounds = %v, want %v", got, want)
	}
}

func TestPrepareSource_Cover_FillsEdgeToEdge(t *testing.T) {
	// 200x100 source (AR 2.0) into 80x160 target (AR 0.5).
	// FitCover crops the source horizontally; output should be opaque
	// everywhere — no transparent bars.
	src := solidImage(200, 100, color.RGBA{R: 200, G: 200, B: 0, A: 255})
	out := prepareSource(src, FitCover, 10, 10, 8, 16, color.Transparent)
	rgba := out.(*image.RGBA)
	for _, pt := range []image.Point{
		{X: 0, Y: 0}, {X: 79, Y: 0}, {X: 0, Y: 159}, {X: 79, Y: 159}, {X: 40, Y: 80},
	} {
		if a := rgba.RGBAAt(pt.X, pt.Y).A; a != 255 {
			t.Errorf("FitCover should fill edge-to-edge; pixel %v has alpha=%d", pt, a)
		}
	}
}

func TestPrepareSource_Cover_CropsSourceWithMatchingARTarget(t *testing.T) {
	// 200x100 source into 80x80 target (square).
	// AR-preserving cover crops source horizontally to centered 100x100,
	// then scales to target.
	src := solidImage(200, 100, color.RGBA{R: 0, G: 200, B: 200, A: 255})
	// 5 cols × 10 rows, 16x8 cell pixels = 80x80 target (square).
	out := prepareSource(src, FitCover, 5, 10, 16, 8, color.Transparent)
	if out.Bounds() != image.Rect(0, 0, 80, 80) {
		t.Fatalf("FitCover bounds = %v, want 80x80", out.Bounds())
	}
}

func TestPrepareSource_DegenerateDims_ReturnsNil(t *testing.T) {
	src := solidImage(10, 10, color.RGBA{A: 255})
	for _, tc := range []struct {
		name                     string
		cols, rows, cellW, cellH int
	}{
		{"cols=0", 0, 10, 8, 16},
		{"rows=0", 10, 0, 8, 16},
		{"cellW=0", 10, 10, 0, 16},
		{"cellH=0", 10, 10, 8, 0},
		{"negative", -1, 10, 8, 16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := prepareSource(src, FitContain, tc.cols, tc.rows, tc.cellW, tc.cellH, color.Transparent); got != nil {
				t.Fatalf("expected nil for degenerate dims, got bounds %v", got.Bounds())
			}
		})
	}
}

func TestPrepareSource_EmptySource_ReturnsBgFilled(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 0, 0))
	red := color.RGBA{R: 255, A: 255}
	out := prepareSource(src, FitContain, 4, 4, 8, 16, red)
	if out == nil {
		t.Fatal("empty source should return a bg-filled image, got nil")
	}
	rgba := out.(*image.RGBA)
	if c := rgba.RGBAAt(0, 0); c != red {
		t.Errorf("empty source should produce bg-filled output, got %+v", c)
	}
}

// TestPrepareSource_EqualAR_AllFitsAreFullBleed verifies that when the
// source aspect ratio matches the target cell-rect aspect ratio, all three
// fit modes produce a full-bleed bitmap with no transparent letterbox bars.
// Math sanity check: cross-product equality (sw*th == sh*tw) means the
// inscribed and circumscribed rects both equal the target.
func TestPrepareSource_EqualAR_AllFitsAreFullBleed(t *testing.T) {
	// 80x160 source matches a 10x10 cell rect with 8x16 cell pixels.
	src := solidImage(80, 160, color.RGBA{R: 100, G: 200, B: 50, A: 255})
	for _, fit := range []FitMode{FitContain, FitFill, FitCover} {
		t.Run(fitTestName(fit), func(t *testing.T) {
			out := prepareSource(src, fit, 10, 10, 8, 16, color.Transparent)
			if got, want := out.Bounds(), (image.Rect(0, 0, 80, 160)); got != want {
				t.Fatalf("bounds = %v, want %v", got, want)
			}
			rgba := out.(*image.RGBA)
			for _, pt := range []image.Point{
				{X: 0, Y: 0}, {X: 79, Y: 0}, {X: 0, Y: 159}, {X: 79, Y: 159}, {X: 40, Y: 80},
			} {
				if a := rgba.RGBAAt(pt.X, pt.Y).A; a != 255 {
					t.Errorf("equal-AR %v: pixel %v alpha=%d, want opaque (no letterbox)", fit, pt, a)
				}
			}
		})
	}
}

func fitTestName(f FitMode) string {
	switch f {
	case FitFill:
		return "FitFill"
	case FitCover:
		return "FitCover"
	default:
		return "FitContain"
	}
}

// TestPrepareSource_AllFits_OpaqueBg_CompositesTransparentSrc verifies the
// regression case from the deleted composite() helper: a fully transparent
// source over an opaque background should produce a fully opaque output of
// the bg color in every fit mode. Before this contract was enforced,
// FitFill ignored bg entirely and FitCover wrote src with draw.Src,
// leaving transparent areas in the encoded bitmap.
func TestPrepareSource_AllFits_OpaqueBg_CompositesTransparentSrc(t *testing.T) {
	// Fully transparent source — every pixel contributes nothing.
	src := solidImage(80, 160, color.RGBA{R: 0, G: 0, B: 0, A: 0})
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	for _, fit := range []FitMode{FitContain, FitFill, FitCover} {
		t.Run(fitTestName(fit), func(t *testing.T) {
			out := prepareSource(src, fit, 10, 10, 8, 16, red)
			rgba := out.(*image.RGBA)
			for _, pt := range []image.Point{
				{X: 0, Y: 0}, {X: 79, Y: 0}, {X: 0, Y: 159}, {X: 79, Y: 159}, {X: 40, Y: 80},
			} {
				c := rgba.RGBAAt(pt.X, pt.Y)
				if c != red {
					t.Errorf("%v: pixel %v = %+v, want opaque red (translucent src should composite over bg)", fit, pt, c)
				}
			}
		})
	}
}

// TestPrepareSource_Fill_FastPath_OpaqueBgComposites guards the FitFill
// fast path: when src is already at target dims it can return src
// unchanged ONLY if bg is transparent. With an opaque bg the fast path
// must not bypass compositing.
func TestPrepareSource_Fill_FastPath_OpaqueBgComposites(t *testing.T) {
	// Fully transparent source already at target dims (80x160).
	src := solidImage(80, 160, color.RGBA{R: 0, G: 0, B: 0, A: 0})
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	out := prepareSource(src, FitFill, 10, 10, 8, 16, red)
	if out == image.Image(src) {
		t.Fatal("FitFill must not return src unchanged when bg is opaque (compositing was skipped)")
	}
	c := out.(*image.RGBA).RGBAAt(0, 0)
	if c != red {
		t.Errorf("expected composited red pixel, got %+v", c)
	}
}
