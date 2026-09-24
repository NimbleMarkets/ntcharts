package main

import (
	"bytes"
	"os"
	"testing"
)

// Enable explicitly: native driver access is not available on every CI host.
// Run without -race: gogpu's Metal FFI currently has a checkptr incompatibility.
func TestGPUAllPresets(t *testing.T) {
	if os.Getenv("NTCHARTS_GPU_TEST") != "1" {
		t.Skip("set NTCHARTS_GPU_TEST=1 for real GPU tests")
	}
	g, err := newGPU()
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	for i, p := range presets {
		t.Run(p.name, func(t *testing.T) {
			req := renderRequest{preset: i, width: 160, height: 100, seconds: 1, speed: 0.6, scale: p.scale, detail: p.detail}
			first, err := g.Render(req)
			if err != nil {
				t.Fatal(err)
			}
			req.seconds = 3
			second, err := g.Render(req)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(first.Pix, second.Pix) {
				t.Fatal("shader did not animate")
			}
			varied := false
			for i := 0; i < len(first.Pix); i += 4 {
				if first.Pix[i+3] != 255 {
					t.Fatal("non-opaque pixel")
				}
				if !bytes.Equal(first.Pix[i:i+3], first.Pix[:3]) {
					varied = true
				}
			}
			if !varied {
				t.Fatal("shader produced a uniform frame")
			}
		})
	}
	g.Close()
	if _, err := g.Render(renderRequest{}); err == nil {
		t.Fatal("closed GPU accepted a render")
	}
}
