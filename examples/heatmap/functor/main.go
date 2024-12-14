// Simple Heatmap Demo
// https://adventofcode.com/2024/day/4
//

package main

import (
	"fmt"
	"math"
	"os"
	"slices"

	"github.com/NimbleMarkets/ntcharts/heatmap"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

///////////////////////////////////////////////////////////////////////////////

type MapFunctor interface {
	Name() string             // Descriptive name
	Detail() string           // Equation description
	Map(x, y float64) float64 // maps z = f(x, y)
}

type Func1 struct{}
type Func2 struct{}
type Func3 struct{}

func (Func1) Name() string   { return "f1" }
func (Func1) Detail() string { return "sin(sqrt(x^2 + y^2))" }
func (Func1) Map(x, y float64) float64 {
	return math.Sin(math.Sqrt(x*x + y*y))
}

func (Func2) Name() string   { return "f2" }
func (Func2) Detail() string { return "sin(x) + cos(y)" }
func (Func2) Map(x, y float64) float64 {
	return math.Sin(x) + math.Cos(y)
}

func (Func3) Name() string   { return "f3" }
func (Func3) Detail() string { return "exp(-(x^2 + y^2)/10)" }
func (Func3) Map(x, y float64) float64 {
	return math.Exp(-(x*x + y*y) / 10)
}

///////////////////////////////////////////////////////////////////////////////

type keymap struct {
	quit key.Binding

	functor key.Binding

	gradient key.Binding
	invert   key.Binding

	zoomIn  key.Binding
	zoomOut key.Binding

	transUp    key.Binding
	transDown  key.Binding
	transLeft  key.Binding
	transRight key.Binding
}

// FullHelp returns bindings to show the full help view.
// Implements bubble's [help.KeyMap] interface.
func (k keymap) FullHelp() [][]key.Binding {
	kb := [][]key.Binding{{
		k.functor,
		k.gradient,
		k.invert,
		k.quit,
		k.zoomIn,
		k.zoomOut,
		k.transUp,
		k.transDown,
		k.transLeft,
		k.transRight,
	}}
	return kb
}

// ShortHelp returns bindings to show in the abbreviated help view. It's part
// of the help.KeyMap interface.
func (k keymap) ShortHelp() []key.Binding {
	kb := []key.Binding{
		k.functor,
		k.gradient,
		k.invert,
		k.quit,
		k.zoomIn,
		k.zoomOut,
		k.transUp,
		k.transDown,
		k.transLeft,
		k.transRight,
	}
	return kb
}

func newKeyMap() keymap {
	return keymap{
		functor: key.NewBinding(
			key.WithKeys(" ", "f"),
			key.WithHelp("<space>", "next function"),
		),
		gradient: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "next gradient"),
		),
		invert: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "invert"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		transUp: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "up"),
		),
		transDown: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "down"),
		),
		transLeft: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "left"),
		),
		transRight: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "right"),
		),
		zoomIn: key.NewBinding(
			key.WithKeys("=", "+"),
			key.WithHelp("+", "in"),
		),
		zoomOut: key.NewBinding(
			key.WithKeys("-", "_"),
			key.WithHelp("-", "out"),
		),
	}
}

///////////////////////////////////////////////////////////////////////////////

var modelFunctors = []MapFunctor{Func1{}, Func2{}, Func3{}}

type Model struct {
	Heatmap heatmap.Model
	Zoom    float64
	OriginX float64
	OriginY float64

	currentFunctor int
	currentColor   int
	colors         [][]lipgloss.Color

	keymap keymap
	help   help.Model
}

func NewModel() Model {
	return Model{
		Heatmap:        heatmap.New(10, 10, heatmap.WithValueRange(0, 1)),
		Zoom:           1.0,
		currentFunctor: 0,
		currentColor:   0,
		colors:         appColorScales,
		keymap:         newKeyMap(),
		help:           help.New(),
	}
}

func (m *Model) SampleFunctor() {
	m.Heatmap.ClearData()
	currentFunctor := modelFunctors[m.currentFunctor]
	for i := 0; i <= m.Heatmap.GraphWidth(); i++ {
		for j := 0; j <= m.Heatmap.GraphHeight(); j++ {
			x, y := m.OriginX+float64(i), m.OriginY+float64(j)
			val := currentFunctor.Map(x*m.Zoom, y*m.Zoom)
			m.Heatmap.Push(heatmap.NewHeatPoint(x, y, val))
		}
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

var moveFactor float64 = 0.5

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Heatmap.Resize(msg.Width, msg.Height)
		m.SampleFunctor()
		m.Heatmap.Draw()
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit

		case key.Matches(msg, m.keymap.functor):
			m.currentFunctor++
			if m.currentFunctor >= len(modelFunctors) {
				m.currentFunctor = 0
			}
			m.SampleFunctor()
			m.Heatmap.Draw()
		case
			key.Matches(msg, m.keymap.gradient):
			m.currentColor += 1
			if m.currentColor >= len(m.colors) {
				m.currentColor = 0
			}
			m.Heatmap.ColorScale = m.colors[m.currentColor]
			m.SampleFunctor()
			m.Heatmap.Draw()
			return m, nil

		case key.Matches(msg, m.keymap.invert):
			// invert current gradient index
			slices.Reverse(m.Heatmap.ColorScale)
			m.SampleFunctor()
			m.Heatmap.Draw()
			return m, nil

		case key.Matches(msg, m.keymap.zoomIn):
			m.Zoom += 0.1
			m.SampleFunctor()
			m.Heatmap.Draw()
			return m, nil
		case key.Matches(msg, m.keymap.zoomOut):
			m.Zoom -= 0.1
			m.SampleFunctor()
			m.Heatmap.Draw()
			return m, nil
		case key.Matches(msg, m.keymap.transUp):
			m.OriginY -= (moveFactor / m.Zoom)
			m.SampleFunctor()
			m.Heatmap.Draw()
			return m, nil
		case key.Matches(msg, m.keymap.transUp):
			m.OriginY += (moveFactor / m.Zoom)
			m.SampleFunctor()
			m.Heatmap.Draw()
			return m, nil
		case key.Matches(msg, m.keymap.transLeft):
			m.OriginX -= (moveFactor / m.Zoom)
			m.SampleFunctor()
			m.Heatmap.Draw()
			return m, nil
		case key.Matches(msg, m.keymap.transRight):
			m.OriginX += (moveFactor / m.Zoom)
			m.SampleFunctor()
			m.Heatmap.Draw()
			return m, nil
		}
	}
	return m, nil
}

func (m Model) View() string {
	functor := modelFunctors[m.currentFunctor]
	return fmt.Sprintf("%s\nzoom: %0.1f  o: (%0.1f, %0.1f) | %s  %s | %s\n%s",
		m.Heatmap.View(),
		m.Zoom, m.OriginX, m.OriginY,
		functor.Name(), functor.Detail(),
		appColorScaleNames[m.currentColor],
		m.help.View(m.keymap),
	)
}

///////////////////////////////////////////////////////////////////////////////

func main() {
	m := NewModel()
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}

///////////////////////////////////////////////////////////////////////////////

var appColorScaleNames = []string{
	"red fire",
	"blue fire",
	"thermal",
	"greyscale",
	"red",
}

var appColorScales = [][]lipgloss.Color{
	{ // red fire, thanks Claude!
		lipgloss.Color("#FFFFFF"),
		lipgloss.Color("#FFFAF0"),
		lipgloss.Color("#FFF5DC"),
		lipgloss.Color("#FFF0C1"),
		lipgloss.Color("#FFE5A0"),
		lipgloss.Color("#FFD573"),
		lipgloss.Color("#FFC247"),
		lipgloss.Color("#FFAA33"),
		lipgloss.Color("#FF9124"),
		lipgloss.Color("#FF7216"),
		lipgloss.Color("#FF5500"),
		lipgloss.Color("#FF3300"),
		lipgloss.Color("#FF1100"),
		lipgloss.Color("#E30D00"),
		lipgloss.Color("#CC0000"),
		lipgloss.Color("#990000"),
	},
	{ // blue fire, thanks Claude!
		lipgloss.Color("#FFFFFF"),
		lipgloss.Color("#F5F9FF"),
		lipgloss.Color("#E6F0FF"),
		lipgloss.Color("#CCE5FF"),
		lipgloss.Color("#99CCFF"),
		lipgloss.Color("#6699FF"),
		lipgloss.Color("#3366FF"),
		lipgloss.Color("#0033FF"),
		lipgloss.Color("#0000FF"),
		lipgloss.Color("#000099"),
		lipgloss.Color("#000066"),
		lipgloss.Color("#2B0080"),
		lipgloss.Color("#3D0099"),
		lipgloss.Color("#4B0082"),
		lipgloss.Color("#400066"),
		lipgloss.Color("#2A004D"),
	},
	{ // thermal, thanks Claude!
		lipgloss.Color("#FFFFFF"),
		lipgloss.Color("#FFFFDB"),
		lipgloss.Color("#FFF5BA"),
		lipgloss.Color("#FFE699"),
		lipgloss.Color("#FFD27F"),
		lipgloss.Color("#FFBD66"),
		lipgloss.Color("#FFA64D"),
		lipgloss.Color("#FF8533"),
		lipgloss.Color("#FF6619"),
		lipgloss.Color("#FF4700"),
		lipgloss.Color("#E31400"),
		lipgloss.Color("#C60100"),
		lipgloss.Color("#960000"),
		lipgloss.Color("#640099"),
		lipgloss.Color("#3300CC"),
		lipgloss.Color("#0000FF"),
		lipgloss.Color("#000099"),
		lipgloss.Color("#000066"),
	},
	{ // greyscale
		lipgloss.Color("#000000"), // black
		lipgloss.Color("#111111"),
		lipgloss.Color("#222222"),
		lipgloss.Color("#333333"),
		lipgloss.Color("#444444"),
		lipgloss.Color("#555555"),
		lipgloss.Color("#666666"),
		lipgloss.Color("#777777"),
		lipgloss.Color("#888888"),
		lipgloss.Color("#999999"),
		lipgloss.Color("#AAAAAA"),
		lipgloss.Color("#BBBBBB"),
		lipgloss.Color("#CCCCCC"),
		lipgloss.Color("#DDDDDD"),
		lipgloss.Color("#EEEEEE"),
		lipgloss.Color("#FFFFFF"), // white
	},
	{ // red
		lipgloss.Color("#000000"),
		lipgloss.Color("#110000"),
		lipgloss.Color("#220000"),
		lipgloss.Color("#330000"),
		lipgloss.Color("#440000"),
		lipgloss.Color("#550000"),
		lipgloss.Color("#660000"),
		lipgloss.Color("#770000"),
		lipgloss.Color("#880000"),
		lipgloss.Color("#990000"),
		lipgloss.Color("#AA0000"),
		lipgloss.Color("#BB0000"),
		lipgloss.Color("#CC0000"),
		lipgloss.Color("#DD0000"),
		lipgloss.Color("#EE0000"),
		lipgloss.Color("#FF0000"),
	},
}
