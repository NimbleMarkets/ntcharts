//go:build !js

package main

import tea "charm.land/bubbletea/v2"

// programOptions returns the Bubble Tea options for this platform. Native
// builds keep the default SIGINT/SIGTERM handling.
func programOptions() []tea.ProgramOption { return nil }

// pageQuery is empty natively: options come from the command line.
func pageQuery() string { return "" }

// loadMS is browser-only.
const loadMS = 0.0

// publishReport is nil natively: -report writes the report at exit.
var publishReport func(report)
