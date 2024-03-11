package timeserieslinechart

// File contains methods and objects used during
// timeserieslinechart Model Update() to modify internal state.
// timeserieslinechart is able to zoom in and out of the graph,
// and increase and decrease the X values to simulating moving
// the viewport of the linechart

import (
	"github.com/NimbleMarkets/bubbletea-charts/linechart"
)

// DateUpdateHandler is used by timeserieslinechart to enable
// zooming in and out with the mouse wheels,
// moving the viewing window by holding down mouse button and moving,
// and moving the viewing window with the arrow keys.
// There is only movement along the X axis by day increments.
// Uses linechart Canvas Keymap for keyboard messages.
func DateUpdateHandler() linechart.UpdateHandler {
	const daySeconds = 86400 // number of seconds in a day
	return linechart.XAxisUpdateHandler(daySeconds)
}

// HourUpdateHandler is used by timeserieslinechart to enable
// zooming in and out with the mouse wheels,
// moving the viewing window by holding down mouse button and moving,
// and moving the viewing window with the arrow keys.
// There is only movement along the X axis by hour increments.
// Uses linechart Canvas Keymap for keyboard messages.
func HourUpdateHandler() linechart.UpdateHandler {
	const hourSeconds = 3600 // number of seconds in a hour
	return linechart.XAxisUpdateHandler(hourSeconds)
}

// HourUpdateHandler is used by timeserieslinechart to enable
// zooming in and out with the mouse wheels,
// moving the viewing window by holding down mouse button and moving,
// and moving the viewing window with the arrow keys.
// There is only movement along the X axis by second increments.
// Uses linechart Canvas Keymap for keyboard messages.
func SecondUpdateHandler() linechart.UpdateHandler {
	return linechart.XAxisUpdateHandler(1)
}
