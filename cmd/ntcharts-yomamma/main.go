package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

const (
	heartEmoji        = "💖"
	catEmoji          = "😻"
	fallbackWidth     = 80
	fallbackHeight    = 24
	catClickThreshold = 6
	chaosDuration     = 5 * time.Second
	chaosTickInterval = 160 * time.Millisecond
	pinkPauseDuration = 900 * time.Millisecond
	finalContentWidth = 62
)

var randomEmojis = []string{
	"💖", "💗", "💓", "💝", "😻", "🐱", "🌸", "🌺", "💐",
	"🥰", "🦋", "🌼", "💘", "💕",
}

type phase int

const (
	phaseHearts phase = iota
	phaseChaotic
	phasePink
	phaseMessage
)

type tickMsg time.Time
type goToMessageMsg struct{}

type model struct {
	c        canvas.Model
	width    int
	height   int
	grid     [][]string
	clicks   int
	phase    phase
	name     string
	chaosEnd time.Time
	style    lipgloss.Style
}

func makeGrid(rows, cols int, emoji string) [][]string {
	grid := make([][]string, rows)
	for i := range grid {
		grid[i] = make([]string, cols)
		for j := range grid[i] {
			grid[i][j] = emoji
		}
	}
	return grid
}

func initialModel(name string) model {
	c := canvas.New(fallbackWidth, fallbackHeight,
		canvas.WithStyle(lipgloss.NewStyle()),
		canvas.WithFocus(),
	)

	m := model{
		c:      c,
		width:  fallbackWidth,
		height: fallbackHeight,
		grid:   makeGrid(fallbackHeight, fallbackWidth/2, heartEmoji),
		name:   name,
		phase:  phaseHearts,
		style:  lipgloss.NewStyle().Foreground(lipgloss.Color("#FF69B4")),
	}

	m.updateCanvas()
	return m
}

func (m *model) updateCanvas() {
	lines := make([]string, m.height)
	for y := range m.grid {
		line := strings.Join(m.grid[y], "")
		lines[y] = line
	}
	m.c.SetLinesWithStyle(lines, m.style)
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.c, cmd = m.c.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.c = canvas.New(m.width, m.height,
			canvas.WithStyle(lipgloss.NewStyle()),
			canvas.WithFocus(),
		)

		switch m.phase {
		case phaseHearts:
			m.grid = makeGrid(m.height, m.width/2, heartEmoji)
			m.updateCanvas()
		case phasePink:
			m.grid = makeGrid(m.height, m.width/2, heartEmoji)
			m.updateCanvas()
		case phaseChaotic:
			m.grid = makeGrid(m.height, m.width/2, randomEmojis[0])
			m.updateCanvas()
		}
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.MouseClickMsg:
		if m.phase != phaseHearts {
			return m, cmd
		}

		mouse := msg.Mouse()
		x, y := mouse.X, mouse.Y

		if y >= 0 && y < m.height && x >= 0 && x < m.width {
			col := x / 2
			if col < len(m.grid[y]) && m.grid[y][col] == heartEmoji {
				m.grid[y][col] = catEmoji
				m.clicks++
				m.updateCanvas()

				if m.clicks >= catClickThreshold {
					m.phase = phaseChaotic
					m.chaosEnd = time.Now().Add(chaosDuration)
					m.style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF69B4"))
					return m, tickCmd(chaosTickInterval)
				}
			}
		}

	case tickMsg:
		if m.phase == phaseChaotic {
			if time.Now().After(m.chaosEnd) {
				m.phase = phasePink
				m.grid = makeGrid(m.height, m.width/2, heartEmoji)
				m.style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF69B4"))
				m.updateCanvas()

				return m, tea.Tick(pinkPauseDuration, func(t time.Time) tea.Msg {
					return goToMessageMsg{}
				})
			}

			for i := range m.grid {
				for j := range m.grid[i] {
					m.grid[i][j] = randomEmojis[rand.Intn(len(randomEmojis))]
				}
			}
			m.updateCanvas()
			return m, tickCmd(chaosTickInterval)
		}

	case goToMessageMsg:
		m.phase = phaseMessage
	}

	return m, cmd
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Large full glyph heart built from 💖 emojis
func finalView(width, height int, name string) string {
	heartLines := []string{
		"      💖💖💖💖💖💖         💖💖💖💖💖💖💖     ",
		"    💖💖💖💖💖💖💖💖      💖💖💖💖💖💖💖💖    ",
		"  💖💖💖💖💖💖💖💖💖💖  💖💖💖💖💖💖💖💖💖💖  ",
		" 💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖 ",
		"💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖  ",
		"💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖  ",
		" 💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖   ",
		"  💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖    ",
		"   💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖     ",
		"    💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖      ",
		"     💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖       ",
		"      💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖        ",
		"       💖💖💖💖💖💖💖💖💖💖💖💖💖💖💖         ",
		"        💖💖💖💖💖💖💖💖💖💖💖💖💖💖          ",
		"         💖💖💖💖💖💖💖💖💖💖💖💖💖           ",
		"          💖💖💖💖💖💖💖💖💖💖💖💖            ",
		"           💖💖💖💖💖💖💖💖💖💖💖             ",
		"            💖💖💖💖💖💖💖💖💖💖              ",
		"             💖💖💖💖💖💖💖💖💖               ",
		"               💖💖💖💖💖💖💖                 ",
		"                💖💖💖💖💖💖                  ",
		"                  💖💖💖💖                    ",
	}

	heartStr := strings.Join(heartLines, "\n")
	heartStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF1493")).
		Bold(true)

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF69B4")).
		Bold(true).
		Italic(true).
		Align(lipgloss.Center).
		Width(finalContentWidth).
		Render("HAPPY MOTHER'S DAY")

	nameLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFB6C1")).
		Bold(true).
		Align(lipgloss.Center).
		Width(finalContentWidth).
		Render(name)

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"", // line 2 under title → name is now exactly two lines under
		nameLine,
		"",
		heartStyle.Render(heartStr),
	)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m model) View() tea.View {
	var str string
	switch m.phase {
	case phaseMessage:
		str = finalView(m.width, m.height, m.name)
	default:
		str = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Render(m.c.View())
	}

	v := tea.NewView(str)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run . \"Your message or name here\"")
		fmt.Println("Example: go run . \"Love you Mom!\"")
		os.Exit(1)
	}

	rand.Seed(time.Now().UnixNano())

	m := initialModel(os.Args[1])

	p := tea.NewProgram(m)
	fmt.Println("💖  Mother's Day TUI launching... click the hearts anywhere on the full screen! 💖")

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
