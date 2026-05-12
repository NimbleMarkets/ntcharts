//go:build js

package heatmap

import "time"

// yieldToJS gives the JavaScript event loop a slice between batches of
// DrawPoint work in Model.Draw. On Go WASM (GOOS=js), time.Sleep with a
// positive duration schedules a setTimeout that returns control to JS
// until the timer fires; 1ms is the minimum reliably-effective resolution.
//
// Draw runs synchronously on the consumer's Update goroutine, so callers
// won't get new input until Draw returns — yielding here still helps by
// letting background goroutines (fetch resolvers, timer ticks waiting to
// fire) make progress between point batches.
func yieldToJS() { time.Sleep(time.Millisecond) }
