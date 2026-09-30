//go:build js

package main

import (
	"encoding/json"
	"syscall/js"

	tea "charm.land/bubbletea/v2"
)

// programOptions returns the Bubble Tea options for the browser build.
//
// A browser has no POSIX signals, so the handler is useless there, and under
// TinyGo 0.42.0 it is harmful: that runtime's signal_recv returns immediately,
// so the os/signal loop goroutine spins without ever yielding to the browser's
// event loop and the page hangs. TinyGo fixed this after 0.42.0 (PR #5620);
// once the gallery builds with a release that includes it, this option is no
// longer needed but stays harmless.
func programOptions() []tea.ProgramOption { return []tea.ProgramOption{tea.WithoutSignalHandler()} }

// pageQuery returns the page's query string, which carries this build's flags.
func pageQuery() string {
	location := js.Global().Get("location")
	if location.IsUndefined() {
		return ""
	}
	return location.Get("search").String()
}

// loadMS is the page's clock when the program started.
var loadMS = js.Global().Get("performance").Call("now").Float()

// publishReport exposes the report to the page, which has no file system to
// receive -report: the latest one as JSON in globalThis.ntchartsShadersReport,
// and each one in the console.
var publishReport = func(r report) {
	data, err := json.Marshal(r)
	if err != nil {
		return
	}
	js.Global().Set("ntchartsShadersReport", string(data))
	js.Global().Get("console").Call("debug", "ntcharts-shaders-report "+string(data))
}
