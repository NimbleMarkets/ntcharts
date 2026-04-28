package chartpicture

import (
	"bytes"
	"image"
	_ "image/png"
	"strings"
	"testing"

	"github.com/go-analyze/charts"
)

// decodePNG decodes buf as a PNG and returns the resulting image, failing t
// on any error or unexpected format.
func decodePNG(t *testing.T, buf []byte) image.Image {
	t.Helper()
	if len(buf) == 0 {
		t.Fatal("empty buffer")
	}
	img, format, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if format != "png" {
		t.Fatalf("format = %q, want png", format)
	}
	return img
}

func TestLineRecipe(t *testing.T) {
	opt := charts.NewLineChartOptionWithData([][]float64{{1, 2, 3, 4, 5}})
	r := newLineRecipe(opt)
	buf, err := r(640, 480, "")
	if err != nil {
		t.Fatalf("recipe: %v", err)
	}
	img := decodePNG(t, buf)
	if got := img.Bounds().Dx(); got != 640 {
		t.Errorf("width = %d, want 640", got)
	}
	if got := img.Bounds().Dy(); got != 480 {
		t.Errorf("height = %d, want 480", got)
	}
}

func TestBarRecipe(t *testing.T) {
	opt := charts.NewBarChartOptionWithData([][]float64{{10, 20, 30}})
	r := newBarRecipe(opt)
	buf, err := r(400, 300, "dark")
	if err != nil {
		t.Fatalf("recipe: %v", err)
	}
	img := decodePNG(t, buf)
	if img.Bounds().Dx() != 400 || img.Bounds().Dy() != 300 {
		t.Errorf("size = %dx%d, want 400x300", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestEChartsJSONRecipe(t *testing.T) {
	jsonStr := `{
		"type": "line",
		"xAxis": {"data": ["A","B","C","D"]},
		"series": [{"data": [1,2,3,4]}]
	}`
	r := newEChartsJSONRecipe(jsonStr)
	buf, err := r(500, 300, "")
	if err != nil {
		t.Fatalf("recipe: %v", err)
	}
	img := decodePNG(t, buf)
	if img.Bounds().Dx() != 500 || img.Bounds().Dy() != 300 {
		t.Errorf("size = %dx%d, want 500x300", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestEChartsJSONRecipeBadJSON(t *testing.T) {
	r := newEChartsJSONRecipe("not json")
	_, err := r(400, 300, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ECharts") {
		t.Errorf("error = %q, want it to mention ECharts", err.Error())
	}
}

func TestPainterFuncRecipe(t *testing.T) {
	called := false
	r := newPainterFuncRecipe(func(p *charts.Painter) error {
		called = true
		return p.LineChart(charts.NewLineChartOptionWithData([][]float64{{5, 4, 3}}))
	})
	buf, err := r(320, 240, "")
	if err != nil {
		t.Fatalf("recipe: %v", err)
	}
	if !called {
		t.Fatal("painter func was not invoked")
	}
	img := decodePNG(t, buf)
	if img.Bounds().Dx() != 320 || img.Bounds().Dy() != 240 {
		t.Errorf("size = %dx%d, want 320x240", img.Bounds().Dx(), img.Bounds().Dy())
	}
}
