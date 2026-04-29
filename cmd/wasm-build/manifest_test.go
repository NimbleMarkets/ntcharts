package main

import (
	"strings"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	yaml := `
groups:
  - title: Lines
    demos:
      - name: quickstart
        title: Quickstart
        blurb: Time-series chart with mouse + keyboard zoom.
        source: ./examples/quickstart
      - name: wavelines
        title: Wavelines
        blurb: Looping wave pattern.
        source: ./examples/linechart/wavelines
  - title: Heatmap
    demos:
      - name: heatmap-perlin
        title: Heatmap (Perlin)
        blurb: Color-mapped 2D Perlin noise.
        source: ./examples/heatmap/perlin
`
	m, err := loadManifest(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("loadManifest: %v", err)
	}
	if len(m.Groups) != 2 {
		t.Fatalf("want 2 groups, got %d", len(m.Groups))
	}
	if m.Groups[0].Title != "Lines" || len(m.Groups[0].Demos) != 2 {
		t.Errorf("Lines group: got %+v", m.Groups[0])
	}
	if m.Groups[1].Demos[0].Name != "heatmap-perlin" {
		t.Errorf("Heatmap demo name: got %q", m.Groups[1].Demos[0].Name)
	}
}

func TestLoadManifestRejectsUnsafeDemoNames(t *testing.T) {
	tests := []string{
		"../escape",
		"quick/start",
		"-flag",
		"Uppercase",
		"two--hyphens",
		"trailing-",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := loadManifest(strings.NewReader(validManifestWith("name", name)))
			if err == nil {
				t.Fatal("expected invalid demo name to be rejected")
			}
			if !strings.Contains(err.Error(), "invalid name") {
				t.Fatalf("expected invalid name error, got %v", err)
			}
		})
	}
}

func TestLoadManifestRejectsUnsafeDemoSources(t *testing.T) {
	tests := []string{
		"-toolexec=helper",
		"examples/quickstart",
		"./cmd/wasm-build",
		"./examples/../cmd/wasm-build",
		"./examples/",
		"./examples//quickstart",
		`./examples\quickstart`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			_, err := loadManifest(strings.NewReader(validManifestWith("source", source)))
			if err == nil {
				t.Fatal("expected invalid demo source to be rejected")
			}
			if !strings.Contains(err.Error(), "invalid source") {
				t.Fatalf("expected invalid source error, got %v", err)
			}
		})
	}
}

func TestLoadManifestRepoManifestPassesValidation(t *testing.T) {
	if _, err := loadManifestFile("../../web/demos.yaml"); err != nil {
		t.Fatalf("repo manifest should pass validation: %v", err)
	}
}

func TestAllDemosFlattens(t *testing.T) {
	m := &Manifest{Groups: []Group{
		{Title: "A", Demos: []Demo{{Name: "x"}, {Name: "y"}}},
		{Title: "B", Demos: []Demo{{Name: "z"}}},
	}}
	got := m.AllDemos()
	if len(got) != 3 || got[0].Name != "x" || got[2].Name != "z" {
		t.Errorf("AllDemos: %+v", got)
	}
}

func validManifestWith(field, value string) string {
	name := "quickstart"
	source := "./examples/quickstart"
	switch field {
	case "name":
		name = value
	case "source":
		source = value
	}
	return `
groups:
  - title: Lines
    demos:
      - name: ` + name + `
        title: Quickstart
        blurb: Time-series chart with mouse + keyboard zoom.
        source: ` + source + `
`
}
