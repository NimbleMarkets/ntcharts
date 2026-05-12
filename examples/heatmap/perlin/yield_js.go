//go:build js

package main

import "time"

// yieldToJS gives the JavaScript event loop a slice between columns of
// the Perlin noise sample. SampleNoise is called from Update on every
// stopwatch tick AND on every resize; with a wide viewport (e.g. 200×60
// = 12000 noise samples) the cumulative cost stalls JS unless we
// periodically yield. 1ms is the minimum reliably-effective Sleep on
// Go WASM (sub-ms rounds down to 0).
func yieldToJS() { time.Sleep(time.Millisecond) }
