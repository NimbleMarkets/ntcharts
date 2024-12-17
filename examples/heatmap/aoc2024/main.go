// Heatmap displaying XMAS
// https://adventofcode.com/2024/day/4
//
// Download example data from URL above.
//
// go run examples/heatmap/main.go examples/heatmap/aoc2024/4.txt

package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/NimbleMarkets/ntcharts/heatmap"
	"github.com/charmbracelet/lipgloss"
)

///////////////////////////////////////////////////////////////////////////////
// Utility functions

func maxOf(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func signOf(x int) int {
	if x < 0 {
		return -1
	} else if x > 0 {
		return 1
	} else {
		return 0
	}
}

///////////////////////////////////////////////////////////////////////////////

var xmasColorScale = []lipgloss.Color{
	lipgloss.Color("#FF000"),  // red
	lipgloss.Color("#00FF00"), // green
	lipgloss.Color("#FFFFFF"), // white
}

type Board struct {
	puzzle     string
	lines      []string
	lenX, lenY int
}

func NewBoard(puzzle string) *Board {
	// extract all the lines
	lines := strings.Split(puzzle, "\n")
	if len(lines) == 0 {
		return nil
	}

	lenX, lenY := len(lines[0]), len(lines) // assumes uniform input

	return &Board{
		puzzle: puzzle,
		lines:  lines,
		lenX:   maxOf(0, lenX-1),
		lenY:   maxOf(0, lenY-1),
	}
}

func (b *Board) Puzzle() string {
	return b.puzzle
}

func (b *Board) Lines() []string {
	return b.lines
}

func (b *Board) LenX() int {
	return b.lenX
}

func (b *Board) LenY() int {
	return b.lenY
}

// CharAt returns the character at the given position.
// Returns 0 if out-of-bounds
func (b *Board) CharAt(x int, y int) byte {
	if x < 0 || x >= b.lenX || y < 0 || y >= b.lenY {
		// out of bounds, return empty rune
		return 0
	}
	return b.lines[y][x]
}

// StringLine returns a string of characters from the board of max `length`
// The sign of xdir/ydir express unit direction, 0 is no movement.
func (b *Board) StringLine(x, y, length, xdir, ydir int) string {
	// build the string via iteration
	signX, signY := signOf(xdir), signOf(ydir)
	var buffer bytes.Buffer
	for i := 0; i < length; i++ {
		if c := b.CharAt(x, y); c == 0 {
			// out of bounds, stop
			break
		} else {
			buffer.WriteByte(c)
		}
		x += signX
		y += signY
	}
	return buffer.String()
}

///////////////////////////////////////////////////////////////////////////////

type BoardCount struct {
	dataYX     [][]int // [y][x]
	lenX, lenY int
}

func NewBoardCount(lenX int, lenY int) BoardCount {
	var bc BoardCount
	for i := 0; i < lenY; i++ {
		bc.dataYX = append(bc.dataYX, make([]int, lenX)) // 0-filled
	}
	bc.lenX, bc.lenY = lenX, lenY
	return bc
}

func (bc *BoardCount) Data() [][]int {
	return bc.dataYX
}

func (bc *BoardCount) Mark(x, y, length, xdir, ydir int) {
	signX, signY := signOf(xdir), signOf(ydir)
	for i := 0; i < length; i++ {
		if x < 0 || x >= bc.lenX || y < 0 || y >= bc.lenY {
			// out of bounds, stop
			break
		}
		bc.dataYX[y][x]++
		x += signX
		y += signY
	}
}

///////////////////////////////////////////////////////////////////////////////

var letterStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Background(lipgloss.Color("000000"))

func countWordsDir(b *Board, counts *BoardCount, word string, x int, y int, xdir int, ydir int) bool {
	if word == b.StringLine(x, y, len(word), xdir, ydir) {
		counts.Mark(x, y, len(word), xdir, ydir)
		return true
	}
	return false
}

func makeHeatmap(board Board, word string) heatmap.Model {
	// make the heatmap
	hm := heatmap.New(board.LenX(), board.LenY(), heatmap.WithColorScale(xmasColorScale))
	hm.Canvas.Style = letterStyle
	word = strings.ToUpper(word)
	if word == "" {
		return hm
	}

	// collect hit counts across all the directions
	// also paint the canvas
	hm.Canvas.Clear()
	counts := NewBoardCount(board.LenX(), board.LenY())
	for y := 0; y < board.LenY(); y++ {
		for x := 0; x < board.LenX(); x++ {
			letter := board.lines[y][x]
			hm.Canvas.SetRune(canvas.Point{X: x, Y: y}, rune(letter))
			if letter != word[0] {
				continue // quick exit
			}
			countWordsDir(&board, &counts, word, x, y, -1, 0)  // left
			countWordsDir(&board, &counts, word, x, y, +1, 0)  // right
			countWordsDir(&board, &counts, word, x, y, 0, -1)  // up
			countWordsDir(&board, &counts, word, x, y, 0, +1)  // down
			countWordsDir(&board, &counts, word, x, y, -1, -1) // up-left
			countWordsDir(&board, &counts, word, x, y, +1, -1) // up-right
			countWordsDir(&board, &counts, word, x, y, -1, +1) // down-left
			countWordsDir(&board, &counts, word, x, y, +1, +1) // down-right
		}
	}

	// collect all the counts and set the heatmap data
	maxCount := 0
	for y := 0; y < board.LenY(); y++ {
		for x := 0; x < board.LenX(); x++ {
			count := counts.Data()[y][x]
			if count != 0 {
				// we need to go from "board space" (x-left, y-down) to the linechart's  (Y-up,X-right)
				hm.Push(heatmap.NewHeatPointInt(
					x,
					board.LenY()-y,
					float64(count)))
			}
			if count > maxCount {
				maxCount = count
			}
		}
	}

	hm.Model.SetViewXYRange(0, float64(board.LenX()), 0, float64(board.LenY()))
	hm.Model.SetXYRange(0, float64(board.LenX()), 0, float64(board.LenY()))
	hm.SetValueRange(0, float64(maxCount))
	hm.Draw()
	return hm
}

///////////////////////////////////////////////////////////////////////////////

const usage = `usage: %s <aoc2024.4 puzzle file> [<word>]
Creates a heatmap of word-hits for the AdventOfCode 2024 Day 4
puzzle with the word filled.  Default word is 'xmas'.

Find test input:
  https://adventofcode.com/2024/day/4/

Download your input from:
  https://adventofcode.com/2024/day/4/input

Try running: %s examples/heatmap/aoc2024_4.txt

`

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintf(os.Stderr, usage, os.Args[0], os.Args[0])
		os.Exit(1)
	}
	filename := os.Args[1]
	word := "XMAS"
	if len(os.Args) > 2 {
		word = os.Args[2]
	}

	// Open and read aoc2024/4 data file
	puzzle, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err.Error())
		os.Exit(1)
	}

	// Create a new board
	board := NewBoard(string(puzzle))
	if board == nil {
		fmt.Fprintf(os.Stderr, "Error creating board\n")
		os.Exit(1)
	}

	// Make and print heatmap
	hm := makeHeatmap(*board, word)
	fmt.Println(lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder(), true, true, true, true).
		Render(hm.View()))
}
