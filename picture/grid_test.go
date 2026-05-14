package picture

import (
	"image"
	"image/color"
	"testing"

	xdraw "golang.org/x/image/draw"
)

func TestNewGridImage_Geometry(t *testing.T) {
	g := NewGridImage(GridImageConfig{
		LogicalCols:         3,
		LogicalRows:         2,
		TerminalColsPerCell: 2,
		TerminalRowsPerCell: 1,
		CellPixelWidth:      8,
		CellPixelHeight:     16,
	})

	if got, want := g.Image().Bounds(), image.Rect(0, 0, 48, 32); got != want {
		t.Fatalf("Image bounds = %v, want %v", got, want)
	}
	if cols, rows := g.TerminalSize(); cols != 6 || rows != 2 {
		t.Fatalf("TerminalSize = (%d,%d), want (6,2)", cols, rows)
	}
	if got, want := g.CellRect(1, 1), image.Rect(16, 16, 32, 32); got != want {
		t.Fatalf("CellRect = %v, want %v", got, want)
	}
}

func TestGridImage_DrawAndFillCell(t *testing.T) {
	g := NewGridImage(GridImageConfig{
		LogicalCols:         2,
		LogicalRows:         1,
		TerminalColsPerCell: 2,
		CellPixelWidth:      4,
		CellPixelHeight:     4,
		BackgroundColor:     color.RGBA{B: 255, A: 255},
	})

	sprite := image.NewRGBA(image.Rect(0, 0, 1, 1))
	sprite.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	if !g.DrawCell(1, 0, sprite) {
		t.Fatal("DrawCell returned false")
	}

	if got := rgbaAt(g.Image(), 10, 2); got.R < 200 {
		t.Fatalf("drawn cell R = %d, want red sprite", got.R)
	}
	if got := rgbaAt(g.Image(), 2, 2); got.B < 200 {
		t.Fatalf("background cell B = %d, want blue background", got.B)
	}

	if !g.FillCell(0, 0, color.RGBA{G: 255, A: 255}) {
		t.Fatal("FillCell returned false")
	}
	if got := rgbaAt(g.Image(), 2, 2); got.G < 200 {
		t.Fatalf("filled cell G = %d, want green fill", got.G)
	}
}

func TestGridImage_BackgroundScalesToBounds(t *testing.T) {
	bg := image.NewRGBA(image.Rect(0, 0, 1, 1))
	bg.SetRGBA(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})

	g := NewGridImage(GridImageConfig{
		LogicalCols:     2,
		LogicalRows:     2,
		CellPixelWidth:  3,
		CellPixelHeight: 3,
		Background:      bg,
	})

	for _, pt := range []image.Point{{0, 0}, {5, 5}} {
		if got := rgbaAt(g.Image(), pt.X, pt.Y); got.R < 5 || got.G < 10 || got.B < 15 {
			t.Fatalf("background at %v = %#v, want scaled source color", pt, got)
		}
	}
}

func TestGridImage_OutOfRange(t *testing.T) {
	g := NewGridImage(GridImageConfig{LogicalCols: 1, LogicalRows: 1})

	if got := g.CellRect(1, 0); !got.Empty() {
		t.Fatalf("out-of-range CellRect = %v, want empty", got)
	}
	if g.DrawCell(1, 0, image.NewRGBA(image.Rect(0, 0, 1, 1))) {
		t.Fatal("DrawCell out of range returned true")
	}
	if g.FillCell(-1, 0, color.Black) {
		t.Fatal("FillCell out of range returned true")
	}

	empty := NewGridImage(GridImageConfig{LogicalCols: -1, LogicalRows: -1})
	if cols, rows := empty.TerminalSize(); cols != 0 || rows != 0 {
		t.Fatalf("negative TerminalSize = (%d,%d), want (0,0)", cols, rows)
	}
}

func TestGridImage_DefaultScalerIsNearestNeighbor(t *testing.T) {
	sprite := image.NewRGBA(image.Rect(0, 0, 2, 1))
	sprite.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	sprite.SetRGBA(1, 0, color.RGBA{B: 255, A: 255})

	g := NewGridImage(GridImageConfig{
		LogicalCols:     1,
		LogicalRows:     1,
		CellPixelWidth:  4,
		CellPixelHeight: 1,
	})
	if !g.DrawCell(0, 0, sprite) {
		t.Fatal("DrawCell returned false")
	}

	if got := rgbaAt(g.Image(), 1, 0); got.R < 200 || got.B > 20 {
		t.Fatalf("left half = %#v, want nearest red", got)
	}
	if got := rgbaAt(g.Image(), 2, 0); got.B < 200 || got.R > 20 {
		t.Fatalf("right half = %#v, want nearest blue", got)
	}
}

func TestGridImage_UsesConfiguredScaler(t *testing.T) {
	scaler := &recordingScaler{}
	g := NewGridImage(GridImageConfig{
		LogicalCols:     2,
		LogicalRows:     1,
		CellPixelWidth:  2,
		CellPixelHeight: 2,
		Background:      image.NewRGBA(image.Rect(0, 0, 1, 1)),
		Scaler:          scaler,
	})
	if scaler.calls != 1 {
		t.Fatalf("background scaler calls = %d, want 1", scaler.calls)
	}
	if got, want := scaler.rects[0], image.Rect(0, 0, 4, 2); got != want {
		t.Fatalf("background scaler rect = %v, want %v", got, want)
	}

	if !g.DrawCell(1, 0, image.NewRGBA(image.Rect(0, 0, 1, 1))) {
		t.Fatal("DrawCell returned false")
	}
	if scaler.calls != 2 {
		t.Fatalf("scaler calls after DrawCell = %d, want 2", scaler.calls)
	}
	if got, want := scaler.rects[1], image.Rect(2, 0, 4, 2); got != want {
		t.Fatalf("cell scaler rect = %v, want %v", got, want)
	}
	if !g.DrawOverlay(image.NewRGBA(image.Rect(0, 0, 1, 1))) {
		t.Fatal("DrawOverlay returned false")
	}
	if scaler.calls != 3 {
		t.Fatalf("scaler calls after DrawOverlay = %d, want 3", scaler.calls)
	}
	if got, want := scaler.rects[2], image.Rect(0, 0, 4, 2); got != want {
		t.Fatalf("overlay scaler rect = %v, want %v", got, want)
	}
	if got := rgbaAt(g.Image(), 0, 0); got.G != 255 {
		t.Fatalf("configured scaler did not write expected color: %#v", got)
	}
}

type recordingScaler struct {
	calls int
	rects []image.Rectangle
}

func (s *recordingScaler) Scale(dst xdraw.Image, dr image.Rectangle, src image.Image, sr image.Rectangle, op xdraw.Op, opts *xdraw.Options) {
	s.calls++
	s.rects = append(s.rects, dr)
	xdraw.NearestNeighbor.Scale(dst, dr, &image.Uniform{C: color.RGBA{G: 255, A: 255}}, image.Rect(0, 0, 1, 1), op, opts)
}

func rgbaAt(img image.Image, x, y int) color.RGBA {
	return color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
}
