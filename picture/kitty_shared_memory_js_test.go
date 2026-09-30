//go:build js && wasm

package picture

import (
	"bytes"
	"image"
	"regexp"
	"strings"
	"syscall/js"
	"testing"
)

func shmTestRegistry(t *testing.T) js.Value {
	// The registry is the browser's signal; no terminal query is involved.
	resetKittySharedCap(t)
	t.Helper()
	old := js.Global().Get("ghosttyKittySharedMemory")
	r := js.Global().Get("Map").New()
	js.Global().Set("ghosttyKittySharedMemory", r)
	original := tmuxPassthroughEnabled.Load()
	SetTmuxPassthrough(false)
	t.Cleanup(func() { js.Global().Set("ghosttyKittySharedMemory", old); SetTmuxPassthrough(original) })
	return r
}

func shmTestModel() Model {
	m := NewWithConfig(Config{KittyMedium: KittyMediumSharedMemory, Fit: FitFill, CellPixelWidth: 1, CellPixelHeight: 1})
	m.mode = PictureKitty
	m.cols, m.rows = 2, 2
	m.img = randomNRGBA(2, 2)
	return m
}

func TestSharedMemoryFrame(t *testing.T) {
	r := shmTestRegistry(t)
	m := shmTestModel()
	f := m.renderCmd()().(KittyFrameMsg)
	if f.Medium != KittyMediumSharedMemory || f.Format != KittyFormatRGBA {
		t.Fatalf("wrong path: %+v", f)
	}
	name := string(decodeKittyRaw(t, f.APC, "t=s", "S=16", "s=2,v=2", "U=1", "q=2"))
	if !regexp.MustCompile(`^/ntc-[0-9a-f]{20}$`).MatchString(name) {
		t.Fatal(name)
	}
	data := make([]byte, 16)
	js.CopyBytesToGo(data, r.Call("get", name))
	if !bytes.Equal(data, m.img.(*image.NRGBA).Pix) {
		t.Fatal("pixels changed")
	}
	if len(f.APC) > 150 {
		t.Fatal("pixels included in APC")
	}
	other := shmTestModel()
	if other.Update(f) != nil || !r.Call("has", name).Bool() {
		t.Fatal("foreign model consumed frame")
	}
	if m.Update(f) == nil {
		t.Fatal("valid frame dropped")
	}
	// The terminal consumes the entry synchronously when it sees the APC.
	r.Call("delete", name)
	if err := f.sharedMemory.Unlink(); err != nil {
		t.Fatal(err)
	}
}

func TestSharedMemoryDroppedAndOutOfOrderFrames(t *testing.T) {
	r := shmTestRegistry(t)
	m := shmTestModel()
	first := m.renderCmd()().(KittyFrameMsg)
	oldCmd := m.SetImage(randomNRGBA(2, 2))
	newCmd := m.SetImage(randomNRGBA(2, 2))
	newest := newCmd().(KittyFrameMsg)
	if r.Get("size").Int() != 1 || r.Call("has", first.sharedMemory.Name).Bool() {
		t.Fatal("dropped predecessor leaked")
	}
	if oldCmd() != nil {
		t.Fatal("out-of-order command produced stale frame")
	}
	if r.Get("size").Int() != 1 || !r.Call("has", newest.sharedMemory.Name).Bool() {
		t.Fatal("newest frame lost")
	}
	if m.Update(first) != nil {
		t.Fatal("stale frame accepted")
	}
	m.SetImage(nil)
	if r.Get("size").Int() != 0 {
		t.Fatal("clearing image leaked")
	}
	late := m.SetImage(randomNRGBA(2, 2))
	m.Toggle()
	if late() != nil || r.Get("size").Int() != 0 {
		t.Fatal("late render resurrected object in glyph mode")
	}
}

func TestSharedMemoryResizeAndDirectSwitch(t *testing.T) {
	r := shmTestRegistry(t)
	m := shmTestModel()
	f := m.renderCmd()().(KittyFrameMsg)
	m.Update(f)
	resize := m.SetSize(3, 2)().(KittyFrameMsg)
	if !strings.HasPrefix(resize.APC, kittyDeleteImage(m.kittyID)) {
		t.Fatal("resize lost previous-image delete")
	}
	direct := m.SetKittyMedium(KittyMediumDirect)().(KittyFrameMsg)
	if direct.Medium != KittyMediumDirect || direct.Format != KittyFormatPNG {
		t.Fatal("switch did not restore PNG")
	}
	if r.Get("size").Int() != 0 {
		t.Fatal("direct switch leaked")
	}
	if !strings.Contains(direct.APC, "f=100") {
		t.Fatal("wrong direct format")
	}
}

func TestSharedMemoryRepeatedGenerationDropsReplacedObject(t *testing.T) {
	r := shmTestRegistry(t)
	m := shmTestModel()
	first := m.renderCmd()().(KittyFrameMsg)
	next := m.renderCmd()().(KittyFrameMsg)
	if r.Get("size").Int() != 1 {
		t.Fatal("repeated generation leaked")
	}
	if m.Update(first) != nil {
		t.Fatal("accepted an unlinked frame")
	}
	if m.Update(next) == nil {
		t.Fatal("dropped current frame")
	}
	m.SetSize(0, 0)
	if r.Get("size").Int() != 0 {
		t.Fatal("zero size leaked")
	}
}

func TestSharedMemoryCreationFailureFallsBackToDirect(t *testing.T) {
	r := shmTestRegistry(t)
	r.Set("set", js.Global().Get("Function").New("throw new Error('registry unavailable')"))
	m := shmTestModel()
	defer m.SetImage(nil)
	f := m.renderCmd()().(KittyFrameMsg)
	if f.Medium != KittyMediumDirect || f.Format != KittyFormatPNG || f.sharedMemory != nil {
		t.Fatalf("failed shared-memory creation did not fall back: %+v", f)
	}
	if !strings.Contains(f.APC, "f=100") || m.Update(f) == nil {
		t.Fatal("fallback did not produce a usable PNG frame")
	}
	if r.Get("size").Int() != 0 {
		t.Fatal("failed creation leaked a registry entry")
	}
}
