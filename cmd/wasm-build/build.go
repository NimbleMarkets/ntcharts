package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// compileAll builds GOOS=js GOARCH=wasm binaries for every demo, in parallel.
// Uses booba-wasm-build (NOT plain `go build`) because bubbletea/v2 lacks
// js/wasm build tags upstream — booba-wasm-build patches them in via a
// temporary module before compiling.
// Output: <outDir>/demos/<demo.Name>/app.wasm
func compileAll(m *Manifest, outDir string) error {
	demos := m.AllDemos()
	errs := make([]error, len(demos))
	var wg sync.WaitGroup
	for i, d := range demos {
		wg.Add(1)
		go func(i int, d Demo) {
			defer wg.Done()
			errs[i] = compileOne(d, outDir)
		}(i, d)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return fmt.Errorf("demo %q: %w", demos[i].Name, err)
		}
	}
	return nil
}

func compileOne(d Demo, outDir string) error {
	dst := filepath.Join(outDir, "demos", d.Name, "app.wasm")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("go", "run",
		"github.com/NimbleMarkets/go-booba/cmd/booba-wasm-build",
		"-o", dst,
		d.Source,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("[wasm-build] compiling %s -> %s\n", d.Source, dst)
	return cmd.Run()
}

// copyAssets populates <outDir>/_assets/ with the booba runtime
// (wasm_exec.js, booba/booba.js, ghostty-web/ghostty-web.js).
// booba-assets writes files flat into the directory it's pointed at,
// so we point it at <outDir>/_assets directly.
func copyAssets(outDir string) error {
	assetsDir := filepath.Join(outDir, "_assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return err
	}
	cmd := exec.Command("go", "run",
		"github.com/NimbleMarkets/go-booba/cmd/booba-assets",
		assetsDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("[wasm-build] copying booba assets into %s\n", assetsDir)
	return cmd.Run()
}
