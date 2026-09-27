package picture

import "testing"

func TestKittyMediumConfigAndSetter(t *testing.T) {
	m := New()
	if m.KittyMedium() != KittyMediumDirect {
		t.Fatal("default medium")
	}
	m = NewWithConfig(Config{KittyMedium: KittyMediumSharedMemory})
	if m.KittyMedium() != KittyMediumSharedMemory {
		t.Fatal("config ignored")
	}
	seq := m.seq
	if m.SetKittyMedium(KittyMediumSharedMemory) != nil || seq != m.seq {
		t.Fatal("unchanged medium invalidates")
	}
	if m.SetKittyMedium(KittyMedium(99)) != nil || m.KittyMedium() != KittyMediumDirect || m.seq != seq+1 {
		t.Fatal("unknown medium not normalized")
	}
	m = NewWithConfig(Config{KittyMedium: KittyMedium(99)})
	if m.KittyMedium() != KittyMediumDirect {
		t.Fatal("unknown config not normalized")
	}
	m.mode = PictureKitty
	m.cols, m.rows = 1, 1
	m.img = randomNRGBA(8, 16)
	if m.SetKittyMedium(KittyMediumSharedMemory) == nil {
		t.Fatal("medium change did not schedule render")
	}
}

func TestKittySharedMemoryFallback(t *testing.T) {
	if KittySharedMemorySupported() {
		t.Skip("transport available")
	}
	for _, format := range []KittyFormat{KittyFormatPNG, KittyFormatRGBA} {
		m := NewWithConfig(Config{KittyMedium: KittyMediumSharedMemory, KittyFormat: format, Fit: FitFill})
		m.mode = PictureKitty
		m.cols, m.rows = 1, 1
		m.img = randomNRGBA(8, 16)
		f, ok := m.renderCmd()().(KittyFrameMsg)
		if !ok || f.Medium != KittyMediumDirect || f.Format != format || f.APC == "" {
			t.Fatalf("bad fallback: %+v", f)
		}
	}
}
