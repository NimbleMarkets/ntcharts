package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	manifestPath := flag.String("manifest", "web/demos.yaml", "path to demos manifest")
	tmplDir := flag.String("templates", "web/_templates", "path to templates dir")
	outDir := flag.String("out", "web", "output directory")
	skipWasm := flag.Bool("skip-wasm", false, "skip GOOS=js GOARCH=wasm builds (HTML/CSS only)")
	skipAssets := flag.Bool("skip-assets", false, "skip booba runtime asset copy")
	flag.Parse()

	m, err := loadManifestFile(*manifestPath)
	if err != nil {
		log.Fatalf("manifest: %v", err)
	}
	fmt.Printf("[wasm-build] %d demos across %d groups\n",
		len(m.AllDemos()), len(m.Groups))

	if !*skipWasm {
		if err := compileAll(m, *outDir); err != nil {
			log.Fatalf("compile: %v", err)
		}
	}
	if !*skipAssets {
		if err := copyAssets(*outDir); err != nil {
			log.Fatalf("copy assets: %v", err)
		}
	}
	if err := renderSite(m, *tmplDir, *outDir); err != nil {
		log.Fatalf("render: %v", err)
	}
	fmt.Println("[wasm-build] done")
}
