package main

import (
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
	for _, size := range [][2]int{{24, 6}, {60, 16}, {99, 24}, {100, 24}, {160, 48}} {
		for _, full := range []bool{false, true} {
			m, _ := testModel(t)
			m.width, m.height = size[0], size[1]
			m.fullscreen = full
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
