package canvas

// File contains methods and objects used during canvas Model Update()
// to modify internal state.
// canvas Model is able to move the viewport displaying the contents
// either up, down, left and right

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
}

// DefaultKeyMap returns a default KeyMap for canvas.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "move right"),
		),
	}
}

// UpdateHandler callback invoked during an Update()
// and passes in the canvas *Model and bubbletea Msg.
type UpdateHandler func(*Model, tea.Msg)

// DefaultUpdateHandler is used by canvas chart to enable
// moving viewing window using the mouse wheel,
// holding down mouse left button and moving,
// and with the arrow keys.
// Uses canvas Keymap for keyboard messages.
func DefaultUpdateHandler() UpdateHandler {
	var lastPos Point // tracks zone position of last zone mouse position
	return func(m *Model, tm tea.Msg) {
		switch msg := tm.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.KeyMap.Up):
				m.MoveUp()
			case key.Matches(msg, m.KeyMap.Down):
				m.MoveDown()
			case key.Matches(msg, m.KeyMap.Left):
				m.MoveLeft()
			case key.Matches(msg, m.KeyMap.Right):
				m.MoveRight()
			}
		case tea.MouseMsg:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.MoveUp()
			case tea.MouseButtonWheelDown:
				m.MoveDown()
			case tea.MouseButtonWheelRight:
				m.MoveRight()
			case tea.MouseButtonWheelLeft:
				m.MoveLeft()
			}

			if m.zoneManager == nil {
				return
			}
			switch msg.Action {
			case tea.MouseActionPress:
				zInfo := m.zoneManager.Get(m.zoneID)
				if zInfo.InBounds(msg) {
					x, y := zInfo.Pos(msg)
					lastPos = Point{X: x, Y: y} // set position of last click
				}
			case tea.MouseActionMotion: // event occurs when mouse is pressed
				zInfo := m.zoneManager.Get(m.zoneID)
				if zInfo.InBounds(msg) {
					x, y := zInfo.Pos(msg)
					if x > lastPos.X {
						m.MoveRight()
					} else if x < lastPos.X {
						m.MoveLeft()
					}
					if y > lastPos.Y {
						m.MoveDown()
					} else if y < lastPos.Y {
						m.MoveUp()
					}
					lastPos = Point{X: x, Y: y} // update last mouse position
				}
			}
		}
	}
}

// MoveUp moves cursor up if possible.
func (m *Model) MoveUp() {
	if m.cursor.Y > 0 {
		m.cursor.Y -= 1
	}
}

// MoveDown moves cursor down if possible.
func (m *Model) MoveDown() {
	endY := m.cursor.Y + m.ViewHeight
	if endY < m.area.Dy() {
		m.cursor.Y += 1
	}
}

// MoveLeft moves cursor left if possible.
func (m *Model) MoveLeft() {
	if m.cursor.X > 0 {
		m.cursor.X -= 1
	}
}

// MoveRight moves cursor right if possible.
func (m *Model) MoveRight() {
	endX := m.cursor.X + m.ViewWidth
	if endX < m.area.Dx() {
		m.cursor.X += 1
	}
}
