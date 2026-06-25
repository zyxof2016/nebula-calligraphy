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

func TestGlyphCatalogContainsEnoughCommonLearningGlyphs(t *testing.T) {
	catalog := NewInMemoryGlyphCatalog()

	for _, style := range []string{"ou", "yan"} {
		groups := catalog.ListPresetGroups(style)
		unique := map[string]bool{}
		for _, group := range groups {
			for _, glyph := range group.Glyphs {
				unique[glyph.Character] = true
				if glyph.Style != style {
					t.Fatalf("preset glyph style = %q, want %q", glyph.Style, style)
				}
			}
		}

		if len(unique) < 120 {
			t.Fatalf("%s unique common glyphs = %d, want at least 120", style, len(unique))
		}
		for _, character := range []string{"人", "天", "月", "春", "德"} {
			if !unique[character] {
				t.Fatalf("%s common glyph %q not found", style, character)
			}
		}
	}
}

func TestGlyphCatalogPresetGroupsExposeLearningCategories(t *testing.T) {
	catalog := NewInMemoryGlyphCatalog()

	groups := catalog.ListPresetGroups("ou")
	if len(groups) < 5 {
		t.Fatalf("len(groups) = %d, want at least 5", len(groups))
	}
	if groups[0].GroupID == "" || len(groups[0].Glyphs) == 0 {
		t.Fatalf("first group is incomplete: %+v", groups[0])
	}
}

func TestGlyphCatalogDetailIncludesPracticeTemplates(t *testing.T) {
	catalog := NewInMemoryGlyphCatalog()

	detail, ok := catalog.GetDetail("ou-jiuchenggong-shan")
	if !ok {
		t.Fatal("GetDetail() not found")
	}
	if detail.Glyph.Character != "山" {
		t.Fatalf("Character = %q, want 山", detail.Glyph.Character)
	}
	if len(detail.StructureNotes) == 0 {
		t.Fatal("StructureNotes is empty")
	}
	if len(detail.PracticeTemplates) < 3 {
		t.Fatalf("len(PracticeTemplates) = %d, want at least 3", len(detail.PracticeTemplates))
	}
	if detail.PracticeTemplates[0].TemplateType == "" {
		t.Fatal("TemplateType is empty")
	}
}

func TestGlyphCatalogDetailHidesRestrictedGlyphs(t *testing.T) {
	catalog := NewInMemoryGlyphCatalog()

	_, ok := catalog.GetDetail("restricted-test")
	if ok {
		t.Fatal("GetDetail(restricted) = true, want false")
	}
}
