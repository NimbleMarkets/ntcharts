// examples/picture/main.go
//
// A small two-pane demo:
//   - Left pane uses picture.Model with a procedurally-generated image
//     (showing the source-agnostic base API).
//   - Right pane uses pictureurl.Model fetching a fixed picsum URL,
//     with the URL drawn as a caption near the bottom of the box.
//
// Press 'g' to toggle both panes between Glyph and Kitty rendering.
// Press 'q' or ctrl+c to quit.
package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	booba "github.com/NimbleMarkets/go-booba"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/NimbleMarkets/ntcharts/v2/picture/pictureurl"
)

const (
	kittyIDLeft  = 4242
	kittyIDRight = 4243

	imageURL = "https://i0.wp.com/www.hypertalking.com/wp-content/uploads/2023/05/Fuji-01.png"

	leftCaption1 = "01 of 36 view of Mt Fuji by hypertalking"
	leftCaption2 = imageURL
	rightURL     = imageURL

	leftTopLabel  = "picture with embed"
	rightTopLabel = "pictureurl with http"

	// Many image hosts reject the stock Go HTTP User-Agent with a 403,
	// so the example sets an identifying UA via Config.UserAgent.
	userAgent = "ntcharts-picture-example/1.0 (https://github.com/NimbleMarkets/ntcharts)"
)

//go:embed Fuji-01.png
var fujiPNG []byte

type model struct {
	leftPic  picture.Model
	rightPic pictureurl.Model

	// Captured from initialModel; returned from Init. Must be set inside the
	// constructor because Init has a value receiver and cannot persist
	// mutations (e.g. SetURL) made on its local copy of the model.
	initCmd tea.Cmd

	width, height int
}

func initialModel() model {
	left := picture.NewWithConfig(picture.Config{KittyID: kittyIDLeft})
	right := pictureurl.NewWithConfig(pictureurl.Config{
		KittyID:   kittyIDRight,
		UserAgent: userAgent,
	})

	img, _, err := image.Decode(bytes.NewReader(fujiPNG))
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode embedded image: %v\n", err)
		os.Exit(1)
	}
	_ = left.SetImage(img)

	// Kick off the right-pane fetch here so the SetURL state mutation
	// (currentURL, loading flag) is captured in the model returned to
	// bubbletea. Init() can only return the Cmd — not a mutated model.
	initCmd := tea.Batch(left.Init(), right.Init(), right.SetURL(rightURL))

	return model{leftPic: left, rightPic: right, initCmd: initCmd}
}

func (m model) Init() tea.Cmd {
	return m.initCmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "g":
			if c := m.leftPic.Toggle(); c != nil {
				cmds = append(cmds, c)
			}
			if c := m.rightPic.Toggle(); c != nil {
				cmds = append(cmds, c)
			}
		case "f":
			next := nextFit(m.leftPic.Fit())
			if c := m.leftPic.SetFit(next); c != nil {
				cmds = append(cmds, c)
			}
			if c := m.rightPic.SetFit(next); c != nil {
				cmds = append(cmds, c)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		cmds = append(cmds, m.applyLayout()...)
	}

	if c := m.leftPic.Update(msg); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.rightPic.Update(msg); c != nil {
		cmds = append(cmds, c)
	}

	// Re-apply layout each tick. SetSize is a no-op when dims are
	// unchanged, so this is cheap; it picks up the row count change
	// when rightPic.Err() transitions in or out (a one- or two-line
	// status bar appears/disappears between panes and footer).
	cmds = append(cmds, m.applyLayout()...)

	return m, tea.Batch(cmds...)
}

type layoutDims struct {
	innerCols, innerRows int
	leftCaptions         []string
	rightCaptions        []string
	// errorLines is the wrapped pictureurl-error text shown as a status
	// bar between panes and footer. Capped at 2 lines; the bar's vertical
	// space is reserved out of innerRows so panes shrink to make room.
	errorLines []string
	tooSmall   bool
}

const (
	minWidth  = 32
	minHeight = 12
)

// Layout: 1 title + Hpic + N error rows + 1 footer = m.height, so
// Hpic = m.height - 2 - N and pane inner rows = Hpic - 2. When an error
// is active, the status bar wraps to at most two lines and panes shrink
// by that many rows to keep the footer on-screen.
func (m *model) layout() layoutDims {
	var d layoutDims
	if m.width < minWidth || m.height < minHeight {
		d.tooSmall = true
		return d
	}
	paneOuter := (m.width - 1) / 2
	d.innerCols = paneOuter - 2
	if d.innerCols < 1 {
		d.innerCols = 1
	}
	if err := m.rightPic.Err(); err != nil {
		d.errorLines = wrapToLines(fmt.Sprintf("right: error: %v", err), m.width, 2)
	}
	d.innerRows = m.height - 4 - len(d.errorLines)
	if d.innerRows < 1 {
		d.innerRows = 1
	}
	d.leftCaptions = append([]string{leftCaption1}, wrapToLines(leftCaption2, d.innerCols, 2)...)
	d.rightCaptions = d.leftCaptions
	return d
}

func (m *model) applyLayout() []tea.Cmd {
	d := m.layout()
	if d.tooSmall {
		return nil
	}
	var cmds []tea.Cmd
	// Each pane reserves 1 top row for the label and bottom rows for caption lines.
	leftRows := d.innerRows - len(d.leftCaptions) - 1
	if leftRows < 1 {
		leftRows = 1
	}
	rightRows := d.innerRows - len(d.rightCaptions) - 1
	if rightRows < 1 {
		rightRows = 1
	}
	if c := m.leftPic.SetSize(d.innerCols, leftRows); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.rightPic.SetSize(d.innerCols, rightRows); c != nil {
		cmds = append(cmds, c)
	}
	return cmds
}

func (m model) View() tea.View {
	d := m.layout()
	if d.tooSmall {
		v := tea.NewView(lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("9")).
			Render(fmt.Sprintf("Terminal too small (%d × %d)\nneed at least %d × %d",
				m.width, m.height, minWidth, minHeight)))
		v.AltScreen = true
		return v
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Width(m.width).
		Align(lipgloss.Center).
		Render("🖼️  ntcharts · picture")

	// lipgloss v2 Width/Height are outer dimensions; +2 makes the inner
	// content area equal d.innerCols × d.innerRows (matching pic SetSize).
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(d.innerCols+2).
		Height(d.innerRows+2).
		Align(lipgloss.Center, lipgloss.Center)

	leftContent := buildPane(m.leftPic.View().Content, leftTopLabel,
		d.leftCaptions, d.innerCols, d.innerRows)
	// When the right fetch errored, pictureurl.View() returns the long
	// error string; that overflows innerCols and visually wraps inside
	// the bordered pane, making the right look taller than the left.
	// The error is already reported in the status bar below — keep the
	// pane interior empty so both panes render identically.
	rightInner := m.rightPic.View().Content
	if m.rightPic.Err() != nil {
		rightInner = ""
	}
	rightContent := buildPane(rightInner, rightTopLabel,
		d.rightCaptions, d.innerCols, d.innerRows)

	leftBox := paneStyle.Render(leftContent)
	rightBox := paneStyle.Render(rightContent)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)

	mode := "Glyph"
	if m.leftPic.Mode() == picture.PictureKitty {
		mode = "Kitty"
	}

	// Footer: left-aligned status + keybindings, plus a right-aligned
	// Kitty-capability badge. 'g toggle' is appended last and only
	// when Kitty is affirmatively Supported — during the probe window
	// (Unknown) and on non-Kitty terminals (Unsupported), Toggle is a
	// no-op so we hide the hint to match.
	cap := m.leftPic.KittySupported()
	leftParts := []string{
		fmt.Sprintf("mode: %s", mode),
		fmt.Sprintf("fit: %s", fitName(m.leftPic.Fit())),
		"f cycle fit",
		"q quit",
	}
	if cap == picture.KittyCapabilitySupported {
		leftParts = append(leftParts, "g toggle")
	}
	leftText := strings.Join(leftParts, "   ")

	badgeText, badgeColor := kittyBadge(cap)
	badge := lipgloss.NewStyle().
		Foreground(lipgloss.Color(badgeColor)).
		Bold(true).
		Render(badgeText)

	leftWidth := m.width - lipgloss.Width(badge)
	if leftWidth < 0 {
		leftWidth = 0
	}
	leftRendered := lipgloss.NewStyle().
		Width(leftWidth).
		Foreground(lipgloss.Color("242")).
		Render(leftText)

	footer := lipgloss.JoinHorizontal(lipgloss.Top, leftRendered, badge)

	parts := []string{title, panes}
	if len(d.errorLines) > 0 {
		errStyle := lipgloss.NewStyle().
			Width(m.width).
			Foreground(lipgloss.Color("9"))
		for _, line := range d.errorLines {
			parts = append(parts, errStyle.Render(line))
		}
	}
	parts = append(parts, footer)

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, parts...))
	v.AltScreen = true
	return v
}

// buildPane composes a pane's content: a top label row, picture content in
// the middle (centered), and caption lines pinned to the bottom. The picture
// model is expected to be sized so its content fits inside the middle area;
// shorter status text ("Loading…", "Image error: …") gets centered without
// clobbering the labels or captions.
func buildPane(content, top string, captions []string, innerCols, innerRows int) string {
	topStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("12")).
		Foreground(lipgloss.Color("0")).
		Bold(true).
		Width(innerCols).
		Align(lipgloss.Center)
	captionStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("15")).
		Width(innerCols).
		Align(lipgloss.Center)

	lines := make([]string, 0, len(captions)+2)
	lines = append(lines, topStyle.Render(truncate(top, innerCols)))

	upperRows := innerRows - 1 - len(captions)
	if upperRows >= 1 {
		lines = append(lines,
			lipgloss.Place(innerCols, upperRows, lipgloss.Center, lipgloss.Center, content))
	}
	for _, c := range captions {
		lines = append(lines, captionStyle.Render(truncate(c, innerCols)))
	}
	return strings.Join(lines, "\n")
}

func nextFit(f picture.FitMode) picture.FitMode {
	switch f {
	case picture.FitContain:
		return picture.FitFill
	case picture.FitFill:
		return picture.FitCover
	default:
		return picture.FitContain
	}
}

func fitName(f picture.FitMode) string {
	switch f {
	case picture.FitFill:
		return "Fill"
	case picture.FitCover:
		return "Cover"
	default:
		return "Contain"
	}
}

// kittyBadge returns the right-aligned footer badge text and its ANSI
// color: green for Supported, red for Unsupported, yellow during the
// brief Unknown probe window. The Unsupported text disambiguates the
// failure mode so users can diagnose: "no env" means we didn't probe
// (no recognized terminal env vars); "no response" means the probe
// went out but the terminal didn't reply within the timeout.
func kittyBadge(c picture.KittyCapability) (text, color string) {
	switch c {
	case picture.KittyCapabilitySupported:
		return "[ Kitty: yes ]", "10" // bright green
	case picture.KittyCapabilityUnsupported:
		if picture.KittyEnvSignalled() {
			return "[ Kitty: no (no response) ]", "9"
		}
		return "[ Kitty: no (no env) ]", "9"
	default:
		return "[ Kitty: probing… ]", "11" // bright yellow
	}
}

func truncate(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return s[:width]
	}
	// Naive byte-truncate with ellipsis; fine for ASCII URLs.
	return s[:width-1] + "…"
}

// wrapToLines returns up to maxLines lines of width cells each, splitting s on
// rune boundaries. If s is longer than width*maxLines, the last line is
// truncated with an ellipsis. Suitable for ASCII URLs; not Unicode-aware
// beyond rune boundaries.
func wrapToLines(s string, width, maxLines int) []string {
	if width <= 0 || maxLines <= 0 {
		return nil
	}
	if lipgloss.Width(s) <= width {
		return []string{s}
	}
	runes := []rune(s)
	out := make([]string, 0, maxLines)
	for len(runes) > 0 && len(out) < maxLines {
		if len(out) == maxLines-1 && len(runes) > width {
			if width <= 1 {
				out = append(out, string(runes[:width]))
			} else {
				out = append(out, string(runes[:width-1])+"…")
			}
			break
		}
		take := width
		if take > len(runes) {
			take = len(runes)
		}
		out = append(out, string(runes[:take]))
		runes = runes[take:]
	}
	return out
}

func main() {
	// booba.Run is a tea.Program substitute that dispatches to native Bubble Tea
	// or the WASM/ghostty-web bridge depending on build target.
	// See https://github.com/NimbleMarkets/go-booba-example.
	if err := booba.Run(initialModel()); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
