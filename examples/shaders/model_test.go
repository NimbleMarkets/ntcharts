package main

import (
	"errors"
	"image"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/charmbracelet/x/ansi"
)

type fakeRenderer struct{ requests []renderRequest }

func (r *fakeRenderer) Render(req renderRequest) (*image.NRGBA, error) {
	r.requests = append(r.requests, req)
	return image.NewNRGBA(image.Rect(0, 0, req.width, req.height)), nil
}
func testModel(t *testing.T) (*model, *fakeRenderer) {
	t.Helper()
	old := picture.KittySupported()
	picture.ForceKittyCapability(picture.KittyCapabilityUnsupported)
	t.Cleanup(func() { picture.ForceKittyCapability(old) })
	r := &fakeRenderer{}
	m := newModel(r, 0, 60, 16, 0)
	return m, r
}
func TestSingleFrameAndPresetCoalescing(t *testing.T) {
	m, r := testModel(t)
	_, first := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if first == nil || !m.busy {
		t.Fatal("first frame not scheduled")
	}
	if m.render() != nil {
		t.Fatal("queued a second render while busy")
	}
	m.selectPreset(4)
	if m.render() != nil {
		t.Fatal("preset change queued concurrent GPU work")
	}
	_, next := m.Update(first())
	if next == nil || len(r.requests) != 1 {
		t.Fatal("stale render did not schedule replacement")
	}
	got := next().(renderedMsg)
	if r.requests[1].preset != 4 || got.epoch != m.epoch {
		t.Fatal("replacement used stale preset")
	}
}
func TestPauseStopsAfterPresentation(t *testing.T) {
	m, _ := testModel(t)
	_, render := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if m.playing {
		t.Fatal("space did not pause")
	}
	_, present := m.Update(render()) // glyph path requires no async encoding
	_, last := m.Update(present())
	// The pause key requests one final frame; it must not start a continuous loop.
	if last != nil {
		_, present = m.Update(last())
		_, last = m.Update(present())
	}
	if last != nil || m.busy {
		t.Fatal("paused viewer kept rendering")
	}
	if _, cmd := m.Update(tickMsg{}); cmd != nil {
		t.Fatal("old tick restarted paused animation")
	}
}
func TestLayoutFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{24, 6}, {60, 16}, {99, 24}, {100, 24}, {109, 24}, {110, 24}, {160, 48}, {240, 60}} {
		for _, mode := range []struct{ full, source bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
			full := mode.full
			m, _ := testModel(t)
			m.width, m.height = size[0], size[1]
			m.fullscreen = full
			m.source = mode.source
			m.geometry()
			// Use an actual glyph image so this also checks picture placement dimensions.
			m.pic.SetImage(image.NewNRGBA(image.Rect(0, 0, m.rasterW, m.rasterH)))
			lines := strings.Split(m.View().Content, "\n")
			if len(lines) > m.height {
				t.Fatalf("%v fullscreen=%v: %d rows", size, full, len(lines))
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > m.width {
					t.Fatalf("%v: line overflows (%d)", size, ansi.StringWidth(line))
				}
			}
		}
	}
}
func TestRasterMatchesPictureAndBudget(t *testing.T) {
	m, _ := testModel(t)
	picture.ForceKittyCapability(picture.KittyCapabilitySupported)
	m.pic.Toggle()
	m.pic.SetCellPixelSize(18, 36)
	for _, size := range [][2]int{{120, 40}, {200, 70}, {350, 100}} {
		m.width, m.height = size[0], size[1]
		m.density = 40
		m.geometry()
		factor := m.pic.KittyResolutionFactor()
		if m.rasterW != m.cols*max(1, int(18*factor)) || m.rasterH != m.rows*max(1, int(36*factor)) {
			t.Fatal("picture will rescale the GPU image")
		}
		if m.rasterW > 2048 || m.rasterH > 1536 || m.rasterW*m.rasterH > 1536*1024 {
			t.Fatal("raster budget exceeded")
		}
	}
}
func TestSlideshowCancellationAndPause(t *testing.T) {
	m, _ := testModel(t)
	m.slideshow = time.Second
	m.playing = false
	m.Update(slideMsg{})
	if m.selected != 0 {
		t.Fatal("paused slideshow changed preset")
	}
	m.playing = true
	m.slideGeneration++
	if _, cmd := m.Update(slideMsg{}); cmd != nil || m.selected != 0 {
		t.Fatal("old slideshow timer accepted")
	}
	m.Update(slideMsg{generation: m.slideGeneration})
	if m.selected != 1 {
		t.Fatal("slideshow did not advance")
	}
}

func TestNewRenderInvalidatesPreviousWake(t *testing.T) {
	m, _ := testModel(t)
	m.width, m.height = 80, 24
	first := m.render()
	_, present := m.Update(first())
	_, wake := m.Update(present())
	if wake == nil {
		t.Fatal("missing next-frame timer")
	}
	old := m.wakeGeneration
	m.dirty = true
	immediate := m.render()
	if immediate == nil {
		t.Fatal("input did not schedule a new frame")
	}
	_, present = m.Update(immediate())
	m.Update(present())
	if _, cmd := m.Update(tickMsg{generation: old}); cmd != nil {
		t.Fatal("stale timer exceeded the frame-rate cap")
	}
}

func TestDensityDoesNotLosePixelToRounding(t *testing.T) {
	m, _ := testModel(t)
	picture.ForceKittyCapability(picture.KittyCapabilitySupported)
	m.pic.Toggle()
	m.pic.SetCellPixelSize(18, 36)
	m.width, m.height = 100, 30
	m.density = 16
	m.geometry()
	if m.rasterW != m.cols*8 || m.rasterH != m.rows*16 {
		t.Fatalf("unexpected raster: %dx%d", m.rasterW, m.rasterH)
	}
}

func TestSlideshowToggleKeepsConfiguredInterval(t *testing.T) {
	r := &fakeRenderer{}
	m := newModel(r, 0, 60, 16, 3*time.Second)
	m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if m.slideshow != 0 {
		t.Fatalf("a did not pause the slideshow: %v", m.slideshow)
	}
	m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if m.slideshow != 3*time.Second {
		t.Fatalf("slideshow resumed at %v, want the configured 3s", m.slideshow)
	}
}

func TestMosaicRectsTileTheRaster(t *testing.T) {
	rects := mosaicRects(101, 63)
	if rects[0].Min != (image.Point{}) {
		t.Fatalf("top-left tile starts at %v", rects[0].Min)
	}
	if rects[3].Max != (image.Point{X: 101, Y: 63}) {
		t.Fatalf("bottom-right tile ends at %v, want the raster corner", rects[3].Max)
	}
	if rects[1].Min.X-rects[0].Max.X != mosaicGutter || rects[2].Min.Y-rects[0].Max.Y != mosaicGutter {
		t.Fatalf("tiles are not separated by the gutter: %v", rects)
	}
	for i, r := range rects {
		if r.Dx() < 1 || r.Dy() < 1 {
			t.Fatalf("tile %d is empty: %v", i, r)
		}
	}
}

func TestComposeMosaicPlacesTiles(t *testing.T) {
	rects := mosaicRects(40, 20)
	var tiles [4]*image.NRGBA
	for i := range tiles {
		tiles[i] = image.NewNRGBA(image.Rect(0, 0, rects[i].Dx(), rects[i].Dy()))
		for j := range tiles[i].Pix {
			tiles[i].Pix[j] = 0xff
		}
		tiles[i].Pix[0] = byte(i + 1) // red channel of the first pixel identifies the tile
	}
	img := composeMosaic(tiles, 40, 20)
	if img.Bounds() != image.Rect(0, 0, 40, 20) {
		t.Fatalf("composed bounds %v", img.Bounds())
	}
	for i, r := range rects {
		if got := img.NRGBAAt(r.Min.X, r.Min.Y).R; got != byte(i+1) {
			t.Fatalf("tile %d landed wrong: red=%d", i, got)
		}
	}
	if g := img.NRGBAAt(rects[0].Max.X, 0); g.R != 0 || g.A != 0xff {
		t.Fatalf("gutter pixel = %v, want opaque dark", g)
	}
}

func TestMosaicRendersFourPresets(t *testing.T) {
	m, r := testModel(t)
	m.width, m.height = 80, 24
	m.selectPreset(4)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if !m.mosaic || cmd == nil {
		t.Fatal("m did not enter mosaic mode and schedule a frame")
	}
	msg := cmd().(renderedMsg)
	if len(r.requests) != 4 {
		t.Fatalf("mosaic issued %d renders, want 4", len(r.requests))
	}
	rects := mosaicRects(m.rasterW, m.rasterH)
	for i, want := range []int{4, 5, 0, 1} {
		req := r.requests[i]
		if req.preset != want || req.width != rects[i].Dx() || req.height != rects[i].Dy() {
			t.Fatalf("request %d = preset %d %dx%d, want preset %d %dx%d", i, req.preset, req.width, req.height, want, rects[i].Dx(), rects[i].Dy())
		}
	}
	if msg.image.Bounds().Dx() != m.rasterW || msg.image.Bounds().Dy() != m.rasterH {
		t.Fatalf("composed image %v, want %dx%d", msg.image.Bounds(), m.rasterW, m.rasterH)
	}
	m.Update(msg)
	m.Update(presentedMsg{})
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	cmd()
	if m.mosaic || len(r.requests) != 5 {
		t.Fatalf("leaving mosaic should render one frame; mosaic=%v requests=%d", m.mosaic, len(r.requests))
	}
}

func TestSourcePaneLayout(t *testing.T) {
	m, _ := testModel(t)
	m.width, m.height = 140, 30
	m.geometry()
	without := m.cols
	if without != 140-sidebarWidth-2 {
		t.Fatalf("image cols without source = %d, want the width less the sidebar and gap", without)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if !m.source || cmd == nil {
		t.Fatal("s did not enable the source pane and schedule a frame")
	}
	// The pane takes about half of the space beside the sidebar.
	if !m.showSource() || m.sourceWidth() < without/2-1 || m.cols != without-m.sourceWidth()-2 {
		t.Fatalf("image cols with source = %d, pane = %d, without = %d", m.cols, m.sourceWidth(), without)
	}
	m.mosaic = true
	m.geometry()
	if m.showSource() || m.cols != without {
		t.Fatalf("source pane should hide in mosaic mode; cols=%d", m.cols)
	}
	m.mosaic = false
	m.width = 109
	m.geometry()
	if m.showSource() {
		t.Fatal("source pane should hide below 110 columns")
	}
	m.width = 110
	m.geometry()
	if !m.showSource() || m.cols < 40 || m.sourceWidth() < 48 {
		t.Fatalf("source pane at 110 columns leaves %d image cols and %d pane cols", m.cols, m.sourceWidth())
	}
	m.width = 240
	m.geometry()
	if m.sourceWidth() > 100 || m.cols < 100 {
		t.Fatalf("very wide terminal: pane %d, image %d; the pane should cap so the image keeps growing", m.sourceWidth(), m.cols)
	}
}

func TestSourceLinesBodyThenPrelude(t *testing.T) {
	m, _ := testModel(t)
	m.selectPreset(4) // julia: 15 body lines; prelude is 30 lines
	lines := sourceLines(m.selected)
	if len(lines) != 15+1+30 {
		t.Fatalf("%d lines, want body + divider + prelude = 46", len(lines))
	}
	if !strings.HasPrefix(lines[0], "fn shade") {
		t.Fatalf("first line should start the body: %q", lines[0])
	}
	if !strings.Contains(lines[15], "common.wgsl") {
		t.Fatalf("divider missing after the body: %q", lines[15])
	}
	if !strings.HasPrefix(lines[16], "struct Uniforms") {
		t.Fatalf("prelude should follow the divider: %q", lines[16])
	}
	for _, l := range lines {
		if strings.Contains(l, "SHADER_BODY") {
			t.Fatal("placeholder marker leaked into the prelude")
		}
	}
}

func TestGPUErrorIsShownInsteadOfBlankScreen(t *testing.T) {
	m, r := testModel(t)
	m.width, m.height = 80, 24
	m.err = errors.New("WebGPU is not available in this browser")
	if m.render() != nil {
		t.Fatal("render scheduled GPU work after a GPU error")
	}
	if !strings.Contains(m.View().Content, "WebGPU is not available") {
		t.Fatalf("view does not show the GPU error:\n%s", m.View().Content)
	}
	if len(r.requests) != 0 {
		t.Fatal("renderer was called despite the error")
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("q should still quit")
	}
}

func TestProgramOptionsSkipSignalsInBrowser(t *testing.T) {
	// In a browser there are no POSIX signals, and TinyGo 0.42.0's os/signal
	// loop spins without yielding, which hangs the page. Native builds keep
	// Bubble Tea's default SIGINT/SIGTERM handling.
	opts := programOptions()
	if inBrowser && len(opts) != 1 {
		t.Fatalf("browser build should pass exactly WithoutSignalHandler, got %d options", len(opts))
	}
	if !inBrowser && len(opts) != 0 {
		t.Fatalf("native build should keep default signal handling, got %d options", len(opts))
	}
}
