package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderProducesAllPages(t *testing.T) {
	out := t.TempDir()

	m := &Manifest{Groups: []Group{
		{Title: "Lines", Demos: []Demo{
			{Name: "quickstart", Title: "Quickstart", Blurb: "x", Source: "./examples/quickstart"},
		}},
		{Title: "Heatmap", Demos: []Demo{
			{Name: "heatmap-perlin", Title: "Heatmap (Perlin)", Blurb: "Press <kbd>F</kbd> to cycle", Source: "./examples/heatmap/perlin"},
		}},
		{Title: "Shaders", Demos: []Demo{
			{Name: "shaders-tinygo", Title: "GPU Shaders (TinyGo)", Blurb: "z", Source: "./examples/shaders", Toolchain: "tinygo"},
		}},
	}}

	if err := renderSite(m, "../../web/_templates", out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	// Landing page exists, contains both demos in nav.
	mustContain(t, filepath.Join(out, "index.html"),
		"Quickstart", "Heatmap (Perlin)", `href="/ntcharts/demos/quickstart/"`)

	// Per-demo pages exist with correct active marker.
	mustContain(t, filepath.Join(out, "demos", "quickstart", "index.html"),
		"Quickstart", "sidebar-link-active",
		`https://github.com/NimbleMarkets/ntcharts/tree/v2/examples/quickstart`)

	mustContain(t, filepath.Join(out, "demos", "heatmap-perlin", "index.html"),
		"Heatmap (Perlin)", "sidebar-link-active")

	// Blurbs are authored in the manifest and may carry inline HTML such as
	// <kbd>; they must reach the page unescaped, on the landing page and the
	// demo page alike.
	mustContain(t, filepath.Join(out, "index.html"), "Press <kbd>F</kbd> to cycle")
	mustContain(t, filepath.Join(out, "demos", "heatmap-perlin", "index.html"), "Press <kbd>F</kbd> to cycle")

	// Each page loads the wasm_exec shim matching its toolchain.
	mustContain(t, filepath.Join(out, "demos", "quickstart", "index.html"), `src="/ntcharts/_assets/wasm_exec.js"`)
	mustContain(t, filepath.Join(out, "demos", "shaders-tinygo", "index.html"), `src="/ntcharts/_assets/wasm_exec_tinygo.js"`)
}

func mustContain(t *testing.T, path string, needles ...string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, n := range needles {
		if !strings.Contains(string(b), n) {
			t.Errorf("%s missing %q", path, n)
		}
	}
}
