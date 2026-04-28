package chartpicture

import (
	"image"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

// chartRenderedMsg carries the result of a rendered chart for a specific
// generation of the Model. Update() ignores frames whose seq does not match
// the Model's current seq (input/size/theme changed since dispatch).
type chartRenderedMsg struct {
	seq uint64
	img image.Image
	err error
}

// IsPictureMsg reports whether msg is an async update owned by chartpicture
// (chart-render completion) or by the embedded picture.Model (Kitty frames).
func IsPictureMsg(msg tea.Msg) bool {
	if _, ok := msg.(chartRenderedMsg); ok {
		return true
	}
	return picture.IsPictureMsg(msg)
}
