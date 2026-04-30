package heatpicture

import (
	"image"
	"time"
)

// heatRenderedMsg carries the result of an asynchronous high-res heatmap
// sampling job for a specific generation of the Model. Update() ignores
// frames whose modelID (a per-Model atomic counter) does not match the
// receiving Model's, or whose seq does not match the Model's current
// sampling generation. modelID prevents cross-talk when multiple heatpicture
// Models share a tea.Program; seq prevents stale frames from a parameter
// change that has already been superseded.
//
// duration is the time taken inside the render closure to sample and
// build the image — not the end-to-end scheduling latency. The Model
// stores it on every msg arrival (matched or stale) so consumers can
// surface a per-frame cost meter even while seq is churning.
type heatRenderedMsg struct {
	modelID  uint64
	seq      uint64
	img      image.Image
	err      error
	duration time.Duration
}
