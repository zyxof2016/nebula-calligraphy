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
