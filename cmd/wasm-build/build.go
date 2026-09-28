package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// findTinyGo locates the TinyGo compiler: the TINYGO environment variable
// if set, otherwise "tinygo" on PATH.
func findTinyGo() (string, error) {
	if p := os.Getenv("TINYGO"); p != "" {
		return p, nil
	}
	p, err := exec.LookPath("tinygo")
	if err != nil {
		return "", fmt.Errorf("tinygo not found: install TinyGo 0.42.0 or newer, set TINYGO=/path/to/tinygo, or pass -skip-tinygo")
	}
	return p, nil
}

// buildCommand returns the compiler invocation for a demo: the Go toolchain
// by default, or TinyGo (optimized for speed, without debug info) when the
// demo asks for it.
func buildCommand(d Demo, dst, goBin, tinygoBin string) (string, []string) {
	if d.TinyGo() {
		args := append([]string{"build"}, tinygoBuildFlags...)
		return tinygoBin, append(args, "-o", dst, d.Source)
	}
	return goBin, []string{"build", "-o", dst, d.Source}
}

// compileAll builds GOOS=js GOARCH=wasm binaries for every demo, in parallel.
// Output: <outDir>/demos/<demo.Name>/app.wasm
// tinygoBin may be empty when the manifest has no TinyGo demos.
func compileAll(m *Manifest, outDir, tinygoBin string) error {
	// wasm demos build against the bubbletea wasm fork, selected via the
	// wasm.work workspace at the repo root (see comments in that file).
	workFile, err := filepath.Abs("wasm.work")
	if err != nil {
		return err
	}
	if _, err := os.Stat(workFile); err != nil {
		return fmt.Errorf("wasm.work not found (run from the repo root): %w", err)
	}
	demos := m.AllDemos()
	errs := make([]error, len(demos))
	var wg sync.WaitGroup
	for i, d := range demos {
		wg.Add(1)
		go func(i int, d Demo) {
			defer wg.Done()
			errs[i] = compileOne(d, outDir, workFile, tinygoBin)
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

func compileOne(d Demo, outDir, workFile, tinygoBin string) error {
	dst := filepath.Join(outDir, "demos", d.Name, "app.wasm")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	name, args := buildCommand(d, dst, "go", tinygoBin)
	cmd := exec.Command(name, args...)
	// TinyGo's wasm target implies GOOS=js GOARCH=wasm; setting them is harmless.
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm", "GOWORK="+workFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("[wasm-build] compiling %s -> %s\n", d.Source, dst)
	return cmd.Run()
}

// copyTinyGoAssets copies TinyGo's own wasm_exec.js shim (from its
// TINYGOROOT) to <outDir>/_assets/wasm_exec_tinygo.js for TinyGo demos.
func copyTinyGoAssets(outDir, tinygoBin string) error {
	root, err := exec.Command(tinygoBin, "env", "TINYGOROOT").Output()
	if err != nil {
		return fmt.Errorf("tinygo env TINYGOROOT: %w", err)
	}
	src := filepath.Join(strings.TrimSpace(string(root)), "targets", "wasm_exec.js")
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	dst := filepath.Join(outDir, "_assets", "wasm_exec_tinygo.js")
	fmt.Printf("[wasm-build] copying TinyGo wasm_exec.js -> %s\n", dst)
	return os.WriteFile(dst, data, 0o644)
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
	cmd := exec.Command("go", "tool", "booba-assets", assetsDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("[wasm-build] copying booba assets into %s\n", assetsDir)
	return cmd.Run()
}
