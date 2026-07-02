package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/service"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 || args[0] != "validate" {
		fmt.Fprintln(stderr, "usage: calligraphy-glyph-manifest validate <manifest.json>")
		return 2
	}
	path := args[1]
	payload, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "read glyph manifest: %v\n", err)
		return 1
	}
	var manifest service.CopybookGlyphManifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		fmt.Fprintf(stderr, "parse glyph manifest: %v\n", err)
		return 1
	}
	if _, err := service.NewFileGlyphCatalogFromManifest(manifest); err != nil {
		fmt.Fprintf(stderr, "invalid glyph manifest: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "valid glyph manifest: copybook=%s style=%s glyphs=%d\n", manifest.Copybook.CopybookID, manifest.Copybook.Style, len(manifest.Glyphs))
	return 0
}
