//go:build js

package main

import tea "charm.land/bubbletea/v2"

// programOptions returns the Bubble Tea options for the browser build.
//
// A browser has no POSIX signals, so the handler is useless there, and under
// TinyGo 0.42.0 it is harmful: that runtime's signal_recv returns immediately,
// so the os/signal loop goroutine spins without ever yielding to the browser's
// event loop and the page hangs. TinyGo fixed this after 0.42.0 (PR #5620);
// once the gallery builds with a release that includes it, this option is no
// longer needed but stays harmless.
func programOptions() []tea.ProgramOption { return []tea.ProgramOption{tea.WithoutSignalHandler()} }
