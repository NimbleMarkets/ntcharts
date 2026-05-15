package heatpicture

import (
	"image/color"
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

// TestMain defaults the package-wide Kitty capability to Supported so
// the rendering tests (which Toggle into Kitty as setup) proceed past
// picture.Model.Toggle's strict capability gate.
func TestMain(m *testing.M) {
	picture.ForceKittyCapability(picture.KittyCapabilitySupported)
	os.Exit(m.Run())
}

// constSampler returns a Sampler that always reports the same value, useful
// for verifying the rendering pipeline ignores the input coordinates.
func constSampler(v float64) Sampler {
	return func(_, _ float64) float64 { return v }
}

// fullScale is a no-frills two-stop scale used in most Model tests.
var fullScale = []color.Color{
	lipgloss.Color("#000000"),
	lipgloss.Color("#ffffff"),
}

// TestNewDefaults verifies the zero-Config Model has the documented
// defaults (kitty ID, cell pixel size, glyph mode).
func TestNewDefaults(t *testing.T) {
	m := New()
	if m.cellW != 8 || m.cellH != 16 {
		t.Errorf("default cell pixel size = %dx%d, want 8x16", m.cellW, m.cellH)
	}
	if m.Mode() != picture.PictureGlyph {
		t.Errorf("default Mode = %v, want PictureGlyph", m.Mode())
	}
}

// TestNewWithConfigOverrides verifies Config.CellPixelWidth/Height carry
// through to the Model and to the embedded picture.Model.
func TestNewWithConfigOverrides(t *testing.T) {
	m := NewWithConfig(Config{CellPixelWidth: 12, CellPixelHeight: 24, KittyID: 999})
	if m.cellW != 12 || m.cellH != 24 {
		t.Errorf("config cell pixel size = %dx%d, want 12x24", m.cellW, m.cellH)
	}
	if w, h := m.pic.CellPixelSize(); w != 12 || h != 24 {
		t.Errorf("embedded picture cell pixel size = %dx%d, want 12x24", w, h)
	}
}

// TestSetSampler_PreRequisites verifies SetSampler returns nil when state
// is incomplete (no size, no scale) and a non-nil Cmd once both are set
// AND no render is in flight. Validates the Kitty-mode resolution path
// (full cellW × cellH per cell); Glyph-mode resolution is covered
// separately.
func TestSetSampler_PreRequisites(t *testing.T) {
	m := New()
	if cmd := m.SetSampler(constSampler(0.5)); cmd != nil {
		t.Errorf("SetSampler before SetSize/SetColorScale should be no-op, got non-nil Cmd")
	}
	m.SetSize(10, 5)
	if cmd := m.SetSampler(constSampler(0.5)); cmd != nil {
		t.Errorf("SetSampler with size but no color scale should be no-op, got non-nil Cmd")
	}
	// Setting the color scale primes state: a render dispatches because
	// sampler is already stored from the previous SetSampler call.
	if cmd := m.SetColorScale(fullScale); cmd != nil {
		drainHeatRender(t, &m, cmd)
	}
	if cmd := m.Toggle(); cmd != nil { // Kitty mode for full-res sampling
		drainHeatRender(t, &m, cmd)
	}
	cmd := m.SetSampler(constSampler(0.5))
	if cmd == nil {
		t.Fatal("SetSampler with size + scale set + no in-flight render should return a render Cmd")
	}
	msg, ok := cmd().(heatRenderedMsg)
	if !ok {
		t.Fatalf("expected heatRenderedMsg from render Cmd, got %T", cmd())
	}
	if msg.err != nil {
		t.Errorf("render produced error: %v", msg.err)
	}
	if msg.img == nil {
		t.Fatal("render produced nil image")
	}
	// In Kitty mode: 10 cols × 8 cellW = 80px wide; 5 rows × 16 cellH = 80px tall.
	if b := msg.img.Bounds(); b.Dx() != 80 || b.Dy() != 80 {
		t.Errorf("Kitty-mode rendered image dims = %dx%d, want 80x80 (cols×cellW × rows×cellH)", b.Dx(), b.Dy())
	}
}

// drainHeatRender runs cmd and forwards any heatRenderedMsg back into the
// model so renderPending clears and follow-ups (if any) are also drained.
// Used by tests to settle the throttle state before exercising new
// behavior. Takes *Model because the throttle mutates Model state via
// pointer receiver.
func drainHeatRender(t *testing.T, m *Model, cmd tea.Cmd) {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			return
		}
		cmd = m.Update(msg)
	}
}

// TestSetSampler_GlyphMode_SamplesAtHalfBlockResolution verifies that in
// Glyph mode the sampler is invoked at (cols, rows*2) pixels — one pixel
// per half-block — rather than full cellW×cellH per cell. This is a
// performance feature: at large terminal sizes a full-resolution sample
// runs into the millions of pixels per frame even though ansimage
// downscales it all back to cols × rows*2 anyway.
func TestSetSampler_GlyphMode_SamplesAtHalfBlockResolution(t *testing.T) {
	m := New()
	m.SetSize(40, 12) // cells; default Glyph mode
	m.SetColorScale(fullScale)
	cmd := m.SetSampler(constSampler(0.5))
	if cmd == nil {
		t.Fatal("expected render Cmd")
	}
	msg := cmd().(heatRenderedMsg)
	// Glyph: 40 cols × 1 = 40 px wide; 12 rows × 2 = 24 px tall.
	if b := msg.img.Bounds(); b.Dx() != 40 || b.Dy() != 24 {
		t.Errorf("Glyph-mode rendered image dims = %dx%d, want 40x24 (cols × rows*2)", b.Dx(), b.Dy())
	}
}

// TestRender_ConstSampler_FillsWithMidScaleColor verifies the value→color
// pipeline: a constant sampler at midpoint with a black→white scale should
// produce a fully-grey image (every pixel ~127). Uses Glyph mode (default)
// which samples at (cols, rows*2) pixels.
func TestRender_ConstSampler_FillsWithMidScaleColor(t *testing.T) {
	m := New()
	m.SetSize(4, 2) // Glyph: 4×4 px
	m.SetColorScale(fullScale)
	cmd := m.SetSampler(constSampler(0.5))
	if cmd == nil {
		t.Fatal("expected render Cmd")
	}
	msg := cmd().(heatRenderedMsg)
	if msg.img == nil {
		t.Fatal("nil image")
	}
	// Sample a few pixels; t=0.5 in a black→white scale interpolates to ~127.
	for _, p := range []struct{ x, y int }{{0, 0}, {2, 2}, {3, 3}} {
		c := msg.img.At(p.x, p.y)
		r, g, b, _ := c.RGBA()
		want8 := uint32(127)
		got8 := r >> 8
		if got8 < want8-2 || got8 > want8+2 {
			t.Errorf("pixel (%d,%d) R=%d, want ~127 (got rgb=%d,%d,%d)", p.x, p.y, got8, r>>8, g>>8, b>>8)
		}
	}
}

// TestUpdate_RoutesHeatRenderedMsgIntoPicture verifies a fresh
// heatRenderedMsg fed through Update produces a Cmd from the embedded
// picture.Model (which means SetImage was called and dispatched a
// Kitty-encode if applicable).
func TestUpdate_RoutesHeatRenderedMsgIntoPicture(t *testing.T) {
	m := New()
	m.SetSize(2, 1)
	m.SetColorScale(fullScale)
	m.Toggle() // Kitty mode so SetImage returns a non-nil Cmd
	cmd := m.SetSampler(constSampler(0.5))
	if cmd == nil {
		t.Fatal("expected render Cmd")
	}
	frame := cmd().(heatRenderedMsg)
	out := m.Update(frame)
	if out == nil {
		t.Fatal("Update with matching heatRenderedMsg in Kitty mode should return a Cmd from pic.SetImage")
	}
}

// TestUpdate_AcceptsOwnRender_KicksFollowupIfDirty verifies the throttle's
// completion semantics: a heatRenderedMsg from our own renderCmd is always
// accepted (image pushed via pic.SetImage), even when m.seq has advanced
// since the Cmd was scheduled. If a parameter changed during flight
// (dirty=true), a follow-up render is dispatched too.
//
// This is critical for the case where per-frame render cost exceeds the
// inter-setter spacing — without acceptance, every frame would be
// rejected as "stale" and the visible image would freeze.
func TestUpdate_AcceptsOwnRender_KicksFollowupIfDirty(t *testing.T) {
	m := New()
	m.SetSize(2, 1)
	m.SetColorScale(fullScale)
	cmd := m.SetSampler(constSampler(0.5))
	frame := cmd().(heatRenderedMsg)
	// Bump seq by changing a parameter; the in-flight frame is now
	// "stale" relative to current state, but it's still the freshest
	// in-flight result so we accept it AND kick a follow-up.
	m.SetValueRange(0, 10)
	out := m.Update(frame)
	// The frame was accepted; pic should now hold its image.
	if v := m.pic.View().Content; v == "" {
		t.Errorf("expected frame to be accepted (own render), pic.View was empty")
	}
	// Follow-up render Cmd should be dispatched for the dirty state.
	if out == nil {
		t.Fatal("expected follow-up render Cmd from dirty=true")
	}
	if _, ok := out().(heatRenderedMsg); !ok {
		t.Errorf("expected follow-up heatRenderedMsg, got %T", out())
	}
}

// TestUpdate_RejectsCrossModelMsg verifies a heatRenderedMsg with a
// foreign modelID is rejected — protects against cross-talk when
// multiple heatpicture Models share a tea.Program (every msg is broadcast
// to every Model).
func TestUpdate_RejectsCrossModelMsg(t *testing.T) {
	a := New()
	a.SetSize(2, 1)
	a.SetColorScale(fullScale)
	cmdA := a.SetSampler(constSampler(0.5))
	frameA := cmdA().(heatRenderedMsg)

	b := New()
	b.SetSize(2, 1)
	b.SetColorScale(fullScale)
	if out := b.Update(frameA); out != nil {
		t.Errorf("Model b should reject Model a's frame, got non-nil cmd")
	}
	if v := b.pic.View().Content; v != "" {
		t.Errorf("Model b's pic should not have been mutated by Model a's frame; got non-empty View %q", v)
	}
}

// TestUpdate_CellSizeEvent_ResizesAndRerenders verifies a CellSizeEvent
// updates the cell pixel size, propagates to the embedded picture.Model,
// and triggers a fresh render at the new pixel resolution. The throttle
// requires draining any in-flight render before the CellSizeEvent so we
// can observe the dispatched Cmd directly.
func TestUpdate_CellSizeEvent_ResizesAndRerenders(t *testing.T) {
	m := New()
	m.SetSize(2, 1) // cells
	m.SetColorScale(fullScale)
	if cmd := m.SetSampler(constSampler(0.5)); cmd != nil {
		drainHeatRender(t, &m, cmd)
	}
	if cmd := m.Toggle(); cmd != nil {
		drainHeatRender(t, &m, cmd)
	}
	// Drive a CellSizeEvent: 9x18 (non-default).
	out := m.Update(uv.CellSizeEvent{Width: 9, Height: 18})
	if out == nil {
		t.Fatal("CellSizeEvent should trigger a render Cmd when state is sufficient and no render in flight")
	}
	if m.cellW != 9 || m.cellH != 18 {
		t.Errorf("local cell pixel size = %dx%d, want 9x18", m.cellW, m.cellH)
	}
	if w, h := m.pic.CellPixelSize(); w != 9 || h != 18 {
		t.Errorf("embedded picture cell pixel size = %dx%d, want 9x18", w, h)
	}
	frame, ok := out().(heatRenderedMsg)
	if !ok {
		t.Fatalf("expected heatRenderedMsg, got %T", out())
	}
	// Kitty mode at 9x18: 2 cols × 9 = 18; 1 row × 18 = 18.
	if b := frame.img.Bounds(); b.Dx() != 18 || b.Dy() != 18 {
		t.Errorf("rendered image dims = %dx%d, want 18x18 (post-CellSize)", b.Dx(), b.Dy())
	}
}

// TestInit_DispatchesCellSizeRequest mirrors picture/chartpicture/pictureurl:
// Init forwards the cell-size CSI 16 t request so consumers integrate with
// one Init Cmd.
func TestInit_DispatchesCellSizeRequest(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init must return a non-nil Cmd")
	}
	if !batchContainsCSI16t(cmd) {
		t.Error("expected Init to include a CSI 16 t request (possibly batched with the Kitty support probe)")
	}
}

// batchContainsCSI16t walks a Cmd (running it, then walking a BatchMsg
// if produced) and reports whether any sub-Cmd produces a tea.RawMsg
// carrying "\x1b[16t".
func batchContainsCSI16t(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			if batchContainsCSI16t(sub) {
				return true
			}
		}
		return false
	}
	if raw, ok := msg.(tea.RawMsg); ok {
		if seq, _ := raw.Msg.(string); seq == "\x1b[16t" {
			return true
		}
	}
	return false
}

// TestSetSize_ZeroDoesNotPanic verifies SetSize(0,0) is a safe no-op
// (mirrors the picture/canvas clamp pattern; ensures heatpicture doesn't
// regress).
func TestSetSize_ZeroDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetSize(0,0) panicked: %v", r)
		}
	}()
	m := New()
	if cmd := m.SetSize(0, 0); cmd != nil {
		t.Errorf("SetSize(0,0) on empty Model should return nil, got non-nil Cmd")
	}
}

// TestThrottle_BurstCallsDispatchOneRender verifies that bursty setter
// calls (faster than one render finishes) collapse into a single in-flight
// render plus one follow-up — the dirty flag captures intermediate
// changes without queueing N renders behind the goroutine pool.
func TestThrottle_BurstCallsDispatchOneRender(t *testing.T) {
	m := New()
	m.SetSize(4, 2)
	m.SetColorScale(fullScale)

	// Initial SetSampler: state ready, dispatches a render.
	cmd1 := m.SetSampler(constSampler(0.0))
	if cmd1 == nil {
		t.Fatal("first SetSampler should dispatch a render")
	}

	// Second and third SetSampler before the first frame lands: throttled
	// to nil (dirty flag set).
	if cmd := m.SetSampler(constSampler(0.5)); cmd != nil {
		t.Errorf("second SetSampler during in-flight render should return nil, got non-nil")
	}
	if cmd := m.SetSampler(constSampler(1.0)); cmd != nil {
		t.Errorf("third SetSampler during in-flight render should return nil, got non-nil")
	}

	// Deliver the first render's result. Because dirty is set, Update
	// should also dispatch a follow-up render.
	frame := cmd1().(heatRenderedMsg)
	out := m.Update(frame)
	if out == nil {
		t.Fatal("Update should dispatch a follow-up render when dirty was set during flight")
	}
	// Follow-up should be a heatRenderedMsg-producing Cmd.
	if msg, ok := out().(heatRenderedMsg); !ok {
		t.Errorf("follow-up Cmd should produce heatRenderedMsg, got %T", out())
	} else if msg.err != nil {
		t.Errorf("follow-up render error: %v", msg.err)
	}
}

// TestSamplingFactor_HalvesPixelDimensions verifies SetSamplingFactor(0.5)
// shrinks the rendered Kitty source image by half on each axis.
func TestSamplingFactor_HalvesPixelDimensions(t *testing.T) {
	m := New()
	m.SetSize(10, 5)
	m.SetColorScale(fullScale)
	if cmd := m.SetSampler(constSampler(0.5)); cmd != nil {
		drainHeatRender(t, &m, cmd)
	}
	if cmd := m.Toggle(); cmd != nil { // Kitty
		drainHeatRender(t, &m, cmd)
	}
	if cmd := m.SetSamplingFactor(0.5); cmd != nil {
		frame := cmd().(heatRenderedMsg)
		if frame.img == nil {
			t.Fatal("nil image at SamplingFactor 0.5")
		}
		// Full Kitty: 10*8 × 5*16 = 80×80. At 0.5: 40×40.
		if b := frame.img.Bounds(); b.Dx() != 40 || b.Dy() != 40 {
			t.Errorf("dims at SamplingFactor 0.5 = %dx%d, want 40x40", b.Dx(), b.Dy())
		}
	} else {
		t.Fatal("SetSamplingFactor(0.5) should dispatch a render at the new resolution")
	}
}

// TestSamplingFactor_ConfigDefaultIsOne verifies that a zero / out-of-range
// Config.SamplingFactor falls back to 1.0 (full resolution).
func TestSamplingFactor_ConfigDefaultIsOne(t *testing.T) {
	m := New()
	if got := m.SamplingFactor(); got != 1.0 {
		t.Errorf("default SamplingFactor = %v, want 1.0", got)
	}
	m2 := NewWithConfig(Config{SamplingFactor: -1})
	if got := m2.SamplingFactor(); got != 1.0 {
		t.Errorf("negative SamplingFactor should clamp to 1.0, got %v", got)
	}
	m3 := NewWithConfig(Config{SamplingFactor: 2})
	if got := m3.SamplingFactor(); got != 1.0 {
		t.Errorf(">1 SamplingFactor should clamp to 1.0, got %v", got)
	}
}

// TestIsHeatPictureMsg verifies the helper recognizes both heatpicture-owned
// and picture-owned async messages so consumers can gate forwarding through
// a single check.
func TestIsHeatPictureMsg(t *testing.T) {
	if !IsHeatPictureMsg(heatRenderedMsg{}) {
		t.Error("expected IsHeatPictureMsg to match heatRenderedMsg")
	}
	if !IsHeatPictureMsg(uv.CellSizeEvent{}) {
		t.Error("expected IsHeatPictureMsg to match uv.CellSizeEvent (delegates to picture.IsPictureMsg)")
	}
	if IsHeatPictureMsg("hello") {
		t.Error("expected IsHeatPictureMsg to reject unrelated messages")
	}
}

func TestModel_Fit_FromConfig(t *testing.T) {
	m := NewWithConfig(Config{Fit: picture.FitFill})
	if got := m.Fit(); got != picture.FitFill {
		t.Fatalf("Fit from Config not honored: got %v want FitFill", got)
	}
}

func TestModel_Anchor_FromConfig(t *testing.T) {
	m := NewWithConfig(Config{Anchor: picture.AnchorTop})
	if got := m.Anchor(); got != picture.AnchorTop {
		t.Fatalf("Anchor from Config not honored: got %v want AnchorTop", got)
	}
}

func TestModel_SetFit_Forwards(t *testing.T) {
	m := New()
	m.SetSize(20, 10)
	if cmd := m.SetFit(picture.FitCover); cmd != nil {
		t.Fatalf("SetFit in Glyph mode should return nil, got %v", cmd)
	}
	if got := m.Fit(); got != picture.FitCover {
		t.Fatalf("SetFit didn't take: got %v want FitCover", got)
	}
}

func TestModel_SetAnchor_Forwards(t *testing.T) {
	m := New()
	m.SetSize(20, 10)
	if cmd := m.SetAnchor(picture.AnchorTop); cmd != nil {
		t.Fatalf("SetAnchor in Glyph mode should return nil, got %v", cmd)
	}
	if got := m.Anchor(); got != picture.AnchorTop {
		t.Fatalf("SetAnchor didn't take: got %v want AnchorTop", got)
	}
}
