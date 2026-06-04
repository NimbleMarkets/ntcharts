package picture

import (
	"image"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTmuxPassthrough(t *testing.T) {
	// Ensure we restore state after test
	defer SetTmuxPassthrough(false)

	// 1. Test default value corresponds to TMUX environment presence
	expectedDefault := os.Getenv("TMUX") != ""
	if tmuxPassthroughEnabled.Load() != expectedDefault {
		t.Errorf("expected tmux passthrough to be %v by default; got %v", expectedDefault, tmuxPassthroughEnabled.Load())
	}

	testSeq := "\x1b_Ga=d,d=I,i=123,q=2\x1b\\"
	if got := tmuxWrap(testSeq); got != testSeq {
		t.Errorf("expected no-op when disabled; got %q, want %q", got, testSeq)
	}

	// 2. Test enabled
	SetTmuxPassthrough(true)
	if !tmuxPassthroughEnabled.Load() {
		t.Error("expected tmux passthrough to be enabled")
	}

	wrapped := tmuxWrap(testSeq)
	expected := ansi.TmuxPassthrough(testSeq)
	if wrapped != expected {
		t.Errorf("expected tmuxWrap to match ansi.TmuxPassthrough when enabled; got %q, want %q", wrapped, expected)
	}

	// Ensure ESC characters are doubled
	if !strings.HasPrefix(wrapped, "\x1bPtmux;\x1b") {
		t.Errorf("expected wrapped sequence to start with tmux passthrough prefix; got %q", wrapped)
	}
	if !strings.HasSuffix(wrapped, "\x1b\\") {
		t.Errorf("expected wrapped sequence to end with ST; got %q", wrapped)
	}
	// The original sequence has 2 ESC characters. In the wrapped sequence, inside the tmux passthrough block
	// they should be doubled. So total ESC characters should be:
	// 1 (prefix ESC) + 2 (first ESC doubled) + 2 (second ESC doubled) + 1 (suffix ESC) = 6.
	escCount := strings.Count(wrapped, "\x1b")
	if escCount != 6 {
		t.Errorf("expected 6 ESC characters in wrapped sequence, got %d (wrapped sequence: %q)", escCount, wrapped)
	}

	// 3. Test buildKittyAPC wrapped
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	apc := buildKittyAPC(img, 45, 2, 2)
	if !strings.HasPrefix(apc, "\x1bPtmux;\x1b") {
		t.Errorf("expected buildKittyAPC output to be wrapped in tmux passthrough; got %q", apc)
	}

	// 4. Test kittyDeleteImage wrapped
	del := kittyDeleteImage(45)
	if !strings.HasPrefix(del, "\x1bPtmux;\x1b") {
		t.Errorf("expected kittyDeleteImage output to be wrapped in tmux passthrough; got %q", del)
	}

	// 5. Test kittyEnvSignal with tmux passthrough
	SetTmuxPassthrough(false)
	clearKittyEnv(t)
	if kittyEnvSignal() {
		t.Errorf("expected kittyEnvSignal to return false with empty env; vars: KITTY_WINDOW_ID=%q, KITTY_INSTALLATION_DIR=%q, GHOSTTY_RESOURCES_DIR=%q, WEZTERM_EXECUTABLE=%q, WEZTERM_PANE=%q, TERM=%q, TERM_PROGRAM=%q",
			os.Getenv("KITTY_WINDOW_ID"), os.Getenv("KITTY_INSTALLATION_DIR"),
			os.Getenv("GHOSTTY_RESOURCES_DIR"), os.Getenv("WEZTERM_EXECUTABLE"),
			os.Getenv("WEZTERM_PANE"), os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"))
	}
	SetTmuxPassthrough(true)
	if !kittyEnvSignal() {
		t.Error("expected kittyEnvSignal to return true when tmux passthrough is enabled")
	}

	// 6. Test restore
	SetTmuxPassthrough(false)
	if got := tmuxWrap(testSeq); got != testSeq {
		t.Errorf("expected restore to disable wrapping; got %q, want %q", got, testSeq)
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	// Preserve and restore cap and passthrough after testing
	origPassthrough := tmuxPassthroughEnabled.Load()
	origKittyCap := KittySupported()
	defer func() {
		tmuxPassthroughEnabled.Store(origPassthrough)
		ForceKittyCapability(origKittyCap)
	}()

	tests := []struct {
		envName             string
		envVal              string
		expectPassthrough   bool
		expectKittyCap      KittyCapability
		checkKittyCap       bool
	}{
		{"NTCHARTS_TMUX_PASSTHROUGH", "true", true, KittyCapabilityUnknown, false},
		{"NTCHARTS_TMUX_PASSTHROUGH", "1", true, KittyCapabilityUnknown, false},
		{"NTCHARTS_TMUX_PASSTHROUGH", "false", false, KittyCapabilityUnknown, false},
		{"NTCHARTS_TMUX_PASSTHROUGH", "0", false, KittyCapabilityUnknown, false},
		{"NTCHARTS_KITTY", "supported", false, KittyCapabilitySupported, true},
		{"NTCHARTS_KITTY", "unsupported", false, KittyCapabilityUnsupported, true},
	}

	for _, tc := range tests {
		t.Run(tc.envName+"="+tc.envVal, func(t *testing.T) {
			// Clear checked variables
			t.Setenv("NTCHARTS_TMUX_PASSTHROUGH", "")
			t.Setenv("NTCHARTS_KITTY", "")
			t.Setenv("TMUX", "")
			ForceKittyCapability(KittyCapabilityUnknown)

			t.Setenv(tc.envName, tc.envVal)
			parseEnvOverrides()

			if !tc.checkKittyCap {
				if tmuxPassthroughEnabled.Load() != tc.expectPassthrough {
					t.Errorf("expected passthrough to be %v; got %v", tc.expectPassthrough, tmuxPassthroughEnabled.Load())
				}
			} else {
				if KittySupported() != tc.expectKittyCap {
					t.Errorf("expected kitty capability to be %v; got %v", tc.expectKittyCap, KittySupported())
				}
			}
		})
	}
}

