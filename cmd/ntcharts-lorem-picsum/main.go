// examples/picture/main.go
package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/NimbleMarkets/ntcharts/v2/picture/pictureurl"
)

const (
	catalogPages    = 3
	catalogLimit    = 100
	startingID      = "110"
	kittyIDRight    = 44
	imageCacheLimit = 8
)

type model struct {
	leftPic      pictureurl.Model // Glyph
	rightPic     pictureurl.Model // Kitty (deferred toggle until probe resolves)
	rightToggled bool             // true once rightPic has been switched into Kitty mode

	items  []picsumItem
	cursor int

	catalogMode  bool
	catalogTable table.Model
	catalogRows  []int
	catalogSort  catalogSort
	filterMode   bool
	authorFilter string

	width, height int

	inputMode bool
	inputBuf  string

	status string
}

func initialModel() model {
	leftPic := pictureurl.NewWithConfig(pictureurl.Config{CacheLimit: imageCacheLimit})
	rightPic := pictureurl.NewWithConfig(pictureurl.Config{KittyID: kittyIDRight, CacheLimit: imageCacheLimit})
	// rightPic should render in Kitty mode, but Toggle() silently no-ops when
	// the Kitty capability is still Unknown — and the probe hasn't run yet at
	// construction time. The toggle is retried in Update once the probe
	// resolves (see ensureRightKitty).

	return model{
		leftPic:      leftPic,
		rightPic:     rightPic,
		catalogTable: newCatalogTable(),
		status:       "Loading catalog…",
	}
}

func (m model) Init() tea.Cmd {
	// Both pane Init()s must be called so the picture.Model issues the
	// Kitty support probe and cell-size query; otherwise capability stays
	// Unknown forever and the rightPic deferred toggle never fires.
	return tea.Batch(
		m.leftPic.Init(),
		m.rightPic.Init(),
		fetchListCmd(catalogPages, catalogLimit),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

		keyHandled := false
		if m.catalogMode {
			if m.filterMode {
				switch msg.String() {
				case "esc":
					m.filterMode = false
				case "enter":
					m.filterMode = false
				case "backspace":
					if len(m.authorFilter) > 0 {
						m.authorFilter = m.authorFilter[:len(m.authorFilter)-1]
						m.updateCatalogTable()
					}
				default:
					if s := msg.String(); len(s) == 1 {
						m.authorFilter += s
						m.updateCatalogTable()
					}
				}
			} else {
				switch msg.String() {
				case "esc":
					m.catalogMode = false
					m.status = "Returned to image"
				case "/":
					m.filterMode = true
				case "s":
					m.catalogSort = m.catalogSort.next()
					m.updateCatalogTable()
				case "enter":
					if idx := m.selectedCatalogIndex(); idx >= 0 {
						m.cursor = idx
						m.catalogMode = false
						cmds = append(cmds, m.setCurrentURL()...)
						m.status = fmt.Sprintf("Selected ID %s (%d of %d)",
							m.items[m.cursor].ID, m.cursor+1, len(m.items))
					}
				default:
					var c tea.Cmd
					m.catalogTable, c = m.catalogTable.Update(msg)
					if c != nil {
						cmds = append(cmds, c)
					}
				}
			}
			keyHandled = true
		}

		if !keyHandled && m.inputMode {
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
			keyHandled = true
		}
		if !keyHandled {
			switch msg.String() {
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

			case "c":
				if len(m.items) == 0 {
					m.status = "Catalog is not loaded"
				} else {
					m.catalogMode = true
					m.updateCatalogTable()
					m.catalogTable.Focus()
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
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateCatalogTable()
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
		m.updateCatalogTable()
		cmds = append(cmds, m.setCurrentURL()...)
	}

	// Forward every msg to each pane; Update routes its own types and ignores others.
	if c := m.leftPic.Update(msg); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.rightPic.Update(msg); c != nil {
		cmds = append(cmds, c)
	}

	if c := m.ensureRightKitty(); c != nil {
		cmds = append(cmds, c)
	}

	return m, tea.Batch(cmds...)
}

// ensureRightKitty toggles rightPic into Kitty mode the first time the probe
// resolves to Supported. The eager Toggle() at construction time is a silent
// no-op while the capability is still Unknown.
func (m *model) ensureRightKitty() tea.Cmd {
	if m.rightToggled {
		return nil
	}
	if m.rightPic.KittySupported() != picture.KittyCapabilitySupported {
		return nil
	}
	m.rightToggled = true
	return m.rightPic.Toggle()
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

	catalogMinWidth  = 48
	catalogMinHeight = 8
)

type catalogSort int

const (
	sortIDAsc catalogSort = iota
	sortIDDesc
	sortAuthorAsc
	sortAuthorDesc
)

func (s catalogSort) next() catalogSort {
	switch s {
	case sortIDAsc:
		return sortIDDesc
	case sortIDDesc:
		return sortAuthorAsc
	case sortAuthorAsc:
		return sortAuthorDesc
	default:
		return sortIDAsc
	}
}

func (s catalogSort) String() string {
	switch s {
	case sortIDDesc:
		return "ID desc"
	case sortAuthorAsc:
		return "Author asc"
	case sortAuthorDesc:
		return "Author desc"
	default:
		return "ID asc"
	}
}

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
	d.innerRows = m.height - 15
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

func newCatalogTable() table.Model {
	styles := table.DefaultStyles()
	styles.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Padding(0, 1)
	styles.Selected = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("12"))

	return table.New(
		table.WithFocused(true),
		table.WithStyles(styles),
	)
}

func catalogRows(items []picsumItem, indices []int) []table.Row {
	rows := make([]table.Row, 0, len(indices))
	for _, idx := range indices {
		it := items[idx]
		rows = append(rows, table.Row{
			it.ID,
			fmt.Sprintf("%d × %d", it.Width, it.Height),
			it.Author,
			it.URL,
		})
	}
	return rows
}

func catalogIndices(items []picsumItem, filter string, sortBy catalogSort) []int {
	filter = strings.ToLower(strings.TrimSpace(filter))
	indices := make([]int, 0, len(items))
	for i, it := range items {
		if filter == "" || strings.Contains(strings.ToLower(it.Author), filter) {
			indices = append(indices, i)
		}
	}

	sort.SliceStable(indices, func(i, j int) bool {
		left, right := items[indices[i]], items[indices[j]]
		switch sortBy {
		case sortIDDesc:
			return compareIDs(left.ID, right.ID) > 0
		case sortAuthorAsc:
			if c := strings.Compare(strings.ToLower(left.Author), strings.ToLower(right.Author)); c != 0 {
				return c < 0
			}
			return compareIDs(left.ID, right.ID) < 0
		case sortAuthorDesc:
			if c := strings.Compare(strings.ToLower(left.Author), strings.ToLower(right.Author)); c != 0 {
				return c > 0
			}
			return compareIDs(left.ID, right.ID) < 0
		default:
			return compareIDs(left.ID, right.ID) < 0
		}
	})

	return indices
}

func compareIDs(left, right string) int {
	leftNum, leftErr := strconv.Atoi(left)
	rightNum, rightErr := strconv.Atoi(right)
	if leftErr == nil && rightErr == nil {
		switch {
		case leftNum < rightNum:
			return -1
		case leftNum > rightNum:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(left, right)
}

func catalogColumns(width int) []table.Column {
	contentWidth := width - 8 // table cell padding: two cells per four columns

	idWidth := 7
	sizeWidth := 12
	authorWidth := contentWidth / 3
	if authorWidth < 14 {
		authorWidth = 14
	}
	if authorWidth > 32 {
		authorWidth = 32
	}
	urlWidth := contentWidth - idWidth - sizeWidth - authorWidth
	if urlWidth < 12 {
		urlWidth = 12
		authorWidth = contentWidth - idWidth - sizeWidth - urlWidth
		if authorWidth < 8 {
			authorWidth = 8
		}
	}

	return []table.Column{
		{Title: "ID", Width: idWidth},
		{Title: "Native", Width: sizeWidth},
		{Title: "Author", Width: authorWidth},
		{Title: "Source", Width: urlWidth},
	}
}

func (m *model) updateCatalogTable() {
	prevSelection := m.cursor
	if selected := m.selectedCatalogIndex(); selected >= 0 {
		prevSelection = selected
	}

	if m.width > 0 {
		m.catalogTable.SetWidth(m.width)
		m.catalogTable.SetColumns(catalogColumns(m.width))
	}
	if m.height > 3 {
		m.catalogTable.SetHeight(m.height - 3)
	}
	m.catalogRows = catalogIndices(m.items, m.authorFilter, m.catalogSort)
	m.catalogTable.SetRows(catalogRows(m.items, m.catalogRows))
	if len(m.catalogRows) > 0 {
		cursor := 0
		for i, idx := range m.catalogRows {
			if idx == prevSelection {
				cursor = i
				break
			}
		}
		m.catalogTable.SetCursor(cursor)
	}
}

func (m model) selectedCatalogIndex() int {
	cursor := m.catalogTable.Cursor()
	if cursor < 0 || cursor >= len(m.catalogRows) {
		return -1
	}
	return m.catalogRows[cursor]
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
	if m.catalogMode {
		return m.catalogView()
	}

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

	badge := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(m.statusBadge())

	// lipgloss v2 Width/Height are OUTER dimensions (including border).
	// Add 2 so the inner content area equals d.innerCols × d.innerRows,
	// which matches the size the picture model is rendering at.
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(d.innerCols+2).
		Height(d.innerRows+2).
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
		Render("←/→ prev·next   c catalog   g jump   r reload   q quit")

	return tea.NewView(
		title + "\n" +
			badge + "\n\n" +
			panes + "\n\n" +
			details + "\n\n" +
			footer,
	)
}

// statusBadge formats the Kitty probe state and the current render mode of
// each pane on a single line. Green/yellow/red color the probe state to make
// at-a-glance terminal-capability debugging easier.
func (m model) statusBadge() string {
	probe := m.leftPic.KittySupported()
	probeColor := "11" // yellow (unknown)
	switch probe {
	case picture.KittyCapabilitySupported:
		probeColor = "10" // green
	case picture.KittyCapabilityUnsupported:
		probeColor = "9" // red
	}
	probeStr := lipgloss.NewStyle().
		Foreground(lipgloss.Color(probeColor)).
		Render("kitty:" + capabilityString(probe))

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	return dim.Render("[ ") + probeStr +
		dim.Render("  ·  L:") + modeBadge(m.leftPic.Mode()) +
		dim.Render("  ·  R:") + modeBadge(m.rightPic.Mode()) +
		dim.Render(" ]")
}

func capabilityString(c picture.KittyCapability) string {
	switch c {
	case picture.KittyCapabilitySupported:
		return "supported"
	case picture.KittyCapabilityUnsupported:
		return "unsupported"
	default:
		return "unknown"
	}
}

func modeBadge(mode picture.PictureMode) string {
	if mode == picture.PictureKitty {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("kitty")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("glyph")
}

func (m model) catalogView() tea.View {
	if m.width < catalogMinWidth || m.height < catalogMinHeight {
		return tea.NewView(lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("9")).
			Render(fmt.Sprintf("Terminal too small (%d × %d)\nneed at least %d × %d",
				m.width, m.height, catalogMinWidth, catalogMinHeight)))
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Width(m.width).
		Align(lipgloss.Center).
		Render("ntcharts · picsum catalog")

	filter := m.authorFilter
	if filter == "" {
		filter = "none"
	}
	prompt := ""
	if m.filterMode {
		prompt = "  author: " + m.authorFilter + "▌"
	}
	meta := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color("242")).
		Render(fmt.Sprintf("sort: %s   filter: %s   rows: %d/%d%s",
			m.catalogSort, filter, len(m.catalogRows), len(m.items), prompt))

	footer := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color("242")).
		Render("↑/↓ move   s sort   / author filter   enter select   esc image   q quit")

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		title,
		meta,
		m.catalogTable.View(),
		footer,
	))
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
