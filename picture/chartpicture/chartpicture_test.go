package chartpicture

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/go-analyze/charts"
)

// TestInit_DispatchesCellSizeRequest verifies the chartpicture wrapper's Init
// bubbles the picture-layer cell-size request so consumers can call a single
// Init Cmd and have terminal-reported cell dims auto-applied for Kitty
// placement.
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

func TestNewDefaults(t *testing.T) {
	m := New()
	w, h := m.pixelSize() // before any size, returns floor
	if w < 200 || h < 120 {
		t.Errorf("pixelSize floor = %dx%d, want at least 200x120", w, h)
	}
	if m.cellW != 10 || m.cellH != 20 {
		t.Errorf("cell size = %dx%d, want 10x20", m.cellW, m.cellH)
	}
}

func TestNewWithConfigOverrides(t *testing.T) {
	m := NewWithConfig(Config{CellWidthPx: 8, CellHeightPx: 16, Theme: "dark"})
	if m.cellW != 8 || m.cellH != 16 {
		t.Errorf("cell size = %dx%d, want 8x16", m.cellW, m.cellH)
	}
	if m.theme != "dark" {
		t.Errorf("theme = %q, want dark", m.theme)
	}
}

// TestSetSize_NegativeClampedToZero verifies SetSize never lets negative dims
// reach the stored cols/rows. 0 is honored as "no size yet" by renderCmd; the
// clamp at the boundary keeps consumers from defending upstream of the call.
func TestSetSize_NegativeClampedToZero(t *testing.T) {
	m := New()
	m.SetSize(-5, -10)
	if m.cols != 0 || m.rows != 0 {
		t.Errorf("expected SetSize(-5,-10) to clamp to (0,0), got (%d,%d)", m.cols, m.rows)
	}
	if cmd := m.SetSize(0, 0); cmd != nil {
		t.Errorf("SetSize(0,0) after clamped SetSize(-5,-10) should be no-op, got non-nil Cmd")
	}
}

func TestSetLineChartOptionDeferredUntilSize(t *testing.T) {
	m := New()
	cmd := m.SetLineChartOption(charts.NewLineChartOptionWithData([][]float64{{1, 2, 3}}))
	if cmd != nil {
		t.Fatal("setter returned a Cmd before SetSize; expected nil")
	}
	if m.recipe == nil {
		t.Fatal("recipe was not stored")
	}
}

func TestSetSizeReturnsRenderCmdWhenRecipeSet(t *testing.T) {
	m := New()
	m.SetLineChartOption(charts.NewLineChartOptionWithData([][]float64{{1, 2}}))
	cmd := m.SetSize(20, 10)
	if cmd == nil {
		t.Fatal("SetSize returned nil; expected render Cmd")
	}
}

func TestStaleFrameDropped(t *testing.T) {
	m := New()
	m.SetSize(40, 12)
	cmd1 := m.SetLineChartOption(charts.NewLineChartOptionWithData([][]float64{{1, 2}}))
	// Simulate a second mutation before the first frame lands.
	m.SetSize(60, 20) // bumps seq
	stale := cmd1().(chartRenderedMsg)
	if stale.seq == m.seq {
		t.Fatal("seq should differ after second mutation")
	}
	out := m.Update(stale)
	if out != nil {
		t.Fatalf("stale msg should be ignored, got Cmd %v", out)
	}
}

func TestUpdateIgnoresRenderedMsgFromOtherModel(t *testing.T) {
	left := New()
	right := New()
	left.SetSize(40, 12)
	right.SetSize(40, 12)

	cmd := left.SetEChartsJSON("not json")
	if cmd == nil {
		t.Fatal("SetEChartsJSON should return a render Cmd")
	}
	if rightCmd := right.SetLineChartOption(charts.NewLineChartOptionWithData([][]float64{{1, 2}})); rightCmd == nil {
		t.Fatal("right SetLineChartOption should return a render Cmd")
	}

	msg := cmd()
	rendered, ok := msg.(chartRenderedMsg)
	if !ok {
		t.Fatalf("expected chartRenderedMsg, got %T", msg)
	}
	if rendered.err == nil {
		t.Fatal("precondition: left render should fail")
	}
	if rendered.seq != right.seq {
		t.Fatalf("precondition: seq mismatch, got left %d right %d", rendered.seq, right.seq)
	}

	if out := right.Update(rendered); out != nil {
		t.Fatalf("message from another model should be ignored, got Cmd %v", out)
	}
	if err := right.Err(); err != nil {
		t.Fatalf("right model should not receive foreign error, got %v", err)
	}
}

func TestRenderErrorSurfacedViaErrAndView(t *testing.T) {
	m := New()
	m.SetSize(40, 12)
	cmd := m.SetEChartsJSON("not json")
	msg := cmd().(chartRenderedMsg)
	if msg.err == nil {
		t.Fatal("expected err in chartRenderedMsg")
	}
	m.Update(msg)
	if m.Err() == nil {
		t.Fatal("Err() returned nil after error msg")
	}
	if !strings.Contains(m.View().Content, "Chart error") {
		t.Errorf("View() = %q; want it to contain \"Chart error\"", m.View().Content)
	}
}

func TestSetLineChartOptionRendersAfterSize(t *testing.T) {
	m := New()
	if cmd := m.SetSize(40, 12); cmd != nil {
		// SetSize before any recipe should not produce a Cmd.
		_ = cmd
	}
	cmd := m.SetLineChartOption(charts.NewLineChartOptionWithData([][]float64{{1, 2, 3, 4, 5}}))
	if cmd == nil {
		t.Fatal("setter returned nil after SetSize")
	}
	msg := cmd()
	rendered, ok := msg.(chartRenderedMsg)
	if !ok {
		t.Fatalf("msg type = %T, want chartRenderedMsg", msg)
	}
	if rendered.err != nil {
		t.Fatalf("render error: %v", rendered.err)
	}
	if rendered.img == nil {
		t.Fatal("rendered img is nil")
	}
	if rendered.seq != m.seq {
		t.Errorf("rendered.seq = %d, m.seq = %d", rendered.seq, m.seq)
	}
}

func TestModel_Fit_FromConfig(t *testing.T) {
	m := NewWithConfig(Config{Fit: picture.FitFill})
	if got := m.Fit(); got != picture.FitFill {
		t.Fatalf("Fit from Config not honored: got %v want FitFill", got)
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
