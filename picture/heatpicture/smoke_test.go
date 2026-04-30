package heatpicture

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// TestSmoke_GlyphView_RendersNonEmpty drives a heatpicture.Model through
// the whole lifecycle (Init → SetSize → SetColorScale → SetSampler →
// heatRenderedMsg) and asserts that View().Content is non-empty in default
// (Glyph) mode. This is the path the perlin demo takes; if it fails, the
// demo will show an empty terminal too.
func TestSmoke_GlyphView_RendersNonEmpty(t *testing.T) {
	scale := []color.Color{
		lipgloss.Color("#000000"),
		lipgloss.Color("#ff0000"),
		lipgloss.Color("#ffffff"),
	}
	m := New()
	m.SetValueRange(-1, 1)
	m.SetXYRange(0, 1, 0, 1)
	m.SetColorScale(scale)
	if cmd := m.SetSampler(constSampler(0.5)); cmd != nil {
		t.Fatalf("SetSampler before SetSize should be no-op (no size yet), got non-nil Cmd: ran=%v", cmd != nil)
	}

	// Simulate the WindowSizeMsg path: 80x20 cells.
	cmd := m.SetSize(80, 20)
	if cmd == nil {
		t.Fatal("SetSize with sampler+scale set should return a render Cmd")
	}
	msg, ok := cmd().(heatRenderedMsg)
	if !ok {
		t.Fatalf("expected heatRenderedMsg from SetSize Cmd, got %T", cmd())
	}
	if msg.err != nil {
		t.Fatalf("render produced error: %v", msg.err)
	}
	if msg.img == nil {
		t.Fatal("render produced nil image")
	}

	// Route the msg through Update like a real tea.Program would.
	if c := m.Update(msg); c != nil {
		// Glyph mode returns nil from pic.SetImage; we accept either path.
		_ = c
	}

	view := m.View()
	if view.Content == "" {
		t.Fatal("View().Content empty after full pipeline; demo would show blank terminal")
	}
	// At least one ANSI escape should be present (color-styled half-blocks).
	if !strings.Contains(view.Content, "\x1b[") {
		t.Errorf("View().Content has no ANSI escapes; expected styled half-blocks. First 200 chars: %q",
			view.Content[:min(200, len(view.Content))])
	}
}

// TestSmoke_PerlinDemoFlow mimics the heatpicture/perlin demo more closely:
// Init returns a Cmd; SetSampler before SetSize returns nil; SetSize after
// produces the first frame. Confirms the demo's startup ordering doesn't
// have a hidden race with the no-sampler-yet branch.
func TestSmoke_PerlinDemoFlow(t *testing.T) {
	scale := []color.Color{
		lipgloss.Color("#000000"),
		lipgloss.Color("#ffffff"),
	}
	m := New()
	m.SetValueRange(-1, 1)
	m.SetXYRange(0, 1, 0, 1)
	m.SetColorScale(scale)

	// Init flow: hp.Init returns CSI 16 t cmd; SetSampler returns nil
	// because no size yet; we ignore Init and SetSampler's Cmds.
	_ = m.Init()
	if cmd := m.SetSampler(constSampler(0.0)); cmd != nil {
		t.Fatal("SetSampler before SetSize should be no-op")
	}

	// Now the simulated WindowSizeMsg arrives.
	cmd := m.SetSize(40, 10)
	if cmd == nil {
		t.Fatal("SetSize should produce a render Cmd")
	}

	frame := cmd().(heatRenderedMsg)
	if frame.img == nil {
		t.Fatal("nil image in frame")
	}
	if cmd2 := m.Update(frame); cmd2 != nil {
		// Glyph mode: pic.SetImage returns nil. Accept either.
		_ = cmd2
	}
	if m.View().Content == "" {
		t.Fatal("post-pipeline View is empty")
	}
}

// Verify the smoke test invariants are visible to vet too (no surprise
// nil panics).
var _ = tea.WindowSizeMsg{}
