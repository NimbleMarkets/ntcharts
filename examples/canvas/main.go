package main

import (
	"fmt"
	"os"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var highlightStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("1")). // red
	Background(lipgloss.Color("2"))  // green

var c1Style = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var c2Style = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("14")). // cyan
	Foreground(lipgloss.Color("212")).      // pink
	Background(lipgloss.Color("227"))       // yellow

type model struct {
	c1 canvas.Model
	c2 canvas.Model
	zM *zone.Manager
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.c1.Focused() {
				m.c1.Blur()
				m.c2.Focus()
			} else {
				m.c1.Focus()
				m.c2.Blur()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			if m.zM.Get(m.c1.GetZoneID()).InBounds(msg) { // switch to canvas 1 if clicked on it
				m.c2.Blur()
				m.c1.Focus()
			} else if m.zM.Get(m.c2.GetZoneID()).InBounds(msg) { // switch to canvas 2 if clicked on it
				m.c1.Blur()
				m.c2.Focus()
			} else {
				m.c1.Blur()
				m.c2.Blur()
			}
		}
	}

	if m.c1.Focused() {
		m.c1, _ = m.c1.Update(msg)
	} else {
		m.c2, _ = m.c2.Update(msg)
	}

	return m, nil
}

func (m model) View() string {
	s := "left click to select canvas, or `enter` to toggle between canvas\n"
	s += "click and drag, mouse wheel scroll or arrow keys to move viewport around\n"
	s += "`q/ctrl+c` to quit\n"
	s += lipgloss.JoinHorizontal(lipgloss.Top,
		c1Style.Render(m.c1.View()),
		c2Style.Render(m.c2.View())) + "\n"
	return m.zM.Scan(s) // call zone Manager.Scan() at root model
}

func getExampleCanvas1() (c canvas.Model) {
	c1 := canvas.New(20, 10)

	// set all contents at once with []string
	c1.SetLines([]string{
		"   ", // line 0 - missing characters will be displayed as ' ' up to width of Canvas
		" ██████████████████ ", // line 1
		" █                █ ", // line 2
		" █ ██████████████ █ ", // line 3
		" █ █                ", // line 4
		"                █ █ ", // line 5
		" █ ██████████████ █ ", // line 6
		" █                █ ", // line 7
		" ██████████████████ ", // line 8
		// line 9 - missing line will be displayed as ' ' up to width of Canvas
	})

	// Canvas coordinate system uses (0,0) as top left of Canvas
	// set runes in line 4 using string starting at (X,Y) coordinates (7,4) (\u2588 is █)
	c1.SetString(canvas.Point{7, 4}, " IMBLE   \u2588 █ this will be dropped")

	// set runes in line 5 using []rune starting at (X,Y) coordinates (1,5)
	// (\u2588 and 0x2588 are both █)
	c1.SetRunes(canvas.Point{1, 5}, []rune{'\u2588', ' ', 0x2588, ' ', ' ', ' ', 'M', 'A', 'R', 'K', 'E', 'T', 'S'})

	// set specific Cell at coordinates (7, 4)
	c1.SetCell(canvas.Point{7, 4}, canvas.NewCellWithStyle('N', highlightStyle))

	// set specific Cell styles at coordinates (7, 5)
	c1.SetCellStyle(canvas.Point{7, 5}, highlightStyle) // 'M'

	c1.ViewHeight = 6
	c1.ViewWidth = 12
	return c1
}

func getExampleCanvas2(zm *zone.Manager) (c canvas.Model) {
	// set all contents at once with []string
	contents := []string{
		" ███████████████████  ",
		"  ███████████████████ ",
		"     ███       ███    ",
		"     ███       ███    ",
		"   ███████████████    ",
		"    ██████████████    ",
		"      ██       ███    ",
		"       █       ███    ",
		"                ██    ",
		"                 █    ",
	}

	// creating canvas with options
	c2 := canvas.New(30, 10,
		canvas.WithZoneManager(zm),
		canvas.WithViewHeight(6),
		canvas.WithViewWidth(24),
		canvas.WithLines(contents))
	return c2
}

func main() {
	// Mouse support is enabled using the bubblezone module
	// 1. create a new bubblezone.Manager with bubblezone.New()
	// 2. call canvas.SetZoneManager() on the new bubblezone.Manager
	// 3. create bubbletea.Program with mouse support enabled, e.g. tea.WithMouseCellMotion()
	// 4. bubblezone.Manager.Scan() must be called in bubbletea Model.View()
	//    wrapping the returned string output
	z := zone.New()
	m := model{getExampleCanvas1(), getExampleCanvas2(z), z}
	m.c1.SetZoneManager(z)
	m.c1.Focus()

	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
