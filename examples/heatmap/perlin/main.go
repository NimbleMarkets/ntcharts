//
// Based on ECharts heat map demo:
//   https://echarts.apache.org/examples/en/editor.html?c=heatmap-large-piecewise

package main

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/NimbleMarkets/ntcharts/heatmap"
	"github.com/aquilax/go-perlin"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/pflag"
)

///////////////////////////////////////////////////////////////////////////////

type keymap struct {
	start    key.Binding
	stop     key.Binding
	reset    key.Binding
	gradient key.Binding
	invert   key.Binding
	quit     key.Binding

	zoomIn  key.Binding
	zoomOut key.Binding

	ap1 key.Binding
	am1 key.Binding
	bp1 key.Binding
	bm1 key.Binding
	np1 key.Binding
	nm1 key.Binding
	sp1 key.Binding
	sm1 key.Binding
}

func newKeyMap() keymap {
	return keymap{
		start: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("<space>", "start"),
		),
		stop: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("<space>", "stop"),
		),
		reset: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reset"),
		),
		gradient: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "next gradient"),
		),
		invert: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "invert gradient"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		zoomIn: key.NewBinding(
			key.WithKeys("-", "_"),
			key.WithHelp("-", "zoom out"),
		),
		zoomOut: key.NewBinding(
			key.WithKeys("+", "="),
			key.WithHelp("+", "zoom in"),
		),
		ap1: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "a+1"),
		),
		am1: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "a-1"),
		),
		bp1: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "b+10"),
		),
		bm1: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "b-10"),
		),
		np1: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "n+1"),
		),
		nm1: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "n-1"),
		),
		sp1: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "seed+1"),
		),
		sm1: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "seed-1"),
		),
	}
}

///////////////////////////////////////////////////////////////////////////////

type PerlinModel struct {
	Alpha, Beta float64
	N, Seed     int64
	Zoom        float64

	Heatmap heatmap.Model

	gradientIndex int
	gradients     [][]lipgloss.Color

	timeout   time.Duration
	stopwatch stopwatch.Model
	keymap    keymap
	help      help.Model
	quitting  bool
}

func NewPerlinModel(alpha, beta float64, n, seed int64, timeoutMS int64) *PerlinModel {
	intervalDuration := time.Duration(timeoutMS) * time.Millisecond
	return &PerlinModel{
		Alpha: alpha,
		Beta:  beta,
		N:     n,
		Seed:  seed,
		Zoom:  0.01,
		Heatmap: heatmap.New(80, 30,
			heatmap.WithAutoValueRange(),
			heatmap.WithColorScale(appColorScales[0])),
		gradientIndex: 0,
		gradients:     appColorScales,
		timeout:       intervalDuration,
		stopwatch:     stopwatch.NewWithInterval(intervalDuration),
		keymap:        newKeyMap(),
		help:          help.New(),
	}
}

func (m *PerlinModel) SampleNoise() {
	p := perlin.NewPerlin(m.Alpha, m.Beta, int32(m.N), m.Seed)
	m.Heatmap.ClearData()
	for i := 0; i <= m.Heatmap.GraphWidth(); i++ {
		for j := 0; j <= m.Heatmap.GraphHeight(); j++ {
			noise := p.Noise2D(float64(i)*m.Zoom, float64(j)*m.Zoom)
			m.Heatmap.Push(heatmap.NewHeatPoint(float64(i), float64(j), noise))
		}
	}
}

func (m *PerlinModel) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.start,
		m.keymap.stop,
		m.keymap.quit,
		m.keymap.reset,
		m.keymap.gradient,
		m.keymap.invert,
		m.keymap.zoomIn,
		m.keymap.zoomOut,
	}) + m.help.ShortHelpView([]key.Binding{
		m.keymap.ap1,
		m.keymap.am1,
		m.keymap.bp1,
		m.keymap.bm1,
		m.keymap.np1,
		m.keymap.nm1,
		m.keymap.sp1,
		m.keymap.sm1,
	})
}

///////////////////////////////////////////////////////////////////////////////

func (m *PerlinModel) Init() tea.Cmd {
	m.SampleNoise()
	m.Heatmap.Draw()
	return tea.Batch(m.Heatmap.Init(), m.stopwatch.Init())
}

// Update forwards BubbleTea Msg to underlying canvas.
func (m *PerlinModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Heatmap.Resize(msg.Width, msg.Height)
		m.SampleNoise()
		m.Heatmap.Draw()
		return m, nil

	case stopwatch.TickMsg:
		var cmd tea.Cmd
		m.Alpha += 0.01
		m.Beta += 0.01
		m.SampleNoise()
		m.Heatmap.Draw()
		m.stopwatch, cmd = m.stopwatch.Update(msg)
		return m, cmd

	case stopwatch.StartStopMsg:
		var cmd tea.Cmd
		m.stopwatch, cmd = m.stopwatch.Update(msg)
		m.keymap.stop.SetEnabled(m.stopwatch.Running())
		m.keymap.start.SetEnabled(!m.stopwatch.Running())
		return m, cmd

	case timer.TimeoutMsg:
		return m, m.stopwatch.Start()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keymap.gradient):
			m.gradientIndex += 1
			if m.gradientIndex >= len(m.gradients) {
				m.gradientIndex = 0
			}
			m.Heatmap.ColorScale = m.gradients[m.gradientIndex]
			return m, nil

		case key.Matches(msg, m.keymap.invert):
			// invert current gradient index
			slices.Reverse(m.Heatmap.ColorScale)
			return m, nil

		case key.Matches(msg, m.keymap.start, m.keymap.stop):
			return m, m.stopwatch.Toggle()

		case key.Matches(msg, m.keymap.zoomIn):
			m.Zoom += 0.01
			return m, nil

		case key.Matches(msg, m.keymap.zoomOut):
			m.Zoom -= 0.01
			return m, nil

		case key.Matches(msg, m.keymap.ap1):
			m.Alpha += 1
			return m, nil
		case key.Matches(msg, m.keymap.am1):
			m.Alpha -= 1
			return m, nil
		case key.Matches(msg, m.keymap.bp1):
			m.Beta += 1
			return m, nil
		case key.Matches(msg, m.keymap.bm1):
			m.Beta -= 1
			return m, nil
		case key.Matches(msg, m.keymap.np1):
			m.N += 1
			return m, nil
		case key.Matches(msg, m.keymap.nm1):
			m.N -= 1
			return m, nil
		case key.Matches(msg, m.keymap.sp1):
			m.Seed += 1
			return m, nil
		case key.Matches(msg, m.keymap.sm1):
			m.Seed -= 1
			return m, nil
		}
	}
	return m, nil
}

// View returns a string used by the bubbletea framework to display the barchart.
func (m *PerlinModel) View() (r string) {
	info := fmt.Sprintf("\n%s  alpha: %.4f  beta: %.2f  n: %d  seed: %d  zoom: %.3f\n",
		appColorScaleNames[m.gradientIndex], m.Alpha, m.Beta, m.N, m.Seed, m.Zoom)
	return m.Heatmap.View() + info + m.helpView()
}

/////////////////////////////////////////////////////////////////////////////

var usageFormat string = "usage:  %s [--help] [options]\n"

func main() {
	var alpha, beta, timeoutSecs float64
	var seed, n int64
	var showHelp bool

	pflag.Float64VarP(&alpha, "alpha", "a", 1, "Perlin Alpha")
	pflag.Float64VarP(&beta, "beta", "b", 2, "Perlin Beta")
	pflag.Int64VarP(&n, "num", "n", 4, "Perlin Start Num")
	pflag.Int64VarP(&seed, "seed", "s", 100, "Perlin Seed")
	pflag.Float64VarP(&timeoutSecs, "time", "t", 0.15, "time between updates, in seconds")
	pflag.BoolVarP(&showHelp, "help", "", false, "show help")
	pflag.Parse()

	if showHelp {
		fmt.Fprintf(os.Stdout, usageFormat, os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}

	m := NewPerlinModel(alpha, beta, n, seed, int64(timeoutSecs*1000))
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
