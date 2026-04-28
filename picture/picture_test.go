package picture

import (
	"image"
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// smallImage returns a 4×4 *image.RGBA filled with a solid color.
func smallImage(c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestModel_GlyphSmoke_RendersNonEmpty(t *testing.T) {
	m := New()
	if cmd := m.SetSize(40, 20); cmd != nil {
		t.Fatalf("SetSize should return nil in Glyph mode, got %v", cmd)
	}
	if cmd := m.SetImage(smallImage(color.RGBA{R: 200, G: 50, B: 50, A: 255})); cmd != nil {
		t.Fatalf("SetImage should return nil in Glyph mode, got %v", cmd)
	}
	got := m.View().Content
	if got == "" {
		t.Fatal("expected non-empty Glyph render, got empty string")
	}
}

func TestModel_NoImage_ViewIsEmpty(t *testing.T) {
	m := New()
	m.SetSize(40, 20)
	if got := m.View().Content; got != "" {
		t.Fatalf("expected empty View() with no image, got %q", got)
	}
}

func TestModel_SetImageNil_ClearsView(t *testing.T) {
	m := New()
	m.SetSize(40, 20)
	m.SetImage(smallImage(color.RGBA{R: 0, G: 200, B: 0, A: 255}))
	if m.View().Content == "" {
		t.Fatal("precondition: View should be non-empty after SetImage")
	}
	m.SetImage(nil)
	if got := m.View().Content; got != "" {
		t.Fatalf("expected empty View() after SetImage(nil), got %q", got)
	}
}

func TestModel_KittyMode_ViewSilentUntilFrameDelivered(t *testing.T) {
	m := New()
	m.SetSize(20, 10)
	if cmd := m.Toggle(); cmd != nil {
		// Toggle to Kitty before any image: nothing to render, want nil.
		t.Fatalf("Toggle on empty Model should return nil, got non-nil cmd")
	}
	if m.Mode() != PictureKitty {
		t.Fatal("expected Kitty mode after Toggle")
	}

	cmd := m.SetImage(smallImage(color.RGBA{R: 50, G: 50, B: 200, A: 255}))
	if cmd == nil {
		t.Fatal("SetImage in Kitty mode should return a render Cmd, got nil")
	}

	// Before delivering the frame, View should be empty (silent mid-render).
	if got := m.View().Content; got != "" {
		t.Fatalf("expected empty View() before KittyFrameMsg, got %q", got)
	}

	// Execute the Cmd to produce the KittyFrameMsg.
	msg := cmd()
	frame, ok := msg.(KittyFrameMsg)
	if !ok {
		t.Fatalf("expected KittyFrameMsg, got %T", msg)
	}

	// Feed it back via Update; should produce a tea.Raw of the APC.
	out := m.Update(frame)
	if out == nil {
		t.Fatal("Update with matching KittyFrameMsg should return a Cmd, got nil")
	}
	if rawMsg := out(); rawMsg == nil {
		t.Fatal("Cmd from Update should produce a non-nil tea.Msg")
	}

	if got := m.View().Content; got == "" {
		t.Fatal("expected non-empty View() after frame applied")
	}
}

func TestModel_KittyMode_StaleFrameIgnored(t *testing.T) {
	m := New()
	m.SetSize(20, 10)
	m.Toggle() // Kitty
	cmd := m.SetImage(smallImage(color.RGBA{R: 0, G: 0, B: 0, A: 255}))
	if cmd == nil {
		t.Fatal("expected non-nil Cmd")
	}
	frame := cmd().(KittyFrameMsg)

	// Bump seq by setting a new image; the previously-built frame is now stale.
	m.SetImage(smallImage(color.RGBA{R: 255, G: 255, B: 255, A: 255}))

	if out := m.Update(frame); out != nil {
		t.Fatalf("expected stale KittyFrameMsg to be ignored (nil Cmd), got non-nil")
	}
}

func TestModel_KittyMode_RejectsOtherModelsFrame(t *testing.T) {
	// Two Models with default config both default to KittyID=43 and, after
	// the same sequence of mutations, the same seq. Without a per-Model
	// discriminator, a frame from a is accepted by b in tea.Program (which
	// broadcasts every msg to every Model).
	a := New()
	a.SetSize(20, 10)
	a.Toggle()
	cmdA := a.SetImage(smallImage(color.RGBA{R: 100, G: 0, B: 0, A: 255}))
	if cmdA == nil {
		t.Fatal("expected non-nil Cmd from a.SetImage")
	}
	frameA := cmdA().(KittyFrameMsg)

	b := New()
	b.SetSize(20, 10)
	b.Toggle()
	if cmd := b.SetImage(smallImage(color.RGBA{R: 0, G: 100, B: 0, A: 255})); cmd == nil {
		t.Fatal("expected non-nil Cmd from b.SetImage")
	}

	if out := b.Update(frameA); out != nil {
		t.Fatalf("expected b to reject Model a's KittyFrameMsg (cross-talk), got non-nil cmd")
	}
	if got := b.View().Content; got == frameA.Grid {
		t.Fatal("expected b's View not to be set to a's grid")
	}
}

// Compile-time check that Update accepts arbitrary tea.Msg without panicking.
type unrelatedMsg struct{}

func TestModel_Update_IgnoresUnknownMessages(t *testing.T) {
	m := New()
	if out := m.Update(unrelatedMsg{}); out != nil {
		t.Fatalf("expected nil Cmd for unknown message type, got %v", out)
	}
	if out := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24}); out != nil {
		t.Fatalf("expected nil Cmd for WindowSizeMsg, got %v", out)
	}
}
