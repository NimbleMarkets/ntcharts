package decoders

import (
	"bytes"
	_ "embed"
	"image"
	"testing"
)

// Tiny 4x4 fixtures, one per x/image format this package registers. The test
// decodes them through the generic image.Decode to guard against a format being
// dropped from the package's blank imports. (The standard-library formats —
// png, jpeg, gif — are exercised throughout the wider test suite.)
var (
	//go:embed testdata/tiny.webp
	embedWebP []byte
	//go:embed testdata/tiny.bmp
	embedBMP []byte
	//go:embed testdata/tiny.tiff
	embedTIFF []byte
)

// TestRegistersXImageFormats confirms importing this package registers the
// golang.org/x/image decoders (WebP, BMP, TIFF) with image.Decode.
func TestRegistersXImageFormats(t *testing.T) {
	cases := []struct {
		format string
		data   []byte
	}{
		{"webp", embedWebP},
		{"bmp", embedBMP},
		{"tiff", embedTIFF},
	}

	for _, tc := range cases {
		t.Run(tc.format, func(t *testing.T) {
			img, format, err := image.Decode(bytes.NewReader(tc.data))
			if err != nil {
				t.Fatalf("image.Decode %s: %v (decoder not registered?)", tc.format, err)
			}
			if format != tc.format {
				t.Errorf("format = %q, want %q", format, tc.format)
			}
			if b := img.Bounds(); b.Dx() != 4 || b.Dy() != 4 {
				t.Errorf("decoded bounds = %v, want 4x4", b)
			}
		})
	}
}
