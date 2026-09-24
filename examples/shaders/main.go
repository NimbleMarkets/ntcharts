// shaders is a native GPU shader gallery displayed through ntcharts/picture.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "shaders:", err)
		os.Exit(1)
	}
}
func run() error {
	name := flag.String("preset", "plasma", "initial preset (see -list)")
	list := flag.Bool("list", false, "list bundled shaders")
	transport := flag.String("transport", "png", "transport (png)")
	fps := flag.Int("fps", 60, "maximum application frame rate (1..120)")
	density := flag.Int("density", 16, "vertical source pixels per terminal row (2..40)")
	slideshow := flag.Duration("slideshow", 0, "switch presets automatically, e.g. 8s (0 disables)")
	fullscreen := flag.Bool("fullscreen", false, "start with an unobstructed image")
	duration := flag.Duration("duration", 0, "quit after this duration (0 disables)")
	report := flag.String("report", "", "write final timing and transport statistics as JSON")
	snapshot := flag.String("snapshot", "", "render all six shaders to PNGs in this directory, without a terminal")
	flag.Parse()
	if *list {
		for _, p := range presets {
			fmt.Printf("%-14s %s\n", p.name, p.description)
		}
		return nil
	}
	index, err := presetIndex(*name)
	if err != nil {
		return err
	}
	if *fps < 1 || *fps > 120 || *density < 2 || *density > 40 || *slideshow < 0 || *duration < 0 {
		return errors.New("invalid fps, density or duration")
	}
	if *transport != "png" {
		return errors.New("only png transport is supported")
	}
	gpu, err := newGPU()
	if err != nil {
		return fmt.Errorf("initialize GPU: %w", err)
	}
	defer gpu.Close()
	if *snapshot != "" {
		return snapshots(gpu, *snapshot)
	}
	m := newModel(gpu, index, *fps, *density, *slideshow)
	m.fullscreen = *fullscreen
	m.duration = *duration
	_, err = tea.NewProgram(m).Run()
	var reportErr error
	if *report != "" {
		data, marshalErr := json.MarshalIndent(struct {
			GPU                        string         `json:"gpu"`
			EncodedFrames              map[string]int `json:"encoded_frames"`
			Presets                    map[string]int `json:"rendered_presets"`
			AppFPS, RenderMS, EncodeMS float64
			Width, Height              int
		}{gpu.name, m.encodedFrames, m.renderedPresets, m.fps, m.renderMS, m.encodeMS, m.rasterW, m.rasterH}, "", "  ")
		reportErr = marshalErr
		if reportErr == nil {
			reportErr = os.WriteFile(*report, append(data, '\n'), 0600)
		}
	}
	return errors.Join(err, m.err, reportErr)
}
func snapshots(g *gpuRenderer, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	sheet := image.NewNRGBA(image.Rect(0, 0, 960, 400))
	for i, p := range presets {
		r := renderRequest{preset: i, width: 640, height: 400, seconds: 3.5, speed: 0.6, scale: p.scale, detail: p.detail}
		start := time.Now()
		img, err := g.Render(r)
		if err != nil {
			return err
		}
		fmt.Printf("%-14s %7.2f ms (render + readback)\n", p.name, float64(time.Since(start))/float64(time.Millisecond))
		if err := writePNG(filepath.Join(dir, p.name+".png"), img); err != nil {
			return err
		}
		// A nearest-neighbor contact sheet makes visual checking repeatable.
		thumb := image.NewNRGBA(image.Rect(0, 0, 320, 200))
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				thumb.SetNRGBA(x, y, img.NRGBAAt(x*2, y*2))
			}
		}
		pos := image.Pt((i%3)*320, (i/3)*200)
		draw.Draw(sheet, image.Rectangle{Min: pos, Max: pos.Add(thumb.Bounds().Size())}, thumb, image.Point{}, draw.Src)
	}
	return writePNG(filepath.Join(dir, "gallery.png"), sheet)
}
func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return errors.Join(png.Encode(f, img), f.Close())
}
