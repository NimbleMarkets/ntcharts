package main

import (
	"fmt"
	"io"
	"os"

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
		}
	}
	return &m, nil
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
