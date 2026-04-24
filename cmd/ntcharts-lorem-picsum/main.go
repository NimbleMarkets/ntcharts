// examples/picture/main.go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture/pictureurl"
)

const (
	catalogPages = 3
	catalogLimit = 100
	startingID   = "110"
	kittyIDRight = 44
)

type model struct {
	leftPic  pictureurl.Model // Glyph
	rightPic pictureurl.Model // Kitty

	items  []picsumItem
	cursor int

	width, height int

	inputMode bool
	inputBuf  string

	status string
}

func initialModel() model {
	leftPic := pictureurl.New()
	rightPic := pictureurl.NewWithConfig(pictureurl.Config{KittyID: kittyIDRight})
	// right starts in Glyph by default; flip to Kitty. URL is empty so the
	// returned Cmd is nil and can be ignored here.
	_ = rightPic.Toggle()

	return model{
		leftPic:  leftPic,
		rightPic: rightPic,
		status:   "Loading catalog…",
	}
}

func (m model) Init() tea.Cmd {
	return fetchListCmd(catalogPages, catalogLimit)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.inputMode {
			switch msg.String() {
			case "esc":
				m.inputMode = false
				m.inputBuf = ""
				m.status = "Jump cancelled"
			case "enter":
				idx := findIndex(m.items, m.inputBuf)
				if idx < 0 {
					m.status = fmt.Sprintf("ID %q not in catalog", m.inputBuf)
				} else {
					m.cursor = idx
					cmds = append(cmds, m.setCurrentURL()...)
					m.status = fmt.Sprintf("Jumped to ID %s (%d of %d)",
						m.items[m.cursor].ID, m.cursor+1, len(m.items))
				}
				m.inputMode = false
				m.inputBuf = ""
			case "backspace":
				if n := len(m.inputBuf); n > 0 {
					m.inputBuf = m.inputBuf[:n-1]
				}
			default:
				if s := msg.String(); len(s) == 1 && s[0] >= '0' && s[0] <= '9' {
					m.inputBuf += s
				}
			}
			break
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "left", "h":
			if len(m.items) > 0 && m.cursor > 0 {
				m.cursor--
				cmds = append(cmds, m.setCurrentURL()...)
				m.status = fmt.Sprintf("ID %s (%d of %d)",
					m.items[m.cursor].ID, m.cursor+1, len(m.items))
			}

		case "right", "l":
			if len(m.items) > 0 && m.cursor < len(m.items)-1 {
				m.cursor++
				cmds = append(cmds, m.setCurrentURL()...)
				m.status = fmt.Sprintf("ID %s (%d of %d)",
					m.items[m.cursor].ID, m.cursor+1, len(m.items))
			}

		case "g":
			if len(m.items) > 0 {
				m.inputMode = true
				m.inputBuf = ""
			}

		case "r":
			if len(m.items) == 0 {
				m.status = "Retrying catalog…"
				cmds = append(cmds, fetchListCmd(catalogPages, catalogLimit))
			} else {
				if c := m.leftPic.Reload(); c != nil {
					cmds = append(cmds, c)
				}
				if c := m.rightPic.Reload(); c != nil {
					cmds = append(cmds, c)
				}
				m.status = fmt.Sprintf("Reloaded ID %s", m.items[m.cursor].ID)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		cmds = append(cmds, m.applyLayout()...)

	case picsumListLoadedMsg:
		if msg.err != nil {
			m.status = "Catalog error: " + msg.err.Error()
			return m, nil
		}
		m.items = msg.items
		idx := findIndex(m.items, startingID)
		if idx < 0 {
			idx = 0
		}
		m.cursor = idx
		m.status = fmt.Sprintf("Loaded %d images", len(m.items))
		cmds = append(cmds, m.setCurrentURL()...)
	}

	// Forward every msg to each pane; Update routes its own types and ignores others.
	if c := m.leftPic.Update(msg); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.rightPic.Update(msg); c != nil {
		cmds = append(cmds, c)
	}

	return m, tea.Batch(cmds...)
}

// setCurrentURL points both panes at the URL for items[cursor]. It re-applies
// the layout first so the panes are sized for the new item's aspect ratio
// before the fetch completes.
func (m *model) setCurrentURL() []tea.Cmd {
	if len(m.items) == 0 {
		return nil
	}
	cmds := m.applyLayout()
	url := imageURL(m.items[m.cursor].ID)
	if c := m.leftPic.SetURL(url); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.rightPic.SetURL(url); c != nil {
		cmds = append(cmds, c)
	}
	return cmds
}

// layoutDims holds the pane geometry for the current window size and the
// currently selected item's aspect ratio.
type layoutDims struct {
	innerCols, innerRows int // bordered-pane inner area (width/height)
	fitCols, fitRows     int // aspect-preserved image cell region (≤ inner*)
	tooSmall             bool
}

const (
	minWidth  = 24
	minHeight = 16
)

// Height budget: 1 title + 1 blank + Hpic (bordered panes) + 1 blank
// + 7 details (bordered, 5 content) + 1 blank + 1 footer = H, so Hpic = H-12
// and inner rows per pane = Hpic - 2.
// Width: two panes + 1-col gutter fill W, so pane outer = (W-1)/2 and
// inner cols = pane outer - 2.
func (m *model) layout() layoutDims {
	var d layoutDims
	if m.width < minWidth || m.height < minHeight {
		d.tooSmall = true
		return d
	}
	paneOuter := (m.width - 1) / 2
	d.innerCols = paneOuter - 2
	d.innerRows = m.height - 14
	if d.innerCols < 1 {
		d.innerCols = 1
	}
	if d.innerRows < 1 {
		d.innerRows = 1
	}

	imgW, imgH := 0, 0
	if len(m.items) > 0 && m.cursor >= 0 && m.cursor < len(m.items) {
		imgW = m.items[m.cursor].Width
		imgH = m.items[m.cursor].Height
	}
	d.fitCols, d.fitRows = fitCells(imgW, imgH, d.innerCols, d.innerRows)
	return d
}

// fitCells scales (imgW, imgH) image pixels into a (maxCols × maxRows) cell
// region while preserving aspect ratio. Assumes terminal cells have a 1:2
// width:height pixel ratio (so one cell row covers ~2 pixel rows of image).
// Returns the dimensions of the cell region to use.
func fitCells(imgW, imgH, maxCols, maxRows int) (int, int) {
	if imgW <= 0 || imgH <= 0 || maxCols <= 0 || maxRows <= 0 {
		return maxCols, maxRows
	}
	rows := maxRows
	cols := 2 * rows * imgW / imgH
	if cols > maxCols {
		cols = maxCols
		rows = cols * imgH / (2 * imgW)
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}

// applyLayout sizes both panes to the current aspect-preserved fit.
func (m *model) applyLayout() []tea.Cmd {
	d := m.layout()
	if d.tooSmall {
		return nil
	}
	var cmds []tea.Cmd
	if c := m.leftPic.SetSize(d.fitCols, d.fitRows); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.rightPic.SetSize(d.fitCols, d.fitRows); c != nil {
		cmds = append(cmds, c)
	}
	return cmds
}

func (m model) View() tea.View {
	d := m.layout()
	if d.tooSmall {
		return tea.NewView(lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("9")).
			Render(fmt.Sprintf("Terminal too small (%d × %d)\nneed at least %d × %d",
				m.width, m.height, minWidth, minHeight)))
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Width(m.width).
		Align(lipgloss.Center).
		Render("🖼️  ntcharts · picture — picsum browser")

	// lipgloss v2 Width/Height are OUTER dimensions (including border).
	// Add 2 so the inner content area equals d.innerCols × d.innerRows,
	// which matches the size the picture model is rendering at.
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(d.innerCols + 2).
		Height(d.innerRows + 2).
		Align(lipgloss.Center, lipgloss.Center)

	leftBox := paneStyle.Render(m.leftPic.View().Content)
	rightBox := paneStyle.Render(m.rightPic.View().Content)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)

	var detailsBody string
	switch {
	case m.inputMode:
		detailsBody = fmt.Sprintf("Jump to ID: %s▌   (Enter confirms, Esc cancels)", m.inputBuf)
	case len(m.items) == 0:
		detailsBody = m.status
	default:
		it := m.items[m.cursor]
		detailsBody = fmt.Sprintf(
			"ID: %s   Author: %s\nNative: %d × %d\nURL: %s\nDownload: %s\n[%d of %d]   %s",
			it.ID, it.Author,
			it.Width, it.Height,
			it.URL,
			it.DownloadURL,
			m.cursor+1, len(m.items),
			m.status,
		)
	}

	details := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(m.width - 2).
		Height(5).
		Render(detailsBody)

	footer := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color("242")).
		Render("←/→ prev·next   g jump   r reload   q quit")

	return tea.NewView(
		title + "\n\n" +
			panes + "\n\n" +
			details + "\n\n" +
			footer,
	)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
