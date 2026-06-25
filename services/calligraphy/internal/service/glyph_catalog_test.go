package service

import "testing"

func TestGlyphCatalogSearchFiltersPublishedGlyphs(t *testing.T) {
	catalog := NewInMemoryGlyphCatalog()

	result := catalog.Search(GlyphSearchParams{
		Character: "山",
		Style:     "ou",
	})

	if len(result) != 1 {
		t.Fatalf("len(Search()) = %d, want 1", len(result))
	}
	if result[0].Character != "山" {
		t.Fatalf("Character = %q, want 山", result[0].Character)
	}
	if result[0].LicenseStatus == "restricted" {
		t.Fatal("restricted glyph should not be returned")
	}
}

func TestGlyphCatalogSearchCanFilterCopybook(t *testing.T) {
	catalog := NewInMemoryGlyphCatalog()

	result := catalog.Search(GlyphSearchParams{
		Character:  "水",
		Style:      "yan",
		CopybookID: "duobaota",
	})

	if len(result) != 1 {
		t.Fatalf("len(Search()) = %d, want 1", len(result))
	}
	if result[0].CopybookID != "duobaota" {
		t.Fatalf("CopybookID = %q, want duobaota", result[0].CopybookID)
	}
}

func TestGlyphCatalogSearchReturnsEmptyForRestrictedOnly(t *testing.T) {
	catalog := NewInMemoryGlyphCatalog()

	result := catalog.Search(GlyphSearchParams{
		Character: "禁",
	})

	if len(result) != 0 {
		t.Fatalf("len(Search()) = %d, want 0", len(result))
	}
}
