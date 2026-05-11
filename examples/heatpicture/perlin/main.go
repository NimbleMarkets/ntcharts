// examples/heatpicture/perlin/main.go
//
// High-resolution Perlin-noise heatmap via the Kitty graphics protocol.
// Each frame samples the Perlin function at full terminal-pixel resolution
// (cellPixelW × cols by cellPixelH × rows), maps the value through a
// gradient with linear interpolation between stops, and pushes the
// resulting bitmap through picture.Model. Compared to examples/heatmap/
// perlin (which renders one solid color per terminal cell), this demo
// shows smooth anti-aliased gradients with no cell-quantization artifacts.
//
// Controls mirror the cell-grid version:
//   <space>           start / stop animation
//   r                 reset alpha/beta
//   g                 next gradient
//   i                 invert gradient
//   - / +             zoom out / in
//   a/z, s/x, d/c, f/v   adjust alpha, beta, n, seed
//   q / ctrl+c        quit

package main

import (
	"fmt"
	"image/color"
	"os"
	"runtime"
	"slices"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/stopwatch"
	"charm.land/bubbles/v2/timer"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	booba "github.com/NimbleMarkets/go-booba"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/NimbleMarkets/ntcharts/v2/picture/heatpicture"
	"github.com/aquilax/go-perlin"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/spf13/pflag"
)

const (
	tickPeriod      = 100 * time.Millisecond
	kittyTickStride = 2 // sample every 10th tick in Kitty/WASM (~1 fps)
)

type keymap struct {
	start    key.Binding
	stop     key.Binding
	reset    key.Binding
	gradient key.Binding
	invert   key.Binding
	toggle   key.Binding
	quit     key.Binding

	zoomIn  key.Binding
	zoomOut key.Binding
	factor  key.Binding

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
		start:    key.NewBinding(key.WithKeys(" "), key.WithHelp("<space>", "start")),
		stop:     key.NewBinding(key.WithKeys(" "), key.WithHelp("<space>", "stop")),
		reset:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reset")),
		gradient: key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "next gradient")),
		invert:   key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "invert")),
		toggle:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "kitty/glyph")),
		quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		zoomIn:   key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "zoom in")),
		zoomOut:  key.NewBinding(key.WithKeys("-", "_"), key.WithHelp("-", "zoom out")),
		factor:   key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "sampling factor")),
		ap1:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "a+1")),
		am1:      key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "a-1")),
		bp1:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "b+1")),
		bm1:      key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "b-1")),
		np1:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "n+1")),
		nm1:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "n-1")),
		sp1:      key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "seed+1")),
		sm1:      key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "seed-1")),
	}
}

type model struct {
	hp heatpicture.Model

	alpha, beta float64
	n, seed     int64
	zoom        float64

	// kittyTickCount throttles Perlin re-sampling in Kitty mode under
	// browser-WASM. Each Kitty render is ~400ms of PNG encode + ~700KB
	// of APC bytes, far too expensive at the Glyph 100ms tick rate.
	// Sampling once every kittyTickStride ticks gives ~1 Kitty frame
	// per second — visibly animating without locking the JS thread.
	kittyTickCount int

	gradientIndex int
	gradients     [][]color.Color
	gradientNames []string

	stopwatch stopwatch.Model
	keymap    keymap
	help      help.Model
}

func newModel(alpha, beta float64, n, seed int64) *model {
	hp := heatpicture.New()
	m := &model{
		hp:            hp,
		alpha:         alpha,
		beta:          beta,
		n:             n,
		seed:          seed,
		zoom:          0.05,
		gradients:     appColorScales,
		gradientNames: appColorScaleNames,
		stopwatch:     stopwatch.New(stopwatch.WithInterval(tickPeriod)),
		keymap:        newKeyMap(),
		help:          help.New(),
	}
	// Perlin samples in (-1, 1)-ish range; map [-1, 1] data domain onto the
	// pixel grid so zoom interacts intuitively (smaller zoom = larger view).
	m.hp.SetValueRange(-1, 1)
	m.hp.SetXYRange(0, 1, 0, 1)
	m.hp.SetColorScale(m.gradients[0])
	if runtime.GOOS == "js" && runtime.GOARCH == "wasm" {
		// In browser-WASM, Go's runtime puts every goroutine on the
		// browser's main thread, so a full-resolution Kitty render at
		// 100ms ticks locks the UI. Compute scales quadratically with
		// the factor; 0.25 cuts per-frame Perlin work by 16× while
		// staying visually indistinguishable at typical demo sizes.
		// Native targets keep the heatpicture default (1.0).
		m.hp.SetSamplingFactor(0.25)
	}
	return m
}

// nextSamplingFactor cycles through 1.0 → 0.5 → 0.25 → 1.0 so a single
// keypress steps quality vs. animation cost. Falls back to 1.0 for any
// out-of-cycle value.
func nextSamplingFactor(curr float64) float64 {
	switch {
	case curr >= 0.99: // ~1.0
		return 0.5
	case curr >= 0.49: // ~0.5
		return 0.25
	default:
		return 1.0
	}
}

func (m *model) sampler() heatpicture.Sampler {
	p := perlin.NewPerlin(m.alpha, m.beta, int32(m.n), m.seed)
	zoom := m.zoom
	return func(x, y float64) float64 {
		// Scale data-space [0,1] coords by an animation-friendly base
		// scale modulated by zoom; a smaller zoom yields a coarser view.
		return p.Noise2D(x/zoom, y/zoom)
	}
}

func (m *model) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.start, m.keymap.stop, m.keymap.quit,
		m.keymap.reset, m.keymap.gradient, m.keymap.invert,
		m.keymap.toggle, m.keymap.factor,
		m.keymap.zoomIn, m.keymap.zoomOut,
	}) + "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.ap1, m.keymap.am1,
		m.keymap.bp1, m.keymap.bm1,
		m.keymap.np1, m.keymap.nm1,
		m.keymap.sp1, m.keymap.sm1,
	})
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.hp.Init(),
		m.hp.SetSampler(m.sampler()),
		m.stopwatch.Init(),
	)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Reserve 4 rows: blank + info + 2 help bars.
		if c := m.hp.SetSize(msg.Width, msg.Height-4); c != nil {
			cmds = append(cmds, c)
		}
		return m, tea.Batch(cmds...)
	case uv.CellSizeEvent:
		// fall through to m.hp.Update below to trigger re-render

	case stopwatch.TickMsg:
		var cmd tea.Cmd
		m.stopwatch, cmd = m.stopwatch.Update(msg)
		cmds = append(cmds, cmd)
		// Throttle Perlin re-sampling in Kitty/WASM. The Glyph mode
		// renders ~10 frames/sec from this 100ms tick; in Kitty/WASM,
		// each render is ~400ms of compute + transmission, so we drop
		// to ~1 frame/sec. Native and Glyph paths sample every tick.
		skipSample := false
		if runtime.GOOS == "js" && runtime.GOARCH == "wasm" && m.hp.Mode() == picture.PictureKitty {
			m.kittyTickCount++
			skipSample = m.kittyTickCount%kittyTickStride != 0
		}
		if !skipSample {
			m.alpha += 0.01
			m.beta += 0.01
			if c := m.hp.SetSampler(m.sampler()); c != nil {
				cmds = append(cmds, c)
			}
		}
		return m, tea.Batch(cmds...)

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
			return m, tea.Quit
		case key.Matches(msg, m.keymap.start, m.keymap.stop):
			return m, m.stopwatch.Toggle()
		case key.Matches(msg, m.keymap.reset):
			m.alpha, m.beta = 1, 2
			if c := m.hp.SetSampler(m.sampler()); c != nil {
				cmds = append(cmds, c)
			}
		case key.Matches(msg, m.keymap.gradient):
			m.gradientIndex = (m.gradientIndex + 1) % len(m.gradients)
			if c := m.hp.SetColorScale(m.gradients[m.gradientIndex]); c != nil {
				cmds = append(cmds, c)
			}
		case key.Matches(msg, m.keymap.invert):
			scale := slices.Clone(m.gradients[m.gradientIndex])
			slices.Reverse(scale)
			m.gradients[m.gradientIndex] = scale
			if c := m.hp.SetColorScale(scale); c != nil {
				cmds = append(cmds, c)
			}
		case key.Matches(msg, m.keymap.toggle):
			if c := m.hp.Toggle(); c != nil {
				cmds = append(cmds, c)
			}
		case key.Matches(msg, m.keymap.factor):
			next := nextSamplingFactor(m.hp.SamplingFactor())
			if c := m.hp.SetSamplingFactor(next); c != nil {
				cmds = append(cmds, c)
			}
		case key.Matches(msg, m.keymap.zoomIn):
			m.zoom *= 1.1
			if c := m.hp.SetSampler(m.sampler()); c != nil {
				cmds = append(cmds, c)
			}
		case key.Matches(msg, m.keymap.zoomOut):
			m.zoom /= 1.1
			if c := m.hp.SetSampler(m.sampler()); c != nil {
				cmds = append(cmds, c)
			}
		case key.Matches(msg, m.keymap.ap1):
			m.alpha += 1
			cmds = append(cmds, m.hp.SetSampler(m.sampler()))
		case key.Matches(msg, m.keymap.am1):
			m.alpha -= 1
			cmds = append(cmds, m.hp.SetSampler(m.sampler()))
		case key.Matches(msg, m.keymap.bp1):
			m.beta += 1
			cmds = append(cmds, m.hp.SetSampler(m.sampler()))
		case key.Matches(msg, m.keymap.bm1):
			m.beta -= 1
			cmds = append(cmds, m.hp.SetSampler(m.sampler()))
		case key.Matches(msg, m.keymap.np1):
			m.n += 1
			cmds = append(cmds, m.hp.SetSampler(m.sampler()))
		case key.Matches(msg, m.keymap.nm1):
			m.n -= 1
			cmds = append(cmds, m.hp.SetSampler(m.sampler()))
		case key.Matches(msg, m.keymap.sp1):
			m.seed += 1
			cmds = append(cmds, m.hp.SetSampler(m.sampler()))
		case key.Matches(msg, m.keymap.sm1):
			m.seed -= 1
			cmds = append(cmds, m.hp.SetSampler(m.sampler()))
		}
		return m, tea.Batch(cmds...)
	}

	if c := m.hp.Update(msg); c != nil {
		cmds = append(cmds, c)
	}
	return m, tea.Batch(cmds...)
}

func (m *model) View() tea.View {
	hpContent := m.hp.View().Content
	pxW, pxH := m.hp.SamplePixelSize()
	cellW, cellH := m.hp.CellPixelSize()
	info := fmt.Sprintf("\n%s  α: %.3f  β: %.3f  n: %d  seed: %d  zoom: %.3f  sf: %.2f  mode: %v  Δ: %s  frames: %d  composites: %d  px: %d×%d  cell: %d×%d",
		m.gradientNames[m.gradientIndex], m.alpha, m.beta, m.n, m.seed, m.zoom,
		m.hp.SamplingFactor(),
		m.hp.Mode(),
		m.hp.LastRenderDuration().Round(time.Microsecond*100),
		m.hp.FrameCount(),
		m.hp.CompositeCount(),
		pxW, pxH,
		cellW, cellH)
	v := tea.NewView(hpContent + info + m.helpView())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func main() {
	var alpha, beta float64
	var seed, n int64
	var showHelp bool
	pflag.Float64VarP(&alpha, "alpha", "a", 1, "Perlin Alpha")
	pflag.Float64VarP(&beta, "beta", "b", 2, "Perlin Beta")
	pflag.Int64VarP(&n, "num", "n", 4, "Perlin Start Num")
	pflag.Int64VarP(&seed, "seed", "s", 100, "Perlin Seed")
	pflag.BoolVarP(&showHelp, "help", "", false, "show help")
	pflag.Parse()
	if showHelp {
		fmt.Fprintf(os.Stdout, "usage:  %s [--help] [options]\n", os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}

	m := newModel(alpha, beta, n, seed)
	if err := booba.Run(m); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

var appColorScaleNames = []string{
	"red fire", "blue fire", "thermal", "greyscale",
}

var appColorScales = [][]color.Color{
	{ // red fire
		lipgloss.Color("#FFFFFF"), lipgloss.Color("#FFFAF0"), lipgloss.Color("#FFE5A0"),
		lipgloss.Color("#FFC247"), lipgloss.Color("#FF7216"), lipgloss.Color("#FF3300"),
		lipgloss.Color("#CC0000"), lipgloss.Color("#660000"),
	},
	{ // blue fire
		lipgloss.Color("#FFFFFF"), lipgloss.Color("#E6F0FF"), lipgloss.Color("#99CCFF"),
		lipgloss.Color("#3366FF"), lipgloss.Color("#0033FF"), lipgloss.Color("#000099"),
		lipgloss.Color("#3D0099"), lipgloss.Color("#2A004D"),
	},
	{ // thermal
		lipgloss.Color("#FFFFFF"), lipgloss.Color("#FFE699"), lipgloss.Color("#FFA64D"),
		lipgloss.Color("#FF6619"), lipgloss.Color("#E31400"), lipgloss.Color("#960000"),
		lipgloss.Color("#3300CC"), lipgloss.Color("#000066"),
	},
	{ // greyscale
		lipgloss.Color("#000000"), lipgloss.Color("#333333"), lipgloss.Color("#666666"),
		lipgloss.Color("#999999"), lipgloss.Color("#CCCCCC"), lipgloss.Color("#FFFFFF"),
	},
}
