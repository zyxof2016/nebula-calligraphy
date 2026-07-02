package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidateGlyphManifest(t *testing.T) {
	path := writeManifestFixture(t, `{
	  "schema_version": "calligraphy.glyph_manifest.v1",
	  "copybook": {
	    "copybook_id": "jiuchenggong",
	    "title": "九成宫醴泉铭",
	    "style": "ou",
	    "calligrapher": "欧阳询",
	    "source_url": "https://commons.wikimedia.org/wiki/Category:%E4%B9%9D%E6%88%90%E5%AE%AE%E9%86%B4%E6%B3%89%E9%8A%98",
	    "license_status": "public_domain",
	    "attribution": "Wikimedia Commons public-domain scan"
	  },
	  "glyphs": [
	    {
	      "glyph_id": "ou-jiuchenggong-yong-original",
	      "character": "永",
	      "source_image": "s3://nebula-calligraphy/copybooks/jiuchenggong/page-001.png",
	      "crop_box": {"x": 100, "y": 120, "width": 80, "height": 96, "unit": "px"},
	      "license_status": "public_domain",
	      "review_status": "published"
	    }
	  ]
	}`)
	var out bytes.Buffer

	code := run([]string{"validate", path}, &out, &out)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0; output: %s", code, out.String())
	}
	if !strings.Contains(out.String(), "glyphs=1") {
		t.Fatalf("output = %q, want glyph count", out.String())
	}
}

func TestRunRejectsInvalidGlyphManifest(t *testing.T) {
	path := writeManifestFixture(t, `{"schema_version":"bad"}`)
	var out bytes.Buffer

	code := run([]string{"validate", path}, &out, &out)

	if code == 0 {
		t.Fatalf("run() code = 0, want failure; output: %s", out.String())
	}
	if !strings.Contains(out.String(), "invalid glyph manifest") {
		t.Fatalf("output = %q, want invalid manifest message", out.String())
	}
}

func writeManifestFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
