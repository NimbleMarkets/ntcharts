package picture

import (
	"image/color"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi/kitty"
)

// resetKittyCapability restores the package-level state to Unknown
// between tests. The probe's sync.Once isn't reset — these tests don't
// call QueryKittySupport, they manipulate state directly via
// ForceKittyCapability and the package-internal record* helpers.
func resetKittyCapability(t *testing.T) {
	t.Helper()
	ForceKittyCapability(KittyCapabilityUnknown)
}

func TestKittyCapability_DefaultUnknown(t *testing.T) {
	resetKittyCapability(t)
	if got := KittySupported(); got != KittyCapabilityUnknown {
		t.Fatalf("KittySupported default = %v, want KittyCapabilityUnknown", got)
	}
}

func TestForceKittyCapability_RoundTrips(t *testing.T) {
	resetKittyCapability(t)
	for _, c := range []KittyCapability{
		KittyCapabilitySupported,
		KittyCapabilityUnsupported,
		KittyCapabilityUnknown,
	} {
		ForceKittyCapability(c)
		if got := KittySupported(); got != c {
			t.Errorf("after Force(%v): KittySupported = %v", c, got)
		}
	}
}

// TestRecordKittyResponse_OurProbeMarksSupported verifies a Kitty graphics
// event carrying our probe's image ID resolves Unknown → Supported.
func TestRecordKittyResponse_OurProbeMarksSupported(t *testing.T) {
	resetKittyCapability(t)
	recordKittyResponse(uv.KittyGraphicsEvent{
		Options: kitty.Options{ID: kittyProbeID},
	})
	if got := KittySupported(); got != KittyCapabilitySupported {
		t.Fatalf("after probe response: KittySupported = %v, want Supported", got)
	}
}

// TestRecordKittyResponse_OtherIDIgnored verifies a Kitty event for a
// different image ID (e.g., a regular image's response) does NOT
// promote Unknown — we only trust responses to our specific probe.
func TestRecordKittyResponse_OtherIDIgnored(t *testing.T) {
	resetKittyCapability(t)
	recordKittyResponse(uv.KittyGraphicsEvent{
		Options: kitty.Options{ID: 12345}, // not our probe ID
	})
	if got := KittySupported(); got != KittyCapabilityUnknown {
		t.Fatalf("non-probe response should not change capability; got %v", got)
	}
}

// TestRecordKittyTimeout_FromUnknownMarksUnsupported verifies the
// timeout tick resolves Unknown → Unsupported.
func TestRecordKittyTimeout_FromUnknownMarksUnsupported(t *testing.T) {
	resetKittyCapability(t)
	recordKittyTimeout()
	if got := KittySupported(); got != KittyCapabilityUnsupported {
		t.Fatalf("after timeout: KittySupported = %v, want Unsupported", got)
	}
}

// TestRecordKittyTimeout_DoesNotOverwriteSupported verifies a late
// timeout after a successful probe response is a no-op (CompareAndSwap
// from Unknown fails).
func TestRecordKittyTimeout_DoesNotOverwriteSupported(t *testing.T) {
	resetKittyCapability(t)
	recordKittyResponse(uv.KittyGraphicsEvent{Options: kitty.Options{ID: kittyProbeID}})
	recordKittyTimeout()
	if got := KittySupported(); got != KittyCapabilitySupported {
		t.Fatalf("late timeout should not overwrite Supported; got %v", got)
	}
}

// TestRecordKittyResponse_DoesNotOverwriteUnsupported verifies a
// late-arriving response after the probe timed out is a no-op
// (CompareAndSwap from Unknown fails).
func TestRecordKittyResponse_DoesNotOverwriteUnsupported(t *testing.T) {
	resetKittyCapability(t)
	recordKittyTimeout()
	recordKittyResponse(uv.KittyGraphicsEvent{Options: kitty.Options{ID: kittyProbeID}})
	if got := KittySupported(); got != KittyCapabilityUnsupported {
		t.Fatalf("late response should not overwrite Unsupported; got %v", got)
	}
}

// TestModel_Toggle_NoOpWhenKittyUnsupported verifies the gating: with
// the capability forced Unsupported, Toggle in Glyph mode does NOT
// switch to Kitty (which would emit garbage to a non-Kitty terminal).
func TestModel_Toggle_NoOpWhenKittyUnsupported(t *testing.T) {
	resetKittyCapability(t)
	ForceKittyCapability(KittyCapabilityUnsupported)
	t.Cleanup(func() { resetKittyCapability(t) })

	m := New()
	if m.Mode() != PictureGlyph {
		t.Fatal("precondition: New should default to Glyph mode")
	}
	if cmd := m.Toggle(); cmd != nil {
		t.Fatalf("Toggle when Kitty unsupported should return nil Cmd, got %v", cmd)
	}
	if m.Mode() != PictureGlyph {
		t.Fatalf("Toggle should not enter Kitty when unsupported; mode = %v", m.Mode())
	}
}

// TestModel_Toggle_AllowedWhenKittySupported verifies Toggle still
// works when capability is Supported.
func TestModel_Toggle_AllowedWhenKittySupported(t *testing.T) {
	resetKittyCapability(t)
	ForceKittyCapability(KittyCapabilitySupported)
	t.Cleanup(func() { resetKittyCapability(t) })

	m := New()
	m.Toggle()
	if m.Mode() != PictureKitty {
		t.Fatalf("Toggle should enter Kitty when supported; mode = %v", m.Mode())
	}
}

// TestModel_Toggle_AllowedWhenKittyUnknown verifies Toggle is permissive
// in the Unknown state — we'd rather attempt Kitty during the brief
// probe window than reject pre-resolution Toggle attempts on real
// Kitty terminals.
func TestModel_Toggle_AllowedWhenKittyUnknown(t *testing.T) {
	resetKittyCapability(t)
	t.Cleanup(func() { resetKittyCapability(t) })

	m := New()
	if got := KittySupported(); got != KittyCapabilityUnknown {
		t.Fatalf("precondition: should start Unknown, got %v", got)
	}
	m.Toggle()
	if m.Mode() != PictureKitty {
		t.Fatalf("Toggle should attempt Kitty when capability is Unknown; mode = %v", m.Mode())
	}
}

// TestModel_KittySupported_DelegatesToPackage verifies the Model method
// reads from the package-level state.
func TestModel_KittySupported_DelegatesToPackage(t *testing.T) {
	resetKittyCapability(t)
	t.Cleanup(func() { resetKittyCapability(t) })

	m := New()
	ForceKittyCapability(KittyCapabilitySupported)
	if got := m.KittySupported(); got != KittyCapabilitySupported {
		t.Fatalf("Model.KittySupported = %v, want Supported", got)
	}
}

// TestIsPictureMsg_IncludesKittyGraphicsEvent verifies the helper
// matches the Kitty query response so consumers gating message
// forwarding on it route the response into Update.
func TestIsPictureMsg_IncludesKittyGraphicsEvent(t *testing.T) {
	if !IsPictureMsg(uv.KittyGraphicsEvent{}) {
		t.Error("expected IsPictureMsg to recognize uv.KittyGraphicsEvent")
	}
}

// TestModel_Update_KittyGraphicsEventRecordsSupport verifies routing a
// probe-ID response through Update.recordKittyResponse() resolves the
// capability — the end-to-end integration path consumers depend on.
func TestModel_Update_KittyGraphicsEventRecordsSupport(t *testing.T) {
	resetKittyCapability(t)
	t.Cleanup(func() { resetKittyCapability(t) })

	m := New()
	m.Update(uv.KittyGraphicsEvent{Options: kitty.Options{ID: kittyProbeID}})
	if got := KittySupported(); got != KittyCapabilitySupported {
		t.Fatalf("after Update routes response: KittySupported = %v, want Supported", got)
	}
}

// TestModel_Update_KittyProbeTickMarksUnsupported verifies routing the
// timeout tick through Update resolves Unknown → Unsupported.
func TestModel_Update_KittyProbeTickMarksUnsupported(t *testing.T) {
	resetKittyCapability(t)
	t.Cleanup(func() { resetKittyCapability(t) })

	m := New()
	m.Update(kittyProbeTickMsg{})
	if got := KittySupported(); got != KittyCapabilityUnsupported {
		t.Fatalf("after Update routes tick: KittySupported = %v, want Unsupported", got)
	}
}

// Sanity helper: silences unused-import warnings if a test file edit
// removes the only reference to color.
var _ = color.Transparent
