package picture

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi/kitty"
)

// decodeKittyRaw parses a chunked APC, checks the first chunk carries every
// option in wantFirst and no f=100 (RGBA is the protocol default and is
// serialized without an f= key), and returns the base64-decoded payload.
func decodeKittyRaw(t *testing.T, apc string, wantFirst ...string) []byte {
	t.Helper()
	var payload strings.Builder
	chunks := strings.Split(strings.TrimSuffix(apc, "\x1b\\"), "\x1b\\")
	for i, chunk := range chunks {
		if !strings.HasPrefix(chunk, "\x1b_G") {
			t.Fatalf("invalid APC %q", chunk[:min(len(chunk), 40)])
		}
		control, data, _ := strings.Cut(chunk[3:], ";")
		if len(data) > kitty.MaxChunkSize || len(data)%4 != 0 {
			t.Fatalf("invalid chunk length %d", len(data))
		}
		if i == 0 {
			if strings.Contains(control, "f=100") {
				t.Errorf("raw frame advertises PNG: %s", control)
			}
			for _, want := range wantFirst {
				if !strings.Contains(control, want) {
					t.Errorf("missing %s in %s", want, control)
				}
			}
		} else if strings.Contains(control, "a=") || strings.Contains(control, "f=") || strings.Contains(control, "s=") {
			t.Errorf("repeated first-chunk options: %s", control)
		}
		payload.WriteString(data)
	}
	data, err := base64.StdEncoding.DecodeString(payload.String())
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func randomNRGBA(w, h int) *image.NRGBA {
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	random := rand.New(rand.NewPCG(3, 4))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = uint8(random.Uint32()), uint8(random.Uint32()), uint8(random.Uint32()), uint8(random.Uint32())
	}
	return src
}

func TestKittyRGBARoundTripNRGBA(t *testing.T) {
	original := tmuxPassthroughEnabled.Load()
	defer SetTmuxPassthrough(original)
	SetTmuxPassthrough(false)
	// Sub-image with a non-zero origin and row padding, so the encoder
	// must walk rows rather than copy Pix wholesale.
	full := randomNRGBA(240, 160)
	src := full.SubImage(image.Rect(7, 9, 211, 145)).(*image.NRGBA)
	w, h := src.Bounds().Dx(), src.Bounds().Dy()

	apc := buildKittyAPC(src, 45, 2, 3, KittyFormatRGBA)
	data := decodeKittyRaw(t, apc, "a=T", fmt.Sprintf("s=%d", w), fmt.Sprintf("v=%d", h), "i=45", "c=2", "r=3", "U=1", "q=2")
	if strings.Contains(apc, "f=100") {
		t.Fatal("RGBA frame advertises PNG")
	}
	if len(data) != w*h*4 {
		t.Fatalf("payload %d bytes, want %d", len(data), w*h*4)
	}
	for y := range h {
		row := src.Pix[y*src.Stride : y*src.Stride+w*4]
		if !bytes.Equal(data[y*w*4:(y+1)*w*4], row) {
			t.Fatalf("row %d differs", y)
		}
	}
}

func TestKittyRGBAUnpremultipliesRGBA(t *testing.T) {
	// prepareSource hands the encoder a premultiplied *image.RGBA. Kitty
	// wants straight alpha, so translucent pixels must be divided back out.
	src := image.NewRGBA(image.Rect(0, 0, 3, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 200, G: 100, B: 50, A: 255})                                        // opaque: bytes pass through
	src.SetRGBA(1, 0, color.RGBAModel.Convert(color.NRGBA{R: 200, G: 100, B: 50, A: 128}).(color.RGBA)) // translucent
	src.SetRGBA(2, 0, color.RGBA{})                                                                     // fully transparent

	data := decodeKittyRaw(t, buildKittyAPC(src, 1, 1, 1, KittyFormatRGBA), "s=3", "v=1")
	if got, want := data[0:4], []byte{200, 100, 50, 255}; !bytes.Equal(got, want) {
		t.Fatalf("opaque pixel = %v, want %v", got, want)
	}
	want := color.NRGBAModel.Convert(src.RGBAAt(1, 0)).(color.NRGBA)
	if got := data[4:8]; got[0] != want.R || got[1] != want.G || got[2] != want.B || got[3] != want.A {
		t.Fatalf("translucent pixel = %v, want %v", got, want)
	}
	if got := data[8:12]; !bytes.Equal(got, []byte{0, 0, 0, 0}) {
		t.Fatalf("transparent pixel = %v, want zeros", got)
	}
}

func TestWriteKittyRGBA(t *testing.T) {
	full := randomNRGBA(12, 8)
	premultiplied := image.NewRGBA(full.Bounds())
	draw.Draw(premultiplied, full.Bounds(), full, image.Point{}, draw.Src)
	premultiplied.SetRGBA(3, 2, color.RGBA{})
	rect := image.Rect(2, 1, 9, 6)
	for _, src := range []image.Image{
		full.SubImage(rect),
		premultiplied.SubImage(rect),
		image.NewGray(rect),
	} {
		t.Run(fmt.Sprintf("%T", src), func(t *testing.T) {
			// A caller-owned buffer need not start zeroed. Include guard bytes
			// to detect writes past the tightly packed pixel data.
			n := 4 * rect.Dx() * rect.Dy()
			storage := bytes.Repeat([]byte{0xff}, n+8)
			dst := storage[4 : n+4]
			writeKittyRGBA(dst, src)
			at := 0
			for y := rect.Min.Y; y < rect.Max.Y; y++ {
				for x := rect.Min.X; x < rect.Max.X; x++ {
					c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
					if !bytes.Equal(dst[at:at+4], []byte{c.R, c.G, c.B, c.A}) {
						t.Fatalf("pixel (%d,%d): %v != %v", x, y, dst[at:at+4], c)
					}
					at += 4
				}
			}
			if !bytes.Equal(storage[:4], bytes.Repeat([]byte{0xff}, 4)) ||
				!bytes.Equal(storage[n+4:], bytes.Repeat([]byte{0xff}, 4)) {
				t.Fatal("wrote outside pixel buffer")
			}
		})
	}
}

func TestKittyPNGFormatUnchanged(t *testing.T) {
	src := randomNRGBA(64, 32)
	if got, want := buildKittyAPC(src, 45, 2, 3, KittyFormatPNG), buildKittyAPC(src, 45, 2, 3, KittyFormat(99)); got != want {
		t.Fatal("unknown formats must fall back to PNG")
	}
	if !strings.Contains(buildKittyAPC(src, 45, 2, 3, KittyFormatPNG), "f=100") {
		t.Fatal("PNG frame lost f=100")
	}
}

func TestKittyFormatConfigAndSetter(t *testing.T) {
	m := New()
	if m.KittyFormat() != KittyFormatPNG {
		t.Fatalf("default format = %v, want PNG", m.KittyFormat())
	}
	m = NewWithConfig(Config{KittyFormat: KittyFormatRGBA})
	if m.KittyFormat() != KittyFormatRGBA {
		t.Fatalf("configured format = %v, want RGBA", m.KittyFormat())
	}
	if cmd := m.SetKittyFormat(KittyFormatRGBA); cmd != nil {
		t.Fatal("setting the same format should be a no-op")
	}
	if m.SetKittyFormat(KittyFormatPNG); m.KittyFormat() != KittyFormatPNG {
		t.Fatal("SetKittyFormat did not apply")
	}
	if m.SetKittyFormat(KittyFormat(99)); m.KittyFormat() != KittyFormatPNG {
		t.Fatal("unknown format should normalise to PNG")
	}
}

func TestKittyFrameMsgCarriesFormat(t *testing.T) {
	original := tmuxPassthroughEnabled.Load()
	defer SetTmuxPassthrough(original)
	SetTmuxPassthrough(false)
	m := NewWithConfig(Config{KittyFormat: KittyFormatRGBA, Fit: FitFill})
	m.mode = PictureKitty
	m.cols, m.rows = 2, 1
	m.img = randomNRGBA(16, 16)
	cmd := m.renderCmd()
	if cmd == nil {
		t.Fatal("renderCmd returned nil")
	}
	frame, ok := cmd().(KittyFrameMsg)
	if !ok {
		t.Fatal("renderCmd did not produce a KittyFrameMsg")
	}
	if frame.Format != KittyFormatRGBA {
		t.Fatalf("frame format = %v, want RGBA", frame.Format)
	}
	// RGBA (f=32) is the protocol default, so the serializer omits f=; a
	// raw frame is one without f=100 and with explicit dimensions.
	if strings.Contains(frame.APC, "f=100") || !strings.Contains(frame.APC, "s=16,v=16") {
		t.Fatalf("frame APC is not raw RGBA: %.80s", frame.APC)
	}
}

func BenchmarkKittyFormats(b *testing.B) {
	original := tmuxPassthroughEnabled.Load()
	defer SetTmuxPassthrough(original)
	SetTmuxPassthrough(false)
	// noise: worst case for PNG, translucent so RGBA takes the un-premultiply
	// path. gradient: opaque smooth content, the realistic chart-frame case.
	noise := image.NewRGBA(image.Rect(0, 0, 1280, 960))
	draw.Draw(noise, noise.Bounds(), randomNRGBA(1280, 960), image.Point{}, draw.Src)
	gradient := image.NewRGBA(image.Rect(0, 0, 1280, 960))
	for y := range 960 {
		for x := range 1280 {
			gradient.SetRGBA(x, y, color.RGBA{R: uint8(x / 5), G: uint8(y / 4), B: uint8((x + y) / 9), A: 255})
		}
	}
	for _, src := range []struct {
		name string
		img  *image.RGBA
	}{{"noise", noise}, {"gradient", gradient}} {
		for _, f := range []struct {
			name   string
			format KittyFormat
		}{{"png", KittyFormatPNG}, {"rgba", KittyFormatRGBA}} {
			b.Run(src.name+"/"+f.name, func(b *testing.B) {
				b.ReportAllocs()
				var n int
				for b.Loop() {
					n = len(buildKittyAPC(src.img, 45, 160, 60, f.format))
				}
				b.ReportMetric(float64(n), "APC-bytes/op")
			})
		}
	}
}

// Zlib frames carry raw pixels compressed with zlib (o=z): RGB for an opaque
// image, RGBA for one with translucency. Skipping PNG's per-row filter trials
// makes the encode about three times faster for a comparable payload.
func TestKittyZlibRoundTrip(t *testing.T) {
	opaque := image.NewRGBA(image.Rect(0, 0, 5, 3))
	for i := 0; i < len(opaque.Pix); i += 4 {
		opaque.Pix[i], opaque.Pix[i+1], opaque.Pix[i+2], opaque.Pix[i+3] = byte(i), byte(i*7), byte(i*13), 255
	}
	apc := buildKittyAPC(opaque, 7, 5, 3, KittyFormatZlib)
	data := decodeKittyRaw(t, apc, "o=z", "f=24", "s=5", "v=3", "i=7")
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(r)
	want := make([]byte, 0, 45)
	for i := 0; i < len(opaque.Pix); i += 4 {
		want = append(want, opaque.Pix[i], opaque.Pix[i+1], opaque.Pix[i+2])
	}
	if !bytes.Equal(raw, want) {
		t.Fatalf("opaque payload %v, want RGB %v", raw, want)
	}
	translucent := randomNRGBA(4, 2)
	apc = buildKittyAPC(translucent, 8, 4, 2, KittyFormatZlib)
	if strings.Contains(apc, "f=24") {
		t.Fatal("translucent frame sent as RGB")
	}
	data = decodeKittyRaw(t, apc, "o=z", "s=4", "v=2") // RGBA is the protocol default: no f= key.
	r, _ = zlib.NewReader(bytes.NewReader(data))
	raw, _ = io.ReadAll(r)
	if !bytes.Equal(raw, translucent.Pix) {
		t.Fatal("translucent payload is not the straight-alpha RGBA rows")
	}
	if KittyFormatZlib.String() != "zlib" || normalizeKittyFormat(KittyFormatZlib) != KittyFormatZlib || normalizeKittyFormat(KittyFormat(9)) != KittyFormatPNG {
		t.Fatal("format naming or normalization")
	}
}

func BenchmarkKittyFrameFormats(b *testing.B) {
	page := image.NewRGBA(image.Rect(0, 0, 2000, 1000))
	for y := 0; y < 1000; y++ {
		for x := 0; x < 2000; x++ {
			c := color.RGBA{255, 255, 255, 255}
			if (x*7+y*13)%23 < 4 {
				c = color.RGBA{30, 30, 30, 255}
			}
			page.SetRGBA(x, y, c)
		}
	}
	for _, f := range []KittyFormat{KittyFormatPNG, KittyFormatZlib, KittyFormatRGBA} {
		b.Run(f.String(), func(b *testing.B) {
			var n int
			b.ReportAllocs()
			for b.Loop() {
				n = len(buildKittyAPC(page, 1, 200, 50, f))
			}
			b.ReportMetric(float64(n)/1e6, "MB/frame")
		})
	}
}
