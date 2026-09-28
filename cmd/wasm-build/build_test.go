package main

import (
	"strings"
	"testing"
)

func TestBuildCommandPerToolchain(t *testing.T) {
	goDemo := Demo{Name: "shaders", Source: "./examples/shaders"}
	name, args := buildCommand(goDemo, "web/demos/shaders/app.wasm", "go", "tinygo")
	if name != "go" || strings.Join(args, " ") != "build -o web/demos/shaders/app.wasm ./examples/shaders" {
		t.Errorf("go: %s %v", name, args)
	}

	tg := Demo{Name: "shaders-tinygo", Source: "./examples/shaders", Toolchain: "tinygo"}
	name, args = buildCommand(tg, "web/demos/shaders-tinygo/app.wasm", "go", "/opt/tinygo/bin/tinygo")
	joined := strings.Join(args, " ")
	if name != "/opt/tinygo/bin/tinygo" || !strings.HasPrefix(joined, "build -target=wasm") ||
		!strings.Contains(joined, "-opt=2") || !strings.Contains(joined, "-no-debug") ||
		!strings.HasSuffix(joined, "-o web/demos/shaders-tinygo/app.wasm ./examples/shaders") {
		t.Errorf("tinygo: %s %v", name, args)
	}
}
