package main

import (
	"fmt"
	"image"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

type tickMsg struct{ generation uint64 }
type slideMsg struct{ generation uint64 }
type quitMsg struct{}
type renderedMsg struct {
	epoch   uint64
	image   *image.NRGBA
	elapsed time.Duration
	err     error
}
type encodedMsg struct {
	frame   tea.Msg
	elapsed time.Duration
}
type presentedMsg struct{}
type model struct {
	encodedFrames, renderedPresets                                     map[string]int
	pic                                                                picture.Model
	renderer                                                           frameRenderer
	selected, parameter, width, height, cols, rows, density, targetFPS int
	rasterW, rasterH                                                   int
	speed, scale, color, detail                                        float32
	seconds                                                            float64
	lastClock, frameStarted, lastPresented                             time.Time
	renderMS, encodeMS, fps                                            float64
	bytes                                                              int
	transport                                                          string
	playing, fullscreen, forceGlyph, busy, dirty                       bool
	epoch, slideGeneration, wakeGeneration                             uint64
	slideshow, duration                                                time.Duration
	err                                                                error
}

func newModel(r frameRenderer, index, fps, density int, slideshow time.Duration) *model {
	p := presets[index]
	return &model{encodedFrames: make(map[string]int), renderedPresets: make(map[string]int), renderer: r, selected: index, speed: 0.6, scale: p.scale, detail: p.detail, density: density, targetFPS: fps,
		playing: true, dirty: true, slideshow: slideshow, transport: "probing", pic: picture.NewWithConfig(picture.Config{
			Fit: picture.FitFill, CellPixelWidth: 8, CellPixelHeight: 16,
		})}
}
func (m *model) Init() tea.Cmd {
	var quit tea.Cmd
	if m.duration > 0 {
		quit = tea.Tick(m.duration, func(time.Time) tea.Msg { return quitMsg{} })
	}
	return tea.Batch(m.pic.Init(), m.slide(), quit)
}
func (m *model) slide() tea.Cmd {
	if m.slideshow <= 0 {
		return nil
	}
	generation := m.slideGeneration
	return tea.Tick(m.slideshow, func(time.Time) tea.Msg { return slideMsg{generation} })
}
func (m *model) selectPreset(index int) {
	m.selected = (index + len(presets)) % len(presets)
	m.scale, m.detail = presets[m.selected].scale, presets[m.selected].detail
	m.seconds = 0
	m.lastClock = time.Now()
	m.dirty = true
	m.epoch++
}
func (m *model) showChrome() bool { return !m.fullscreen && m.width >= 40 && m.height >= 8 }

func (m *model) geometry() {
	m.cols, m.rows = max(1, m.width), max(1, m.height)
	if m.showChrome() {
		m.rows = max(1, m.height-5)
		if m.width >= 100 {
			m.cols = m.width - 26
		}
	}
	// The following setters invalidate pending frames. We render one fresh image
	// below, instead of executing commands that would re-encode the previous one.
	m.pic.SetSize(m.cols, m.rows)
	cw, ch := m.pic.CellPixelSize()
	factor := math.Min(1, float64(m.density)/float64(ch))
	factor = math.Min(factor, math.Min(2048/float64(m.cols*cw), 1536/float64(m.rows*ch)))
	factor = math.Min(factor, math.Sqrt((1536*1024)/float64(m.cols*m.rows*cw*ch)))
	factor = math.Min(1, math.Nextafter(factor, math.Inf(1)))
	m.pic.SetKittyResolutionFactor(factor)
	m.rasterW = m.cols * max(1, int(float64(cw)*factor))
	m.rasterH = m.rows * max(1, int(float64(ch)*factor))
	if m.pic.Mode() == picture.PictureGlyph {
		m.rasterW = min(m.cols, 2048)
		m.rasterH = min(m.rows*2, 1536)
	}
}
func (m *model) render() tea.Cmd {
	if m.busy || m.width < 1 || m.height < 1 || (!m.playing && !m.dirty) || m.err != nil {
		return nil
	}
	// Very large terminal grids can exceed the per-pixel GPU limit even at one
	// source pixel per cell; keep the UI responsive without submitting invalid work.
	m.geometry()
	if m.rasterW > 2048 || m.rasterH > 1536 {
		return nil
	}
	now := time.Now()
	if m.playing && !m.lastClock.IsZero() {
		m.seconds += min(now.Sub(m.lastClock).Seconds(), 0.25)
	}
	m.lastClock = now
	m.frameStarted = now
	m.wakeGeneration++
	m.busy = true
	m.dirty = false
	epoch := m.epoch
	req := renderRequest{preset: m.selected, width: m.rasterW, height: m.rasterH, seconds: float32(m.seconds), speed: m.speed, scale: m.scale, color: m.color, detail: m.detail}
	renderer := m.renderer
	return func() tea.Msg {
		start := time.Now()
		img, err := renderer.Render(req)
		return renderedMsg{epoch, img, time.Since(start), err}
	}
}
func smooth(old, next float64) float64 {
	if old == 0 {
		return next
	}
	return old*0.8 + next*0.2
}
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case quitMsg:
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.epoch++
		m.dirty = true
		m.geometry()
		return m, m.render()
	case uv.CellSizeEvent:
		m.pic.SetCellPixelSize(msg.Width, msg.Height)
		m.epoch++
		m.dirty = true
		m.geometry()
		return m, m.render()
	case slideMsg:
		if msg.generation != m.slideGeneration || m.slideshow <= 0 {
			return m, nil
		}
		if m.playing {
			m.selectPreset(m.selected + 1)
		}
		return m, tea.Batch(m.slide(), m.render())
	case tickMsg:
		if msg.generation != m.wakeGeneration {
			return m, nil
		}
		return m, m.render()
	case renderedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.busy = false
			return m, tea.Quit
		}
		if msg.epoch != m.epoch {
			m.busy = false
			return m, m.render()
		}
		m.renderedPresets[presets[m.selected].name]++
		m.renderMS = smooth(m.renderMS, float64(msg.elapsed)/float64(time.Millisecond))
		cmd := m.pic.SetImage(msg.image)
		if cmd == nil {
			m.bytes = 0
			m.encodeMS = 0
			m.transport = "glyph"
			return m, func() tea.Msg { return presentedMsg{} }
		}
		return m, func() tea.Msg { start := time.Now(); frame := cmd(); return encodedMsg{frame, time.Since(start)} }
	case encodedMsg:
		m.encodeMS = smooth(m.encodeMS, float64(msg.elapsed)/float64(time.Millisecond))
		if frame, ok := msg.frame.(picture.KittyFrameMsg); ok {
			m.encodedFrames["png"]++
			m.bytes = len(frame.APC)
			m.transport = "png"
		}
		return m, tea.Sequence(m.pic.Update(msg.frame), func() tea.Msg { return presentedMsg{} })
	case presentedMsg:
		now := time.Now()
		if m.playing && !m.lastPresented.IsZero() {
			m.fps = smooth(m.fps, 1/now.Sub(m.lastPresented).Seconds())
		}
		m.lastPresented = now
		m.busy = false
		if m.dirty {
			return m, m.render()
		}
		if !m.playing {
			return m, nil
		}
		delay := max(time.Duration(0), time.Second/time.Duration(m.targetFPS)-time.Since(m.frameStarted))
		generation := m.wakeGeneration
		return m, tea.Tick(delay, func(time.Time) tea.Msg { return tickMsg{generation} })
	case tea.KeyPressMsg:
		var extra tea.Cmd
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "left", "h":
			m.selectPreset(m.selected - 1)
		case "right", "l":
			m.selectPreset(m.selected + 1)
		case "1", "2", "3", "4", "5", "6":
			m.selectPreset(int(msg.String()[0] - '1'))
		case "space":
			m.playing = !m.playing
			m.lastClock = time.Now()
			m.lastPresented = time.Time{}
			m.fps = 0
		case "f", "ctrl+f":
			m.fullscreen = !m.fullscreen
			m.geometry()
			m.epoch++
		case "esc":
			m.fullscreen = false
			m.geometry()
			m.epoch++
		case "r":
			m.seconds = 0
			m.lastClock = time.Now()
			m.epoch++
		case "g":
			m.forceGlyph = !m.forceGlyph
			if m.forceGlyph && m.pic.Mode() == picture.PictureKitty || !m.forceGlyph && m.pic.Mode() == picture.PictureGlyph {
				extra = m.pic.Toggle()
			}
			m.geometry()
			m.epoch++
		case "a":
			m.slideGeneration++
			if m.slideshow > 0 {
				m.slideshow = 0
			} else {
				m.slideshow = 8 * time.Second
				extra = m.slide()
			}
		case "up":
			m.parameter = (m.parameter + 3) % 4
		case "down", "tab":
			m.parameter = (m.parameter + 1) % 4
		case "[", "-":
			m.adjust(-1)
		case "]", "+", "=":
			m.adjust(1)
		case ",":
			m.density = max(2, m.density-2)
			m.geometry()
			m.epoch++
		case ".":
			m.density = min(40, m.density+2)
			m.geometry()
			m.epoch++
		default:
			return m, nil
		}
		m.dirty = true
		return m, tea.Batch(extra, m.render())
	}
	cmd := m.pic.Update(msg)
	if !m.forceGlyph && m.pic.Mode() == picture.PictureGlyph && picture.KittySupported() == picture.KittyCapabilitySupported {
		toggle := m.pic.Toggle()
		m.epoch++
		m.dirty = true
		return m, tea.Batch(cmd, toggle, m.render())
	}
	return m, cmd
}
func (m *model) adjust(direction float32) {
	switch m.parameter {
	case 0:
		m.speed = max(0, min(3, m.speed+direction*0.05))
	case 1:
		m.scale = max(0.2, min(6, m.scale+direction*0.1))
	case 2:
		m.color += direction * 0.025
	case 3:
		m.detail = max(0, min(3, m.detail+direction*0.05))
	}
	m.epoch++
}

var accent = lipgloss.NewStyle().Foreground(lipgloss.Color("#70ead1")).Bold(true)
var muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b91aa"))

func (m *model) View() tea.View {
	if m.width < 1 || m.height < 1 {
		return tea.NewView("Starting GPU shader gallery…")
	}
	imageView := m.pic.String()
	if imageView == "" {
		imageView = lipgloss.NewStyle().Width(m.cols).Height(m.rows).Render("Rendering…")
	}
	content := imageView
	if m.showChrome() {
		if m.width >= 100 {
			lines := []string{accent.Render("SHADER GALLERY"), ""}
			for i, p := range presets {
				label := fmt.Sprintf(" %d  %s", i+1, p.name)
				if i == m.selected {
					label = accent.Render("›" + label[1:])
				} else {
					label = muted.Render(label)
				}
				lines = append(lines, label, "")
			}
			lines = append(lines, muted.Render("← → browse"), muted.Render("a   slideshow"))
			if len(lines) > m.rows {
				lines = lines[:m.rows]
			}
			sidebar := lipgloss.NewStyle().Width(24).Height(m.rows).Render(strings.Join(lines, "\n"))
			content = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, "  ", imageView)
		}
		state := "PLAY"
		if !m.playing {
			state = "PAUSED"
		}
		if m.slideshow > 0 {
			state += " · SLIDESHOW"
		}
		title := accent.Render("NTCHARTS / SHADERS") + "   " + presets[m.selected].title + "   " + muted.Render(state)
		stats := fmt.Sprintf("%s · %.0f app fps · %d×%d · R %.1f / E %.1f ms · %s", m.transport, m.fps, m.rasterW, m.rasterH, m.renderMS, m.encodeMS, formatBytes(m.bytes))
		values := []string{fmt.Sprintf("speed %.2f", m.speed), fmt.Sprintf("scale %.2f", m.scale), fmt.Sprintf("color %.2f", m.color), fmt.Sprintf("detail %.2f", m.detail)}
		values[m.parameter] = accent.Render("[" + values[m.parameter] + "]")
		params := strings.Join(values, "   ") + fmt.Sprintf("   density %d", m.density)
		help := "←→ preset · space pause · f full · ↑↓ select · [] edit · q quit"
		note := presets[m.selected].description
		clip := func(s string) string { return ansi.Truncate(s, m.width, "") }
		content = clip(title) + "\n" + clip(muted.Render(stats)) + "\n" + content + "\n" + clip(params) + "\n" + clip(muted.Render(help)) + "\n" + clip(muted.Render(note))
	}
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}
func formatBytes(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KiB", float64(n)/1024)
	}
	return fmt.Sprintf("%.2f MiB", float64(n)/(1024*1024))
}
