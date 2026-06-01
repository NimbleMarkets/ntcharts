// examples/picture/simple/main.go
//
// The minimal picture demo: a single embedded image rendered with
// picture.Model, sized to fill the terminal. Press 'q' or ctrl+c to quit.
//
// The core picture package renders an image.Image and registers no decoders,
// so the program decodes its own source. This file imports just image/png for
// the embedded PNG below (the minimal "bring your own decoder" path). For all
// common formats in one line instead, blank-import the helper:
//
//	import _ "github.com/NimbleMarkets/ntcharts/v2/picture/decoders"
//
// which registers PNG, JPEG, GIF, WebP, BMP, and TIFF. examples/picture/main.go
// uses that helper.
//
// For the fuller demo (HTTP fetching, fit cycling, Glyph/Kitty toggle, and
// a Kitty-capability badge) see examples/picture/main.go.
package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	_ "image/png" // decode the embedded PNG asset
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

// kittyID is distinct from the 4242/4243 used by examples/picture/main.go so
// the two demos don't collide if a terminal reuses Kitty image IDs.
const kittyID = 4244

//go:embed Fuji-01.png
var fujiPNG []byte

type model struct {
	pic           picture.Model
	width, height int
}

func initialModel() (model, error) {
	pic := picture.NewWithConfig(picture.Config{KittyID: kittyID})

	img, _, err := image.Decode(bytes.NewReader(fujiPNG))
	if err != nil {
		return model{}, fmt.Errorf("decode embedded image: %w", err)
	}
	// SetImage's mutation is captured in pic before it goes into the model;
	// its Cmd is nil in the default Glyph mode, so it's safe to discard.
	_ = pic.SetImage(img)

	return model{pic: pic}, nil
}

// Init kicks off the picture's cell-size and Kitty-capability probes.
// picture.Model.Init only returns Cmds (it mutates no state), so it's fine
// to call here on the value receiver's copy.
func (m model) Init() tea.Cmd {
	return m.pic.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve one bottom row for the footer.
		if c := m.pic.SetSize(m.width, m.height-1); c != nil {
			cmds = append(cmds, c)
		}
	}

	if c := m.pic.Update(msg); c != nil {
		cmds = append(cmds, c)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	footer := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color("242")).
		Render("q quit")

	imageBox := lipgloss.Place(m.width, m.height-1,
		lipgloss.Center, lipgloss.Center, m.pic.View().Content)

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, imageBox, footer))
	v.AltScreen = true
	return v
}

func main() {
	m, err := initialModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
