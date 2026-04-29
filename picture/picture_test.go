package picture

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
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

func TestModel_View_NeitherModeEndsWithTrailingNewline(t *testing.T) {
	// pixterm/ansimage appends \n after every row including the last, while
	// the Kitty path's buildKittyGrid omits it. Without normalization, the
	// Glyph render carries a trailing \n that lipgloss.JoinVertical and
	// other newline-aware layout code interpret as an extra empty row,
	// making the surrounding box jump by one row when toggling modes.
	const cols, rows = 40, 20

	m := New()
	if cmd := m.SetSize(cols, rows); cmd != nil {
		t.Fatalf("SetSize: %v", cmd)
	}
	if cmd := m.SetImage(smallImage(color.RGBA{R: 100, G: 100, B: 100, A: 255})); cmd != nil {
		t.Fatalf("glyph SetImage should return nil, got %v", cmd)
	}

	glyph := m.View().Content
	if glyph == "" {
		t.Fatal("expected non-empty Glyph render")
	}
	if strings.HasSuffix(glyph, "\n") {
		t.Errorf("Glyph render must not end with a trailing \\n; layouts that count newlines (lipgloss.JoinVertical etc.) will see an extra row")
	}

	cmd := m.Toggle()
	if cmd == nil {
		t.Fatal("Toggle to Kitty should return a render Cmd")
	}
	msg := cmd()
	frame, ok := msg.(KittyFrameMsg)
	if !ok {
		t.Fatalf("expected KittyFrameMsg from Toggle, got %T", msg)
	}
	m.Update(frame)

	kitty := m.View().Content
	if kitty == "" {
		t.Fatal("expected non-empty Kitty render")
	}
	if strings.HasSuffix(kitty, "\n") {
		t.Errorf("Kitty render must not end with a trailing \\n")
	}
}

func TestModel_KittyMode_ToggleWithImage_EmitsFrame(t *testing.T) {
	// Toggle Glyph→Kitty while an image is already set: the Cmd must produce
	// a KittyFrameMsg, and after Update applies it the View must be non-empty.
	m := New()
	m.SetSize(20, 10)
	if cmd := m.SetImage(smallImage(color.RGBA{R: 0, G: 200, B: 0, A: 255})); cmd != nil {
		t.Fatalf("SetImage in Glyph mode should return nil, got %v", cmd)
	}

	cmd := m.Toggle()
	if cmd == nil {
		t.Fatal("Toggle Glyph→Kitty with active image should return a render Cmd")
	}
	if m.Mode() != PictureKitty {
		t.Fatal("expected Kitty mode after Toggle")
	}

	msg := cmd()
	frame, ok := msg.(KittyFrameMsg)
	if !ok {
		t.Fatalf("expected KittyFrameMsg from Toggle's render Cmd, got %T", msg)
	}
	if frame.APC == "" || frame.Grid == "" {
		t.Fatalf("expected populated frame, got APC=%q Grid=%q", frame.APC, frame.Grid)
	}

	if out := m.Update(frame); out == nil {
		t.Fatal("Update with the Toggle-emitted frame should return a Cmd")
	}
	if got := m.View().Content; got == "" {
		t.Fatal("expected non-empty View() after Toggle's frame applied")
	}
}

func TestModel_KittyMode_SetImage_KeepsOldGridUntilNewFrame(t *testing.T) {
	// Animation flicker fix: when only the image changes (same size, same
	// kittyID), the placeholder grid is byte-identical pre- and post-render.
	// SetImage must not blank kittyGrid synchronously; the old grid keeps
	// the previously-resident image visible until the new APC overwrites
	// it at the same kittyID.
	m := New()
	m.SetSize(20, 10)
	m.Toggle() // Kitty
	cmd := m.SetImage(smallImage(color.RGBA{R: 255, A: 255}))
	if cmd == nil {
		t.Fatal("expected non-nil render Cmd from initial SetImage")
	}
	frame1 := cmd().(KittyFrameMsg)
	m.Update(frame1)
	grid1 := m.View().Content
	if grid1 == "" {
		t.Fatal("precondition: expected non-empty View after first frame applied")
	}

	// Set a new image. Do NOT deliver the new frame. The grid must remain
	// the previous one so animation has no blank window.
	if cmd2 := m.SetImage(smallImage(color.RGBA{G: 255, A: 255})); cmd2 == nil {
		t.Fatal("expected non-nil render Cmd from second SetImage")
	}
	if got := m.View().Content; got != grid1 {
		t.Fatalf("View().Content blanked between frames; got %q, want previous grid (still placing image at kittyID)", got)
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

// TestKittyAPC_PrescalesSourceToCellRect verifies the encoded APC carries an
// image whose pixel dimensions exactly match the cell rectangle
// (cols × cellPixelW × rows × cellPixelH). With the pre-scale, Kitty's
// AR-preserve fit equals fill regardless of the terminal's actual cell pixel
// ratio.
func TestKittyAPC_PrescalesSourceToCellRect(t *testing.T) {
	m := New()
	m.SetSize(10, 5)
	m.Toggle() // Kitty
	cmd := m.SetImage(smallImage(color.RGBA{R: 1, A: 255}))
	if cmd == nil {
		t.Fatal("expected non-nil Cmd from SetImage in Kitty mode")
	}
	frame := cmd().(KittyFrameMsg)
	img := decodeKittyAPCImage(t, frame.APC)

	// Defaults: 10*8 × 5*16 = 80×80.
	if got := img.Bounds(); got.Dx() != 80 || got.Dy() != 80 {
		t.Errorf("expected encoded image 80×80, got %d×%d", got.Dx(), got.Dy())
	}
}

// TestKittyAPC_OmitsCropOptions verifies the APC doesn't carry w=,h= source
// crop options. The source is already pre-scaled to full placement dimensions,
// so crop options would be a no-op and misleading.
func TestKittyAPC_OmitsCropOptions(t *testing.T) {
	m := New()
	m.SetSize(10, 5)
	m.Toggle() // Kitty
	cmd := m.SetImage(smallImage(color.RGBA{R: 1, A: 255}))
	if cmd == nil {
		t.Fatal("expected non-nil Cmd from SetImage in Kitty mode")
	}
	frame := cmd().(KittyFrameMsg)
	if hasAPCOption(frame.APC, "w") {
		t.Errorf("APC should not contain w= source-crop option, got %q", frame.APC)
	}
	if hasAPCOption(frame.APC, "h") {
		t.Errorf("APC should not contain h= source-crop option, got %q", frame.APC)
	}
}

// TestSetCellPixelSize_AffectsEncodedDimensions verifies that SetCellPixelSize
// updates the dimensions of subsequently encoded Kitty image sources.
func TestSetCellPixelSize_AffectsEncodedDimensions(t *testing.T) {
	m := New()
	m.SetSize(10, 5)
	m.Toggle() // Kitty
	if cmd := m.SetCellPixelSize(9, 17); cmd != nil {
		// No image yet, render Cmd should be nil.
		t.Fatalf("SetCellPixelSize before any image should return nil, got %v", cmd)
	}
	cmd := m.SetImage(smallImage(color.RGBA{R: 1, G: 2, B: 3, A: 255}))
	if cmd == nil {
		t.Fatal("expected non-nil Cmd from SetImage in Kitty mode")
	}
	frame := cmd().(KittyFrameMsg)
	img := decodeKittyAPCImage(t, frame.APC)
	// 10 cols * 9 = 90; 5 rows * 17 = 85.
	if got := img.Bounds(); got.Dx() != 90 || got.Dy() != 85 {
		t.Errorf("expected encoded image 90×85, got %d×%d", got.Dx(), got.Dy())
	}
}

// TestSetCellPixelSize_NoOpOnSameValue verifies that re-setting the current
// cell pixel size does not churn the renderer (returns nil).
func TestSetCellPixelSize_NoOpOnSameValue(t *testing.T) {
	m := New()
	if cmd := m.SetCellPixelSize(8, 16); cmd != nil {
		t.Errorf("default 8x16, SetCellPixelSize(8,16) should be no-op, got non-nil Cmd")
	}
}

// TestSetCellPixelSize_ClampsNonPositive verifies that zero or negative inputs
// are clamped to 1 (a valid display pixel count).
func TestSetCellPixelSize_ClampsNonPositive(t *testing.T) {
	m := New()
	m.SetCellPixelSize(0, -5)
	gotW, gotH := m.CellPixelSize()
	if gotW != 1 || gotH != 1 {
		t.Errorf("expected non-positive inputs clamped to 1, got w=%d h=%d", gotW, gotH)
	}
}

// TestSetCellPixelSize_RerendersInFlightImage verifies that changing the cell
// pixel size while an image is set in Kitty mode triggers a re-render.
func TestSetCellPixelSize_RerendersInFlightImage(t *testing.T) {
	m := New()
	m.SetSize(10, 5)
	m.Toggle() // Kitty
	if cmd := m.SetImage(smallImage(color.RGBA{R: 1, A: 255})); cmd == nil {
		t.Fatal("expected non-nil Cmd from SetImage")
	}
	cmd := m.SetCellPixelSize(10, 20)
	if cmd == nil {
		t.Fatal("expected non-nil render Cmd from SetCellPixelSize while image is set in Kitty mode")
	}
	frame, ok := cmd().(KittyFrameMsg)
	if !ok {
		t.Fatalf("expected KittyFrameMsg, got %T", cmd())
	}
	img := decodeKittyAPCImage(t, frame.APC)
	// 10 cols * 10 = 100; 5 rows * 20 = 100.
	if got := img.Bounds(); got.Dx() != 100 || got.Dy() != 100 {
		t.Errorf("expected encoded image 100×100 after SetCellPixelSize(10,20), got %d×%d", got.Dx(), got.Dy())
	}
}

// TestCellPixelSize_DefaultsAreReportedByGetter verifies the getter returns
// the default 8x16 when no Config override and no SetCellPixelSize call.
func TestCellPixelSize_DefaultsAreReportedByGetter(t *testing.T) {
	m := New()
	w, h := m.CellPixelSize()
	if w != 8 || h != 16 {
		t.Errorf("default CellPixelSize should be 8x16, got %dx%d", w, h)
	}
}

// TestNewWithConfig_OverridesCellPixelSize verifies Config.CellPixelWidth/
// Height are honored at construction.
func TestNewWithConfig_OverridesCellPixelSize(t *testing.T) {
	m := NewWithConfig(Config{CellPixelWidth: 12, CellPixelHeight: 24})
	w, h := m.CellPixelSize()
	if w != 12 || h != 24 {
		t.Errorf("Config override should apply, got %dx%d", w, h)
	}
}

// TestRequestCellSize_EmitsRawWindowOp16 verifies that RequestCellSize returns
// a Cmd whose result is a tea.RawMsg carrying the CSI 16 t (XTWINOPS) escape,
// which asks the terminal to report its cell pixel size.
func TestRequestCellSize_EmitsRawWindowOp16(t *testing.T) {
	cmd := RequestCellSize()
	if cmd == nil {
		t.Fatal("RequestCellSize must return a non-nil Cmd")
	}
	msg := cmd()
	raw, ok := msg.(tea.RawMsg)
	if !ok {
		t.Fatalf("expected tea.RawMsg, got %T", msg)
	}
	seq, ok := raw.Msg.(string)
	if !ok {
		t.Fatalf("RawMsg.Msg should be a string, got %T", raw.Msg)
	}
	if seq != "\x1b[16t" {
		t.Errorf("expected CSI 16 t (\\x1b[16t), got %q", seq)
	}
}

// TestUpdate_AppliesCellSizeEvent verifies Update auto-applies a
// uv.CellSizeEvent via SetCellPixelSize and returns the resulting render Cmd
// when an image is in flight in Kitty mode.
func TestUpdate_AppliesCellSizeEvent(t *testing.T) {
	m := New()
	m.SetSize(10, 5)
	m.Toggle() // Kitty
	if cmd := m.SetImage(smallImage(color.RGBA{R: 1, A: 255})); cmd == nil {
		t.Fatal("expected non-nil Cmd from SetImage")
	}

	cmd := m.Update(uv.CellSizeEvent{Width: 9, Height: 17})
	if cmd == nil {
		t.Fatal("Update with CellSizeEvent should return a render Cmd in Kitty mode with an image set")
	}
	frame, ok := cmd().(KittyFrameMsg)
	if !ok {
		t.Fatalf("expected KittyFrameMsg, got %T", cmd())
	}
	img := decodeKittyAPCImage(t, frame.APC)
	// 10*9 = 90; 5*17 = 85.
	if got := img.Bounds(); got.Dx() != 90 || got.Dy() != 85 {
		t.Errorf("expected encoded image 90×85 after CellSizeEvent, got %d×%d", got.Dx(), got.Dy())
	}
	if w, h := m.CellPixelSize(); w != 9 || h != 17 {
		t.Errorf("expected CellPixelSize 9x17 after CellSizeEvent, got %dx%d", w, h)
	}
}

// TestUpdate_CellSizeEventNoImage verifies Update returns nil for a cell-size
// event when there is no image to render, but still updates the stored size
// so the next render uses it.
func TestUpdate_CellSizeEventNoImage(t *testing.T) {
	m := New()
	if cmd := m.Update(uv.CellSizeEvent{Width: 12, Height: 24}); cmd != nil {
		t.Errorf("Update with no image should return nil Cmd, got %v", cmd)
	}
	if w, h := m.CellPixelSize(); w != 12 || h != 24 {
		t.Errorf("expected stored cell size to update to 12x24, got %dx%d", w, h)
	}
}

// TestInit_DispatchesCellSizeRequest verifies Model.Init returns the same
// cell-size query as RequestCellSize so consumers can batch m.pic.Init() into
// their own Init Cmd and have terminal-reported cell dims auto-applied.
func TestInit_DispatchesCellSizeRequest(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init must return a non-nil Cmd")
	}
	raw, ok := cmd().(tea.RawMsg)
	if !ok {
		t.Fatalf("expected tea.RawMsg, got %T", cmd())
	}
	if seq, _ := raw.Msg.(string); seq != "\x1b[16t" {
		t.Errorf("expected CSI 16 t, got %q", seq)
	}
}

// TestIsPictureMsg_IncludesCellSizeEvent verifies the helper matches the
// terminal's CSI 16 t reply. Update auto-applies it via SetCellPixelSize, so
// consumers that gate message forwarding on IsPictureMsg (or on a wrapper
// that delegates to it) must route the event to Update or Kitty placements
// stay at the default 8×16 cell-pixel size.
func TestIsPictureMsg_IncludesCellSizeEvent(t *testing.T) {
	if !IsPictureMsg(uv.CellSizeEvent{Width: 8, Height: 16}) {
		t.Error("expected IsPictureMsg to recognize uv.CellSizeEvent — Update auto-applies it, so consumers gating forwarding on this helper must see it match")
	}
}

// TestIsPictureMsg_KittyFrameStillMatches pins the original behavior so
// broadening the helper for CellSizeEvent doesn't accidentally drop the
// existing KittyFrameMsg case.
func TestIsPictureMsg_KittyFrameStillMatches(t *testing.T) {
	if !IsPictureMsg(KittyFrameMsg{}) {
		t.Error("regression: KittyFrameMsg should still match")
	}
}

// TestIsPictureMsg_RejectsUnrelated pins the helper's negative case so the
// type switch can't drift to "match everything".
func TestIsPictureMsg_RejectsUnrelated(t *testing.T) {
	if IsPictureMsg("hello") {
		t.Error("expected IsPictureMsg to reject unrelated messages")
	}
}

// TestSetSize_NegativeClampedToZero verifies SetSize never lets negative dims
// reach the stored state. 0 is the natural sentinel for "no size yet" — used
// by renderCmd's <=0 guard — and clamping at the boundary keeps consumers
// (and the demo layer) from having to defend against negative-dim arithmetic
// upstream of the call.
func TestSetSize_NegativeClampedToZero(t *testing.T) {
	m := New()
	m.SetSize(-5, -10)
	if m.cols != 0 || m.rows != 0 {
		t.Errorf("expected SetSize(-5,-10) to clamp to (0,0), got (%d,%d)", m.cols, m.rows)
	}
	// SetSize(0, 0) after clamped SetSize(-5,-10) should be a no-op (already 0,0).
	if cmd := m.SetSize(0, 0); cmd != nil {
		t.Errorf("SetSize(0,0) after clamped SetSize(-5,-10) should be no-op, got non-nil Cmd")
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

func decodeKittyAPCImage(t *testing.T, apc string) image.Image {
	t.Helper()
	pngBytes, ok := extractAPCPayload(apc)
	if !ok {
		t.Fatal("could not extract PNG payload from APC")
	}
	img, _, err := image.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return img
}

func extractAPCPayload(apc string) ([]byte, bool) {
	const (
		prefix = "\x1b_G"
		suffix = "\x1b\\"
	)
	var encoded strings.Builder
	rest := apc
	for {
		start := strings.Index(rest, prefix)
		if start < 0 {
			break
		}
		seq := rest[start+len(prefix):]
		semi := strings.IndexByte(seq, ';')
		if semi < 0 {
			return nil, false
		}
		payloadAndSuffix := seq[semi+1:]
		end := strings.Index(payloadAndSuffix, suffix)
		if end < 0 {
			return nil, false
		}
		encoded.WriteString(payloadAndSuffix[:end])
		rest = payloadAndSuffix[end+len(suffix):]
	}
	if encoded.Len() == 0 {
		return nil, false
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded.String())
	return decoded, err == nil
}

func hasAPCOption(apc, key string) bool {
	const (
		prefix = "\x1b_G"
		suffix = "\x1b\\"
	)
	rest := apc
	for {
		start := strings.Index(rest, prefix)
		if start < 0 {
			return false
		}
		seq := rest[start+len(prefix):]
		semi := strings.IndexByte(seq, ';')
		if semi < 0 {
			return false
		}
		for _, opt := range strings.Split(seq[:semi], ",") {
			if strings.HasPrefix(opt, key+"=") {
				return true
			}
		}
		end := strings.Index(seq[semi+1:], suffix)
		if end < 0 {
			return false
		}
		rest = seq[semi+1+end+len(suffix):]
	}
}
