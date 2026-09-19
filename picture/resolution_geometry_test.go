package picture

import (
	"fmt"
	"image/color"
	"strings"
	"testing"
)

func TestKittyScaledAnimationGeometry(t *testing.T) {
	previous := KittySupported()
	ForceKittyCapability(KittyCapabilitySupported)
	t.Cleanup(func() { ForceKittyCapability(previous) })
	for _, factor := range []float64{1, 0.5, 0.3, 0.01} {
		t.Run(fmt.Sprint(factor), func(t *testing.T) {
			m := NewWithConfig(Config{KittyResolutionFactor: factor})
			m.SetSize(4, 3)
			m.Toggle()
			img := smallImage(color.RGBA{R: 180, A: 255})
			first := m.SetImage(img)().(KittyFrameMsg)
			commitFrame(t, &m, first)
			oldGrid := m.View().Content
			next := m.SetImage(img)().(KittyFrameMsg)
			if strings.Contains(next.APC, kittyDeletePrefix) {
				t.Fatal("unchanged scaled frame deletes the visible image")
			}
			if m.View().Content != oldGrid {
				t.Fatal("placeholder grid disappeared during encoding")
			}
			commitFrame(t, &m, next)
			resized := m.SetSize(5, 3)().(KittyFrameMsg)
			if !strings.HasPrefix(resized.APC, kittyDeletePrefix) {
				t.Fatal("resize needs placement reset")
			}
		})
	}
}

func TestKittyResolutionChangeResetsOnlyOnce(t *testing.T) {
	previous := KittySupported()
	ForceKittyCapability(KittyCapabilitySupported)
	t.Cleanup(func() { ForceKittyCapability(previous) })
	m := New()
	m.SetSize(4, 3)
	m.Toggle()
	img := smallImage(color.RGBA{B: 180, A: 255})
	commitFrame(t, &m, m.SetImage(img)().(KittyFrameMsg))
	changed := m.SetKittyResolutionFactor(0.5)().(KittyFrameMsg)
	if !strings.HasPrefix(changed.APC, kittyDeletePrefix) {
		t.Fatal("resolution change needs reset")
	}
	commitFrame(t, &m, changed)
	next := m.SetImage(img)().(KittyFrameMsg)
	if strings.Contains(next.APC, kittyDeletePrefix) {
		t.Fatal("unchanged resolution keeps deleting image")
	}
}
