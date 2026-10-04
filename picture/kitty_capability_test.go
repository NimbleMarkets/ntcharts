package picture

import (
	"errors"
	"image/color"
	"os"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi/kitty"
)

// TestMain defaults the package-wide Kitty capability to Supported so
// the rendering-pipeline tests (which Toggle into Kitty as setup) can
// proceed without each one having to opt in. Capability-specific
// tests reset to Unknown via resetKittyCapability(t) for isolation.
func TestMain(m *testing.M) {
	ForceKittyCapability(KittyCapabilitySupported)
	kittySharedCap.Store(int32(KittyCapabilitySupported))
	// Package init enables passthrough when TMUX is set; pin it off so the
	// suite behaves the same inside and outside a tmux session.
	SetTmuxPassthrough(false)
	code := m.Run()
	_ = kittySharedProbe.Swap(nil).Unlink()
	os.Exit(code)
}

// resetKittyCapability sets the package-level state to Unknown for the
// duration of a test, and registers a cleanup that restores it to
// Supported (the test-suite default) when the test ends. This keeps
// capability-specific tests from leaking state into rendering tests
// that follow. The probe's sync.Once isn't reset — these tests don't
// call QueryKittySupport, they manipulate state directly via
// ForceKittyCapability and the package-internal record* helpers.
func resetKittyCapability(t *testing.T) {
	t.Helper()
	ForceKittyCapability(KittyCapabilityUnknown)
	t.Cleanup(func() { ForceKittyCapability(KittyCapabilitySupported) })
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

// TestRecordKittyResponse_OverridesUnsupportedFromTimeout verifies a
// real response is authoritative even if the timeout fired first.
// Bubbletea's eventLoop ordering can deliver the timeout msg before
// the response in the input queue; in that case the response should
// still flip capability to Supported, not be silently dropped.
func TestRecordKittyResponse_OverridesUnsupportedFromTimeout(t *testing.T) {
	resetKittyCapability(t)
	recordKittyTimeout() // sets Unsupported
	recordKittyResponse(uv.KittyGraphicsEvent{Options: kitty.Options{ID: kittyProbeID}})
	if got := KittySupported(); got != KittyCapabilitySupported {
		t.Fatalf("response should override Unsupported set by an earlier timeout; got %v", got)
	}
}

// TestModel_Toggle_NoOpWhenKittyUnsupported verifies the gating: with
// the capability forced Unsupported, Toggle in Glyph mode does NOT
// switch to Kitty (which would emit garbage to a non-Kitty terminal).
func TestModel_Toggle_NoOpWhenKittyUnsupported(t *testing.T) {
	resetKittyCapability(t)
	ForceKittyCapability(KittyCapabilityUnsupported)
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
	m := New()
	m.Toggle()
	if m.Mode() != PictureKitty {
		t.Fatalf("Toggle should enter Kitty when supported; mode = %v", m.Mode())
	}
}

// TestModel_Toggle_BlockedWhenKittyUnknown verifies Toggle is strict
// in the Unknown state — better to require an explicit second toggle
// after the probe resolves than to risk emitting Kitty escapes to a
// non-Kitty terminal during the early-startup probe window.
func TestModel_Toggle_BlockedWhenKittyUnknown(t *testing.T) {
	resetKittyCapability(t)
	m := New()
	if got := KittySupported(); got != KittyCapabilityUnknown {
		t.Fatalf("precondition: should start Unknown, got %v", got)
	}
	if cmd := m.Toggle(); cmd != nil {
		t.Fatalf("Toggle when capability is Unknown should return nil, got %v", cmd)
	}
	if m.Mode() != PictureGlyph {
		t.Fatalf("Toggle should NOT enter Kitty when capability is Unknown; mode = %v", m.Mode())
	}
}

// TestModel_KittySupported_DelegatesToPackage verifies the Model method
// reads from the package-level state.
func TestModel_KittySupported_DelegatesToPackage(t *testing.T) {
	resetKittyCapability(t)
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
	m := New()
	m.Update(kittyProbeTickMsg{})
	if got := KittySupported(); got != KittyCapabilityUnsupported {
		t.Fatalf("after Update routes tick: KittySupported = %v, want Unsupported", got)
	}
}

// TestKittyEnvSignal_RecognizedTerminals verifies kittyEnvSignal
// returns true for each known Kitty-capable terminal indicator. Uses
// t.Setenv (per-test scoped, restored on cleanup) and explicitly
// clears every checked variable first so prior environment state can't
// leak into the assertion.
func TestKittyEnvSignal_RecognizedTerminals(t *testing.T) {
	cases := []struct {
		name   string
		envVar string
		envVal string
	}{
		{"kitty/KITTY_WINDOW_ID", "KITTY_WINDOW_ID", "1"},
		{"kitty/KITTY_INSTALLATION_DIR", "KITTY_INSTALLATION_DIR", "/usr/lib/kitty"},
		{"ghostty/GHOSTTY_RESOURCES_DIR", "GHOSTTY_RESOURCES_DIR", "/usr/share/ghostty"},
		{"wezterm/WEZTERM_EXECUTABLE", "WEZTERM_EXECUTABLE", "/usr/bin/wezterm"},
		{"wezterm/WEZTERM_PANE", "WEZTERM_PANE", "0"},
		{"TERM=xterm-kitty", "TERM", "xterm-kitty"},
		{"TERM=xterm-ghostty", "TERM", "xterm-ghostty"},
		{"TERM_PROGRAM=ghostty", "TERM_PROGRAM", "ghostty"},
		{"TERM_PROGRAM=WezTerm", "TERM_PROGRAM", "WezTerm"},
		{"TERM_PROGRAM=kitty", "TERM_PROGRAM", "kitty"},
		{"TERM_PROGRAM=iTerm.app", "TERM_PROGRAM", "iTerm.app"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearKittyEnv(t)
			t.Setenv(tc.envVar, tc.envVal)
			if !kittyEnvSignal() {
				t.Errorf("expected kittyEnvSignal()=true with %s=%q, got false",
					tc.envVar, tc.envVal)
			}
		})
	}
}

// TestKittyEnvSignal_UnknownTerminalReturnsFalse verifies that with
// every recognized indicator unset, the gate is closed.
func TestKittyEnvSignal_UnknownTerminalReturnsFalse(t *testing.T) {
	clearKittyEnv(t)
	if kittyEnvSignal() {
		t.Error("expected kittyEnvSignal()=false with no recognized env var set")
	}
}

// TestKittyEnvSignal_NonKittyTerminalProgram verifies a non-Kitty
// TERM_PROGRAM (e.g., Apple Terminal.app) is NOT mistaken for Kitty.
func TestKittyEnvSignal_NonKittyTerminalProgram(t *testing.T) {
	clearKittyEnv(t)
	t.Setenv("TERM_PROGRAM", "Apple_Terminal")
	t.Setenv("TERM", "xterm-256color")
	if kittyEnvSignal() {
		t.Error("expected kittyEnvSignal()=false for Apple_Terminal")
	}
}

// clearKittyEnv unsets every env var consulted by kittyEnvSignal so
// tests start from a known baseline. t.Setenv automatically restores
// values on test cleanup.
func clearKittyEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{
		"KITTY_WINDOW_ID",
		"KITTY_INSTALLATION_DIR",
		"GHOSTTY_RESOURCES_DIR",
		"WEZTERM_EXECUTABLE",
		"WEZTERM_PANE",
		"TERM",
		"TERM_PROGRAM",
	} {
		t.Setenv(v, "")
	}
}

// Sanity helper: silences unused-import warnings if a test file edit
// removes the only reference to color.
var _ = color.Transparent

// resetKittySharedCap sets the shared-memory capability to Unknown for a
// test and restores the suite default (Supported) afterwards.
func resetKittySharedCap(t *testing.T) {
	t.Helper()
	kittySharedCap.Store(int32(KittyCapabilityUnknown))
	t.Cleanup(func() { kittySharedCap.Store(int32(KittyCapabilitySupported)) })
}

func sharedCap() KittyCapability { return KittyCapability(kittySharedCap.Load()) }

func sharedProbeReply(payload string) uv.KittyGraphicsEvent {
	return uv.KittyGraphicsEvent{Options: kitty.Options{ID: kittySharedProbeID}, Payload: []byte(payload)}
}

// TestRecordKittyResponse_SharedMemoryProbe verifies only an OK reply to
// the t=s query enables the shared-memory medium.
func TestRecordKittyResponse_SharedMemoryProbe(t *testing.T) {
	for payload, want := range map[string]KittyCapability{
		"OK":                          KittyCapabilitySupported,
		"EBADF:Failed to open shm":    KittyCapabilityUnsupported,
		"EINVAL:Unsupported medium s": KittyCapabilityUnsupported,
	} {
		resetKittySharedCap(t)
		recordKittyResponse(sharedProbeReply(payload))
		if got := sharedCap(); got != want {
			t.Errorf("reply %q: shared-memory capability = %v, want %v", payload, got, want)
		}
	}
}

// TestRecordKittyResponse_KittyProbeLeavesSharedMemoryUnknown verifies the
// t=d query's reply says nothing about t=s.
func TestRecordKittyResponse_KittyProbeLeavesSharedMemoryUnknown(t *testing.T) {
	resetKittySharedCap(t)
	recordKittyResponse(uv.KittyGraphicsEvent{Options: kitty.Options{ID: kittyProbeID}, Payload: []byte("OK")})
	if got := sharedCap(); got != KittyCapabilityUnknown {
		t.Fatalf("shared-memory capability = %v, want Unknown", got)
	}
}

// TestRecordKittyTimeout_SharedMemory verifies an unanswered t=s query
// resolves to Unsupported, and a late OK still wins.
func TestRecordKittyTimeout_SharedMemory(t *testing.T) {
	resetKittySharedCap(t)
	recordKittyTimeout()
	if got := sharedCap(); got != KittyCapabilityUnsupported {
		t.Fatalf("after timeout: shared-memory capability = %v, want Unsupported", got)
	}
	recordKittyResponse(sharedProbeReply("OK"))
	if got := sharedCap(); got != KittyCapabilitySupported {
		t.Fatalf("late OK: shared-memory capability = %v, want Supported", got)
	}
}

func TestKittyUnavailable_Reasons(t *testing.T) {
	resetKittyCapability(t)
	prevEnv := kittyEnvSignalled.Load()
	t.Cleanup(func() { kittyEnvSignalled.Store(prevEnv) })

	cases := []struct {
		name     string
		cap      KittyCapability
		signaled bool
		want     error
	}{
		{"unknown", KittyCapabilityUnknown, false, ErrKittyProbePending},
		{"supported", KittyCapabilitySupported, false, nil},
		{"unsupported, env not signalled", KittyCapabilityUnsupported, false, ErrKittyNotDetected},
		{"unsupported, probe timed out", KittyCapabilityUnsupported, true, ErrKittyProbeTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ForceKittyCapability(tc.cap)
			kittyEnvSignalled.Store(tc.signaled)
			if got := KittyUnavailable(); !errors.Is(got, tc.want) {
				t.Fatalf("KittyUnavailable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestModel_ToggleBlocked(t *testing.T) {
	resetKittyCapability(t)
	m := New()

	ForceKittyCapability(KittyCapabilityUnknown)
	if err := m.ToggleBlocked(); !errors.Is(err, ErrKittyProbePending) {
		t.Fatalf("Glyph mode, probe pending: ToggleBlocked = %v, want ErrKittyProbePending", err)
	}
	m.Toggle()
	if m.Mode() != PictureGlyph {
		t.Fatal("blocked Toggle must stay in Glyph mode")
	}

	ForceKittyCapability(KittyCapabilitySupported)
	if err := m.ToggleBlocked(); err != nil {
		t.Fatalf("Glyph mode, supported: ToggleBlocked = %v, want nil", err)
	}
	m.Toggle()
	if m.Mode() != PictureKitty {
		t.Fatal("Toggle should enter Kitty mode once supported")
	}

	// Leaving Kitty is never blocked, whatever the capability says.
	ForceKittyCapability(KittyCapabilityUnsupported)
	if err := m.ToggleBlocked(); err != nil {
		t.Fatalf("Kitty mode: ToggleBlocked = %v, want nil", err)
	}
}
