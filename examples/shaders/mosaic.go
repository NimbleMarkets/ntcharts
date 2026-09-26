package main

import (
	"image"
	"image/color"
	"image/draw"
)

// mosaicGutter is the width in source pixels of the dark separator between
// the four tiles of a mosaic frame.
const mosaicGutter = 2

// mosaicMinSize is the smallest raster edge that can hold two one-pixel
// tiles and the gutter; smaller rasters fall back to a single shader.
const mosaicMinSize = 2 + mosaicGutter

// mosaicRects splits a w×h raster into four tiles in reading order
// (top-left, top-right, bottom-left, bottom-right) separated by the gutter.
// The right and bottom tiles absorb any odd pixel so the tiles always cover
// the raster exactly.
func mosaicRects(w, h int) [4]image.Rectangle {
	leftW := max(1, (w-mosaicGutter)/2)
	topH := max(1, (h-mosaicGutter)/2)
	rightX := leftW + mosaicGutter
	bottomY := topH + mosaicGutter
	return [4]image.Rectangle{
		image.Rect(0, 0, leftW, topH),
		image.Rect(rightX, 0, w, topH),
		image.Rect(0, bottomY, leftW, h),
		image.Rect(rightX, bottomY, w, h),
	}
}

// composeMosaic places four tiles, sized by mosaicRects, into one opaque
// w×h image. The gutter between tiles is black.
func composeMosaic(tiles [4]*image.NRGBA, w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{A: 0xff}), image.Point{}, draw.Src)
	for i, r := range mosaicRects(w, h) {
		if tiles[i] != nil {
			draw.Draw(img, r, tiles[i], tiles[i].Bounds().Min, draw.Src)
		}
	}
	return img
}
