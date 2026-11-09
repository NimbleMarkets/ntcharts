// ntcharts - Copyright (c) 2026 Neomantra Corp.

package picture

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzPrepareSourceAndGlyph(f *testing.F) {
	f.Add(uint8(4), uint8(4), uint8(6), uint8(3), uint8(2), uint8(4), uint8(0), uint8(0), true, []byte{255, 0, 0, 255, 0, 255, 0, 128})
	f.Add(uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(2), uint8(1), uint8(4), false, []byte{10, 20, 30, 0})
	f.Add(uint8(0), uint8(0), uint8(3), uint8(2), uint8(1), uint8(2), uint8(2), uint8(8), true, []byte{})

	f.Fuzz(func(t *testing.T, srcWRaw, srcHRaw, colsRaw, rowsRaw, cwRaw, chRaw, fitRaw, anchorRaw uint8, useNRGBA bool, pix []byte) {
		srcW := int(srcWRaw % 16)
		srcH := int(srcHRaw % 16)
		cols := int(colsRaw%12) + 1
		rows := int(rowsRaw%8) + 1
		cellW := int(cwRaw%8) + 1
		cellH := (int(chRaw%4) + 1) * 2 // even height for appendGlyph

		var src image.Image
		if useNRGBA {
			nrgba := image.NewNRGBA(image.Rect(0, 0, srcW, srcH))
			for i := 0; i < len(nrgba.Pix) && i < len(pix); i++ {
				nrgba.Pix[i] = pix[i]
			}
			src = nrgba
		} else {
			rgba := image.NewRGBA(image.Rect(0, 0, srcW, srcH))
			for i := 0; i < len(rgba.Pix) && i < len(pix); i++ {
				rgba.Pix[i] = pix[i]
			}
			src = rgba
		}

		fit := FitMode(fitRaw % 4)
		anchor := FitAnchor(anchorRaw % 10)
		var bg color.Color = color.RGBA{R: 16, G: 24, B: 32, A: 255}
		if len(pix) > 0 && pix[0]&1 == 1 {
			bg = color.Transparent
		}

		var scratch fitScratch
		prepared := prepareSourceWithScratch(src, fit, cols, rows, cellW, cellH, bg, anchor, &scratch)
		if prepared == nil {
			t.Fatalf("prepareSource returned nil for positive dims")
		}
		pb := prepared.Bounds()
		if pb.Dx() != cols*cellW || pb.Dy() != rows*cellH {
			t.Fatalf("prepared bounds %v != %dx%d", pb, cols*cellW, rows*cellH)
		}

		glyphBytes := appendGlyph(nil, prepared)
		if !utf8.Valid(glyphBytes) {
			t.Fatalf("appendGlyph produced invalid UTF-8")
		}
	})
}

func FuzzKittyEncode(f *testing.F) {
	f.Add(uint8(4), uint8(4), uint8(4), uint8(2), uint8(0), []byte{100, 150, 200, 255})
	f.Add(uint8(3), uint8(3), uint8(2), uint8(2), uint8(1), []byte{0, 50, 100, 128})
	f.Add(uint8(2), uint8(2), uint8(1), uint8(1), uint8(2), []byte{255, 255, 255, 255})

	f.Fuzz(func(t *testing.T, wRaw, hRaw, colsRaw, rowsRaw, fmtRaw uint8, pix []byte) {
		w := int(wRaw%12) + 1
		h := int(hRaw%12) + 1
		cols := int(colsRaw%8) + 1
		rows := int(rowsRaw%8) + 1

		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for i := 0; i < len(img.Pix); i++ {
			if len(pix) > 0 {
				img.Pix[i] = pix[i%len(pix)]
			}
		}

		formats := []KittyFormat{KittyFormatPNG, KittyFormatRGBA, KittyFormatZlib}
		kf := formats[int(fmtRaw)%len(formats)]
		apc := buildKittyAPC(img, 42, cols, rows, kf, -1)
		if apc == "" {
			t.Fatalf("buildKittyAPC returned empty string for %dx%d format %v", w, h, kf)
		}
		if !strings.Contains(apc, "\x1b_G") {
			t.Fatalf("buildKittyAPC output missing APC prefix")
		}
	})
}

func FuzzGridImage(f *testing.F) {
	f.Add(uint8(4), uint8(3), uint8(2), uint8(1), uint8(4), uint8(8), int8(1), int8(1))
	f.Add(uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), int8(-1), int8(0))

	f.Fuzz(func(t *testing.T, lCols, lRows, tCols, tRows, cpW, cpH uint8, cx, cy int8) {
		grid := NewGridImage(GridImageConfig{
			LogicalCols:         int(lCols%8) - 1,
			LogicalRows:         int(lRows%8) - 1,
			TerminalColsPerCell: int(tCols % 4),
			TerminalRowsPerCell: int(tRows % 4),
			CellPixelWidth:      int(cpW % 12),
			CellPixelHeight:     int(cpH % 16),
			BackgroundColor:     color.RGBA{R: 10, G: 20, B: 30, A: 255},
		})
		sprite := image.NewRGBA(image.Rect(0, 0, 4, 4))
		_ = grid.CellRect(int(cx), int(cy))
		_ = grid.FillCell(int(cx), int(cy), color.RGBA{R: 200, G: 100, B: 50, A: 255})
		_ = grid.DrawCell(int(cx), int(cy), sprite)
		_ = grid.DrawOverlay(sprite)
		if grid.Image() == nil {
			t.Fatalf("GridImage.Image() returned nil")
		}
	})
}
