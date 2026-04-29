package picture

import (
	"bytes"
	"fmt"
	"image"
	"strings"

	"github.com/charmbracelet/x/ansi/kitty"
	"golang.org/x/image/draw"
)

// buildKittyAPC encodes img as a Kitty graphics APC sequence. cellPixelW and
// cellPixelH are the terminal cell pixel dimensions; multiplied by cols and
// rows they form the source pixel size that matches the c×r cell rectangle.
// Kitty preserves source aspect ratio when placing into cells, so pre-scaling
// the source to the cell-rect pixel dimensions makes that fit a true fill.
func buildKittyAPC(img image.Image, id, cols, rows, cellPixelW, cellPixelH int) string {
	targetW := cols * cellPixelW
	targetH := rows * cellPixelH
	srcForAPC := img
	if b := img.Bounds(); b.Dx() != targetW || b.Dy() != targetH {
		scaled := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
		draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, b, draw.Src, nil)
		srcForAPC = scaled
	}

	var buf bytes.Buffer
	opts := &kitty.Options{
		Action:           kitty.TransmitAndPut,
		Transmission:     kitty.Direct,
		Format:           kitty.PNG,
		ID:               id,
		Columns:          cols,
		Rows:             rows,
		VirtualPlacement: true,
		Quite:            2,
		Chunk:            true,
	}
	if err := kitty.EncodeGraphics(&buf, srcForAPC, opts); err != nil {
		return ""
	}
	return buf.String()
}

func buildKittyGrid(cols, rows, imageID int) string {
	r := (imageID >> 16) & 0xff
	g := (imageID >> 8) & 0xff
	b := imageID & 0xff
	sgr := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	reset := "\x1b[39m"

	var sb strings.Builder
	sb.Grow((cols*4 + len(sgr) + len(reset) + 1) * rows)

	for y := 0; y < rows; y++ {
		sb.WriteString(sgr)
		rowDia := kitty.Diacritic(y)
		for x := 0; x < cols; x++ {
			sb.WriteRune(kitty.Placeholder)
			sb.WriteRune(rowDia)
			sb.WriteRune(kitty.Diacritic(x))
		}
		sb.WriteString(reset)
		if y < rows-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func kittyDeleteImage(id int) string {
	return fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
}
