package main

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	Groups []Group `yaml:"groups"`
}

type Group struct {
	Title string `yaml:"title"`
	Demos []Demo `yaml:"demos"`
}

type Demo struct {
	Name   string `yaml:"name"`
	Title  string `yaml:"title"`
	Blurb  string `yaml:"blurb"`
	Source string `yaml:"source"`
	// Toolchain selects the compiler: "" (Go) or "tinygo".
	Toolchain string `yaml:"toolchain"`
}

// BlurbHTML returns the blurb as trusted HTML. Blurbs are authored in the
// repository's own manifest and may use inline markup such as <kbd>.
func (d Demo) BlurbHTML() template.HTML { return template.HTML(d.Blurb) }

// TinyGo reports whether the demo is compiled with TinyGo.
func (d Demo) TinyGo() bool { return d.Toolchain == "tinygo" }

// WasmExec returns the site path of the wasm_exec.js shim matching the
// demo's toolchain; Go and TinyGo each ship their own.
func (d Demo) WasmExec() string {
	if d.TinyGo() {
		return "/ntcharts/_assets/wasm_exec_tinygo.js"
	}
	return "/ntcharts/_assets/wasm_exec.js"
}

// HasTinyGo reports whether any demo needs the TinyGo toolchain.
func (m *Manifest) HasTinyGo() bool {
	for _, d := range m.AllDemos() {
		if d.TinyGo() {
			return true
		}
	}
	return false
}

// WithoutTinyGo returns a copy of the manifest with TinyGo demos removed,
// dropping any group left empty. The receiver is not modified.
func (m *Manifest) WithoutTinyGo() *Manifest {
	out := &Manifest{}
	for _, g := range m.Groups {
		var demos []Demo
		for _, d := range g.Demos {
			if !d.TinyGo() {
				demos = append(demos, d)
			}
		}
		if len(demos) > 0 {
			out.Groups = append(out.Groups, Group{Title: g.Title, Demos: demos})
		}
	}
	return out
}

func loadManifest(r io.Reader) (*Manifest, error) {
	var m Manifest
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if len(m.Groups) == 0 {
		return nil, fmt.Errorf("manifest has no groups")
	}
	for gi, g := range m.Groups {
		if g.Title == "" {
			return nil, fmt.Errorf("group %d: title required", gi)
		}
		for di, d := range g.Demos {
			if d.Name == "" || d.Title == "" || d.Source == "" {
				return nil, fmt.Errorf("group %q demo %d: name/title/source required", g.Title, di)
			}
			if !validDemoName(d.Name) {
				return nil, fmt.Errorf("group %q demo %d: invalid name %q: use lowercase letters, digits, and single hyphens", g.Title, di, d.Name)
			}
			if !validDemoSource(d.Source) {
				return nil, fmt.Errorf("group %q demo %q: invalid source %q: use a ./examples/... package path without traversal", g.Title, d.Name, d.Source)
			}
			if d.Toolchain != "" && d.Toolchain != "tinygo" {
				return nil, fmt.Errorf("group %q demo %q: unknown toolchain %q: use \"tinygo\" or leave unset for Go", g.Title, d.Name, d.Toolchain)
			}
		}
	}
	return &m, nil
}

func validDemoName(name string) bool {
	if name == "" {
		return false
	}
	prevHyphen := true // rejects a leading hyphen.
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			prevHyphen = false
		case r >= '0' && r <= '9':
			prevHyphen = false
		case r == '-':
			if prevHyphen {
				return false
			}
			prevHyphen = true
		default:
			return false
		}
	}
	return !prevHyphen
}

func validDemoSource(source string) bool {
	const prefix = "./examples/"
	if !strings.HasPrefix(source, prefix) || strings.Contains(source, `\`) {
		return false
	}
	rest := strings.TrimPrefix(source, prefix)
	if rest == "" {
		return false
	}
	for _, part := range strings.Split(rest, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func loadManifestFile(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return loadManifest(f)
}

func (m *Manifest) AllDemos() []Demo {
	var out []Demo
	for _, g := range m.Groups {
		out = append(out, g.Demos...)
	}
	return out
}
