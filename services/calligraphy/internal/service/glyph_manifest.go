package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

const glyphManifestSchemaVersion = "calligraphy.glyph_manifest.v1"

type CopybookGlyphManifest struct {
	SchemaVersion string              `json:"schema_version"`
	Copybook      CopybookManifest    `json:"copybook"`
	Glyphs        []GlyphManifestItem `json:"glyphs"`
}

type CopybookManifest struct {
	CopybookID    string `json:"copybook_id"`
	Title         string `json:"title"`
	Style         string `json:"style"`
	Calligrapher  string `json:"calligrapher"`
	SourceURL     string `json:"source_url"`
	LicenseStatus string `json:"license_status"`
	Attribution   string `json:"attribution"`
}

type GlyphManifestItem struct {
	GlyphID        string        `json:"glyph_id"`
	Character      string        `json:"character"`
	Style          string        `json:"style,omitempty"`
	CopybookID     string        `json:"copybook_id,omitempty"`
	Calligrapher   string        `json:"calligrapher,omitempty"`
	SourceImage    string        `json:"source_image"`
	CropBox        model.CropBox `json:"crop_box"`
	LicenseStatus  string        `json:"license_status,omitempty"`
	ReviewStatus   string        `json:"review_status"`
	StructureNotes []string      `json:"structure_notes,omitempty"`
	BrushworkNotes []string      `json:"brushwork_notes,omitempty"`
}

type FileGlyphCatalog struct {
	glyphs           []model.Glyph
	details          map[string]model.GlyphDetail
	copybookID       string
	copybookTitle    string
	copybookStyle    string
	copybookDescribe string
}

func NewFileGlyphCatalog(path string) (*FileGlyphCatalog, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest CopybookGlyphManifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return nil, err
	}
	return NewFileGlyphCatalogFromManifest(manifest)
}

func NewFileGlyphCatalogFromManifest(manifest CopybookGlyphManifest) (*FileGlyphCatalog, error) {
	if err := validateGlyphManifest(manifest); err != nil {
		return nil, err
	}

	catalog := &FileGlyphCatalog{
		copybookID:       manifest.Copybook.CopybookID,
		copybookTitle:    manifest.Copybook.Title,
		copybookStyle:    manifest.Copybook.Style,
		copybookDescribe: manifest.Copybook.Attribution,
		details:          map[string]model.GlyphDetail{},
	}
	for _, item := range manifest.Glyphs {
		glyph := model.Glyph{
			GlyphID:       strings.TrimSpace(item.GlyphID),
			Character:     item.Character,
			Style:         firstNonEmpty(item.Style, manifest.Copybook.Style),
			CopybookID:    firstNonEmpty(item.CopybookID, manifest.Copybook.CopybookID),
			Calligrapher:  firstNonEmpty(item.Calligrapher, manifest.Copybook.Calligrapher),
			SourceImage:   strings.TrimSpace(item.SourceImage),
			CropBox:       normalizedCropBox(item.CropBox),
			LicenseStatus: firstNonEmpty(item.LicenseStatus, manifest.Copybook.LicenseStatus),
			ReviewStatus:  firstNonEmpty(item.ReviewStatus, "draft"),
		}
		catalog.glyphs = append(catalog.glyphs, glyph)
		catalog.details[glyph.GlyphID] = model.GlyphDetail{
			Glyph:             glyph,
			StructureNotes:    notesOrDefault(item.StructureNotes, "观察中宫、重心和主笔走向。"),
			BrushworkNotes:    notesOrDefault(item.BrushworkNotes, "以原碑笔意为准，先慢写再求连贯。"),
			PracticeTemplates: defaultPracticeTemplates(),
		}
	}
	return catalog, nil
}

func (c *FileGlyphCatalog) Search(params GlyphSearchParams) []model.Glyph {
	var matches []model.Glyph
	for _, glyph := range c.glyphs {
		if !glyphIsPublic(glyph) {
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

func (c *FileGlyphCatalog) GetDetail(glyphID string) (model.GlyphDetail, bool) {
	detail, ok := c.details[glyphID]
	if !ok || !glyphIsPublic(detail.Glyph) {
		return model.GlyphDetail{}, false
	}
	return detail, true
}

func (c *FileGlyphCatalog) ListPresetGroups(style string) []model.GlyphPresetGroup {
	if style == "" {
		style = c.copybookStyle
	}
	glyphs := c.Search(GlyphSearchParams{Style: style})
	if len(glyphs) == 0 {
		return nil
	}
	return []model.GlyphPresetGroup{{
		GroupID:     "copybook-" + c.copybookID,
		Title:       c.copybookTitle,
		Description: c.copybookDescribe,
		Glyphs:      glyphs,
	}}
}

type CompositeGlyphCatalog struct {
	catalogs []GlyphCatalog
}

func NewCompositeGlyphCatalog(catalogs ...GlyphCatalog) *CompositeGlyphCatalog {
	return &CompositeGlyphCatalog{catalogs: catalogs}
}

func (c *CompositeGlyphCatalog) Search(params GlyphSearchParams) []model.Glyph {
	var matches []model.Glyph
	seen := map[string]bool{}
	for _, catalog := range c.catalogs {
		if catalog == nil {
			continue
		}
		for _, glyph := range catalog.Search(params) {
			key := glyphIdentityKey(glyph)
			if seen[key] {
				continue
			}
			seen[key] = true
			matches = append(matches, glyph)
		}
	}
	return matches
}

func (c *CompositeGlyphCatalog) GetDetail(glyphID string) (model.GlyphDetail, bool) {
	for _, catalog := range c.catalogs {
		if catalog == nil {
			continue
		}
		if detail, ok := catalog.GetDetail(glyphID); ok {
			return detail, true
		}
	}
	return model.GlyphDetail{}, false
}

func (c *CompositeGlyphCatalog) ListPresetGroups(style string) []model.GlyphPresetGroup {
	if len(c.catalogs) == 0 {
		return nil
	}
	groups := c.catalogs[len(c.catalogs)-1].ListPresetGroups(style)
	for i := len(c.catalogs) - 2; i >= 0; i-- {
		if c.catalogs[i] == nil {
			continue
		}
		replacements := c.catalogs[i].Search(GlyphSearchParams{Style: style})
		if len(replacements) == 0 {
			continue
		}
		byCharacter := map[string]model.Glyph{}
		for _, glyph := range replacements {
			byCharacter[glyph.Character] = glyph
		}
		for groupIndex := range groups {
			for glyphIndex, glyph := range groups[groupIndex].Glyphs {
				if replacement, ok := byCharacter[glyph.Character]; ok {
					groups[groupIndex].Glyphs[glyphIndex] = replacement
				}
			}
		}
	}
	return groups
}

func validateGlyphManifest(manifest CopybookGlyphManifest) error {
	if manifest.SchemaVersion != glyphManifestSchemaVersion {
		return fmt.Errorf("unsupported glyph manifest schema_version %q", manifest.SchemaVersion)
	}
	copybook := manifest.Copybook
	required := map[string]string{
		"copybook.copybook_id":    copybook.CopybookID,
		"copybook.title":          copybook.Title,
		"copybook.style":          copybook.Style,
		"copybook.calligrapher":   copybook.Calligrapher,
		"copybook.source_url":     copybook.SourceURL,
		"copybook.license_status": copybook.LicenseStatus,
		"copybook.attribution":    copybook.Attribution,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required for traceability", field)
		}
	}
	if !validLicenseStatus(copybook.LicenseStatus) {
		return fmt.Errorf("copybook.license_status %q is invalid", copybook.LicenseStatus)
	}
	for index, item := range manifest.Glyphs {
		if err := validateGlyphManifestItem(index, item, copybook); err != nil {
			return err
		}
	}
	return nil
}

func validateGlyphManifestItem(index int, item GlyphManifestItem, copybook CopybookManifest) error {
	prefix := fmt.Sprintf("glyphs[%d]", index)
	if strings.TrimSpace(item.GlyphID) == "" {
		return errors.New(prefix + ".glyph_id is required")
	}
	if utf8.RuneCountInString(item.Character) != 1 {
		return errors.New(prefix + ".character must contain exactly one rune")
	}
	if strings.TrimSpace(item.SourceImage) == "" {
		return errors.New(prefix + ".source_image is required")
	}
	if item.CropBox.X < 0 || item.CropBox.Y < 0 || item.CropBox.Width <= 0 || item.CropBox.Height <= 0 {
		return errors.New(prefix + ".crop_box must use non-negative origin and positive size")
	}
	licenseStatus := firstNonEmpty(item.LicenseStatus, copybook.LicenseStatus)
	if !validLicenseStatus(licenseStatus) {
		return fmt.Errorf("%s.license_status %q is invalid", prefix, licenseStatus)
	}
	if !validReviewStatus(firstNonEmpty(item.ReviewStatus, "draft")) {
		return fmt.Errorf("%s.review_status %q is invalid", prefix, item.ReviewStatus)
	}
	return nil
}

func validLicenseStatus(status string) bool {
	switch status {
	case "public_domain", "licensed", "restricted", "pending_review":
		return true
	default:
		return false
	}
}

func validReviewStatus(status string) bool {
	switch status {
	case "draft", "expert_reviewed", "published", "rejected":
		return true
	default:
		return false
	}
}

func glyphIsPublic(glyph model.Glyph) bool {
	return glyph.LicenseStatus != "restricted" && glyph.ReviewStatus == "published"
}

func glyphIdentityKey(glyph model.Glyph) string {
	return strings.ToLower(glyph.Style) + "\x00" + glyph.Character + "\x00" + strings.ToLower(glyph.CopybookID)
}

func normalizedCropBox(crop model.CropBox) model.CropBox {
	if crop.Unit == "" {
		crop.Unit = "px"
	}
	return crop
}

func notesOrDefault(notes []string, fallback string) []string {
	if len(notes) > 0 {
		return notes
	}
	return []string{fallback}
}

func defaultPracticeTemplates() []model.PracticeTemplate {
	return []model.PracticeTemplate{
		{TemplateType: "copy", GridType: "mi", Title: "米字格临摹", Description: "适合观察重心和斜向笔势。"},
		{TemplateType: "copy", GridType: "jiugong", Title: "九宫格结构", Description: "适合拆解上下左右比例。"},
		{TemplateType: "outline", GridType: "mi", Title: "双钩练习", Description: "适合描摹外轮廓和收放关系。"},
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
