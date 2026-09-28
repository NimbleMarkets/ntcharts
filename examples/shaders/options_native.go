//go:build !js

package main

import tea "charm.land/bubbletea/v2"

// programOptions returns the Bubble Tea options for this platform. Native
// builds keep the default SIGINT/SIGTERM handling.
func programOptions() []tea.ProgramOption { return nil }
