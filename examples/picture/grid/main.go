package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	booba "github.com/NimbleMarkets/go-booba"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	uv "github.com/charmbracelet/ultraviolet"
)

const (
	boardCols           = 14
	boardRows           = 8
	terminalColsPerCell = 2
	kittyID             = 5151
)

type point struct {
	x int
	y int
}

type model struct {
	pic picture.Model

	player point
	target point
	walls  map[point]bool

	score int
}

func initialModel() model {
	pic := picture.NewWithConfig(picture.Config{
		KittyID: kittyID,
		Fit:     picture.FitFill,
	})

	m := model{
		pic:    pic,
		player: point{1, 1},
		target: point{11, 5},
		walls: map[point]bool{
			{4, 2}: true,
			{5, 2}: true,
			{6, 2}: true,
			{6, 3}: true,
			{6, 4}: true,
			{9, 5}: true,
			{9, 6}: true,
		},
	}

	cols, rows := terminalSize()
	_ = m.pic.SetSize(cols, rows)
	_ = m.renderBoard()
	return m
}

func (m model) Init() tea.Cmd {
	return m.pic.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	redraw := false
	forwardPicture := true

	switch msg := msg.(type) {
	case tea.KeyMsg:
		next := m.player
		moved := false
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "g":
			if c := m.pic.Toggle(); c != nil {
				cmds = append(cmds, c)
			}
		case "left", "a", "h":
			next.x--
			moved = true
		case "right", "d", "l":
			next.x++
			moved = true
		case "up", "w", "k":
			next.y--
			moved = true
		case "down", "s", "j":
			next.y++
			moved = true
		}

		if moved && m.canMove(next) {
			m.player = next
			if m.player == m.target {
				m.score++
				m.target = m.nextTarget()
			}
			redraw = true
		}
	case uv.CellSizeEvent:
		// Recompose the source bitmap at the terminal's real cell size.
		if c := m.pic.SetCellPixelSize(msg.Width, msg.Height); c != nil {
			cmds = append(cmds, c)
		}
		redraw = true
		// SetCellPixelSize was applied above; forwarding would apply it twice.
		forwardPicture = false
	}

	if forwardPicture {
		if c := m.pic.Update(msg); c != nil {
			cmds = append(cmds, c)
		}
	}
	if redraw {
		if c := m.renderBoard(); c != nil {
			cmds = append(cmds, c)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	board := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		Render(m.pic.View().Content)

	mode := "Glyph"
	if m.pic.Mode() == picture.PictureKitty {
		mode = "Kitty"
	}
	cols, rows := terminalSize()
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Render("GridImage")
	header := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", m.kittyBadge())
	status := fmt.Sprintf("mode: %s  pos: %d,%d  target: %d,%d  score: %d  size: %dx%d",
		mode, m.player.x, m.player.y, m.target.x, m.target.y, m.score, cols, rows)
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(
		"arrows/WASD move  g toggle Kitty  q quit",
	)

	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, status, board, help))
	view.AltScreen = true
	return view
}

func (m model) kittyBadge() string {
	label := "KITTY CHECKING"
	fg := lipgloss.Color("0")
	bg := lipgloss.Color("11")
	switch m.pic.KittySupported() {
	case picture.KittyCapabilitySupported:
		label = "KITTY AVAILABLE"
		bg = lipgloss.Color("10")
	case picture.KittyCapabilityUnsupported:
		label = "KITTY UNAVAILABLE"
		bg = lipgloss.Color("9")
	}
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(fg).
		Background(bg).
		Padding(0, 1).
		Render(label)
}

func (m model) canMove(p point) bool {
	if p.x < 0 || p.x >= boardCols || p.y < 0 || p.y >= boardRows {
		return false
	}
	return !m.walls[p]
}

func (m model) nextTarget() point {
	for y := 0; y < boardRows; y++ {
		for x := 0; x < boardCols; x++ {
			p := point{
				x: (m.target.x + x + 3) % boardCols,
				y: (m.target.y + y + 2) % boardRows,
			}
			if p != m.player && !m.walls[p] {
				return p
			}
		}
	}
	return m.target
}

func terminalSize() (cols, rows int) {
	return boardCols * terminalColsPerCell, boardRows
}

func (m *model) renderBoard() tea.Cmd {
	cellW, cellH := m.pic.CellPixelSize()
	grid := picture.NewGridImage(picture.GridImageConfig{
		LogicalCols:         boardCols,
		LogicalRows:         boardRows,
		TerminalColsPerCell: terminalColsPerCell,
		TerminalRowsPerCell: 1,
		CellPixelWidth:      cellW,
		CellPixelHeight:     cellH,
		Background:          boardBackground(),
	})

	// DrawOverlay keeps full-board effects separate from cell sprite placement.
	gridLines := image.NewRGBA(grid.Image().Bounds())
	drawLogicGrid(gridLines, grid.CellRect(0, 0))
	grid.DrawOverlay(gridLines)
	for p := range m.walls {
		grid.DrawCell(p.x, p.y, wallSprite())
	}
	grid.DrawCell(m.target.x, m.target.y, targetSprite())
	grid.DrawCell(m.player.x, m.player.y, playerSprite())

	cols, rows := grid.TerminalSize()
	if c := m.pic.SetSize(cols, rows); c != nil {
		return tea.Batch(c, m.pic.SetImage(grid.Image()))
	}
	return m.pic.SetImage(grid.Image())
}

func boardBackground() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	fill(img, img.Bounds(), color.RGBA{20, 24, 32, 255})
	fill(img, image.Rect(0, 0, 64, 8), color.RGBA{27, 36, 47, 255})
	fill(img, image.Rect(0, 56, 64, 64), color.RGBA{15, 18, 26, 255})
	return img
}

func playerSprite() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	fillCircle(img, image.Rect(8, 8, 56, 56), color.RGBA{96, 214, 151, 255})
	fillCircle(img, image.Rect(25, 18, 31, 24), color.RGBA{8, 20, 15, 255})
	fillCircle(img, image.Rect(38, 18, 44, 24), color.RGBA{8, 20, 15, 255})
	fill(img, image.Rect(25, 39, 43, 43), color.RGBA{14, 56, 38, 255})
	return img
}

func targetSprite() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	fillCircle(img, image.Rect(14, 12, 50, 52), color.RGBA{230, 83, 91, 255})
	fillCircle(img, image.Rect(24, 8, 36, 20), color.RGBA{110, 184, 91, 255})
	fillCircle(img, image.Rect(28, 20, 38, 30), color.RGBA{255, 160, 165, 255})
	return img
}

func wallSprite() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	fill(img, image.Rect(5, 5, 59, 59), color.RGBA{78, 88, 101, 255})
	fill(img, image.Rect(9, 9, 55, 55), color.RGBA{96, 110, 126, 255})
	fill(img, image.Rect(12, 12, 52, 24), color.RGBA{126, 140, 155, 255})
	fill(img, image.Rect(12, 37, 52, 49), color.RGBA{65, 74, 87, 255})
	return img
}

func drawLogicGrid(img *image.RGBA, cell image.Rectangle) {
	if cell.Empty() {
		return
	}
	gridColor := color.RGBA{70, 196, 210, 80}
	bounds := img.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x += cell.Dx() {
		fill(img, image.Rect(x, bounds.Min.Y, x+1, bounds.Max.Y), gridColor)
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y += cell.Dy() {
		fill(img, image.Rect(bounds.Min.X, y, bounds.Max.X, y+1), gridColor)
	}
}

func fill(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	draw.Draw(img, rect.Intersect(img.Bounds()), &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func fillCircle(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() {
		return
	}
	cx2 := rect.Min.X + rect.Max.X - 1
	cy2 := rect.Min.Y + rect.Max.Y - 1
	rx := rect.Dx()
	ry := rect.Dy()
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		dy := 2*y - cy2
		for x := rect.Min.X; x < rect.Max.X; x++ {
			dx := 2*x - cx2
			if dx*dx*ry*ry+dy*dy*rx*rx <= rx*rx*ry*ry {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func main() {
	if err := booba.Run(initialModel()); err != nil {
		log.Fatal(err)
	}
}
