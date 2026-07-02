package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileGlyphCatalogLoadsTraceablePublishedGlyphs(t *testing.T) {
	manifestPath := writeGlyphManifestFixture(t, `{
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
	      "review_status": "published",
	      "structure_notes": ["中宫收紧，纵势取正。"],
	      "brushwork_notes": ["首点沉着，横折处转锋清楚。"]
	    },
	    {
	      "glyph_id": "ou-jiuchenggong-draft",
	      "character": "未",
	      "source_image": "s3://nebula-calligraphy/copybooks/jiuchenggong/page-001.png",
	      "crop_box": {"x": 1, "y": 1, "width": 8, "height": 8, "unit": "px"},
	      "license_status": "public_domain",
	      "review_status": "draft"
	    }
	  ]
	}`)

	catalog, err := NewFileGlyphCatalog(manifestPath)
	if err != nil {
		t.Fatalf("NewFileGlyphCatalog() error = %v", err)
	}

	result := catalog.Search(GlyphSearchParams{Character: "永", Style: "ou", CopybookID: "jiuchenggong"})
	if len(result) != 1 {
		t.Fatalf("len(Search()) = %d, want 1", len(result))
	}
	if result[0].GlyphID != "ou-jiuchenggong-yong-original" {
		t.Fatalf("GlyphID = %q, want ou-jiuchenggong-yong-original", result[0].GlyphID)
	}
	if result[0].CropBox.Unit != "px" || result[0].CropBox.Width != 80 {
		t.Fatalf("CropBox = %+v, want px crop from manifest", result[0].CropBox)
	}

	detail, ok := catalog.GetDetail("ou-jiuchenggong-yong-original")
	if !ok {
		t.Fatal("GetDetail() not found")
	}
	if detail.StructureNotes[0] != "中宫收紧，纵势取正。" {
		t.Fatalf("StructureNotes = %#v, want manifest notes", detail.StructureNotes)
	}
	if len(catalog.Search(GlyphSearchParams{Character: "未"})) != 0 {
		t.Fatal("draft glyph should not be searchable")
	}
}

func TestFileGlyphCatalogRejectsUntraceableManifest(t *testing.T) {
	manifestPath := writeGlyphManifestFixture(t, `{
	  "schema_version": "calligraphy.glyph_manifest.v1",
	  "copybook": {
	    "copybook_id": "jiuchenggong",
	    "title": "九成宫醴泉铭",
	    "style": "ou",
	    "calligrapher": "欧阳询",
	    "license_status": "public_domain"
	  },
	  "glyphs": []
	}`)

	_, err := NewFileGlyphCatalog(manifestPath)
	if err == nil {
		t.Fatal("NewFileGlyphCatalog() error = nil, want traceability error")
	}
}

func TestCompositeGlyphCatalogPrefersManifestGlyphs(t *testing.T) {
	manifestPath := writeGlyphManifestFixture(t, `{
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
	      "glyph_id": "ou-jiuchenggong-shan-original",
	      "character": "山",
	      "source_image": "s3://nebula-calligraphy/copybooks/jiuchenggong/page-001.png",
	      "crop_box": {"x": 100, "y": 120, "width": 80, "height": 96, "unit": "px"},
	      "license_status": "public_domain",
	      "review_status": "published"
	    }
	  ]
	}`)
	fileCatalog, err := NewFileGlyphCatalog(manifestPath)
	if err != nil {
		t.Fatalf("NewFileGlyphCatalog() error = %v", err)
	}
	catalog := NewCompositeGlyphCatalog(fileCatalog, NewInMemoryGlyphCatalog())

	result := catalog.Search(GlyphSearchParams{Character: "山", Style: "ou", CopybookID: "jiuchenggong"})
	if len(result) != 1 {
		t.Fatalf("len(Search()) = %d, want de-duplicated manifest result", len(result))
	}
	if result[0].GlyphID != "ou-jiuchenggong-shan-original" {
		t.Fatalf("GlyphID = %q, want manifest glyph first", result[0].GlyphID)
	}
}

func writeGlyphManifestFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
