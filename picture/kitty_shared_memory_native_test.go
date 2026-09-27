//go:build linux || (darwin && cgo)

package picture

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"
)

func TestSharedMemoryTmuxWrapsOnce(t *testing.T) {
	original := tmuxPassthroughEnabled.Load()
	SetTmuxPassthrough(true)
	defer SetTmuxPassthrough(original)
	apc, obj, err := buildKittySharedMemoryAPC(randomNRGBA(2, 2), 45, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Unlink()
	if strings.Count(apc, "tmux;") != 1 || !strings.HasPrefix(apc, "\x1bPtmux;\x1b\x1b_G") {
		t.Fatalf("shared-memory APC must have exactly one tmux wrapper: %q", apc)
	}
}

func TestNativeSharedMemoryFrameLifecycle(t *testing.T) {
	original := tmuxPassthroughEnabled.Load()
	SetTmuxPassthrough(false)
	defer SetTmuxPassthrough(original)
	m := NewWithConfig(Config{KittyMedium: KittyMediumSharedMemory, Fit: FitFill, CellPixelWidth: 1, CellPixelHeight: 1})
	m.mode = PictureKitty
	m.cols, m.rows = 2, 2
	m.img = randomNRGBA(2, 2)
	defer m.SetImage(nil)
	first := m.renderCmd()().(KittyFrameMsg)
	if first.Medium != KittyMediumSharedMemory || first.Format != KittyFormatRGBA {
		t.Fatalf("native shared memory unavailable: %+v", first)
	}
	name := string(decodeKittyRaw(t, first.APC, "t=s", "S=16", "s=2,v=2", "U=1", "q=2"))
	if name != first.sharedMemory.Name {
		t.Fatal("wrong shared-memory reference")
	}
	if m.Update(first) == nil {
		t.Fatal("frame not submitted")
	}
	next := m.SetImage(randomNRGBA(2, 2))().(KittyFrameMsg)
	if first.sharedMemory.Done() {
		t.Fatal("submitted frame unlinked before terminal could read")
	}
	// A frame that never reaches Update may be discarded immediately.
	last := m.SetImage(randomNRGBA(2, 2))().(KittyFrameMsg)
	if !next.sharedMemory.Done() {
		t.Fatal("dropped frame not unlinked")
	}
	m.SetImage(nil)
	if !first.sharedMemory.Done() || !last.sharedMemory.Done() {
		t.Fatal("model clear leaked buffers")
	}
}

func BenchmarkKittySharedMemory(b *testing.B) {
	src := image.NewRGBA(image.Rect(0, 0, 1280, 960))
	draw.Draw(src, src.Bounds(), image.NewUniform(color.RGBA{R: 20, G: 40, B: 60, A: 255}), image.Point{}, draw.Src)
	b.ReportAllocs()
	for b.Loop() {
		_, obj, err := buildKittySharedMemoryAPC(src, 45, 160, 60)
		if err != nil {
			b.Fatal(err)
		}
		if err := obj.Unlink(); err != nil {
			b.Fatal(err)
		}
	}
}
