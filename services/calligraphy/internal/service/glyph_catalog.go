package service

import (
	"strings"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

type GlyphSearchParams = model.GlyphSearchParams

type GlyphCatalog interface {
	Search(params GlyphSearchParams) []model.Glyph
}

type InMemoryGlyphCatalog struct {
	glyphs []model.Glyph
}

func NewInMemoryGlyphCatalog() *InMemoryGlyphCatalog {
	return &InMemoryGlyphCatalog{glyphs: []model.Glyph{
		seedGlyph("ou-jiuchenggong-shan", "山", "ou", "jiuchenggong", "欧阳询", "licensed", "published"),
		seedGlyph("ou-jiuchenggong-shui", "水", "ou", "jiuchenggong", "欧阳询", "licensed", "published"),
		seedGlyph("yan-duobaota-shui", "水", "yan", "duobaota", "颜真卿", "licensed", "published"),
		seedGlyph("yan-duobaota-qing", "清", "yan", "duobaota", "颜真卿", "licensed", "published"),
		seedGlyph("restricted-test", "禁", "ou", "private-copybook", "测试", "restricted", "published"),
	}}
}

func (c *InMemoryGlyphCatalog) Search(params GlyphSearchParams) []model.Glyph {
	var matches []model.Glyph
	for _, glyph := range c.glyphs {
		if glyph.LicenseStatus == "restricted" || glyph.ReviewStatus != "published" {
			continue
		}
		if params.Character != "" && glyph.Character != params.Character {
			continue
		}
		if params.Style != "" && !strings.EqualFold(glyph.Style, params.Style) {
			continue
		}
		if params.CopybookID != "" && !strings.EqualFold(glyph.CopybookID, params.CopybookID) {
			continue
		}
		matches = append(matches, glyph)
	}
	return matches
}

func seedGlyph(id, character, style, copybookID, calligrapher, licenseStatus, reviewStatus string) model.Glyph {
	return model.Glyph{
		GlyphID:       id,
		Character:     character,
		Style:         style,
		CopybookID:    copybookID,
		Calligrapher:  calligrapher,
		SourceImage:   "s3://nebula-calligraphy/seed/" + id + ".png",
		CropBox:       model.CropBox{X: 0, Y: 0, Width: 1, Height: 1, Unit: "ratio"},
		LicenseStatus: licenseStatus,
		ReviewStatus:  reviewStatus,
	}
}
