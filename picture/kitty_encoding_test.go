package picture

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

func decodeKittyPNG(t *testing.T, apc string) []byte {
	t.Helper()
	var payload strings.Builder
	chunks := strings.Split(strings.TrimSuffix(apc, "\x1b\\"), "\x1b\\")
	for i, chunk := range chunks {
		if !strings.HasPrefix(chunk, "\x1b_G") {
			t.Fatalf("invalid APC %q", chunk)
		}
		control, data, _ := strings.Cut(chunk[3:], ";")
		if len(data) > kitty.MaxChunkSize || len(data)%4 != 0 {
			t.Fatalf("invalid chunk length %d", len(data))
		}
		if !strings.Contains(control, "q=2") {
			t.Fatalf("missing quiet mode: %s", control)
		}
		if i == 0 {
			for _, want := range []string{"a=T", "f=100", "i=45", "c=2", "r=3", "U=1"} {
				if !strings.Contains(control, want) {
					t.Errorf("missing %s in %s", want, control)
				}
			}
		} else if strings.Contains(control, "a=") || strings.Contains(control, "i=") {
			t.Errorf("repeated first-chunk options: %s", control)
		}
		if len(chunks) > 1 {
			want := "m=1"
			if i == len(chunks)-1 {
				want = "m=0"
			}
			if !strings.Contains(control, want) {
				t.Errorf("missing %s in %s", want, control)
			}
		}
		payload.WriteString(data)
	}
	data, err := base64.StdEncoding.DecodeString(payload.String())
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestKittyBestSpeedPNGRoundTrip(t *testing.T) {
	original := tmuxPassthroughEnabled.Load()
	defer SetTmuxPassthrough(original)
	SetTmuxPassthrough(false)
	// Nonzero bounds, row padding, transparent and translucent pixels.
	src := image.NewNRGBA(image.Rect(0, 0, 240, 160)).SubImage(image.Rect(7, 9, 211, 145)).(*image.NRGBA)
	random := rand.New(rand.NewPCG(1, 2))
	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			src.SetNRGBA(x, y, color.NRGBA{uint8(random.Uint32()), uint8(random.Uint32()), uint8(random.Uint32()), uint8(random.Uint32())})
		}
	}
	apc := buildKittyAPC(src, 45, 2, 3)
	data := decodeKittyPNG(t, apc)
	var expected bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&expected, src); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, expected.Bytes()) {
		t.Fatal("payload does not use PNG BestSpeed")
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < src.Bounds().Dy(); y++ {
		for x := 0; x < src.Bounds().Dx(); x++ {
			want := src.NRGBAAt(x+src.Bounds().Min.X, y+src.Bounds().Min.Y)
			got := color.NRGBAModel.Convert(decoded.At(x, y)).(color.NRGBA)
			if got != want {
				t.Fatalf("pixel %d,%d: %v != %v", x, y, got, want)
			}
		}
	}
	SetTmuxPassthrough(true)
	if got := buildKittyAPC(src, 45, 2, 3); got != ansi.TmuxPassthrough(apc) {
		t.Fatal("tmux framing changed")
	}
}

func TestKittyPNGChunkBoundaries(t *testing.T) {
	// 3072 binary bytes encode to exactly 4096 base64 bytes.
	for _, size := range []int{1, 3071, 3072, 3073, 6144, 6145} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			data := bytes.Repeat([]byte{0x42}, size)
			got := decodeKittyPNG(t, buildKittyPNGAPC(data, 45, 2, 3))
			if !bytes.Equal(got, data) {
				t.Fatal("chunking lost bytes")
			}
		})
	}
}

func TestKittyFramingMatchesCharm(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	random := rand.New(rand.NewPCG(5, 6))
	for i := range src.Pix {
		src.Pix[i] = byte(random.Uint32())
	}
	var data, legacy bytes.Buffer
	if err := png.Encode(&data, src); err != nil {
		t.Fatal(err)
	}
	opts := &kitty.Options{Action: kitty.TransmitAndPut, Transmission: kitty.Direct, Format: kitty.PNG, ID: 45, Columns: 2, Rows: 3, VirtualPlacement: true, Quite: 2, Chunk: true}
	if err := kitty.EncodeGraphics(&legacy, src, opts); err != nil {
		t.Fatal(err)
	}
	if got := buildKittyPNGAPC(data.Bytes(), 45, 2, 3); got != legacy.String() {
		t.Fatal("payload framing differs from Charm")
	}
}

func TestKittyPayloadFramingMatchesCharm(t *testing.T) {
	for _, height := range []int{0, 1, 1023, 1024, 1025, 2048} {
		for _, chunk := range []bool{false, true} {
			for _, quiet := range []byte{0, 1, 2} {
				t.Run(fmt.Sprintf("height=%d/chunk=%v/quiet=%d", height, chunk, quiet), func(t *testing.T) {
					// RGB gives exactly 4096 base64 bytes at height 1024.
					img := image.NewNRGBA(image.Rect(0, 0, 1, height))
					data := make([]byte, 3*height)
					opts := kitty.Options{Action: kitty.TransmitAndPut, Transmission: kitty.Direct,
						Format: kitty.RGB, ID: 45, Quiet: quiet, Chunk: chunk,
						ChunkFormatter: ansi.TmuxPassthrough}
					localOpts := opts
					var upstream, local bytes.Buffer
					if err := kitty.EncodeGraphics(&upstream, img, &opts); err != nil {
						t.Fatal(err)
					}
					if err := encodeKittyGraphicsData(&local, data, &localOpts); err != nil {
						t.Fatal(err)
					}
					if local.String() != upstream.String() {
						t.Fatal("payload framing differs from upstream Charm")
					}
				})
			}
		}
	}
}

func TestKittyPayloadWriteError(t *testing.T) {
	writeErr := errors.New("writer failed")
	if err := encodeKittyGraphicsData(kittyFailWriter{writeErr}, []byte("PNG payload"), kittyPNGOptions(45, 2, 3)); !errors.Is(err, writeErr) {
		t.Fatalf("lost writer error: %v", err)
	}
}

type kittyFailWriter struct{ err error }

func (w kittyFailWriter) Write([]byte) (int, error) { return 0, w.err }

func BenchmarkKittyPNG(b *testing.B) {
	for _, content := range []string{"smooth", "noise"} {
		img := image.NewNRGBA(image.Rect(0, 0, 1134, 756))
		random := rand.New(rand.NewPCG(1, 2))
		for y := 0; y < 756; y++ {
			for x := 0; x < 1134; x++ {
				c := color.NRGBA{uint8(x * 255 / 1134), uint8(y * 255 / 756), uint8((x + y) * 255 / 1890), 255}
				if content == "noise" {
					c = color.NRGBA{uint8(random.Uint32()), uint8(random.Uint32()), uint8(random.Uint32()), 255}
				}
				img.SetNRGBA(x, y, c)
			}
		}
		for _, level := range []struct {
			name  string
			value png.CompressionLevel
		}{{"default", png.DefaultCompression}, {"best-speed", png.BestSpeed}} {
			b.Run(content+"/"+level.name, func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(img.Pix)))
				for b.Loop() {
					var data bytes.Buffer
					encoder := png.Encoder{CompressionLevel: level.value}
					if err := encoder.Encode(&data, img); err != nil {
						b.Fatal(err)
					}
					apc := buildKittyPNGAPC(data.Bytes(), 45, 2, 3)
					b.ReportMetric(float64(len(apc)), "APC-bytes/op")
				}
			})
		}
	}
}
