package linechart

// File contains methods and objects used during linechart Model Update()
// to modify internal state.
// linechart is able to zoom in and out of the graph,
// and increase and decrease the X and Y values to simulating moving
// the viewport of the linechart

import (
	"github.com/NimbleMarkets/bubbletea-charts/canvas"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// UpdateHandler callback invoked during an Update()
// and passes in the linechart Model and bubbletea Msg.
type UpdateHandler func(*Model, tea.Msg)

// DefaultUpdateHandler is used by linechart to enable
// zooming in and out with the mouse wheels,
// moving the viewing window by holding down mouse button and moving,
// and moving the viewing window with the arrow keys.
// Uses linechart Canvas Keymap for keyboard messages.
func DefaultUpdateHandler() UpdateHandler {
	var lastPos canvas.Point
	return func(m *Model, tm tea.Msg) {
		switch msg := tm.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.Canvas.KeyMap.Up):
				m.MoveUp(1)
			case key.Matches(msg, m.Canvas.KeyMap.Down):
				m.MoveDown(1)
			case key.Matches(msg, m.Canvas.KeyMap.Left):
				m.MoveLeft(1)
			case key.Matches(msg, m.Canvas.KeyMap.Right):
				m.MoveRight(1)
			}
		case tea.MouseMsg:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				// zoom in limited values cannot cross
				m.ZoomIn(1, 1)
			case tea.MouseButtonWheelDown:
				// zoom out limited by max values
				m.ZoomOut(1, 1)
			}

			if m.GetZoneManager() == nil {
				return
			}
			switch msg.Action {
			case tea.MouseActionPress:
				zInfo := m.GetZoneManager().Get(m.GetZoneID())
				if zInfo.InBounds(msg) {
					x, y := zInfo.Pos(msg)
					lastPos = canvas.Point{X: x, Y: y}
				}
			case tea.MouseActionMotion:
				zInfo := m.GetZoneManager().Get(m.GetZoneID())
				if zInfo.InBounds(msg) {
					x, y := zInfo.Pos(msg)
					if x > lastPos.X {
						m.MoveRight(1)
					} else if x < lastPos.X {
						m.MoveLeft(1)
					}
					if y > lastPos.Y {
						m.MoveDown(1)
					} else if y < lastPos.Y {
						m.MoveUp(1)
					}
					lastPos = canvas.Point{X: x, Y: y}
				}
			}
		}
	}
}

// XAxisUpdateHandler is used by linechart to enable
// zooming in and out with the mouse wheels,
// moving the viewing window by holding down mouse button and moving,
// and moving the viewing window with the arrow keys.
// There is only movement along the X axis.
// Uses linechart Canvas Keymap for keyboard messages.
func XAxisUpdateHandler() UpdateHandler {
	var lastPos canvas.Point
	return func(m *Model, tm tea.Msg) {
		switch msg := tm.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.Canvas.KeyMap.Left):
				m.MoveLeft(1)
			case key.Matches(msg, m.Canvas.KeyMap.Right):
				m.MoveRight(1)
			}
		case tea.MouseMsg:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				// zoom in limited values cannot cross
				m.ZoomIn(1, 0)
			case tea.MouseButtonWheelDown:
				// zoom out limited by max values
				m.ZoomOut(1, 0)
			}

			if m.GetZoneManager() == nil {
				return
			}
			switch msg.Action {
			case tea.MouseActionPress:
				zInfo := m.GetZoneManager().Get(m.GetZoneID())
				if zInfo.InBounds(msg) {
					x, y := zInfo.Pos(msg)
					lastPos = canvas.Point{X: x, Y: y}
				}
			case tea.MouseActionMotion:
				zInfo := m.GetZoneManager().Get(m.GetZoneID())
				if zInfo.InBounds(msg) {
					x, y := zInfo.Pos(msg)
					if x > lastPos.X {
						m.MoveRight(1)
					} else if x < lastPos.X {
						m.MoveLeft(1)
					}
					lastPos = canvas.Point{X: x, Y: y}
				}
			}
		}
	}
}

// YAxisUpdateHandler is used by steamlinechart to enable
// zooming in and out with the mouse wheels,
// moving the viewing window by holding down mouse button and moving,
// and moving the viewing window with the arrow keys.
// There is only movement along the Y axis.
// Uses linechart Canvas Keymap for keyboard messages.
func YAxisUpdateHandler() UpdateHandler {
	var lastPos canvas.Point
	return func(m *Model, tm tea.Msg) {
		switch msg := tm.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.Canvas.KeyMap.Up):
				m.MoveUp(1)
			case key.Matches(msg, m.Canvas.KeyMap.Down):
				m.MoveDown(1)
			}
		case tea.MouseMsg:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				// zoom in limited values cannot cross
				m.ZoomIn(0, 1)
			case tea.MouseButtonWheelDown:
				// zoom out limited by max values
				m.ZoomOut(0, 1)
			}
			switch msg.Action {
			case tea.MouseActionPress:
				zInfo := m.GetZoneManager().Get(m.GetZoneID())
				if zInfo.InBounds(msg) {
					x, y := zInfo.Pos(msg)
					lastPos = canvas.Point{X: x, Y: y} // set position of last click
				}
			case tea.MouseActionMotion: // event occurs when mouse is pressed
				zInfo := m.GetZoneManager().Get(m.GetZoneID())
				if zInfo.InBounds(msg) {
					x, y := zInfo.Pos(msg)
					if y > lastPos.Y {
						m.MoveDown(1)
					} else if y < lastPos.Y {
						m.MoveUp(1)
					}
					lastPos = canvas.Point{X: x, Y: y} // update last mouse position
				}
			}
		}
	}
}

// ZoomIn will update display X and Y values to simulate
// zooming into the linechart by given increments.
func (m *Model) ZoomIn(x, y float64) {
	m.SetViewXYRange(
		m.viewMinX+x,
		m.viewMaxX-x,
		m.viewMinY+y,
		m.viewMaxY-y,
	)
}

// ZoomOut will update display X and Y values to simulate
// zooming into the linechart by given increments.
func (m *Model) ZoomOut(x, y float64) {
	m.SetViewXYRange(
		m.viewMinX-x,
		m.viewMaxX+x,
		m.viewMinY-y,
		m.viewMaxY+y,
	)
}

// MoveLeft will update display Y values to simulate
// moving left on the linechart by given increment
func (m *Model) MoveLeft(i float64) {
	if (m.viewMinX - i) >= m.MinX() {
		m.SetViewXRange(m.viewMinX-i, m.viewMaxX-i)
	}
}

// MoveRight will update display Y values to simulate
// moving right on the linechart by given increment.
func (m *Model) MoveRight(i float64) {
	if (m.viewMaxX + i) <= m.MaxX() {
		m.SetViewXRange(m.viewMinX+i, m.viewMaxX+i)
	}
}

// MoveUp will update display X values to simulate
// moving up on the linechart chart by given increment.
func (m *Model) MoveUp(i float64) {
	if (m.viewMaxY + i) <= m.MaxY() {
		m.SetViewYRange(m.viewMinY+i, m.viewMaxY+i)
	}
}

// MoveDown will update display Y values to simulate
// moving down on the linechart chart by given increment.
func (m *Model) MoveDown(i float64) {
	if (m.viewMinY - i) >= m.MinY() {
		m.SetViewYRange(m.viewMinY-i, m.viewMaxY-i)
	}
}
