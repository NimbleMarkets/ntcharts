package picture

import (
	"strings"
	"testing"
)

func TestKittyPlacementDepth(t *testing.T) {
	old := tmuxPassthroughEnabled.Load()
	SetTmuxPassthrough(false)
	defer SetTmuxPassthrough(old)
	for _, format := range []KittyFormat{KittyFormatPNG, KittyFormatRGBA} {
		m := NewWithConfig(Config{KittyZ: -1, KittyFormat: format})
		m.mode = PictureKitty
		m.cols, m.rows = 2, 2
		m.img = randomNRGBA(2, 2)
		f := m.renderCmd()().(KittyFrameMsg)
		if !strings.Contains(f.APC, "z=-1") {
			t.Fatalf("placement depth absent: %.90s", f.APC)
		}
	}
}
