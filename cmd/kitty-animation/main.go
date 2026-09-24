// kitty-animation exercises the real picture/Bubble Tea upload lifecycle.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	uv "github.com/charmbracelet/ultraviolet"
)

type nextFrame struct{}
type displayed struct{}
type stop struct{}
type encoded struct {
	frame   tea.Msg
	elapsed time.Duration
}
type sample struct {
	Transport    string  `json:"transport"`
	EncodeMS     float64 `json:"encode_ms"`
	PayloadBytes int     `json:"payload_bytes"`
	Error        string  `json:"error,omitempty"`
}
type model struct {
	pic           picture.Model
	frames, count int
	width, height int
	interval      time.Duration
	samples       []sample
	err           error
	started       time.Time
}

func (m *model) Init() tea.Cmd {
	m.started = time.Now()
	return tea.Batch(m.pic.Init(), tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return nextFrame{} }), tea.Tick(time.Minute, func(time.Time) tea.Msg { return stop{} }))
}
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case stop:
		m.err = fmt.Errorf("animation timed out after %d frames", m.count)
		return m, tea.Quit
	case uv.CellSizeEvent:
		// Keep the requested raster fixed for comparable transport measurements.
		return m, nil
	case nextFrame:
		if picture.KittySupported() != picture.KittyCapabilitySupported {
			if picture.KittySupported() == picture.KittyCapabilityUnsupported {
				m.err = fmt.Errorf("Kitty protocol unsupported")
				return m, tea.Quit
			}
			return m, tea.Tick(20*time.Millisecond, func(time.Time) tea.Msg { return nextFrame{} })
		}
		if m.pic.Mode() == picture.PictureGlyph {
			m.pic.Toggle()
		}
		img := image.NewNRGBA(image.Rect(0, 0, m.width, m.height))
		for y := 0; y < m.height; y++ {
			for x := 0; x < m.width; x++ {
				i := y*img.Stride + 4*x
				img.Pix[i] = byte((x + m.count*7) % 256)
				img.Pix[i+1] = byte((y + m.count*3) % 256)
				img.Pix[i+2] = byte((x ^ y) + m.count)
				img.Pix[i+3] = 255
			}
		}
		cmd := m.pic.SetImage(img)
		if cmd == nil {
			m.err = fmt.Errorf("no frame command")
			return m, tea.Quit
		}
		return m, func() tea.Msg {
			start := time.Now()
			frame := cmd()
			return encoded{frame: frame, elapsed: time.Since(start)}
		}
	case encoded:
		frame, ok := msg.frame.(picture.KittyFrameMsg)
		if !ok {
			m.err = fmt.Errorf("unexpected frame %T", msg.frame)
			return m, tea.Quit
		}
		s := sample{Transport: "png", EncodeMS: float64(msg.elapsed) / float64(time.Millisecond), PayloadBytes: len(frame.APC)}
		m.samples = append(m.samples, s)
		m.count++
		return m, tea.Sequence(m.pic.Update(frame), func() tea.Msg { return displayed{} })
	case displayed:
		if m.count >= m.frames {
			return m, tea.Quit
		}
		return m, tea.Tick(m.interval, func(time.Time) tea.Msg { return nextFrame{} })
	}
	return m, m.pic.Update(msg)
}
func (m *model) View() tea.View {
	view := m.pic.View()
	view.AltScreen = true
	return view
}
func main() {
	transport := flag.String("transport", "png", "transport (png)")
	frames := flag.Int("frames", 60, "frames to encode and present")
	output := flag.String("output", "", "JSON report filename")
	width := flag.Int("width", 1280, "raster width, a positive multiple of 80")
	height := flag.Int("height", 960, "raster height, a positive multiple of 30")
	interval := flag.Duration("interval", 16*time.Millisecond, "pause between presentations")
	flag.Parse()
	if *frames < 1 || *frames > 10000 || *width < 80 || *height < 30 || *width%80 != 0 || *height%30 != 0 || *interval < 0 {
		fmt.Fprintln(os.Stderr, "invalid dimensions, frames or interval")
		os.Exit(2)
	}
	if *transport != "png" {
		fmt.Fprintln(os.Stderr, "only png transport is supported")
		os.Exit(2)
	}
	m := &model{pic: picture.NewWithConfig(picture.Config{Fit: picture.FitFill, CellPixelWidth: *width / 80, CellPixelHeight: *height / 30}), frames: *frames, width: *width, height: *height, interval: *interval}
	m.pic.SetSize(80, 30)
	_, runErr := tea.NewProgram(m).Run()
	result := struct {
		Requested      string `json:"requested"`
		Width, Height  int
		ElapsedSeconds float64  `json:"elapsed_seconds"`
		Samples        []sample `json:"samples"`
		Error          string   `json:"error,omitempty"`
	}{Requested: *transport, Width: *width, Height: *height, ElapsedSeconds: time.Since(m.started).Seconds(), Samples: m.samples}
	for _, err := range []error{m.err, runErr} {
		if err != nil {
			result.Error += err.Error() + "; "
		}
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	if *output != "" {
		if err := os.WriteFile(*output, append(data, '\n'), 0600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if result.Error != "" {
		os.Exit(1)
	}
}
