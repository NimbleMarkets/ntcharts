package main

import (
	"embed"
	"fmt"
	"strings"
)

// These self-contained shaders share one uniform layout and need no assets.
//
//go:embed shaders/*.wgsl
var shaderFiles embed.FS

type preset struct {
	name, title, description string
	scale, detail            float32
}

var presets = []preset{
	{"plasma", "Chromatic plasma", "Interfering waves and luminous color bands", 2.5, 0.8},
	{"kaleidoscope", "Prismatic kaleidoscope", "Rotating mirrored petals and concentric ribbons", 2.0, 0.8},
	{"tunnel", "Neon tunnel", "A spiraling flight through a luminous lattice", 1.1, 0.8},
	{"flow", "Liquid aurora", "Layered flowing noise and warped color fields", 2.5, 0.8},
	{"julia", "Julia bloom", "An animated complex-plane fractal", 1.8, 0.8},
	{"orbits", "Chrome orbits", "Raymarched metallic forms above a checker floor", 1.0, 0.8},
}

func shaderSource(index int) string {
	common, err := shaderFiles.ReadFile("shaders/common.wgsl")
	if err != nil {
		panic(err)
	}
	body, err := shaderFiles.ReadFile("shaders/" + presets[index].name + ".wgsl")
	if err != nil {
		panic(err)
	}
	return strings.Replace(string(common), "// SHADER_BODY", string(body), 1)
}
func presetIndex(name string) (int, error) {
	for i, p := range presets {
		if name == p.name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown preset %q (use -list)", name)
}
