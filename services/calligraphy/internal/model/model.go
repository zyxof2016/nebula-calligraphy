package model

type CropBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"`
}

type Glyph struct {
	GlyphID       string  `json:"glyph_id"`
	Character     string  `json:"character"`
	Style         string  `json:"style"`
	CopybookID    string  `json:"copybook_id"`
	Calligrapher  string  `json:"calligrapher"`
	SourceImage   string  `json:"source_image"`
	CropBox       CropBox `json:"crop_box"`
	LicenseStatus string  `json:"license_status"`
	ReviewStatus  string  `json:"review_status"`
}

type PracticeTemplate struct {
	TemplateType string `json:"template_type"`
	GridType     string `json:"grid_type"`
	Title        string `json:"title"`
	Description  string `json:"description"`
}

type GlyphDetail struct {
	Glyph             Glyph              `json:"glyph"`
	StructureNotes    []string           `json:"structure_notes"`
	BrushworkNotes    []string           `json:"brushwork_notes"`
	PracticeTemplates []PracticeTemplate `json:"practice_templates"`
}

type GlyphPresetGroup struct {
	GroupID     string  `json:"group_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Glyphs      []Glyph `json:"glyphs"`
}

type GlyphSearchParams struct {
	Character  string
	Style      string
	CopybookID string
}

type PaperSpec struct {
	Format   string  `json:"format"`
	WidthCM  float64 `json:"width_cm"`
	HeightCM float64 `json:"height_cm"`
}

type SignatureSpec struct {
	Text string `json:"text"`
}

type LayoutRequest struct {
	Text       string        `json:"text"`
	Style      string        `json:"style"`
	CopybookID string        `json:"copybook_id"`
	Paper      PaperSpec     `json:"paper"`
	Direction  string        `json:"direction"`
	MarginCM   float64       `json:"margin_cm"`
	Signature  SignatureSpec `json:"signature"`
	SealCount  int           `json:"seal_count"`
}

type LayoutResult struct {
	LayoutID       string      `json:"layout_id"`
	NormalizedText string      `json:"normalized_text"`
	CharacterCount int         `json:"character_count"`
	Style          string      `json:"style"`
	CopybookID     string      `json:"copybook_id"`
	Paper          PaperSpec   `json:"paper"`
	Direction      string      `json:"direction"`
	MarginCM       float64     `json:"margin_cm"`
	Columns        int         `json:"columns"`
	Rows           int         `json:"rows"`
	GlyphSizeCM    float64     `json:"glyph_size_cm"`
	Slots          []GlyphSlot `json:"slots"`
	SignatureSlots []TextSlot  `json:"signature_slots,omitempty"`
	SealSlots      []TextSlot  `json:"seal_slots,omitempty"`
}

type GlyphSlot struct {
	Index     int     `json:"index"`
	Character string  `json:"character"`
	Column    int     `json:"column"`
	Row       int     `json:"row"`
	XCM       float64 `json:"x_cm"`
	YCM       float64 `json:"y_cm"`
	SizeCM    float64 `json:"size_cm"`
}

type TextSlot struct {
	Index  int     `json:"index"`
	Text   string  `json:"text"`
	XCM    float64 `json:"x_cm"`
	YCM    float64 `json:"y_cm"`
	SizeCM float64 `json:"size_cm"`
}

type CreateArtworkDraftRequest struct {
	OwnerUserID    string            `json:"owner_user_id"`
	Layout         LayoutRequest     `json:"layout"`
	GlyphOverrides map[string]string `json:"glyph_overrides,omitempty"`
}

type ArtworkDraft struct {
	ArtworkID      string            `json:"artwork_id"`
	OwnerUserID    string            `json:"owner_user_id"`
	Text           string            `json:"text"`
	Layout         LayoutResult      `json:"layout"`
	GlyphOverrides map[string]string `json:"glyph_overrides,omitempty"`
	Exports        []ExportRecord    `json:"exports,omitempty"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
}

type CreateExportRequest struct {
	Format       string `json:"format"`
	TemplateType string `json:"template_type"`
}

type ExportRecord struct {
	ExportID      string `json:"export_id"`
	ArtworkID     string `json:"artwork_id"`
	Format        string `json:"format"`
	TemplateType  string `json:"template_type"`
	ContentType   string `json:"content_type"`
	StorageKey    string `json:"storage_key,omitempty"`
	SHA256        string `json:"sha256"`
	ByteSize      int    `json:"byte_size"`
	InlineContent string `json:"inline_content,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type CreateFavoriteRequest struct {
	GlyphID string `json:"glyph_id"`
}

type FavoriteGlyph struct {
	OwnerUserID string `json:"owner_user_id"`
	GlyphID     string `json:"glyph_id"`
	Character   string `json:"character"`
	Style       string `json:"style"`
	CopybookID  string `json:"copybook_id"`
	CreatedAt   string `json:"created_at"`
}

type CreatePracticeRecordRequest struct {
	GlyphID      string `json:"glyph_id"`
	TemplateType string `json:"template_type"`
	GridType     string `json:"grid_type"`
}

type PracticeRecord struct {
	PracticeID   string `json:"practice_id"`
	OwnerUserID  string `json:"owner_user_id"`
	GlyphID      string `json:"glyph_id"`
	Character    string `json:"character"`
	Style        string `json:"style"`
	TemplateType string `json:"template_type"`
	GridType     string `json:"grid_type"`
	CreatedAt    string `json:"created_at"`
}

type LearningProfile struct {
	OwnerUserID     string           `json:"owner_user_id"`
	Favorites       []FavoriteGlyph  `json:"favorites"`
	RecentPractice  []PracticeRecord `json:"recent_practice"`
	PracticeCount   int              `json:"practice_count"`
	FavoriteCount   int              `json:"favorite_count"`
	LastPracticedAt string           `json:"last_practiced_at,omitempty"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type User struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

type AuthSession struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
