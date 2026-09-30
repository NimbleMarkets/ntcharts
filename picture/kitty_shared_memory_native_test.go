//go:build linux || (darwin && cgo)

package picture

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"
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

func TestNativeSharedMemoryWaitsForProbe(t *testing.T) {
	original := tmuxPassthroughEnabled.Load()
	SetTmuxPassthrough(false)
	defer SetTmuxPassthrough(original)
	resetKittySharedCap(t)
	m := NewWithConfig(Config{KittyMedium: KittyMediumSharedMemory, Fit: FitFill, CellPixelWidth: 1, CellPixelHeight: 1})
	m.mode = PictureKitty
	m.cols, m.rows = 2, 2
	m.img = randomNRGBA(2, 2)
	defer m.SetImage(nil)
	for _, c := range []KittyCapability{KittyCapabilityUnknown, KittyCapabilityUnsupported} {
		kittySharedCap.Store(int32(c))
		f := m.SetImage(randomNRGBA(2, 2))().(KittyFrameMsg)
		if f.Medium != KittyMediumDirect || f.sharedMemory != nil || strings.Contains(f.APC, "t=s") {
			t.Fatalf("capability %v: sent a shared-memory frame: %+v", c, f)
		}
	}
	recordKittyResponse(sharedProbeReply("OK"))
	if f := m.SetImage(randomNRGBA(2, 2))().(KittyFrameMsg); f.Medium != KittyMediumSharedMemory {
		t.Fatalf("probed terminal got a direct frame: %+v", f)
	}
}

func TestNativeSharedMemoryQuery(t *testing.T) {
	original := tmuxPassthroughEnabled.Load()
	SetTmuxPassthrough(false)
	defer SetTmuxPassthrough(original)
	resetKittySharedCap(t)
	apc := kittySharedQueryAPC()
	probe := kittySharedProbe.Load()
	if probe == nil {
		t.Fatal("no probe object")
	}
	defer probe.Unlink()
	name := string(decodeKittyRaw(t, apc, "a=q", "t=s", "f=32", "s=1,v=1", "S=4", fmt.Sprintf("i=%d", kittySharedProbeID)))
	if name != probe.Name {
		t.Fatalf("query names %q, want %q", name, probe.Name)
	}
	if strings.Contains(apc, "q=") {
		t.Fatalf("query suppresses its reply: %q", apc)
	}
	recordKittyTimeout()
	if !probe.Done() || kittySharedProbe.Load() != nil {
		t.Fatal("unanswered probe object not released")
	}
}

// queryKittySupportRaw reruns the once-only startup probe and returns what
// it writes to the terminal.
func queryKittySupportRaw(t *testing.T) string {
	t.Helper()
	original := tmuxPassthroughEnabled.Load()
	SetTmuxPassthrough(false)
	t.Cleanup(func() { SetTmuxPassthrough(original) })
	kittyQueryOnce = sync.Once{}
	cmds := collectBatchCmds(QueryKittySupport())
	if len(cmds) == 0 {
		t.Fatal("no probe sent")
	}
	t.Cleanup(func() { _ = kittySharedProbe.Swap(nil).Unlink() })
	raw, _ := cmds[0]().(tea.RawMsg)
	seq, _ := raw.Msg.(string)
	return seq
}

func TestQueryKittySupportProbesSharedMemoryFirst(t *testing.T) {
	resetKittyCapability(t)
	resetKittySharedCap(t)
	t.Setenv("KITTY_WINDOW_ID", "1")
	seq := queryKittySupportRaw(t)
	shared, direct := strings.Index(seq, "a=q,t=s"), strings.Index(seq, "a=q,t=d")
	if shared < 0 || direct < shared {
		t.Fatalf("want the t=s query before the t=d query: %q", seq)
	}
}

func TestQueryKittySupportProbesSharedMemoryWhenForced(t *testing.T) {
	resetKittySharedCap(t)
	seq := queryKittySupportRaw(t)
	if !strings.Contains(seq, "a=q,t=s") || strings.Contains(seq, "a=q,t=d") {
		t.Fatalf("want only the t=s query: %q", seq)
	}
}
