package service

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

const (
	defaultMarginCM  = 2.5
	defaultDirection = "vertical_rtl"
	defaultStyle     = "ou"
)

type LayoutEngine struct{}

func NewLayoutEngine() *LayoutEngine {
	return &LayoutEngine{}
}

func (e *LayoutEngine) Preview(req model.LayoutRequest) (model.LayoutResult, error) {
	text := normalizeCompositionText(req.Text)
	if text == "" {
		return model.LayoutResult{}, errors.New("layout text must contain at least one calligraphy character")
	}
	if req.Paper.WidthCM <= 0 || req.Paper.HeightCM <= 0 {
		return model.LayoutResult{}, errors.New("paper width_cm and height_cm must be positive")
	}

	margin := req.MarginCM
	if margin == 0 {
		margin = defaultMarginCM
	}
	if margin*2 >= req.Paper.WidthCM || margin*2 >= req.Paper.HeightCM {
		return model.LayoutResult{}, fmt.Errorf("margin_cm %.2f leaves no drawable area", margin)
	}

	direction := req.Direction
	if direction == "" {
		direction = defaultDirection
	}
	if direction != defaultDirection {
		return model.LayoutResult{}, fmt.Errorf("unsupported direction %q", direction)
	}

	style := req.Style
	if style == "" {
		style = defaultStyle
	}

	characters := splitRunes(text)
	signatureReserve := signatureReserveCM(req)
	availableWidth := req.Paper.WidthCM - 2*margin - signatureReserve
	availableHeight := req.Paper.HeightCM - 2*margin
	if availableWidth <= 0 || availableHeight <= 0 {
		return model.LayoutResult{}, errors.New("paper area is too small for content and signature")
	}

	columns, rows, cellWidth, cellHeight, glyphSize := chooseVerticalGrid(len(characters), availableWidth, availableHeight)
	slots := make([]model.GlyphSlot, 0, len(characters))
	for i, character := range characters {
		column := i / rows
		row := i % rows
		x := req.Paper.WidthCM - margin - signatureReserve - cellWidth*(float64(column)+0.5)
		y := margin + cellHeight*(float64(row)+0.5)
		slots = append(slots, model.GlyphSlot{
			Index:     i,
			Character: character,
			Column:    column,
			Row:       row,
			XCM:       round2(x),
			YCM:       round2(y),
			SizeCM:    round2(glyphSize),
		})
	}

	result := model.LayoutResult{
		LayoutID:       "preview",
		NormalizedText: text,
		CharacterCount: len(characters),
		Style:          style,
		CopybookID:     req.CopybookID,
		Paper:          req.Paper,
		Direction:      direction,
		MarginCM:       round2(margin),
		Columns:        columns,
		Rows:           rows,
		GlyphSizeCM:    round2(glyphSize),
		Slots:          slots,
	}
	result.SignatureSlots = buildSignatureSlots(req, margin, glyphSize)
	result.SealSlots = buildSealSlots(req, margin, glyphSize)
	return result, nil
}

func normalizeCompositionText(text string) string {
	var builder strings.Builder
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		switch r {
		case '，', '。', '、', '；', '：', '！', '？', '《', '》', '「', '」', '『', '』', '（', '）':
			continue
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func splitRunes(text string) []string {
	out := make([]string, 0, len(text))
	for _, r := range text {
		out = append(out, string(r))
	}
	return out
}

func chooseVerticalGrid(count int, width, height float64) (int, int, float64, float64, float64) {
	bestColumns := 1
	bestRows := count
	bestGlyphSize := 0.0
	bestCellWidth := width
	bestCellHeight := height / float64(count)
	bestScore := math.Inf(-1)

	for columns := 1; columns <= count; columns++ {
		rows := int(math.Ceil(float64(count) / float64(columns)))
		cellWidth := width / float64(columns)
		cellHeight := height / float64(rows)
		glyphSize := math.Min(cellWidth, cellHeight) * 0.75
		aspectPenalty := math.Abs(cellWidth-cellHeight) * 0.05
		emptyPenalty := float64(columns*rows-count) * 0.1
		score := glyphSize - aspectPenalty - emptyPenalty
		if score > bestScore {
			bestScore = score
			bestColumns = columns
			bestRows = rows
			bestGlyphSize = glyphSize
			bestCellWidth = cellWidth
			bestCellHeight = cellHeight
		}
	}

	return bestColumns, bestRows, bestCellWidth, bestCellHeight, bestGlyphSize
}

func signatureReserveCM(req model.LayoutRequest) float64 {
	if req.Signature.Text == "" && req.SealCount <= 0 {
		return 0
	}
	reserve := req.Paper.WidthCM * 0.12
	if reserve < 3 {
		return 3
	}
	if reserve > 8 {
		return 8
	}
	return reserve
}

func buildSignatureSlots(req model.LayoutRequest, margin, glyphSize float64) []model.TextSlot {
	if req.Signature.Text == "" {
		return nil
	}
	size := math.Min(glyphSize*0.55, 1.2)
	if size <= 0 {
		size = 1
	}
	slots := make([]model.TextSlot, 0, len([]rune(req.Signature.Text)))
	for i, r := range req.Signature.Text {
		slots = append(slots, model.TextSlot{
			Index:  i,
			Text:   string(r),
			XCM:    round2(margin),
			YCM:    round2(margin + float64(i)*size*1.35),
			SizeCM: round2(size),
		})
	}
	return slots
}

func buildSealSlots(req model.LayoutRequest, margin, glyphSize float64) []model.TextSlot {
	if req.SealCount <= 0 {
		return nil
	}
	size := math.Min(glyphSize*0.8, 1.8)
	if size <= 0 {
		size = 1.4
	}
	slots := make([]model.TextSlot, 0, req.SealCount)
	for i := 0; i < req.SealCount; i++ {
		slots = append(slots, model.TextSlot{
			Index:  i,
			Text:   "seal",
			XCM:    round2(margin),
			YCM:    round2(req.Paper.HeightCM - margin - float64(i+1)*size*1.15),
			SizeCM: round2(size),
		})
	}
	return slots
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
