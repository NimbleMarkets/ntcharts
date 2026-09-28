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

func TestManifestToolchain(t *testing.T) {
	m, err := loadManifest(strings.NewReader(`
groups:
  - title: Shaders
    demos:
      - name: shaders
        title: GPU Shaders
        blurb: go
        source: ./examples/shaders
      - name: shaders-tinygo
        title: GPU Shaders (TinyGo)
        blurb: tinygo
        source: ./examples/shaders
        toolchain: tinygo
`))
	if err != nil {
		t.Fatalf("loadManifest: %v", err)
	}
	demos := m.AllDemos()
	if demos[0].TinyGo() || demos[0].WasmExec() != "/ntcharts/_assets/wasm_exec.js" {
		t.Errorf("default demo: tinygo=%v wasm_exec=%q", demos[0].TinyGo(), demos[0].WasmExec())
	}
	if !demos[1].TinyGo() || demos[1].WasmExec() != "/ntcharts/_assets/wasm_exec_tinygo.js" {
		t.Errorf("tinygo demo: tinygo=%v wasm_exec=%q", demos[1].TinyGo(), demos[1].WasmExec())
	}
	if !m.HasTinyGo() {
		t.Error("HasTinyGo should report the tinygo demo")
	}

	_, err = loadManifest(strings.NewReader(`
groups:
  - title: X
    demos:
      - name: a
        title: A
        blurb: b
        source: ./examples/a
        toolchain: rustc
`))
	if err == nil || !strings.Contains(err.Error(), "toolchain") {
		t.Fatalf("unknown toolchain should be rejected, got %v", err)
	}
}

func TestWithoutTinyGoDropsThoseDemos(t *testing.T) {
	m := &Manifest{Groups: []Group{
		{Title: "Shaders", Demos: []Demo{
			{Name: "shaders", Title: "Go", Source: "./examples/shaders"},
			{Name: "shaders-tinygo", Title: "TinyGo", Source: "./examples/shaders", Toolchain: "tinygo"},
		}},
		{Title: "Only TinyGo", Demos: []Demo{
			{Name: "x-tinygo", Title: "X", Source: "./examples/x", Toolchain: "tinygo"},
		}},
	}}
	got := m.WithoutTinyGo()
	if len(got.Groups) != 1 || len(got.Groups[0].Demos) != 1 || got.Groups[0].Demos[0].Name != "shaders" {
		t.Fatalf("WithoutTinyGo = %+v, want only the Go shaders demo and no empty groups", got.Groups)
	}
	if len(m.Groups[0].Demos) != 2 {
		t.Fatal("WithoutTinyGo must not modify the receiver")
	}
}

func TestDemoRunCommand(t *testing.T) {
	for _, tc := range []struct {
		source, toolchain, want string
	}{
		{"./examples/barchart/vertical", "", "github.com/NimbleMarkets/ntcharts/examples/v2/barchart/vertical"},
		{"./examples/shaders", "", "github.com/NimbleMarkets/ntcharts/examples/shaders/v2"},
		{"./examples/shaders-extra", "", "github.com/NimbleMarkets/ntcharts/examples/v2/shaders-extra"},
	} {
		t.Run(tc.source+tc.toolchain, func(t *testing.T) {
			d := Demo{Source: tc.source, Toolchain: tc.toolchain}
			if got, want := d.RunCommand(), "go run "+tc.want+"@latest"; got != want {
				t.Fatalf("RunCommand = %q, want %q", got, want)
			}
		})
	}
}

func TestTinyGoDemoRunCommandBuildsTheBrowserWasm(t *testing.T) {
	d := Demo{Name: "shaders-tinygo", Source: "./examples/shaders", Toolchain: "tinygo"}
	cmd := d.RunCommand()
	for _, want := range []string{
		"git clone https://github.com/NimbleMarkets/ntcharts",
		"GOWORK=$PWD/wasm.work tinygo build -target=wasm -opt=2 -no-debug -o app.wasm ./examples/shaders",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("tinygo RunCommand lacks %q:\n%s", want, cmd)
		}
	}
	if strings.Contains(cmd, "go run") {
		t.Errorf("tinygo RunCommand should not advertise go run:\n%s", cmd)
	}
	if got := d.RunLabel(); !strings.Contains(got, "TinyGo "+tinygoVersion) {
		t.Errorf("RunLabel = %q, want the TinyGo %s requirement", got, tinygoVersion)
	}
	if got := (Demo{Source: "./examples/quickstart"}).RunLabel(); got != "Run in your terminal:" {
		t.Errorf("Go RunLabel = %q", got)
	}
}
