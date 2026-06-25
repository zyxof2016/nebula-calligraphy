package service

import (
	"testing"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

func TestLayoutPreviewUsesVerticalRightToLeftSlots(t *testing.T) {
	engine := NewLayoutEngine()

	result, err := engine.Preview(model.LayoutRequest{
		Text:      "山水清音",
		Style:     "ou",
		Direction: "vertical_rtl",
		Paper: model.PaperSpec{
			Format:   "doufang",
			WidthCM:  69,
			HeightCM: 68,
		},
		MarginCM: 3,
	})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	if result.CharacterCount != 4 {
		t.Fatalf("CharacterCount = %d, want 4", result.CharacterCount)
	}
	if len(result.Slots) != 4 {
		t.Fatalf("len(Slots) = %d, want 4", len(result.Slots))
	}
	if result.Slots[0].Character != "山" || result.Slots[1].Character != "水" {
		t.Fatalf("slot characters = %q, %q; want 山, 水", result.Slots[0].Character, result.Slots[1].Character)
	}
	if result.Slots[0].Column != 0 || result.Slots[0].Row != 0 {
		t.Fatalf("first slot grid = column %d row %d, want 0/0", result.Slots[0].Column, result.Slots[0].Row)
	}
	if !(result.Slots[0].XCM > result.Slots[2].XCM) {
		t.Fatalf("vertical_rtl x ordering invalid: first column x %.2f, second column x %.2f", result.Slots[0].XCM, result.Slots[2].XCM)
	}
	if result.GlyphSizeCM <= 0 {
		t.Fatalf("GlyphSizeCM = %.2f, want positive", result.GlyphSizeCM)
	}
}

func TestLayoutPreviewNormalizesPunctuationAndWhitespace(t *testing.T) {
	engine := NewLayoutEngine()

	result, err := engine.Preview(model.LayoutRequest{
		Text: "山，水。 清 音",
		Paper: model.PaperSpec{
			WidthCM:  34,
			HeightCM: 138,
		},
	})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	if result.NormalizedText != "山水清音" {
		t.Fatalf("NormalizedText = %q, want 山水清音", result.NormalizedText)
	}
}

func TestLayoutPreviewRejectsEmptyText(t *testing.T) {
	engine := NewLayoutEngine()

	_, err := engine.Preview(model.LayoutRequest{
		Text: " ，。 ",
		Paper: model.PaperSpec{
			WidthCM:  34,
			HeightCM: 138,
		},
	})
	if err == nil {
		t.Fatal("Preview() error = nil, want error")
	}
}

func TestLayoutPreviewRejectsInvalidMargin(t *testing.T) {
	engine := NewLayoutEngine()

	_, err := engine.Preview(model.LayoutRequest{
		Text:     "山水",
		MarginCM: 20,
		Paper: model.PaperSpec{
			WidthCM:  34,
			HeightCM: 20,
		},
	})
	if err == nil {
		t.Fatal("Preview() error = nil, want error")
	}
}
