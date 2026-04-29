package main

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func renderSite(m *Manifest, tmplDir, outDir string) error {
	funcs := template.FuncMap{
		"trimPrefix": strings.TrimPrefix,
	}

	parse := func(name string) (*template.Template, error) {
		return template.New("").Funcs(funcs).ParseFiles(
			filepath.Join(tmplDir, "sidebar.html.tmpl"),
			filepath.Join(tmplDir, name),
		)
	}

	// Landing page.
	indexTmpl, err := parse("index.html.tmpl")
	if err != nil {
		return fmt.Errorf("parse index template: %w", err)
	}
	if err := writeTemplate(indexTmpl, "index.html.tmpl",
		filepath.Join(outDir, "index.html"),
		map[string]any{"Manifest": m, "Active": ""}); err != nil {
		return err
	}

	// Per-demo pages.
	demoTmpl, err := parse("demo.html.tmpl")
	if err != nil {
		return fmt.Errorf("parse demo template: %w", err)
	}
	for _, d := range m.AllDemos() {
		dst := filepath.Join(outDir, "demos", d.Name, "index.html")
		if err := writeTemplate(demoTmpl, "demo.html.tmpl", dst,
			map[string]any{"Manifest": m, "Active": d.Name, "Demo": d}); err != nil {
			return fmt.Errorf("render demo %q: %w", d.Name, err)
		}
	}
	return nil
}

func writeTemplate(t *template.Template, entryName, dst string, data any) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := t.ExecuteTemplate(io.Writer(f), entryName, data); err != nil {
		return fmt.Errorf("execute %s: %w", entryName, err)
	}
	return nil
}
